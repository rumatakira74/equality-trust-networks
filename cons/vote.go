// Выборочное голосование (9.1): Snowball-подобный алгоритм — повторяющийся
// опрос случайной выборки пиров; веса голосов — по возрасту и аттестации
// кошельков. Голосование только выбирает ветку при конфликте; вердиктов
// о «вине» нет (7.5) — метка flagged только через evidence.
package cons

import (
	"crypto/sha256"
	"encoding/binary"
	"math/rand"

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

// Snowball — голосование по конфликтам (9.1).
type Snowball struct {
	K     int
	Alpha float64
	Beta  int
	// states — конфликт -> состояние; сериализуется детерминированно.
	states map[string]*VoteState
}

// NewSnowball — голосование с параметрами k/α/β.
func NewSnowball(p Params) *Snowball {
	return &Snowball{K: p.SnowballK, Alpha: p.SnowballAlpha, Beta: p.SnowballBeta,
		states: make(map[string]*VoteState)}
}

// State — состояние конфликта по ключу (nil — конфликт не известен).
func (s *Snowball) State(key string) *VoteState { return s.states[key] }

// States — все состояния (детерминированный порядок по key).
func (s *Snowball) States() []*VoteState {
	out := make([]*VoteState, 0, len(s.states))
	for _, k := range sortedKeys(s.states) {
		out = append(out, s.states[k])
	}
	return out
}

// Observe — узел увидел конфликт (две ветки одного wallet|seq):
// инициализирует состояние, предпочтение — ветка, применённая первой
// (FirstWins ядра; голосование может его сменить).
func (s *Snowball) Observe(key string, applied, alternative []byte) *VoteState {
	st, ok := s.states[key]
	if !ok {
		a, b := applied, alternative
		if string(b) < string(a) {
			a, b = b, a
		}
		st = &VoteState{Key: key, A: a, B: b, Pref: applied}
		s.states[key] = st
	}
	return st
}

// PollRound — один раунд опроса по конфликту: k случайных пиров, кворум
// α суммарного веса; β последовательных раундов — локальная финальность.
// Возвращает true, если конфликт финализирован в этом раунде.
func (s *Snowball) PollRound(key string, poller Poller, rng *rand.Rand) bool {
	st := s.states[key]
	if st == nil || st.Final {
		return false
	}
	st.Rounds++
	peers := poller.Sample(s.K, rng)
	total, forA, forB := uint64(0), uint64(0), uint64(0)
	for _, p := range peers {
		pref, weight, ok := p.VotePreference(key)
		if !ok || weight == 0 {
			continue
		}
		total += weight
		switch string(pref) {
		case string(st.A):
			forA += weight
		case string(st.B):
			forB += weight
		}
	}
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
