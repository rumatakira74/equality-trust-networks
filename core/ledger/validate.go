// Валидация и применение транзакций: spec/03 (C1–C11, таблицы R/V/M/T/X/S/G/N/E/O).
// Каждый отказ — точный код E_* из spec/00 0.4. Эквивокация — не ошибка
// валидации: обе ветки проходят C1–C11, конфликт разрешает финализация (9.1).
package ledger

import (
	"bytes"
	"crypto/ecdsa"
	"time"

	"github.com/rumatakira74/equality-trust-networks/core/attest"
	"github.com/rumatakira74/equality-trust-networks/core/crypto"
	"github.com/rumatakira74/equality-trust-networks/core/wire"
)

type Outcome int

const (
	Accepted Outcome = iota
	Rejected
	Conflict
)

func (o Outcome) String() string {
	switch o {
	case Accepted:
		return "accept"
	case Rejected:
		return "reject"
	default:
		return "conflict"
	}
}

// Result — результат применения транзакции.
type Result struct {
	Outcome Outcome
	Code    string // "OK" | "E_*" | "EQUIVOCATION_BRANCH_LOST"
	TxID    [32]byte
}

func accept(txid [32]byte) Result { return Result{Accepted, "OK", txid} }
func reject(code string) Result   { return Result{Rejected, code, [32]byte{}} }

var zero32 = make([]byte, 32)

// Окно вступления смены гардианов в силу — 48 ч (11, решение владельца
// 21.10.2026): константа, не параметр сети (прецедент — отложенный
// перевод и повышение P, D = 48 ч).
const GuardianChangeWindowSeconds uint64 = 48 * 3600

// GuardianChangeLimit — лимит вступивших в силу смен гардианов на
// кошелёк за всё время существования (решение владельца 21.10.2026;
// отменённые и истёкшие лимит не потребляют).
const GuardianChangeLimit uint64 = 150

// applyCommon выполняет C1–C4 (и C5–C7 вне register).
func (l *Ledger) applyCommon(tx *wire.Transaction, txid [32]byte) Result {
	// C1: сеть
	if !bytes.Equal(tx.NetworkID, l.State.Net.NetworkID) {
		return reject("E_NET")
	}
	// C2: версия протокола (13)
	if tx.ProtocolVersion != l.State.Net.ProtocolVersion {
		return reject("E_VER")
	}
	// C3: дубль txid
	if _, ok := l.TxIndex[string(txid[:])]; ok {
		return reject("E_DUP")
	}
	// C4: type соответствует payload
	if !wire.PayloadMatchesType(tx.Type, tx.Payload) {
		return reject("E_TYPE")
	}
	return Result{}
}

// applyChainChecks выполняет C5–C7 (цепочка; register — отдельно).
func (l *Ledger) applyChainChecks(tx *wire.Transaction, txid [32]byte) Result {
	w := l.State.Wallet(tx.Wallet)
	if tx.Type == wire.Register {
		if tx.Seq != 1 {
			return reject("E_SEQ")
		}
		if len(tx.Prev) != 0 && !bytes.Equal(tx.Prev, zero32) {
			return reject("E_PREV")
		}
		return Result{}
	}
	if w == nil || w.Terminal {
		return reject("E_UNKNOWN_WALLET")
	}
	if tx.Seq == w.Seq && !bytes.Equal(txid[:], w.TipTxID) {
		// Конфликтная ветка на той же высоте: эквивокация — не ошибка
		// валидации (spec/03 3.1); ветка проходит C8–C11 и уходит финализатору.
		return Result{Outcome: Conflict, Code: "EQUIVOCATION_BRANCH_LOST", TxID: txid}
	}
	if tx.Seq != w.Seq+1 {
		return reject("E_SEQ")
	}
	if !bytes.Equal(tx.Prev, w.TipTxID) {
		return reject("E_PREV")
	}
	return Result{}
}

