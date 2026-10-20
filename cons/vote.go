// Выборочное голосование (9.1): Snowball-подобный алгоритм — повторяющийся
// опрос случайной выборки пиров; веса голосов — по возрасту и аттестации
// кошельков. Голосование только выбирает ветку при конфликте; вердиктов
// о «вине» нет (7.5) — метка flagged только через evidence.
package cons

import (
	"crypto/sha256"
	"encoding/binary"
	"math/rand"
	"sync"

	"github.com/rumatakira74/equality-trust-networks/core/ledger"
	"github.com/rumatakira74/equality-trust-networks/core/wire"
)

// Poller — опрос пиров (реализация: net — потоки, sim — виртуальная сеть).
type Poller interface {
	// Sample — k случайных достижимых пиров (детерминированно из rng).
	Sample(k int, rng *rand.Rand) []PollablePeer
}

// PollablePeer — опрашиваемый пир.
type PollablePeer interface {
	// VotePreference — предпочтение пира по конфликту key
	// (txid ветки; пусто — пир не знает конфликта).
	VotePreference(key string) ([]byte, uint64, bool)
}

// Weight — вес голоса кошелька: возраст (до +2 за 2 года) и аттестация
// (+1: сеть с device_policy STRICT — все кошельки с TEE-аттестацией,
// R2 на register; посевной круг — genesis-ключи). Дифференциация по
// уровню аттестации отдельных устройств — этап 6 (нужны данные).
func Weight(w *ledger.Wallet, strictPolicy bool, now uint64) uint64 {
	weight := uint64(1)
	ageDays := w.AgeDays(now)
	if ageDays > 730 {
		weight += 2
	} else {
		weight += ageDays / 365
	}
	if strictPolicy || w.Seed {
		weight++
	}
	return weight
}

// VoteState — состояние одного конфликта (wallet|seq) на узле.
type VoteState struct {
	Key     string // wallet|seq
	A, B    []byte // txid веток (канонический порядок: A < B)
	Pref    []byte // текущее предпочтение
	Counter int    // последовательные раунды с Pref
	Final   bool   // локальная финальность (β раундов)
	Rounds  int    // всего раундов
}

// Snowball — голосование по конфликтам (9.1). Потокобезопасен сам по
// себе (race-аудит этапа 9, go test -race): доступ идёт из нескольких
// горутин — postApply (под applyMu), voteLoop, перестройка/финальность,
// тесты и мост; внешний лок-дисциплины «все под applyMu» недостаточно
// (finalityView читает states без applyMu). Внутренний лок; State/States
// возвращают КОПИИ — запись через возвращённый указатель невозможна.
type Snowball struct {
	K     int
	Alpha float64
	Beta  int
	mu    sync.RWMutex
	// states — конфликт -> состояние; сериализуется детерминированно.
	states map[string]*VoteState
}

// NewSnowball — голосование с параметрами k/α/β.
func NewSnowball(p Params) *Snowball {
	return &Snowball{K: p.SnowballK, Alpha: p.SnowballAlpha, Beta: p.SnowballBeta,
		states: make(map[string]*VoteState)}
}

// State — состояние конфликта по ключу (nil — конфликт не известен);
// возвращается копия.
func (s *Snowball) State(key string) *VoteState {
	s.mu.RLock()
	defer s.mu.RUnlock()
	st, ok := s.states[key]
	if !ok {
		return nil
	}
	cp := *st
	return &cp
}

// States — все состояния (детерминированный порядок по key); копии.
func (s *Snowball) States() []*VoteState {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*VoteState, 0, len(s.states))
	for _, k := range sortedKeys(s.states) {
		cp := *s.states[k]
		out = append(out, &cp)
	}
	return out
}

// Observe — узел увидел конфликт (две ветки одного wallet|seq):
// инициализирует состояние, предпочтение — ветка, применённая первой
// (FirstWins ядра; голосование может его сменить).
func (s *Snowball) Observe(key string, applied, alternative []byte) *VoteState {
	s.mu.Lock()
	defer s.mu.Unlock()
	st, ok := s.states[key]
	if !ok {
		a, b := applied, alternative
		if string(b) < string(a) {
			a, b = b, a
		}
		st = &VoteState{Key: key, A: a, B: b, Pref: applied}
		s.states[key] = st
	}
	cp := *st
	return &cp
}

// PollRound — один раунд опроса по конфликту: k случайных пиров, кворум
// α суммарного веса; β последовательных раундов — локальная финальность.
// Возвращает true, если конфликт финализирован в этом раунде.
func (s *Snowball) PollRound(key string, poller Poller, rng *rand.Rand) bool {
	// снапшот под локом; опрос пиров (сеть) — ВНЕ лока: медленный раунд
	// не блокирует Observe из горутины apply (иначе applyMu + сеть =
	// задержка применения транзакций); результат — под локом
	s.mu.Lock()
	st := s.states[key]
	if st == nil || st.Final {
		s.mu.Unlock()
		return false
	}
	st.Rounds++
	snap := *st
	s.mu.Unlock()
	peers := poller.Sample(s.K, rng)
	total, forA, forB := uint64(0), uint64(0), uint64(0)
	for _, p := range peers {
		pref, weight, ok := p.VotePreference(key)
		if !ok || weight == 0 {
			continue
		}
		total += weight
		switch string(pref) {
		case string(snap.A):
			forA += weight
		case string(snap.B):
			forB += weight
		}
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if total == 0 {
		return false
	}
	quorum := s.Alpha * float64(total)
	var winner []byte
	if float64(forA) >= quorum && forA >= forB {
		winner = st.A
	} else if float64(forB) >= quorum {
		winner = st.B
	}
	if winner == nil {
		// кворума нет — счётчик сбрасывается (Snowball)
		st.Counter = 0
		return false
	}
	if string(winner) == string(st.Pref) {
		st.Counter++
	} else {
		st.Pref = winner
		st.Counter = 1
	}
	if st.Counter >= s.Beta {
		st.Final = true
		return true
	}
	return false
}

// Finalized — локально финализированные конфликты (ветка-победитель).
func (s *Snowball) Finalized() map[string][]byte {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make(map[string][]byte)
	for k, st := range s.states {
		if st.Final {
			out[k] = st.Pref
		}
	}
	return out
}

// ConflictKey — ключ конфликта по транзакции.
func ConflictKey(tx *wire.Transaction) string {
	var b [8]byte
	binary.BigEndian.PutUint64(b[:], tx.Seq)
	return string(tx.Wallet) + "|" + string(b[:])
}

func sortedKeys(m map[string]*VoteState) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	// детерминированная сортировка
	for i := 1; i < len(keys); i++ {
		for j := i; j > 0 && keys[j] < keys[j-1]; j-- {
			keys[j], keys[j-1] = keys[j-1], keys[j]
		}
	}
	return keys
}

var _ = sha256.Sum256
