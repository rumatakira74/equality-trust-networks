// Модули (раздел 6, этап 10): pool (6.3), redeem (6.2), contribution (6.4).
// Типизированные операции, не контракты: правила фиксированы протоколом и
// проверяются каждым узлом. Включение — полем modules профиля (раздел 4).
//
// Числа, отсутствующие в Документе, — предположения v1, PENDING №13/№19/№26
// в QUESTIONS.md: N = 2 подтверждения вклада, лимит 5 подтверждений на
// человека в период, потолки on_contribution (1× issuance на человека,
// 10× issuance на период), право закрытия пула — у создателя, судьба
// незаявленных долей — остаются в пуле.
package ledger

import (
	"bytes"

	"github.com/rumatakira74/equality-trust-networks/core/wire"
)

// Предположения v1 (PENDING: вопрос №13 — числа в Документе не заданы).
const (
	// ContributionConfirmN — независимых подтверждений, принимающих заявку (6.4).
	ContributionConfirmN = 2
	// ContributionConfirmLimit — лимит подтверждений на человека в период (6.4).
	ContributionConfirmLimit = 5
	// ContributionPersonCap — потолок on_contribution на человека в период,
	// в долях issuance_amount (6.4).
	ContributionPersonCap = 1
	// ContributionPeriodCap — потолок on_contribution на период по сети,
	// в долях issuance_amount (6.4).
	ContributionPeriodCap = 10
)

// Pool — типизированный счёт пула (6.3): сборы, бюджет программы, общий
// фонд. Прогресс публичен (состояние пересчитывается каждым узлом).
type Pool struct {
	ID            []byte // txid pool_create
	Creator       []byte
	Recipient     []byte
	Goal          uint64
	DeadlineEpoch uint64
	ReleaseMode   wire.PoolReleaseMode
	RefundMode    wire.PoolRefundMode
	Balance       uint64
	Donations     map[string]uint64 // донат → внесено (для возврата)
	Claimed       map[string]bool   // заявитель → получил долю
	Share         uint64            // доля, зафиксированная при выпуске
	Released      bool
	Closed        bool
}

// ContributionClaim — заявка вклада (6.4) и её подтверждения.
type ContributionClaim struct {
	Claimant        []byte
	PoolID          []byte // пул программы для выплаты (0 — эмиссия on_contribution)
	DescriptionHash []byte
	Size            uint64
	Period          uint64
	Confirmers      [][]byte
	Confirmed       bool
	Paid            bool
}

// RedeemInfo — состоявшееся погашение (для проверки receipt, D5).
type RedeemInfo struct {
	Provider   []byte
	Amount     uint64
	ResourceID string
	Slot       *wire.TimeSlot
}

func countActiveWallets(l *Ledger) uint64 {
	n := uint64(0)
	for _, w := range l.State.Wallets {
		if w.Status == wire.StatusActive {
			n++
		}
	}
	return n
}

// moduleEnabled — модуль включён в профиле сети (раздел 4, поле modules).
func (l *Ledger) moduleEnabled(id uint32) bool {
	for _, m := range l.State.Net.Profile.Modules {
		if m == id {
			return true
		}
	}
	return false
}

// spendWithFee — правила списания как у transfer (D3/P2, решение D10):
// мораторий от register, переводимый остаток ≥ amount + fee, fee по
// fee_mode (burn — сжигается, pool — общий фонд сети).
func (l *Ledger) spendWithFee(w *Wallet, amount uint64) error {
	now := l.Clock.NetTime()
	period := l.Clock.Period()
	prof := l.State.Net.Profile
	if now-w.RegisteredAt < prof.TransferMoratoriumDays*86400 {
		return errMoratorium
	}
	fee := l.State.Net.Fee(amount, l.State.FeeRate)
	if w.Spendable(period, prof.FreshTransferWindowPeriods) < amount+fee {
		return errBalance
	}
	w.Spend(amount+fee, period, prof.FreshTransferWindowPeriods)
	if prof.FeeMode == wire.FeeModeBurn {
		l.State.TotalBurned += fee
	} else if prof.FeeMode == wire.FeeModePool {
		l.State.CommonFund += fee
	}
	return nil
}

var (
	errMoratorium = &txError{"E_MORATORIUM"}
	errBalance    = &txError{"E_BALANCE"}
)