// applySigPowTime выполняет C8–C11. Для register ключ берётся из payload.
func (l *Ledger) applySigPowTime(tx *wire.Transaction, registerPub *ecdsa.PublicKey) Result {
	// C10/C11 (ранняя проверка): PoW обязан присутствовать
	if tx.Pow == nil {
		return reject("E_POW_LIGHT")
	}
	w := l.State.Wallet(tx.Wallet)
	pub := registerPub
	if tx.Type != wire.Register && w != nil {
		pub = w.PublicKey
	}
	// C8: подпись ECDSA P-256 над signing_hash
	sh := tx.SigningHash()
	if pub == nil || !crypto.VerifyDER(pub, sh[:], tx.Signature) {
		return reject("E_SIG")
	}
	// C9: допуск timestamp ±30 мин (9.2)
	now := l.Clock.NetTime()
	tol := l.State.Net.Profile.TimestampToleranceSeconds
	if tx.TimestampClaim+tol < now || now+tol < tx.TimestampClaim {
		return reject("E_TIME")
	}
	// C10/C11: PoW
	pre := tx.SigningPreimage()
	hard := tx.Type == wire.Register || tx.Type == wire.Mint
	if hard {
		if tx.Pow.Algorithm != wire.PowArgon2id {
			return reject("E_POW_HARD")
		}
		target := l.State.Net.Pow.TargetHard
		// R7: повторяющийся android_id повышает таргет (слабый сигнал, 7.2)
		if tx.Type == wire.Register {
			reg := tx.Payload.(*wire.RegisterPayload)
			if reg.AndroidID != "" && l.State.AndroidIDSeen[reg.AndroidID] > 0 &&
				l.State.Net.Pow.RepeatAndroidIDTarget > 0 {
				target = l.State.Net.Pow.RepeatAndroidIDTarget
			}
		}
		if !crypto.VerifyHard(pre, tx.Pow.Nonce, target, crypto.Argon2Params{
			MemoryKiB:   l.State.Net.Pow.Argon2MemoryKiB,
			Iterations:  l.State.Net.Pow.Argon2Iterations,
			Parallelism: l.State.Net.Pow.Argon2Parallelism,
		}) {
			return reject("E_POW_HARD")
		}
		// M7: окно челленджа выпуска (7.6). Отклонение D-1 (см. отчёт):
		// векторы этапа 0 не моделируют слот челленджа; проверка вынесена
		// в CheckMintChallengeWindow и включается опцией.
		if tx.Type == wire.Mint && !l.Opts.SkipMintChallengeWindow {
			mp := tx.Payload.(*wire.MintPayload)
			if !l.CheckMintChallengeWindow(tx.Wallet, mp.Period, now) {
				return reject("E_POW_HARD")
			}
		}
	} else {
		if tx.Pow.Algorithm != wire.PowLight {
			return reject("E_POW_LIGHT")
		}
		if !crypto.VerifyLight(pre, tx.Pow.Nonce, l.State.Net.Pow.TargetLight) {
			return reject("E_POW_LIGHT")
		}
	}
	return Result{}
}

// CheckMintChallengeWindow: слот H(address || period || checkpoint_hash(prev epoch))
// и окно ±48 ч (7.6). В headless-ядре checkpoint_hash = 0^32 (genesis),
// сетевая реализация — этап 5.
func (l *Ledger) CheckMintChallengeWindow(addr []byte, period, now uint64) bool {
	win := l.State.Net.Profile.MintChallengeWindowHours * 3600
	if win == 0 {
		return true
	}
	slot := mintChallengeSlot(l.State.GenesisTime, addr, period)
	lo, hi := slot-win, slot+win
	return now >= lo && now <= hi
}

// mintChallengeSlot — детерминированный момент слота внутри периода:
// period_start + (H(...) mod period_seconds). Периоды сети — календарные
// от genesis (9.2): period_start = genesis_time + period × 30 сут
// (конвенция headless-ядра; та же конвенция в net.RealClock).
// Правка этапа 8 (долг этапа 7): раньше отсчёт шёл от 1970 г. — при
// genesis-времени близком к текущему слот выпадал в далёкое будущее/прошлое
// и окно челленджа всегда было закрыто.
func mintChallengeSlot(genesisTime uint64, addr []byte, period uint64) uint64 {
	h := hashThree(addr, period)
	periodStart := genesisTime + period*30*86400
	return periodStart + h%(30*86400)
}

// Apply применяет транзакцию: сеттлемент -> C1–C11 -> типовая таблица -> эффекты.
func (l *Ledger) Apply(tx *wire.Transaction) Result {
	l.syncTime()
	txid := tx.TxID()

	if res := l.applyCommon(tx, txid); res.Code != "" && res.Outcome == Rejected {
		return res
	}
	if res := l.applyChainChecks(tx, txid); res.Outcome == Rejected {
		return res
	} else if res.Outcome == Conflict {
		// ветка проходит подпись/PoW/время, но не применяется
		if res2 := l.applySigPowTime(tx, nil); res2.Outcome == Rejected {
			return res2
		}
		l.recordConflict(tx, txid)
		return res
	}

	var registerPub *ecdsa.PublicKey
	if tx.Type == wire.Register {
		reg := tx.Payload.(*wire.RegisterPayload)
		pub, err := crypto.ParsePubKey(reg.AttestedPublicKey)
		if err != nil {
			return reject("E_ATTEST")
		}
		registerPub = pub
	}
	if res := l.applySigPowTime(tx, registerPub); res.Outcome == Rejected {
		return res
	}

	var res Result
	switch tx.Type {
	case wire.Register:
		res = l.applyRegister(tx)
	case wire.Vouch:
		res = l.applyVouch(tx)
	case wire.Mint:
		res = l.applyMint(tx)
	case wire.Transfer:
		res = l.applyTransfer(tx)
	case wire.Migrate:
		res = l.applyMigrate(tx)
	case wire.Evidence:
		res = l.applyEvidence(tx)
	case wire.GuardianOffer:
		res = l.applyGuardianOffer(tx)
	case wire.PoolCreate:
		res = l.applyPoolCreate(tx)
	case wire.PoolDonate:
		res = l.applyPoolDonate(tx)
	case wire.PoolClose:
		res = l.applyPoolClose(tx)
	case wire.PoolClaim:
		res = l.applyPoolClaim(tx)
	case wire.Redeem:
		res = l.applyRedeem(tx)
	case wire.Receipt:
		res = l.applyReceipt(tx)
	case wire.ContributionClaim:
		res = l.applyContributionClaim(tx)
	case wire.ContributionConfirm:
		res = l.applyContributionConfirm(tx)
	case wire.UpdateGuardians:
		res = l.applyUpdateGuardians(tx)
	default:
		return reject("E_TYPE")
	}
	if res.Outcome != Accepted {
		return res
	}
	// фиксация цепочки и каузального журнала (INV-3)
	w := l.State.Wallet(tx.Wallet)
	if w != nil {
		w.Seq = tx.Seq
		w.TipTxID = append([]byte(nil), txid[:]...)
	}
	l.TxIndex[string(txid[:])] = tx
	l.journal = append(l.journal, tx)
	return accept(txid)
}

