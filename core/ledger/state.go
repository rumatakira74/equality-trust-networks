// Состояние реестра: кошельки, слоты, pending-операции, счётчики INV-8.
package ledger

import (
	"crypto/ecdsa"
	"strings"

	"github.com/rumatakira74/equality-trust-networks/core/crypto"
	"github.com/rumatakira74/equality-trust-networks/core/params"
	"github.com/rumatakira74/equality-trust-networks/core/wire"
)

// Cohort — выпуск периода Period на сумму Amount (окно переводимости, T8).
type Cohort struct {
	Period uint64
	Amount uint64
}

// Slot — занятый слот поручителя (7.4).
type Slot struct {
	Candidate    []byte
	SinceNetTime uint64
	// MintsBase — last_claimed_period кандидата на момент активации кандидата;
	// слот освобождается, когда кандидат завершит slot_free_mint_periods
	// периодов выпуска (7.4).
	MintsBase       uint64
	CandidateActive bool
}

// VouchReceipt — поручительство, полученное кандидатом.
type VouchReceipt struct {
	Voucher []byte
	Epoch   uint64
}

// Wallet — состояние кошелька.
type Wallet struct {
	Address                   []byte
	Fingerprint               []byte
	PublicKey                 *ecdsa.PublicKey
	Status                    wire.WalletStatus
	Flagged                   bool
	Synthetic                 bool
	Seed                      bool
	RegisteredAt              uint64 // net_time регистрации
	RegisteredEpoch           uint64
	ActivatedAt               uint64
	ActivatedEpoch            uint64
	ActivatedPeriod           uint64 // период активации (M4, R-доля)
	Balance                   uint64
	Liquid                    uint64 // переводимая часть (полученная + созревшие когорты)
	Cohorts                   []Cohort
	Locked                    uint64 // pending-резерв отложенных переводов
	LastTouchEpoch            uint64 // якорь демереджа (№31: от последней операции)
	Seq                       uint64
	TipTxID                   []byte
	LastClaimedPeriod         uint64
	DelayedThreshold          uint64 // P
	PendingThreshold          uint64 // незавершённое повышение (0 — нет)
	PendingThresholdNotBefore uint64
	OccupiedSlots             []Slot
	VouchesReceived           []VouchReceipt
	Activators                [][]byte
	PersonalGuardians         [][]byte
	PublicGuardian            bool
	Wards                     uint64
	MintSuspendedUntilPeriod  uint64
	FlaggedEpoch              uint64
	Terminal                  bool // старый адрес после исполнения миграции
	AndroidID                 string
}

// PendingTransfer — отложенный перевод (8.3).
type PendingTransfer struct {
	Seq       uint64
	Recipient []byte
	Amount    uint64
	Fee       uint64
	CreatedAt uint64
	Deadline  uint64
	Canceled  bool
	Executed  bool
}

// PendingMigration — предложенная миграция (11).
type PendingMigration struct {
	NewAddress       []byte
	OldAddress       []byte
	ProposeSeq       uint64
	CreatedAt        uint64
	ExecuteNotBefore uint64
	Nonce            uint64
	Approvals        []wire.GuardianApproval
	Canceled         bool
	Executed         bool
}

// State — полное состояние реестра (пересчитывается каждым узлом, INV-3).
type State struct {
	Net                  *params.Network
	Wallets              map[string]*Wallet
	FingerprintIndex     map[string][]byte
	FingerprintBlacklist map[string]bool
	PendingTransfers     map[string]*PendingTransfer // key: addr|seq
	PendingMigrations    []*PendingMigration
	EvidenceFacts        map[string]bool // key: addr|seq
	// GuardianLastReward — время последней награды гардиану от подопечного
	// (ключ guardian|ward; кулдаун №29: не чаще раза в 12 мес. на подопечного).
	GuardianLastReward map[string]uint64
	TotalMinted        uint64
	TotalBurned        uint64
	Epoch              uint64
	NetTime            uint64
	CurrentPeriod      uint64
	// GenesisTime — время genesis-записи: сетевые периоды и слот челленджа
	// выпуска (7.6) отсчитываются от него (9.2), не от 1970 г.
	GenesisTime      uint64
	SeedRightsActive bool // посевной круг: особые права (до seed_sunset)
	AndroidIDSeen    map[string]int
	// FeeRate — текущая ставка комиссии в micros (10.2, этап 10):
	// инициализируется значением rate_min, пересчитывается на границе
	// эпохи (settleEpoch) детерминированно из истории предложения.
	FeeRate uint64
	// SupplySnapshots — предложение на границах эпох (total_minted, как
	// в чекпоинте total_supply) для адаптивного пересчёта ставки (10.2).
	SupplySnapshots []SupplySnapshot
	// CommonFund — общий фонд сети (fee_mode = POOL, раздел 4).
	CommonFund uint64
}

// SupplySnapshot — предложение на границе эпохи (данные чекпоинта:
// total_supply = total_minted на момент сборки чекпоинта эпохи).
type SupplySnapshot struct {
	Epoch   uint64
	Supply  uint64
}

func key(b []byte) string { return string(b) }

func ptKey(addr []byte, seq uint64) string {
	return string(addr) + "|" + u64str(seq)
}

func u64str(v uint64) string {
	return string([]byte{byte(v >> 56), byte(v >> 48), byte(v >> 40), byte(v >> 32),
		byte(v >> 24), byte(v >> 16), byte(v >> 8), byte(v)})
}