type txError struct{ code string }

func (e *txError) Error() string { return e.code }

// ---------------------------------------------------------------------------
// POOL (spec/03 3.9)
// ---------------------------------------------------------------------------

// applyPoolCreate — P1: получатель — обычный кошелёк из параметров сети
// (providers); дедлайн в будущем; режимы заданы.
func (l *Ledger) applyPoolCreate(tx *wire.Transaction) Result {
	if !l.moduleEnabled(wire.ModulePool) {
		return reject("E_MODULE")
	}
	p := tx.Payload.(*wire.PoolCreatePayload)
	epoch := l.Clock.Epoch()
	if l.State.Wallet(p.Recipient) == nil || !l.inProviders(p.Recipient) {
		return reject("E_POOL_PARAMS")
	}
	if p.DeadlineEpoch <= epoch {
		return reject("E_POOL_PARAMS")
	}
	if p.ReleaseMode != wire.PoolReleaseOnDeadline &&
		p.ReleaseMode != wire.PoolReleaseOnGoal &&
		p.ReleaseMode != wire.PoolReleaseContinuous {
		return reject("E_POOL_PARAMS")
	}
	if p.RefundMode != wire.PoolRefundIfUnmet && p.RefundMode != wire.PoolRefundNone {
		return reject("E_POOL_PARAMS")
	}
	txid := tx.TxID()
	l.State.Pools[string(txid[:])] = &Pool{
		ID:            append([]byte(nil), txid[:]...),
		Creator:       append([]byte(nil), tx.Wallet...),
		Recipient:     append([]byte(nil), p.Recipient...),
		Goal:          p.GoalAmount,
		DeadlineEpoch: p.DeadlineEpoch,
		ReleaseMode:   p.ReleaseMode,
		RefundMode:    p.RefundMode,
		Donations:     make(map[string]uint64),
		Claimed:       make(map[string]bool),
	}
	return accept(txid)
}

func (l *Ledger) inProviders(addr []byte) bool {
	for _, pr := range l.State.Net.Profile.Providers {
		if bytes.Equal(pr, addr) {
			return true
		}
	}
	return false
}

// applyPoolDonate — P2: пул открыт; правила списания как у transfer.
func (l *Ledger) applyPoolDonate(tx *wire.Transaction) Result {
	if !l.moduleEnabled(wire.ModulePool) {
		return reject("E_MODULE")
	}
	p := tx.Payload.(*wire.PoolDonatePayload)
	pool := l.State.Pools[string(p.PoolID)]
	if pool == nil || pool.Closed {
		return reject("E_POOL_STATE")
	}
	if pool.ReleaseMode == wire.PoolReleaseOnDeadline && l.Clock.Epoch() > pool.DeadlineEpoch {
		return reject("E_POOL_STATE") // дедлайн прошёл, выпуск закрыт
	}
	w := l.State.Wallet(tx.Wallet)
	if err := l.spendWithFee(w, p.Amount); err != nil {
		return reject(err.Error())
	}
	pool.Balance += p.Amount
	pool.Donations[string(tx.Wallet)] += p.Amount
	// release_on_goal: цель достигнута — выпуск открыт, доля фиксируется
	if pool.ReleaseMode == wire.PoolReleaseOnGoal && !pool.Released &&
		pool.Goal > 0 && pool.Balance >= pool.Goal {
		pool.Released = true
		pool.Share = pool.Balance / countActiveWallets(l)
	}
	w.LastTouchEpoch = l.Clock.Epoch()
	return accept(tx.TxID())
}