func (l *Ledger) recordConflict(tx *wire.Transaction, txid [32]byte) {
	l.Conflicts = append(l.Conflicts, TxRef{Wallet: tx.Wallet, Seq: tx.Seq, TxID: txid[:]})
	// полная ветка — в журнал хранения (spec/04 INV-1c, этап 5)
	l.conflictTxs = append(l.conflictTxs, tx)
}

// ---------------------------------------------------------------------------
// REGISTER (spec/03 3.2)
// ---------------------------------------------------------------------------

func (l *Ledger) applyRegister(tx *wire.Transaction) Result {
	p := tx.Payload.(*wire.RegisterPayload)
	now := l.Clock.NetTime()
	epoch := l.Clock.Epoch()

	// R1: фингерпринт уникален и не в чёрном списке (7.2, 11)
	fp := crypto.Fingerprint(p.AttestedPublicKey)
	if _, exists := l.State.FingerprintIndex[string(fp)]; exists {
		return reject("E_FINGERPRINT")
	}
	if l.State.FingerprintBlacklist[string(fp)] {
		return reject("E_FINGERPRINT")
	}
	// R2/R3: цепочка аттестации (7.2)
	chain, err := attest.Verify(p.AttestationChain, l.TrustedRoots,
		l.State.Net.MinSecurityLevel, l.State.Net.MaxPatchAgeMonths,
		time.Unix(int64(now), 0).UTC())
	if err != nil {
		if l.State.Net.Profile.DevicePolicy == wire.DevicePolicyStrict {
			return reject("E_ATTEST")
		}
		return reject("E_ATTEST_POLICY")
	}
	// R4: challenge по формуле (вопрос №23, параметризуемая)
	want := attest.ChallengeFormula(p.AttestedPublicKey, crypto.ProtocolID, l.State.Net.NetworkID)
	if !bytes.Equal(p.AttestationChallenge, want) {
		return reject("E_ATTEST")
	}
	// R4b: challenge внутри цепочки аттестации = challenge регистрации —
	// цепочка выдана под эту регистрацию, не переиспользована (этап 6)
	if len(chain.Challenge) == 0 || !bytes.Equal(chain.Challenge, p.AttestationChallenge) {
		return reject("E_ATTEST")
	}
	// R5: synthetic только в песочнице (6.5)
	if p.Synthetic && !l.State.Net.Profile.Sandbox {
		return reject("E_SBOX")
	}
	// R6: личные гардианы (решение D4): не более guardian_personal_count,
	// адреса известны (11)
	if uint64(len(p.PersonalGuardians)) > l.State.Net.Profile.GuardianPersonalCount {
		return reject("E_MIGRATE_GUARDIANS")
	}
	for _, g := range p.PersonalGuardians {
		gw := l.State.Wallet(g)
		if gw == nil {
			return reject("E_MIGRATE_GUARDIANS")
		}
	}

	addr := crypto.DeriveAddress(fp, l.State.Net.NetworkID)
	w := &Wallet{
		Address:           addr,
		Fingerprint:       fp,
		PublicKey:         nil, // заполняется ниже
		Status:            wire.StatusRegistered,
		Synthetic:         p.Synthetic,
		RegisteredAt:      now,
		RegisteredEpoch:   epoch,
		LastTouchEpoch:    epoch,
		DelayedThreshold:  l.State.Net.Profile.DelayedThresholdDefault,
		PersonalGuardians: append([][]byte(nil), p.PersonalGuardians...),
		AndroidID:         p.AndroidID,
	}
	pub, _ := crypto.ParsePubKey(p.AttestedPublicKey)
	w.PublicKey = pub
	if !bytes.Equal(tx.Wallet, addr) {
		return reject("E_FINGERPRINT") // адрес обязан соответствовать ключу
	}
	l.State.Wallets[key(addr)] = w
	l.State.FingerprintIndex[string(fp)] = addr
	if p.AndroidID != "" {
		l.State.AndroidIDSeen[p.AndroidID]++
	}
	return accept(tx.TxID())
}