// NewState создаёт состояние из genesis.
func NewState(net *params.Network) *State {
	return &State{
		Net:                  net,
		Wallets:              make(map[string]*Wallet),
		FingerprintIndex:     make(map[string][]byte),
		FingerprintBlacklist: make(map[string]bool),
		PendingTransfers:     make(map[string]*PendingTransfer),
		EvidenceFacts:        make(map[string]bool),
		GuardianLastReward:   make(map[string]uint64),
		SeedRightsActive:     true,
		AndroidIDSeen:        make(map[string]int),
	}
}

func (s *State) Wallet(addr []byte) *Wallet { return s.Wallets[key(addr)] }

// WalletByKey — кошелёк по ключу карты (ключ = адрес).
func (s *State) WalletByKey(k string) *Wallet {
	// ключ pending-перевода: addr|seq — отрезаем суффикс после '|'
	for i := len(k) - 1; i >= 0; i-- {
		if k[i] == '|' {
			return s.Wallets[k[:i]]
		}
	}
	return s.Wallets[k]
}

// SeedMembers инициализирует посевной круг (решение D8: active из genesis).
// Публичные ключи и индекс фингерпринтов сохраняются из манифеста (долг
// этапа 4: иначе первая транзакция посевного кошелька не проходит C8).
func (s *State) SeedMembers(members []wire.SeedMember, epoch, netTime uint64) {
	for _, m := range members {
		w := &Wallet{
			Address:          m.Address,
			Status:           wire.StatusActive,
			Seed:             true,
			RegisteredAt:     netTime,
			RegisteredEpoch:  epoch,
			ActivatedAt:      netTime,
			ActivatedEpoch:   epoch,
			LastTouchEpoch:   epoch,
			DelayedThreshold: s.Net.Profile.DelayedThresholdDefault,
		}
		if len(m.PublicKey) > 0 {
			if pub, err := crypto.ParsePubKey(m.PublicKey); err == nil {
				w.PublicKey = pub
				fp := crypto.Fingerprint(m.PublicKey)
				w.Fingerprint = fp
				s.FingerprintIndex[string(fp)] = m.Address
			}
		}
		s.Wallets[key(m.Address)] = w
	}
}

// RefreshCohorts переносит созревшие когорты в Liquid (T8: выпуск периода t
// переводим с t + fresh_transfer_window_periods).
func (w *Wallet) RefreshCohorts(currentPeriod, window uint64) {
	if window == 0 {
		w.Liquid += w.sumCohorts()
		w.Cohorts = nil
		return
	}
	keep := w.Cohorts[:0]
	for _, c := range w.Cohorts {
		if c.Period+window <= currentPeriod {
			w.Liquid += c.Amount
		} else {
			keep = append(keep, c)
		}
	}
	w.Cohorts = keep
}

func (w *Wallet) sumCohorts() uint64 {
	var sum uint64
	for _, c := range w.Cohorts {
		sum += c.Amount
	}
	return sum
}

// Spendable — переводимый остаток: Liquid + зрелые когорты − Locked.
func (w *Wallet) Spendable(currentPeriod, window uint64) uint64 {
	w.RefreshCohorts(currentPeriod, window)
	if w.Liquid > w.Locked {
		return w.Liquid - w.Locked
	}
	return 0
}

// TotalBalance — весь остаток (включая незрелые когорты и резерв).
func (w *Wallet) TotalBalance() uint64 {
	return w.Liquid + w.sumCohorts()
}

// Spend списывает amount из переводимых средств: сначала Liquid, затем
// старейшие зрелые когорты (T8). Списания обязаны быть ≤ Spendable.
func (w *Wallet) Spend(amount uint64, currentPeriod, window uint64) {
	w.RefreshCohorts(currentPeriod, window)
	if amount <= w.Liquid {
		w.Liquid -= amount
		return
	}
	amount -= w.Liquid
	w.Liquid = 0
	for i := 0; i < len(w.Cohorts) && amount > 0; i++ {
		if w.Cohorts[i].Period+window > currentPeriod {
			continue // незрелая когорта не тратится
		}
		if w.Cohorts[i].Amount <= amount {
			amount -= w.Cohorts[i].Amount
			w.Cohorts[i].Amount = 0
		} else {
			w.Cohorts[i].Amount -= amount
			amount = 0
		}
	}
	// убрать пустые когорты
	keep := w.Cohorts[:0]
	for _, c := range w.Cohorts {
		if c.Amount > 0 {
			keep = append(keep, c)
		}
	}
	w.Cohorts = keep
}

// AgeDays — возраст кошелька в днях (от register; 6.5).
func (w *Wallet) AgeDays(now uint64) uint64 {
	if now <= w.RegisteredAt {
		return 0
	}
	return (now - w.RegisteredAt) / 86400
}

// CompletedMintPeriods — выполненные периоды выпуска (V3).
func (w *Wallet) CompletedMintPeriods() uint64 {
	if w.LastClaimedPeriod == 0 || w.LastClaimedPeriod < w.ActivatedPeriod {
		return 0
	}
	return w.LastClaimedPeriod - w.ActivatedPeriod + 1
}

// FreeSlots — число свободных слотов.
func (w *Wallet) FreeSlots() int { return len(w.OccupiedSlots) }

var _ = strings.TrimSpace
