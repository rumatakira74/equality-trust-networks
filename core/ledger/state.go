// Состояние реестра: кошельки, слоты, pending-операции, счётчики INV-8.
package ledger

import (
	"crypto/ecdsa"

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
	// GuardianChanges — число вступивших в силу смен гардианов за всё
	// время (счётчик из истории цепочки; лимит 150, 11).
	GuardianChanges          uint64
	MintSuspendedUntilPeriod uint64
	FlaggedEpoch             uint64
	Terminal                 bool // старый адрес после исполнения миграции
	AndroidID                string
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

// PendingGuardianUpdate — опубликованная смена личных гардианов
// (11, решение владельца 21.10.2026): вступает в силу через 48 ч,
// до истечения действует старый список.
type PendingGuardianUpdate struct {
	Wallet       []byte
	ProposeSeq   uint64
	NewGuardians [][]byte
	Approvals    []wire.GuardianApproval
	Nonce        uint64
	CreatedAt    uint64
	EffectiveAt  uint64
	Canceled     bool
	Effective    bool // вступила в силу (список заменён)
	Expired      bool // не вступила: лимит №7 у гардиана превышен
}

// State — полное состояние реестра (пересчитывается каждым узлом, INV-3).
type State struct {
	Net                    *params.Network
	Wallets                map[string]*Wallet
	FingerprintIndex       map[string][]byte
	FingerprintBlacklist   map[string]bool
	PendingTransfers       map[string]*PendingTransfer // key: addr|seq
	PendingMigrations      []*PendingMigration
	PendingGuardianUpdates []*PendingGuardianUpdate
	EvidenceFacts          map[string]bool // key: addr|seq
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
	// Модули (раздел 6, этап 10): пулы, заявки вклада, погашения,
	// квитанции, счётчики подтверждений и потолков on_contribution.
	Pools                     map[string]*Pool
	Claims                    map[string]*ContributionClaim
	Redeems                   map[string]*RedeemInfo
	Receipts                  map[string]bool
	ConfirmsPerPeriod         map[string]uint64 // подтверждающий|период → count
	ContributionPaidPerPeriod map[uint64]uint64 // период → эмитировано (on_contribution)
	ContributionPaidPerson    map[string]uint64 // заявитель|период → выплачено
}

// SupplySnapshot — предложение на границе эпохи (данные чекпоинта:
// total_supply = total_minted на момент сборки чекпоинта эпохи).
type SupplySnapshot struct {
	Epoch  uint64
	Supply uint64
}

func key(b []byte) string { return string(b) }

// cloneWallet — полная копия кошелька (срезы и карты — новые; ключ
// ECDSA общий: иммутабелен). Для отката сеттлемента отклонённой tx.
func cloneWallet(w *Wallet) *Wallet {
	c := *w
	c.Address = append([]byte(nil), w.Address...)
	c.Fingerprint = append([]byte(nil), w.Fingerprint...)
	c.Cohorts = append([]Cohort(nil), w.Cohorts...)
	c.OccupiedSlots = make([]Slot, len(w.OccupiedSlots))
	for i, s := range w.OccupiedSlots {
		c.OccupiedSlots[i] = Slot{
			Candidate:    append([]byte(nil), s.Candidate...),
			SinceNetTime: s.SinceNetTime, MintsBase: s.MintsBase,
			CandidateActive: s.CandidateActive,
		}
	}
	c.VouchesReceived = make([]VouchReceipt, len(w.VouchesReceived))
	for i, r := range w.VouchesReceived {
		c.VouchesReceived[i] = VouchReceipt{Voucher: append([]byte(nil), r.Voucher...), Epoch: r.Epoch}
	}
	c.Activators = make([][]byte, len(w.Activators))
	for i, a := range w.Activators {
		c.Activators[i] = append([]byte(nil), a...)
	}
	c.PersonalGuardians = make([][]byte, len(w.PersonalGuardians))
	for i, g := range w.PersonalGuardians {
		c.PersonalGuardians[i] = append([]byte(nil), g...)
	}
	c.TipTxID = append([]byte(nil), w.TipTxID...)
	return &c
}

// Clone — полная копия состояния (v9.13): сеттлемент к txNow в Apply
// подтверждается только принятием tx, отклонённая tx откатывает
// состояние снапшотом. Копируются все изменяемые сеттлементом данные
// (кошельки, pending-операции, пулы, счётчики); params.Network и
// публичные ключи — общие (иммутабельны).
func (s *State) Clone() *State {
	c := *s
	c.Wallets = make(map[string]*Wallet, len(s.Wallets))
	for k, w := range s.Wallets {
		c.Wallets[k] = cloneWallet(w)
	}
	c.FingerprintIndex = make(map[string][]byte, len(s.FingerprintIndex))
	for k, v := range s.FingerprintIndex {
		c.FingerprintIndex[k] = append([]byte(nil), v...)
	}
	c.FingerprintBlacklist = make(map[string]bool, len(s.FingerprintBlacklist))
	for k, v := range s.FingerprintBlacklist {
		c.FingerprintBlacklist[k] = v
	}
	c.PendingTransfers = make(map[string]*PendingTransfer, len(s.PendingTransfers))
	for k, v := range s.PendingTransfers {
		pv := *v
		pv.Recipient = append([]byte(nil), v.Recipient...)
		c.PendingTransfers[k] = &pv
	}
	c.PendingMigrations = make([]*PendingMigration, len(s.PendingMigrations))
	for i, v := range s.PendingMigrations {
		pm := *v
		pm.NewAddress = append([]byte(nil), v.NewAddress...)
		pm.OldAddress = append([]byte(nil), v.OldAddress...)
		pm.Approvals = append([]wire.GuardianApproval(nil), v.Approvals...)
		c.PendingMigrations[i] = &pm
	}
	c.PendingGuardianUpdates = make([]*PendingGuardianUpdate, len(s.PendingGuardianUpdates))
	for i, v := range s.PendingGuardianUpdates {
		pu := *v
		pu.Wallet = append([]byte(nil), v.Wallet...)
		pu.NewGuardians = make([][]byte, len(v.NewGuardians))
		for j, g := range v.NewGuardians {
			pu.NewGuardians[j] = append([]byte(nil), g...)
		}
		pu.Approvals = append([]wire.GuardianApproval(nil), v.Approvals...)
		c.PendingGuardianUpdates[i] = &pu
	}
	c.EvidenceFacts = make(map[string]bool, len(s.EvidenceFacts))
	for k, v := range s.EvidenceFacts {
		c.EvidenceFacts[k] = v
	}
	c.GuardianLastReward = make(map[string]uint64, len(s.GuardianLastReward))
	for k, v := range s.GuardianLastReward {
		c.GuardianLastReward[k] = v
	}
	c.AndroidIDSeen = make(map[string]int, len(s.AndroidIDSeen))
	for k, v := range s.AndroidIDSeen {
		c.AndroidIDSeen[k] = v
	}
	c.SupplySnapshots = append([]SupplySnapshot(nil), s.SupplySnapshots...)
	c.Pools = make(map[string]*Pool, len(s.Pools))
	for k, v := range s.Pools {
		p := *v
		p.ID = append([]byte(nil), v.ID...)
		p.Creator = append([]byte(nil), v.Creator...)
		p.Recipient = append([]byte(nil), v.Recipient...)
		p.Donations = make(map[string]uint64, len(v.Donations))
		for dk, dv := range v.Donations {
			p.Donations[dk] = dv
		}
		p.Claimed = make(map[string]bool, len(v.Claimed))
		for ck, cv := range v.Claimed {
			p.Claimed[ck] = cv
		}
		c.Pools[k] = &p
	}
	c.Claims = make(map[string]*ContributionClaim, len(s.Claims))
	for k, v := range s.Claims {
		cl := *v
		cl.Claimant = append([]byte(nil), v.Claimant...)
		cl.PoolID = append([]byte(nil), v.PoolID...)
		cl.DescriptionHash = append([]byte(nil), v.DescriptionHash...)
		cl.Confirmers = make([][]byte, len(v.Confirmers))
		for j, cf := range v.Confirmers {
			cl.Confirmers[j] = append([]byte(nil), cf...)
		}
		c.Claims[k] = &cl
	}
	c.Redeems = make(map[string]*RedeemInfo, len(s.Redeems))
	for k, v := range s.Redeems {
		r := *v
		r.Provider = append([]byte(nil), v.Provider...)
		if v.Slot != nil {
			sl := *v.Slot
			r.Slot = &sl
		}
		c.Redeems[k] = &r
	}
	c.Receipts = make(map[string]bool, len(s.Receipts))
	for k, v := range s.Receipts {
		c.Receipts[k] = v
	}
	c.ConfirmsPerPeriod = make(map[string]uint64, len(s.ConfirmsPerPeriod))
	for k, v := range s.ConfirmsPerPeriod {
		c.ConfirmsPerPeriod[k] = v
	}
	c.ContributionPaidPerPeriod = make(map[uint64]uint64, len(s.ContributionPaidPerPeriod))
	for k, v := range s.ContributionPaidPerPeriod {
		c.ContributionPaidPerPeriod[k] = v
	}
	c.ContributionPaidPerson = make(map[string]uint64, len(s.ContributionPaidPerson))
	for k, v := range s.ContributionPaidPerson {
		c.ContributionPaidPerson[k] = v
	}
	return &c
}

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
		Net:                       net,
		Wallets:                   make(map[string]*Wallet),
		FingerprintIndex:          make(map[string][]byte),
		FingerprintBlacklist:      make(map[string]bool),
		PendingTransfers:          make(map[string]*PendingTransfer),
		EvidenceFacts:             make(map[string]bool),
		GuardianLastReward:        make(map[string]uint64),
		SeedRightsActive:          true,
		AndroidIDSeen:             make(map[string]int),
		Pools:                     make(map[string]*Pool),
		Claims:                    make(map[string]*ContributionClaim),
		Redeems:                   make(map[string]*RedeemInfo),
		Receipts:                  make(map[string]bool),
		ConfirmsPerPeriod:         make(map[string]uint64),
		ContributionPaidPerPeriod: make(map[uint64]uint64),
		ContributionPaidPerson:    make(map[string]uint64),
	}
}