// ---------------------------------------------------------------------------
// VOUCH (spec/03 3.3)
// ---------------------------------------------------------------------------

func (l *Ledger) applyVouch(tx *wire.Transaction) Result {
	vp := tx.Payload.(*wire.VouchPayload)
	now := l.Clock.NetTime()
	epoch := l.Clock.Epoch()
	v := l.State.Wallet(tx.Wallet)
	cand := l.State.Wallet(vp.Candidate)

	// V1: поручитель active
	if v.Status != wire.StatusActive {
		return reject("E_VOUCH_STATUS")
	}
	// V1f: vouch от flagged-кошелька запрещён (решение №14, этап 5;
	// переводы и прочее остаются разрешёнными)
	if v.Flagged {
		return reject("E_VOUCH_FLAGGED")
	}
	// V2: возраст >= voucher_min_age_days (от register; 6.5)
	if v.AgeDays(now) < l.State.Net.Profile.VoucherMinAgeDays {
		return reject("E_VOUCH_AGE")
	}
	// V3: >= voucher_min_mint_periods выполненных периодов выпуска
	if v.CompletedMintPeriods() < uint64(l.State.Net.Profile.VoucherMinMintPeriods) {
		return reject("E_VOUCH_PERIODS")
	}
	// V4: свободный слот (7.4: 3/5/10 по возрасту)
	slotCap := l.State.Net.SlotCapacity(v.AgeDays(now), v.Seed, v.AgeDays(now))
	if uint32(len(v.OccupiedSlots)) >= slotCap {
		return reject("E_VOUCH_SLOTS")
	}
	// V5: не самому себе
	if bytes.Equal(tx.Wallet, vp.Candidate) {
		return reject("E_VOUCH_SELF")
	}
	// V6: кандидат registered
	if cand == nil || cand.Status != wire.StatusRegistered {
		return reject("E_VOUCH_CANDIDATE")
	}
	// V7: нет прежнего vouch от этого поручителя этому кандидату
	for _, r := range cand.VouchesReceived {
		if bytes.Equal(r.Voucher, tx.Wallet) {
			return reject("E_VOUCH_DUP")
		}
	}
	// V8: эпоха текущая
	if vp.Epoch != epoch {
		return reject("E_VOUCH_EPOCH")
	}
	// V9: vouch от flagged — PENDING вопрос №14; предположение v1: разрешён
	// (штрафов нет, 7.5); помечено в отчёте этапа 3.

	// эффекты
	v.OccupiedSlots = append(v.OccupiedSlots, Slot{
		Candidate:       append([]byte(nil), vp.Candidate...),
		SinceNetTime:    now,
		CandidateActive: false,
	})
	cand.VouchesReceived = append(cand.VouchesReceived, VouchReceipt{
		Voucher: append([]byte(nil), tx.Wallet...),
		Epoch:   epoch,
	})
	// активация при K поручительствах от разных кошельков (7.4)
	K := uint64(l.State.Net.Profile.VouchK)
	if uint64(len(cand.VouchesReceived)) == K {
		cand.Status = wire.StatusActive
		cand.ActivatedAt = now
		cand.ActivatedEpoch = epoch
		cand.ActivatedPeriod = l.Clock.Period()
		for _, r := range cand.VouchesReceived {
			cand.Activators = append(cand.Activators, r.Voucher)
		}
		// пометить слоты поручителей: кандидат активен
		for _, r := range cand.VouchesReceived {
			vw := l.State.Wallet(r.Voucher)
			for i := range vw.OccupiedSlots {
				if bytes.Equal(vw.OccupiedSlots[i].Candidate, vp.Candidate) {
					vw.OccupiedSlots[i].CandidateActive = true
					vw.OccupiedSlots[i].MintsBase = cand.LastClaimedPeriod
				}
			}
		}
	}
	return accept(tx.TxID())
}

// ---------------------------------------------------------------------------
// MINT (spec/03 3.4)
// ---------------------------------------------------------------------------