// applyPoolClose — P3 (право закрытия — PENDING №26, предположение:
// создатель пула); P5: refund_if_unmet и цель не достигнута — возврат
// донатам (6.3).
func (l *Ledger) applyPoolClose(tx *wire.Transaction) Result {
	if !l.moduleEnabled(wire.ModulePool) {
		return reject("E_MODULE")
	}
	p := tx.Payload.(*wire.PoolClosePayload)
	pool := l.State.Pools[string(p.PoolID)]
	if pool == nil || pool.Closed {
		return reject("E_POOL_STATE")
	}
	if !bytes.Equal(tx.Wallet, pool.Creator) {
		return reject("E_POOL_STATE") // право закрытия — создатель (№26)
	}
	if pool.Released {
		return reject("E_POOL_STATE") // выпущенный пул не закрывается
	}
	// возврат донатам при невыполненной цели (P5)
	if pool.RefundMode == wire.PoolRefundIfUnmet &&
		(pool.Goal == 0 || pool.Balance < pool.Goal) {
		for donor, amount := range pool.Donations {
			if amount == 0 || pool.Balance == 0 {
				continue
			}
			refund := amount
			if refund > pool.Balance {
				refund = pool.Balance
			}
			if dw := l.State.Wallet([]byte(donor)); dw != nil {
				dw.Liquid += refund
			}
			pool.Balance -= refund
		}
	}
	pool.Closed = true
	return accept(tx.TxID())
}

// applyPoolClaim — P4: пул выпущен (release_on_deadline — после дедлайна,
// release_on_goal — при цели, release_continuous — непрерывно); заявитель
// active; равная доля; повторный claim — E_POOL_CLAIMED. Незаявленные
// доли остаются в пуле (PENDING №19).
func (l *Ledger) applyPoolClaim(tx *wire.Transaction) Result {
	if !l.moduleEnabled(wire.ModulePool) {
		return reject("E_MODULE")
	}
	p := tx.Payload.(*wire.PoolClaimPayload)
	pool := l.State.Pools[string(p.PoolID)]
	w := l.State.Wallet(tx.Wallet)
	if pool == nil || pool.Closed {
		return reject("E_POOL_STATE")
	}
	if w.Status != wire.StatusActive {
		return reject("E_STATUS")
	}
	if pool.Claimed[string(tx.Wallet)] {
		return reject("E_POOL_CLAIMED")
	}
	var share uint64
	switch pool.ReleaseMode {
	case wire.PoolReleaseOnDeadline, wire.PoolReleaseOnGoal:
		if !pool.Released {
			return reject("E_POOL_STATE") // выпуск ещё не открыт
		}
		share = pool.Share
	case wire.PoolReleaseContinuous:
		active := countActiveWallets(l)
		if active == 0 {
			return reject("E_POOL_STATE")
		}
		share = pool.Balance / active
	default:
		return reject("E_POOL_STATE")
	}
	if share == 0 || pool.Balance < share {
		return reject("E_POOL_STATE")
	}
	pool.Balance -= share
	pool.Claimed[string(tx.Wallet)] = true
	w.Liquid += share
	w.LastTouchEpoch = l.Clock.Epoch()
	return accept(tx.TxID())
}

// settlePools — граница эпохи: выпуск release_on_deadline открывается
// после дедлайна (доля фиксируется по числу active), непрерывный
// закрывается дедлайном. Детерминированно: состояние и эпоха одинаковы
// на всех узлах.
func (l *Ledger) settlePools(epoch uint64) {
	for _, pool := range l.State.Pools {
		if pool.Closed {
			continue
		}
		if epoch <= pool.DeadlineEpoch {
			continue
		}
		switch pool.ReleaseMode {
		case wire.PoolReleaseOnDeadline:
			if !pool.Released {
				pool.Released = true
				pool.Share = pool.Balance / countActiveWallets(l)
			}
		case wire.PoolReleaseContinuous:
			pool.Closed = true // непрерывный выпуск закончен дедлайном
		}
	}
}

// ---------------------------------------------------------------------------
// REDEEM / RECEIPT (spec/03 3.11)
// ---------------------------------------------------------------------------

// applyRedeem — D1–D4: поставщик из параметров сети, resource_id сети,
// правила списания как у transfer, слот в будущем/настоящем.
func (l *Ledger) applyRedeem(tx *wire.Transaction) Result {
	if !l.moduleEnabled(wire.ModuleRedeem) {
		return reject("E_MODULE")
	}
	p := tx.Payload.(*wire.RedeemPayload)
	now := l.Clock.NetTime()
	if !l.inProviders(p.Provider) {
		return reject("E_REDEEM_PROVIDER")
	}
	if l.State.Net.Profile.ResourceID == "" || p.ResourceID != l.State.Net.Profile.ResourceID {
		return reject("E_REDEEM_RESOURCE")
	}
	if p.Slot != nil && now > p.Slot.End {
		return reject("E_REDEEM_SLOT")
	}
	w := l.State.Wallet(tx.Wallet)
	if err := l.spendWithFee(w, p.Amount); err != nil {
		return reject(err.Error())
	}
	provider := l.State.Wallet(p.Provider)
	if provider != nil {
		provider.Liquid += p.Amount
	}
	txid := tx.TxID()
	l.State.Redeems[string(txid[:])] = &RedeemInfo{
		Provider:   append([]byte(nil), p.Provider...),
		Amount:     p.Amount,
		ResourceID: p.ResourceID,
		Slot:       p.Slot,
	}
	w.LastTouchEpoch = l.Clock.Epoch()
	return accept(txid)
}

