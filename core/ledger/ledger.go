// Ledger — ядро реестра: применение транзакций, сеттлемент, replay (INV-3).
package ledger

import (
	"bytes"
	"crypto/sha256"
	"crypto/x509"
	"errors"
	"fmt"

	"github.com/rumatakira74/equality-trust-networks/core/params"
	"github.com/rumatakira74/equality-trust-networks/core/wire"
)

// Options — режимы ядра.
type Options struct {
	// SkipMintChallengeWindow отключает проверку окна слота челленджа M7.
	// Отклонение D-1 (отчёт этапа 3): векторы этапа 0 не моделируют слот;
	// в продакшене — false (включена).
	SkipMintChallengeWindow bool
}

// Ledger — headless-ядро реестра (сеть — этап 4, финализация — этап 5).
type Ledger struct {
	State        *State
	Clock        EpochSource
	Finalizer    Finalizer
	TrustedRoots []*x509.Certificate
	TxIndex      map[string]*wire.Transaction // txid -> tx (принятые)
	Conflicts    []TxRef                      // конфликтные ветки (ссылки)
	// journal — принятые транзакции в каузальном порядке принятия
	// (порядок финализации; INV-3). Долг этапа 4: порядок вставки, не map.
	journal []*wire.Transaction
	// conflictTxs — конфликтные ветки целиком (spec/04 INV-1c, правка
	// этапа 5): остаются в журнале хранения как альтернативы; выбор
	// фиксируется в finalized_txid_list чекпоинта (cons, этап 5).
	conflictTxs []*wire.Transaction
	Opts        Options
	SeedSunset  *wire.SeedSunset // из манифеста (для settleEpoch)
}

// New создаёт ядро из genesis-пакета (проверка genesis — spec/02 2.6).
func New(gp *wire.GenesisPackage, clock EpochSource, opts Options) (*Ledger, error) {
	if gp == nil || gp.Manifest == nil || gp.Genesis == nil || gp.Manifest.Profile == nil {
		return nil, errors.New("ledger: неполный genesis-пакет")
	}
	m := gp.Manifest
	// 2.6 (1): manifest_hash
	mh := m.ManifestHash()
	if !bytes.Equal(mh[:], gp.Genesis.ManifestHash) {
		return nil, errors.New("ledger: manifest_hash не сходится")
	}
	// 2.6 (2): network_id
	if !bytes.Equal(m.NetworkID, gp.Genesis.NetworkID) {
		return nil, errors.New("ledger: network_id манифеста и genesis не совпадают")
	}
	// 2.6 (4): непротиворечивость профиля
	p := m.Profile
	if len(p.SlotTiers) == 0 {
		return nil, errors.New("ledger: slot_tiers пусты")
	}
	if p.RateMin > p.RateMax {
		return nil, errors.New("ledger: rate_min > rate_max")
	}
	// 2.6 (5): mutable_bounds не содержат защитных параметров
	for _, b := range p.MutableBounds {
		switch b.Param {
		case "vouch_k", "transfer_delay_hours", "transfer_mode", "issuance_mode",
			"issuance_amount", "device_policy", "signing_auth", "checkpoint_mode",
			"params_mutable":
			return nil, fmt.Errorf("ledger: защитный параметр %s в mutable_bounds", b.Param)
		}
	}
	// 2.6 (6): песочница => params_mutable = false (13)
	if p.Sandbox && p.ParamsMutable {
		return nil, errors.New("ledger: sandbox с params_mutable = true")
	}

	net := params.FromManifest(m, gp.Genesis.ProtocolVersion)
	st := NewState(net)
	st.Epoch = gp.Genesis.EpochZero
	st.CurrentPeriod = 0
	st.GenesisTime = gp.Genesis.GenesisTime
	// стартовая ставка комиссии = rate_min (10.2); дальше — адаптивный
	// пересчёт на границах эпох (settleEpoch)
	st.FeeRate = p.RateMin
	// Посевной круг: active из genesis-чекпоинта эпохи 0 (решение D8)
	st.SeedMembers(m.SeedCircle, gp.Genesis.EpochZero, gp.Genesis.GenesisTime)

	l := &Ledger{
		State:      st,
		Clock:      clock,
		Finalizer:  FirstWins{},
		TxIndex:    make(map[string]*wire.Transaction),
		Opts:       opts,
		SeedSunset: m.SeedSunset,
	}
	return l, nil
}

// hashThree — вспомогательный хеш для слота челленджа.
func hashThree(addr []byte, period uint64) uint64 {
	var pb [8]byte
	pb[0] = byte(period >> 56)
	pb[1] = byte(period >> 48)
	pb[2] = byte(period >> 40)
	pb[3] = byte(period >> 32)
	pb[4] = byte(period >> 24)
	pb[5] = byte(period >> 16)
	pb[6] = byte(period >> 8)
	pb[7] = byte(period)
	h := sha256.Sum256(append(append([]byte{}, addr...), pb[:]...))
	var out uint64
	for i := 0; i < 8; i++ {
		out = out<<8 | uint64(h[i])
	}
	return out
}

// SettleNow — доведение сеттлемента до текущего времени источника без
// транзакции-триггера (PENDING №2 отчёта 11.2): replay журнала пинирует
// часы на timestamp_claim, поэтому после рестарта зрелые отложенные
// операции (переводы D, миграции, смены гардианов, порог P) не применяются
// до ближайшей живой транзакции. Вызывается сетевым узлом на старте и
// периодически; повторный вызов после применения — без эффекта
// (Executed/Effective флаги).
func (l *Ledger) SettleNow() { l.syncTime() }

// syncTime — детерминированный сеттлемент к текущему времени источника.
func (l *Ledger) syncTime() {
	now := l.Clock.NetTime()
	epoch := l.Clock.Epoch()
	period := l.Clock.Period()

	for l.State.Epoch < epoch {
		l.State.Epoch++
		l.settleEpoch(l.State.Epoch, now)
	}
	if period > l.State.CurrentPeriod {
		for pp := l.State.CurrentPeriod + 1; pp <= period; pp++ {
			l.settlePeriod(pp, now)
		}
		l.State.CurrentPeriod = period
	}
	if now > l.State.NetTime {
		l.State.NetTime = now
	}
	l.flushPending(now)
}

// AcceptedTxs возвращает все принятые транзакции в каузальном порядке
// принятия (порядок финализации; для replay и INV-3).
func (l *Ledger) AcceptedTxs() []*wire.Transaction {
	out := make([]*wire.Transaction, len(l.journal))
	copy(out, l.journal)
	return out
}

// ConflictBranches — конфликтные ветки целиком в порядке обнаружения
// (spec/04 INV-1c, этап 5): журнал хранения включает обе ветки.
func (l *Ledger) ConflictBranches() []*wire.Transaction {
	out := make([]*wire.Transaction, len(l.conflictTxs))
	copy(out, l.conflictTxs)
	return out
}

// RecordAccepted — фиксация принятой транзакции в каузальный журнал
// (вызывается Apply; отдельные вызовы — для загрузки журнала).
func (l *Ledger) RecordAccepted(tx *wire.Transaction) {
	l.journal = append(l.journal, tx)
}

// RecordConflictTx — фиксация конфликтной ветки (полная транзакция).
func (l *Ledger) RecordConflictTx(tx *wire.Transaction) {
	l.conflictTxs = append(l.conflictTxs, tx)
}