func (l *Ledger) applyMint(tx *wire.Transaction) Result {
	mp := tx.Payload.(*wire.MintPayload)
	w := l.State.Wallet(tx.Wallet)
	period := l.Clock.Period()
	prof := l.State.Net.Profile

	// M1: статус active (10.1)
	if w.Status != wire.StatusActive {
		return reject("E_STATUS")
	}
	// M2: нет действующей приостановки (flagged; 7.5)
	if w.Flagged && period < w.MintSuspendedUntilPeriod {
		return reject("E_FLAGGED_MINT")
	}
	// M3: period строго больше последнего заявленного (10.1); повтор того же
	// периода — двойной выпуск (7.5)
	if mp.Period <= w.LastClaimedPeriod {
		return reject("E_MINT_DOUBLE")
	}
	// база для первого выпуска: activated_period − 1 (M4, «без ретро-
	// начислений»; вопрос №24 — предположение v1)
	base := w.LastClaimedPeriod
	if base == 0 {
		base = w.ActivatedPeriod - 1
	}
	diff := mp.Period - base
	// M4: заявляются завершённые периоды, не раньше периода активации
	if mp.Period >= period { // только завершённые периоды (вопрос №24)
		return reject("E_MINT_PERIOD")
	}
	if mp.Period < w.ActivatedPeriod { // без ретро-начислений (10.1)
		return reject("E_MINT_PERIOD")
	}
	// M5: окно накопления (10.1); quota_reset (раздел 4, 6.2): квота за
	// период, неиспользованное сгорает — заявляется ровно один период,
	// накопления нет
	if prof.IssuanceMode == wire.IssuanceQuotaReset {
		if diff != 1 {
			return reject("E_ACCRUAL_WINDOW")
		}
	} else if diff > prof.AccrualWindowPeriods {
		return reject("E_ACCRUAL_WINDOW")
	}
	// M6: amount = issuance × diff (10.1)
	if mp.Amount != prof.IssuanceAmount*diff {
		return reject("E_MINT_AMOUNT")
	}

	// эффекты (3.4): доля R из первых 12 выпусков (7.4)
	shareTotal := uint64(0)
	for p := base + 1; p <= mp.Period; p++ {
		idx := p - w.ActivatedPeriod + 1
		if idx <= uint64(prof.VoucherRewardPeriods) && len(w.Activators) == int(prof.VouchK) && !w.Flagged {
			shareTotal += prof.IssuanceAmount * prof.VoucherRewardMicros / 1_000_000
		}
	}
	minterGets := mp.Amount - shareTotal
	w.Cohorts = append(w.Cohorts, Cohort{Period: mp.Period, Amount: minterGets})
	l.State.TotalMinted += mp.Amount
	if shareTotal > 0 {
		per := shareTotal / uint64(prof.VouchK)
		rem := shareTotal - per*uint64(prof.VouchK)
		for _, a := range w.Activators {
			aw := l.State.Wallet(a)
			aw.Cohorts = append(aw.Cohorts, Cohort{Period: mp.Period, Amount: per})
			l.State.TotalMinted += 0 // R не увеличивает эмиссию (источник — выпуск поручаемого)
		}
		if rem > 0 {
			w.Cohorts = append(w.Cohorts, Cohort{Period: mp.Period, Amount: rem})
		}
	}
	w.LastClaimedPeriod = mp.Period
	w.LastTouchEpoch = l.Clock.Epoch()
	return accept(tx.TxID())
}

// ---------------------------------------------------------------------------
// TRANSFER (spec/03 3.5)
// ---------------------------------------------------------------------------

func (l *Ledger) applyTransfer(tx *wire.Transaction) Result {
	tp := tx.Payload.(*wire.TransferPayload)
	w := l.State.Wallet(tx.Wallet)
	now := l.Clock.NetTime()
	period := l.Clock.Period()
	prof := l.State.Net.Profile

	switch {
	case tp.Send != nil:
		amount := tp.Send.Amount
		// T1: единица передаётся (раздел 4)
		if prof.TransferMode == wire.TransferModeSoulbound {
			return reject("E_TRANSFER_MODE")
		}
		// T2: restricted — только поставщику из списка (раздел 4)
		if prof.TransferMode == wire.TransferModeRestricted {
			found := false
			for _, pr := range prof.Providers {
				if bytes.Equal(pr, tp.Send.Recipient) {
					found = true
					break
				}
			}
			if !found {
				return reject("E_TRANSFER_RECIPIENT")
			}
		}
		// T3: мораторий от register (7.6; статус не снимает)
		if now-w.RegisteredAt < prof.TransferMoratoriumDays*86400 {
			return reject("E_MORATORIUM")
		}
		fee := l.State.Net.Fee(amount, l.State.FeeRate)
		// T4/T8: переводимый остаток (когорты + liquid − резерв) >= amount + fee
		if w.Spendable(period, prof.FreshTransferWindowPeriods) < amount+fee {
			return reject("E_BALANCE")
		}
		// T6/T7: отложенный перевод выше P (8.3)
		if amount > w.DelayedThreshold && prof.TransferMode == wire.TransferModeFree && !prof.Sandbox {
			w.Locked += amount + fee
			l.State.PendingTransfers[ptKey(tx.Wallet, tx.Seq)] = &PendingTransfer{
				Seq:       tx.Seq,
				Recipient: append([]byte(nil), tp.Send.Recipient...),
				Amount:    amount,
				Fee:       fee,
				CreatedAt: now,
				Deadline:  now + prof.TransferDelayHours*3600,
			}
			return accept(tx.TxID())
		}
		// малый перевод: сразу
		w.Spend(amount+fee, period, prof.FreshTransferWindowPeriods)
		recipient := l.State.Wallet(tp.Send.Recipient)
		if recipient != nil {
			recipient.Liquid += amount
		}
		if prof.FeeMode == wire.FeeModeBurn {
			l.State.TotalBurned += fee
		} else if prof.FeeMode == wire.FeeModePool {
			l.State.CommonFund += fee // общий фонд сети (раздел 4)
		}
		w.LastTouchEpoch = l.Clock.Epoch()
		return accept(tx.TxID())

	case tp.Cancel != nil:
		// X1–X4: отмена отложенного перевода в течение D (8.3)
		pt := l.State.PendingTransfers[ptKey(tx.Wallet, tp.Cancel.PendingSeq)]
		if pt == nil || pt.Canceled || pt.Executed {
			return reject("E_CANCEL_NOTFOUND")
		}
		if now >= pt.Deadline {
			return reject("E_CANCEL_EXPIRED")
		}
		pt.Canceled = true
		w.Locked -= pt.Amount + pt.Fee
		return accept(tx.TxID())

	case tp.SetThreshold != nil:
		// S1/S2: смена P в пределах P_max (8.3)
		if tp.SetThreshold.NewThreshold > prof.DelayedThresholdMax {
			return reject("E_THRESHOLD_BOUNDS")
		}
		if tp.SetThreshold.NewThreshold > w.DelayedThreshold {
			// повышение вступает через 48 ч (иначе принуждающий поднимет порог)
			w.PendingThreshold = tp.SetThreshold.NewThreshold
			w.PendingThresholdNotBefore = now + prof.TransferDelayHours*3600
		} else {
			// понижение — сразу
			w.DelayedThreshold = tp.SetThreshold.NewThreshold
			w.PendingThreshold = 0
		}
		return accept(tx.TxID())
	}
	return reject("E_TYPE")
}