// applyReceipt — D5: подписан поставщиком (C8 уже проверил подпись);
// redeem_txid — валидная REDEEM в пользу этого поставщика; параметры
// совпадают. Запись, балансы не меняет (6.2).
func (l *Ledger) applyReceipt(tx *wire.Transaction) Result {
	if !l.moduleEnabled(wire.ModuleRedeem) {
		return reject("E_MODULE")
	}
	p := tx.Payload.(*wire.ReceiptPayload)
	info := l.State.Redeems[string(p.RedeemTxID)]
	if info == nil {
		return reject("E_RECEIPT_MISMATCH")
	}
	if !bytes.Equal(tx.Wallet, info.Provider) {
		return reject("E_RECEIPT_MISMATCH")
	}
	if p.Amount != info.Amount || p.ResourceID != info.ResourceID {
		return reject("E_RECEIPT_MISMATCH")
	}
	if (p.Slot == nil) != (info.Slot == nil) {
		return reject("E_RECEIPT_MISMATCH")
	}
	if p.Slot != nil && (*p.Slot != *info.Slot) {
		return reject("E_RECEIPT_MISMATCH")
	}
	if l.State.Receipts[string(p.RedeemTxID)] {
		return reject("E_RECEIPT_MISMATCH") // повторная квитанция
	}
	l.State.Receipts[string(p.RedeemTxID)] = true
	return accept(tx.TxID())
}

// ---------------------------------------------------------------------------
// CONTRIBUTION (spec/03 3.12)
// ---------------------------------------------------------------------------

// applyContributionClaim — K1: период текущий или предыдущий (окно —
// предположение v1, PENDING №13); заявитель active.
func (l *Ledger) applyContributionClaim(tx *wire.Transaction) Result {
	if !l.moduleEnabled(wire.ModuleContribution) {
		return reject("E_MODULE")
	}
	p := tx.Payload.(*wire.ContributionClaimPayload)
	w := l.State.Wallet(tx.Wallet)
	period := l.Clock.Period()
	if w.Status != wire.StatusActive {
		return reject("E_STATUS")
	}
	if p.Period != period && p.Period != period-1 {
		return reject("E_CONTRIBUTION_PERIOD")
	}
	if p.Size == 0 || len(p.DescriptionHash) == 0 {
		return reject("E_CONTRIBUTION_PERIOD")
	}
	if len(p.PoolID) > 0 {
		if l.State.Pools[string(p.PoolID)] == nil {
			return reject("E_POOL_STATE") // пула программы нет
		}
	}
	txid := tx.TxID()
	l.State.Claims[string(txid[:])] = &ContributionClaim{
		Claimant:        append([]byte(nil), tx.Wallet...),
		PoolID:          append([]byte(nil), p.PoolID...),
		DescriptionHash: append([]byte(nil), p.DescriptionHash...),
		Size:            p.Size,
		Period:          p.Period,
	}
	return accept(txid)
}