func (s *State) Wallet(addr []byte) *Wallet { return s.Wallets[key(addr)] }

// WalletByKey — кошелёк отправителя по ключу pending-перевода.
// Ключ = addr (32 байта) ‖ '|' ‖ 8 BE-байт seq (ptKey): адрес выбирается
// по фиксированному смещению, не поиском разделителя — байт seq может
// равняться '|' (0x7C, напр. seq = 124), и скан с конца отрезал бы
// неверный адрес: перевод не исполнялся бы никогда, средства застревали
// бы в Locked (аудит z-ai AUDIT-Z-03). Иные длины (чужие ключи) — как
// раньше, прямым ключом.
func (s *State) WalletByKey(k string) *Wallet {
	if len(k) == 41 {
		return s.Wallets[k[:32]]
	}
	return s.Wallets[k]
}

// SeedMembers инициализирует посевной круг (решение D8: active из genesis).
// Манифест хранит публичные ключи (fingerprint-манифест v9.11) — адреса
// выводятся из финального network_id: SHA-256(fingerprint ‖ network_id).
// Публичные ключи и индекс фингерпринтов сохраняются (долг этапа 4: иначе
// первая транзакция посевного кошелька не проходит C8).
func (s *State) SeedMembers(members []wire.SeedMember, networkID []byte, epoch, netTime uint64) {
	for _, m := range members {
		addr := crypto.AddressOfKey(m.PublicKey, networkID)
		w := &Wallet{
			Address:         addr,
			Status:          wire.StatusActive,
			Seed:            true,
			RegisteredAt:    netTime,
			RegisteredEpoch: epoch,
			ActivatedAt:     netTime,
			ActivatedEpoch:  epoch,
			// ActivatedPeriod = 1: посевной круг активен с genesis, но
			// первый заявляемый период выпуска — 1 (период 0 не заявляем:
			// last_claimed_period = 0 — сентинел «ничего не заявлено»,
			// повторная заявка периода 0 неотличима от первой, M3).
			// Без этого при quota_reset (diff = 1) посевной круг не может
			// выпустить ни одной квоты (этап 12: полный путь профиля 8).
			ActivatedPeriod:  1,
			LastTouchEpoch:   epoch,
			DelayedThreshold: s.Net.Profile.DelayedThresholdDefault,
		}
		if len(m.PublicKey) > 0 {
			if pub, err := crypto.ParsePubKey(m.PublicKey); err == nil {
				w.PublicKey = pub
				fp := crypto.Fingerprint(m.PublicKey)
				w.Fingerprint = fp
				s.FingerprintIndex[string(fp)] = addr
			}
		}
		s.Wallets[key(addr)] = w
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

// RecurringSpendable — переводимый остаток в recurring-сети (v9.10,
// раздел 4): единица периода P переводима, пока текущий период ≤ P+1
// (перевод отвергается после границы периода P+1 — сгорание меткой,
// не сброс баланса: когорты остаются в кошельке и истории). Liquid
// в recurring-сети не используется — все единицы несут периодную
// метку (в том числе полученные переводом).
func (w *Wallet) RecurringSpendable(currentPeriod uint64) uint64 {
	var sum uint64
	for _, c := range w.Cohorts {
		if currentPeriod <= c.Period+1 {
			sum += c.Amount
		}
	}
	if sum > w.Locked {
		return sum - w.Locked
	}
	return 0
}

// RecurringSpend списывает amount в recurring-сети из переводимых
// когорт (период P: текущий ≤ P+1), старейшие вперёд; возвращает
// списанные доли когорт — получателю они приходят с теми же метками
// периода (история проверяема). Списание обязано быть ≤ RecurringSpendable.
func (w *Wallet) RecurringSpend(amount uint64, currentPeriod uint64) []Cohort {
	var spent []Cohort
	for i := 0; i < len(w.Cohorts) && amount > 0; i++ {
		c := &w.Cohorts[i]
		if currentPeriod > c.Period+1 {
			continue // сгоревшая когорта не тратится (история остаётся)
		}
		take := c.Amount
		if take > amount {
			take = amount
		}
		c.Amount -= take
		amount -= take
		if take > 0 {
			spent = append(spent, Cohort{Period: c.Period, Amount: take})
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
	return spent
}

// CreditCohorts зачисляет доли когорт кошельку (recurring-сеть):
// полученные единицы сохраняют периодные метки отправителя.
func (w *Wallet) CreditCohorts(spent []Cohort) {
	for _, c := range spent {
		if c.Amount == 0 {
			continue
		}
		merged := false
		for i := range w.Cohorts {
			if w.Cohorts[i].Period == c.Period {
				w.Cohorts[i].Amount += c.Amount
				merged = true
				break
			}
		}
		if !merged {
			w.Cohorts = append(w.Cohorts, c)
		}
	}
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

// FreeSlots — удалён (аудит z-ai AUDIT-Z-10): возвращал число занятых
// слотов вопреки имени; вызовов в проде не было.