// ---------------------------------------------------------------------------
// MIGRATE (spec/03 3.6)
// ---------------------------------------------------------------------------

func (l *Ledger) applyMigrate(tx *wire.Transaction) Result {
	mp := tx.Payload.(*wire.MigratePayload)
	now := l.Clock.NetTime()
	epoch := l.Clock.Epoch()
	prof := l.State.Net.Profile

	switch {
	case mp.Propose != nil:
		newW := l.State.Wallet(tx.Wallet)
		oldW := l.State.Wallet(mp.Propose.OldAddress)
		// G1: новый кошелёк зарегистрирован (собственная REGISTER; 11)
		if newW == nil || newW.Status == wire.StatusUnspecified {
			return reject("E_MIGRATE_NEW")
		}
		// G2: старый существует и не терминален
		if oldW == nil || oldW.Terminal {
			return reject("E_MIGRATE_OLD")
		}
		// G3/G4: гардианы и подписи
		personal := make(map[string]bool)
		for _, g := range oldW.PersonalGuardians {
			personal[string(g)] = true
		}
		validPersonal, validPublic := 0, 0
		seenApprover := make(map[string]bool)
		for i := range mp.Propose.Approvals {
			ap := &mp.Propose.Approvals[i]
			gw := l.State.Wallet(ap.Guardian)
			if gw == nil || seenApprover[string(ap.Guardian)] {
				continue
			}
			// подпись над MigrateStatement
			stmt := &wire.MigrateStatement{
				NetworkID:  l.State.Net.NetworkID,
				OldAddress: mp.Propose.OldAddress,
				NewAddress: tx.Wallet,
				Nonce:      mp.Propose.Nonce,
				Epoch:      epoch,
			}
			sh := stmt.Hash()
			if !crypto.VerifyDER(gw.PublicKey, sh[:], ap.StatementSig) {
				return reject("E_MIGRATE_GUARDIANS")
			}
			seenApprover[string(ap.Guardian)] = true
			if personal[string(ap.Guardian)] {
				// личный: не более guardian_personal_max_wards подопечных
				if gw.Wards >= prof.GuardianPersonalMaxWards {
					return reject("E_MIGRATE_GUARDIANS")
				}
				validPersonal++
			} else {
				// публичный: active, возраст, offer, G_max (11)
				if !gw.PublicGuardian || gw.Status != wire.StatusActive ||
					gw.AgeDays(now) < prof.GuardianMinAgeDays ||
					gw.Wards >= prof.GuardianMaxWards {
					return reject("E_MIGRATE_GUARDIANS")
				}
				validPublic++
			}
		}
		// порог: только личные — 2 из 3; с публичными — 3 из 5 (11)
		total := validPersonal + validPublic
		if validPublic > 0 {
			if uint64(total) < prof.GuardianPublicThreshold {
				return reject("E_MIGRATE_THRESHOLD")
			}
		} else {
			if uint64(total) < prof.GuardianPersonalThreshold {
				return reject("E_MIGRATE_THRESHOLD")
			}
		}
		// G5: nonce не использовался прежними попытками этой пары
		for _, pm := range l.State.PendingMigrations {
			if bytes.Equal(pm.OldAddress, mp.Propose.OldAddress) &&
				bytes.Equal(pm.NewAddress, tx.Wallet) && pm.Nonce == mp.Propose.Nonce {
				return reject("E_MIGRATE_NONCE")
			}
		}
		l.State.PendingMigrations = append(l.State.PendingMigrations, &PendingMigration{
			NewAddress:       append([]byte(nil), tx.Wallet...),
			OldAddress:       append([]byte(nil), mp.Propose.OldAddress...),
			ProposeSeq:       tx.Seq,
			CreatedAt:        now,
			ExecuteNotBefore: now + prof.MigrateDelayDays*86400,
			Nonce:            mp.Propose.Nonce,
			Approvals:        append([]wire.GuardianApproval(nil), mp.Propose.Approvals...),
		})
		return accept(tx.TxID())

	case mp.Cancel != nil:
		// N1/N2: отмена старым ключом в течение задержки (11; решение D3)
		for _, pm := range l.State.PendingMigrations {
			if !pm.Canceled && !pm.Executed &&
				bytes.Equal(pm.NewAddress, mp.Cancel.NewAddress) &&
				pm.ProposeSeq == mp.Cancel.ProposeSeq {
				if now >= pm.ExecuteNotBefore {
					return reject("E_MIGRATE_EXPIRED")
				}
				pm.Canceled = true
				return accept(tx.TxID())
			}
		}
		return reject("E_MIGRATE_NOTFOUND")
	}
	return reject("E_TYPE")
}