// applyContributionConfirm — K2–K6: подтверждающий active, не связан
// поручительствами с заявителем, лимит на человека в период, без дублей;
// N подтверждений принимают заявку и запускают выплату (6.4).
func (l *Ledger) applyContributionConfirm(tx *wire.Transaction) Result {
	if !l.moduleEnabled(wire.ModuleContribution) {
		return reject("E_MODULE")
	}
	p := tx.Payload.(*wire.ContributionConfirmPayload)
	claim := l.State.Claims[string(p.ClaimTxID)]
	if claim == nil || claim.Confirmed {
		return reject("E_CONTRIBUTION_DUP")
	}
	confirmer := l.State.Wallet(tx.Wallet)
	// K2: подтверждающий active
	if confirmer == nil || confirmer.Status != wire.StatusActive {
		return reject("E_CONTRIBUTION_STATUS")
	}
	// K3: не связан поручительствами с заявителем (нет ребра vouch ни в
	// одну сторону); подтверждающий сам себе — связан по определению
	if bytes.Equal(tx.Wallet, claim.Claimant) || l.vouchLinked(confirmer.Address, claim.Claimant) {
		return reject("E_CONTRIBUTION_LINKED")
	}
	// K5: этот подтверждающий ещё не подтверждал эту заявку
	for _, c := range claim.Confirmers {
		if bytes.Equal(c, tx.Wallet) {
			return reject("E_CONTRIBUTION_DUP")
		}
	}
	// K4: лимит подтверждений на человека в период (предположение v1, №13)
	limKey := string(tx.Wallet) + "|" + u64str(claim.Period)
	if l.State.ConfirmsPerPeriod[limKey] >= ContributionConfirmLimit {
		return reject("E_CONTRIBUTION_LIMIT")
	}
	claim.Confirmers = append(claim.Confirmers, append([]byte(nil), tx.Wallet...))
	l.State.ConfirmsPerPeriod[limKey]++
	// K6: N независимых подтверждений принимают заявку — выплата
	if uint64(len(claim.Confirmers)) < ContributionConfirmN {
		return accept(tx.TxID())
	}
	claim.Confirmed = true
	if err := l.payContribution(claim); err != nil {
		return reject(err.Error())
	}
	return accept(tx.TxID())
}

// vouchLinked — ребро поручительства между a и b в любую сторону (6.4).
func (l *Ledger) vouchLinked(a, b []byte) bool {
	wa := l.State.Wallet(a)
	wb := l.State.Wallet(b)
	if wa == nil || wb == nil {
		return false
	}
	for _, v := range wa.VouchesReceived {
		if bytes.Equal(v.Voucher, b) {
			return true
		}
	}
	for _, v := range wb.VouchesReceived {
		if bytes.Equal(v.Voucher, a) {
			return true
		}
	}
	for _, a2 := range wa.Activators {
		if bytes.Equal(a2, b) {
			return true
		}
	}
	for _, a2 := range wb.Activators {
		if bytes.Equal(a2, a) {
			return true
		}
	}
	return false
}

// payContribution — выплата подтверждённого вклада (6.4): из пула
// программы (рекомендуется, инфляции нет) либо ограниченной эмиссией
// on_contribution с потолками на период и на человека (предположение v1,
// PENDING №13).
func (l *Ledger) payContribution(claim *ContributionClaim) error {
	claimant := l.State.Wallet(claim.Claimant)
	if claimant == nil {
		return &txError{"E_CONTRIBUTION_STATUS"}
	}
	if len(claim.PoolID) > 0 {
		pool := l.State.Pools[string(claim.PoolID)]
		if pool == nil || pool.Balance < claim.Size {
			return &txError{"E_POOL_STATE"}
		}
		pool.Balance -= claim.Size
		claimant.Liquid += claim.Size
		claim.Paid = true
		return nil
	}
	// эмиссия on_contribution: только в профиле с этим режимом выпуска
	if l.State.Net.Profile.IssuanceMode != wire.IssuanceOnContribution {
		return &txError{"E_POOL_STATE"}
	}
	prof := l.State.Net.Profile
	personKey := string(claim.Claimant) + "|" + u64str(claim.Period)
	personCap := prof.IssuanceAmount * ContributionPersonCap
	if l.State.ContributionPaidPerson[personKey]+claim.Size > personCap {
		return &txError{"E_CONTRIBUTION_LIMIT"}
	}
	periodCap := prof.IssuanceAmount * ContributionPeriodCap
	if l.State.ContributionPaidPerPeriod[claim.Period]+claim.Size > periodCap {
		return &txError{"E_CONTRIBUTION_LIMIT"}
	}
	l.State.ContributionPaidPerson[personKey] += claim.Size
	l.State.ContributionPaidPerPeriod[claim.Period] += claim.Size
	claimant.Liquid += claim.Size
	l.State.TotalMinted += claim.Size
	claim.Paid = true
	return nil
}