// ---------------------------------------------------------------------------
// EVIDENCE (spec/03 3.7; закрытый перечень 7.5)
// ---------------------------------------------------------------------------

func (l *Ledger) applyEvidence(tx *wire.Transaction) Result {
	ep := tx.Payload.(*wire.EvidencePayload)
	period := l.Clock.Period()
	if ep.TxA == nil || ep.TxB == nil {
		return reject("E_EVIDENCE_SIG")
	}
	// E1: обе подписи верны (ключом одного кошелька)
	w := l.State.Wallet(ep.TxA.Wallet)
	if w == nil {
		return reject("E_EVIDENCE_SIG")
	}
	sha := ep.TxA.SigningHash()
	shb := ep.TxB.SigningHash()
	if !crypto.VerifyDER(w.PublicKey, sha[:], ep.TxA.Signature) ||
		!crypto.VerifyDER(w.PublicKey, shb[:], ep.TxB.Signature) {
		return reject("E_EVIDENCE_SIG")
	}
	// E2: кошелёк один
	if !bytes.Equal(ep.TxA.Wallet, ep.TxB.Wallet) {
		return reject("E_EVIDENCE_WALLET")
	}
	// E3: seq/prev совпадают
	if ep.TxA.Seq != ep.TxB.Seq && !bytes.Equal(ep.TxA.Prev, ep.TxB.Prev) {
		return reject("E_EVIDENCE_SEQ")
	}
	// E4: хеши разные
	ida, idb := ep.TxA.TxID(), ep.TxB.TxID()
	if ida == idb {
		return reject("E_EVIDENCE_HASH")
	}
	// E5: обе позже register (seq >= 2)
	if ep.TxA.Seq < 2 || ep.TxB.Seq < 2 {
		return reject("E_EVIDENCE_REG")
	}
	// E6: kind соответствует содержанию
	if ep.Kind == wire.EvidenceDoubleMint {
		ma, ok1 := ep.TxA.Payload.(*wire.MintPayload)
		mb, ok2 := ep.TxB.Payload.(*wire.MintPayload)
		if !ok1 || !ok2 || ma.Period != mb.Period {
			return reject("E_EVIDENCE_KIND")
		}
	}
	// E7: факт не доказан ранее
	factKey := string(ep.TxA.Wallet) + "|" + u64str(ep.TxA.Seq)
	if l.State.EvidenceFacts[factKey] {
		return reject("E_EVIDENCE_DUP")
	}
	// эффекты: мягкая схема (7.5; вопрос №5 — подтверждён пользователем)
	l.State.EvidenceFacts[factKey] = true
	repeat := w.Flagged
	w.Flagged = true
	w.FlaggedEpoch = l.Clock.Epoch()
	if repeat {
		// решение №12 (этап 5): повторное доказанное нарушение —
		// приостановка выпуска навсегда
		w.MintSuspendedUntilPeriod = ^uint64(0)
	} else {
		w.MintSuspendedUntilPeriod = period + l.State.Net.FlaggedMintSuspensionPeriods
	}
	// денежных штрафов и потери слотов нет (7.5); доля R поручителям
	// прекращается: mint проверяет w.Flagged (M2/R-условие)
	return accept(tx.TxID())
}

// ---------------------------------------------------------------------------
// GUARDIAN_OFFER (spec/03 3.8)
// ---------------------------------------------------------------------------

func (l *Ledger) applyGuardianOffer(tx *wire.Transaction) Result {
	w := l.State.Wallet(tx.Wallet)
	now := l.Clock.NetTime()
	// O1: active
	if w.Status != wire.StatusActive {
		return reject("E_GUARDIAN_STATUS")
	}
	// O2: возраст >= 12 мес. (11)
	if w.AgeDays(now) < l.State.Net.Profile.GuardianMinAgeDays {
		return reject("E_GUARDIAN_AGE")
	}
	w.PublicGuardian = true
	return accept(tx.TxID())
}

// ---------------------------------------------------------------------------
// UPDATE_GUARDIANS (11, решение владельца 21.10.2026)
// ---------------------------------------------------------------------------

func (l *Ledger) applyUpdateGuardians(tx *wire.Transaction) Result {
	up := tx.Payload.(*wire.UpdateGuardiansPayload)
	now := l.Clock.NetTime()

	switch {
	case up.Propose != nil:
		return l.applyUpdateGuardiansPropose(tx, up.Propose, now)
	case up.Cancel != nil:
		// отмена подписью владельца в окне 48 ч (сама подпись tx —
		// текущим ключом, C8); только явный ввод PIN — правило
		// приложения (этап 9)
		for _, pu := range l.State.PendingGuardianUpdates {
			if !pu.Canceled && !pu.Effective && !pu.Expired &&
				bytes.Equal(pu.Wallet, tx.Wallet) &&
				pu.ProposeSeq == up.Cancel.ProposeSeq {
				if now >= pu.EffectiveAt {
					return reject("E_GUARDIAN_UPDATE_EXPIRED")
				}
				pu.Canceled = true
				return accept(tx.TxID())
			}
		}
		return reject("E_GUARDIAN_UPDATE_NOTFOUND")
	}
	return reject("E_TYPE")
}

func (l *Ledger) applyUpdateGuardiansPropose(tx *wire.Transaction,
	p *wire.UpdateGuardiansPropose, now uint64) Result {
	w := l.State.Wallet(tx.Wallet)
	prof := l.State.Net.Profile

	// новый список: ≤ guardian_personal_count (как при register),
	// без дубликатов
	if uint64(len(p.NewGuardians)) > prof.GuardianPersonalCount {
		return reject("E_GUARDIAN_UPDATE")
	}
	seen := make(map[string]bool)
	for _, g := range p.NewGuardians {
		if seen[string(g)] {
			return reject("E_GUARDIAN_UPDATE")
		}
		seen[string(g)] = true
	}
	// новые гардианы: active, не flagged (№14), лимит подопечных №7
	for _, g := range p.NewGuardians {
		gw := l.State.Wallet(g)
		if gw == nil || gw.Status != wire.StatusActive || gw.Flagged {
			return reject("E_GUARDIAN_UPDATE")
		}
		if gw.Wards >= prof.GuardianPersonalMaxWards {
			return reject("E_GUARDIAN_UPDATE")
		}
	}
	// согласие: подпись каждым новым гардианом хеша смены
	stmt := &wire.GuardianChangeStatement{
		NetworkID:    l.State.Net.NetworkID,
		Ward:         tx.Wallet,
		NewGuardians: p.NewGuardians,
		Nonce:        p.Nonce,
	}
	sh := stmt.Hash()
	approved := make(map[string]bool)
	for i := range p.Approvals {
		ap := &p.Approvals[i]
		if !seen[string(ap.Guardian)] || approved[string(ap.Guardian)] {
			return reject("E_GUARDIAN_UPDATE")
		}
		gw := l.State.Wallet(ap.Guardian)
		if gw == nil || gw.PublicKey == nil ||
			!crypto.VerifyDER(gw.PublicKey, sh[:], ap.StatementSig) {
			return reject("E_GUARDIAN_UPDATE")
		}
		approved[string(ap.Guardian)] = true
	}
	for _, g := range p.NewGuardians {
		if !approved[string(g)] {
			return reject("E_GUARDIAN_UPDATE")
		}
	}
	// активная смена только одна
	for _, pu := range l.State.PendingGuardianUpdates {
		if bytes.Equal(pu.Wallet, tx.Wallet) &&
			!pu.Canceled && !pu.Effective && !pu.Expired {
			return reject("E_GUARDIAN_UPDATE_ACTIVE")
		}
	}
	// лимит смен: не более 150 вступивших в силу за всё время;
	// счётчик — из истории цепочки кошелька
	if w.GuardianChanges >= GuardianChangeLimit {
		return reject("E_GUARDIAN_CHANGE_LIMIT")
	}
	l.State.PendingGuardianUpdates = append(l.State.PendingGuardianUpdates, &PendingGuardianUpdate{
		Wallet:       append([]byte(nil), tx.Wallet...),
		ProposeSeq:   tx.Seq,
		NewGuardians: append([][]byte(nil), p.NewGuardians...),
		Approvals:    append([]wire.GuardianApproval(nil), p.Approvals...),
		Nonce:        p.Nonce,
		CreatedAt:    now,
		EffectiveAt:  now + GuardianChangeWindowSeconds,
	})
	return accept(tx.TxID())
}
