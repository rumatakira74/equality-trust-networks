// Валидация и применение транзакций: spec/03 (C1–C11, таблицы R/V/M/T/X/S/G/N/E/O).
// Каждый отказ — точный код E_* из spec/00 0.4. Эквивокация — не ошибка
// валидации: обе ветки проходят C1–C11, конфликт разрешает финализация (9.1).
package ledger

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/x509"
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
	// C9 (допуск timestamp ±30 мин, 9.2) — слой допуска сети (v9.13):
	// проверяется в net при приёме/распространении; ядро и replay C9 не
	// применяют — вердикт полностью детерминирован от txNow.
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
		// M7: окно челленджа выпуска (7.6). Период-якорь слота задаёт
		// манифест (поле 63 mint_slot_mode, тема fix-mint-window):
		// 0 — наследие (слот заявляемого mp.Period), 1 — слот периода
		// ВЫПУСКА (l.tt.period от txNow). Ранее окно ±48 ч считалось
		// вокруг слота заявленного периода: заявка периода P возможна
		// только после конца P (M4), поэтому окно пересекалось с
		// допустимым временем подачи лишь при слоте в последних 48 ч
		// периода — достижимость 1/15 за период, шесть пропусков подряд
		// давали diff > accrual_window навсегда (поглощающее состояние,
		// симуляция A9). В режиме 1 слот лежит внутри текущего периода:
		// пересечение окна ±48 ч с текущим периодом всегда >= 48 ч — у
		// каждого кошелька каждый период есть достижимое окно; слот
		// публично вычислим заранее.
		// Проверка вынесена в CheckMintWindow и включается опцией
		// (SkipMintChallengeWindow — только песочница).
		if tx.Type == wire.Mint && !l.Opts.SkipMintChallengeWindow {
			mp := tx.Payload.(*wire.MintPayload)
			if !l.CheckMintWindow(tx.Wallet, mp.Period, l.tt.now) {
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

// CheckMintChallengeWindow: слот H(address || period) внутри периода
// (period_start + H mod period_seconds) и окно ±48 ч (7.6). period —
// период-ЯКОРЬ слота (какой именно — решает MintWindowAnchor по режиму
// манифеста). Включение hash(чекпоинта предыдущей эпохи) в формулу
// слота — открытый вопрос, PENDING за владельцем; сейчас код считает
// H(address || period) (hashThree).
func (l *Ledger) CheckMintChallengeWindow(addr []byte, period, now uint64) bool {
	win := l.State.Net.Profile.MintChallengeWindowHours * 3600
	if win == 0 {
		return true
	}
	slot := mintChallengeSlot(l.State.GenesisTime, addr, period, l.State.Net.Profile.PeriodSecsOrDefault())
	lo, hi := slot-win, slot+win
	return now >= lo && now <= hi
}

// MintWindowAnchor — период-якорь слота челленджа выпуска по режиму
// манифеста (поле 63 mint_slot_mode, тема fix-mint-window, вариант A
// аудита Sonnet-2 п. 3.1/3.2):
//
//	0 (наследие) — заявляемый период claimedPeriod: окно ±48 ч вокруг
//	  слота внутри заявляемого периода; заявка возможна только после
//	  его конца (M4), поэтому достижимо лишь при слоте в последних
//	  48 ч периода (1/15 кошельков за период);
//	1 — период выпуска (период txNow): слот лежит внутри текущего
//	  периода, окно ±48 ч пересекается с допустимым временем подачи
//	  всегда — достижимо у каждого кошелька каждый период.
//
// Режим — часть network_id (манифест), поэтому вердикт детерминирован
// и одинаков на всех узлах сети.
func (l *Ledger) MintWindowAnchor(claimedPeriod, now uint64) uint64 {
	if l.State.Net.Profile.MintSlotMode == 1 {
		return l.periodAt(now)
	}
	return claimedPeriod
}

// CheckMintWindow — открыто ли окно челленджа для заявки периода
// claimedPeriod в момент now (M7, 7.6): слот периода-якоря по режиму
// манифеста (MintWindowAnchor) в пределах ±mint_challenge_window_hours.
// Единая точка входа для apply и для клиентских запросов (net/mobile) —
// UI и ядро приходят к одному вердикту.
func (l *Ledger) CheckMintWindow(addr []byte, claimedPeriod, now uint64) bool {
	return l.CheckMintChallengeWindow(addr, l.MintWindowAnchor(claimedPeriod, now), now)
}

// MintWindowBounds — ДОСТИЖИМЫЕ границы окна челленджа (unix-секунды,
// включительно) для заявки периода claimedPeriod в момент now:
// [slot−W, slot+W] вокруг слота периода-якоря, в режиме 1 усечённые
// периодом выпуска (аудит п. 3.1: «окно [S_p − W, S_p + W] усечено
// периодом p»). win == 0 (окно выключено) — весь интервал [now, now].
//
// Нужны клиенту для джиттера подачи (аудит п. 3.6: без джиттера пик 49× —
// окна кошельков со слотом в первые 48 ч периода обрезаны границей и
// открываются разом).
func (l *Ledger) MintWindowBounds(addr []byte, claimedPeriod, now uint64) (lo, hi uint64) {
	win := l.State.Net.Profile.MintChallengeWindowHours * 3600
	if win == 0 {
		return now, now
	}
	periodSecs := l.State.Net.Profile.PeriodSecsOrDefault()
	anchor := l.MintWindowAnchor(claimedPeriod, now)
	slot := mintChallengeSlot(l.State.GenesisTime, addr, anchor, periodSecs)
	if slot < win {
		lo = 0
	} else {
		lo = slot - win
	}
	hi = slot + win
	if l.State.Net.Profile.MintSlotMode == 1 {
		periodStart := l.State.GenesisTime + anchor*periodSecs
		if lo < periodStart {
			lo = periodStart
		}
		if periodEnd := periodStart + periodSecs; hi > periodEnd {
			hi = periodEnd
		}
	}
	return lo, hi
}

// mintChallengeSlot — детерминированный момент слота внутри периода:
// period_start + (H(...) mod period_seconds). Периоды сети — календарные
// от genesis (9.2): period_start = genesis_time + period × период сети
// (конвенция headless-ядра — 30 сут; genesis-поле 61 задаёт длину —
// тестовые сети живого прогона). Та же конвенция в net.RealClock.
// Какой период передать (якорь слота) — решает MintWindowAnchor по
// режиму манифеста (поле 63 mint_slot_mode): 0 — заявляемый период
// (наследие), 1 — период выпуска.
// Правка этапа 8 (долг этапа 7): раньше отсчёт шёл от 1970 г. — при
// genesis-времени близком к текущему слот выпадал в далёкое будущее/прошлое
// и окно челленджа всегда было закрыто.
func mintChallengeSlot(genesisTime uint64, addr []byte, period, periodSecs uint64) uint64 {
	h := hashThree(addr, period)
	periodStart := genesisTime + period*periodSecs
	return periodStart + h%periodSecs
}

// Apply применяет транзакцию: C1–C4 -> сеттлемент к txNow (подтверждается
// только принятием) -> C5–C11 -> типовая таблица -> эффекты.
//
// txNow = tx.TimestampClaim (v9.13): единая точка времени live и replay.
// Все часозависимые правила (M2/M4/M7, T3, V2, recurring-сгорание,
// evidence/сроки, ActivatedPeriod, модули) валидируются от txNow;
// periodAt/epochAt вычисляются от txNow арифметически от genesis.
// Вердикт не зависит от часов узла: одинаковый журнал даёт одинаковый
// state_root на любом узле и в replay. C9 (допуск ±30 мин) — слой
// допуска сети (net), ядро и replay его не применяют.
func (l *Ledger) Apply(tx *wire.Transaction) Result {
	txid := tx.TxID()

	if res := l.applyCommon(tx, txid); res.Outcome == Rejected {
		return res
	}
	txNow := tx.TimestampClaim
	l.tt = txTime{now: txNow, epoch: l.epochAt(txNow), period: l.periodAt(txNow)}
	// Сеттлемент к txNow до валидации: правила проверяются против
	// состояния, осевшего к моменту txNow (граница периода между
	// NetTime и txNow успевает сгореть/демереджиться). Подтверждается
	// только принятием: отклонённая tx не двигает сеттлемент — иначе
	// узлы, видевшие отклонённую tx, расходились бы с узлами и replay,
	// которые знают только журнал.
	advancing := txNow > l.State.NetTime
	var snap *State
	if advancing {
		// Предохранитель сеттлемента (H1 аудита Sonnet-2, шаг 3): сдвиг
		// времени реестра за один Apply ограничен MaxSettleShiftSeconds —
		// claim в далёком будущем иначе стоил бы O(эпох × кошельков) под
		// applyMu (анти-DoS). Граница аварийная: сетевые пути держат claim
		// в пределах допуска (net), честный сдвиг — периоды с прошлой
		// принятой tx. Отказ не двигает состояние (сеттлемент не начат).
		if txNow > l.State.NetTime+MaxSettleShiftSeconds {
			return reject("E_TIME_SHIFT")
		}
		snap = l.State.Clone()
		l.settleTo(txNow)
	}
	res := l.applyChecked(tx, txid)
	if res.Outcome == Rejected && advancing {
		l.State = snap
	}
	return res
}

// applyChecked — применение после сеттлемента к txNow (вызывающий
// держит l.tt; отклонение откатывает сеттлемент).
func (l *Ledger) applyChecked(tx *wire.Transaction, txid [32]byte) Result {
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
	now := l.tt.now
	epoch := l.tt.epoch

	// R1: фингерпринт уникален и не в чёрном списке (7.2, 11)
	fp := crypto.Fingerprint(p.AttestedPublicKey)
	if _, exists := l.State.FingerprintIndex[string(fp)]; exists {
		return reject("E_FINGERPRINT")
	}
	if l.State.FingerprintBlacklist[string(fp)] {
		return reject("E_FINGERPRINT")
	}
	// R2/R3: цепочка аттестации (7.2). Срок действия сертификатов
	// проверяется на момент tx.TimestampClaim (этап 14, A2-срок): листы
	// короткоживущие (~2 мес), replay старых регистраций обязан давать
	// тот же вердикт, что и live-apply, — поэтому не часы узла, а
	// timestamp_claim (он же пинирует replay, C9). Более широкий
	// txNow-рефакторинг — этап фикса времени (A7), не здесь.
	chain, err := attest.Verify(p.AttestationChain, l.TrustedRoots,
		l.State.Net.MinSecurityLevel, l.State.Net.MaxPatchAgeMonths,
		time.Unix(int64(tx.TimestampClaim), 0).UTC())
	if err != nil {
		if l.State.Net.Profile.DevicePolicy == wire.DevicePolicyStrict {
			return reject("E_ATTEST")
		}
		return reject("E_ATTEST_POLICY")
	}
	// R4c (этап 14, A1 — привязка листа): SPKI сертификата-листа равен
	// регистрируемому AttestedPublicKey (сравнение байтов SPKI).
	// Без этого цепочка, полученная для одного ключа, регистрирует
	// другой: фингерпринт и адрес считались бы от неподтверждённого
	// ключа («один человек ≈ один кошелёк» не работает). Идентичность
	// адресов не меняется: fingerprint остаётся от ключа, совпадающего
	// с листом. Проверка у каждого узла, replay даёт тот же вердикт.
	leafSPKI, err := x509.MarshalPKIXPublicKey(chain.Leaf.PublicKey)
	regPub, perr := crypto.ParsePubKey(p.AttestedPublicKey)
	if err != nil || perr != nil {
		return reject("E_ATTEST_KEY_MISMATCH")
	}
	regSPKI, err := x509.MarshalPKIXPublicKey(regPub)
	if err != nil || !bytes.Equal(leafSPKI, regSPKI) {
		return reject("E_ATTEST_KEY_MISMATCH")
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
	now := l.tt.now
	epoch := l.tt.epoch
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
		cand.ActivatedPeriod = l.tt.period
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
	period := l.tt.period
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
	// период, неиспользованное сгорает — заявляется один период,
	// накопления нет.
	//
	// Дефект M5 (аудит Sonnet-2 п. 3.3, тема fix-mint-window): требование
	// diff == 1 делало пропуск периода необратимым. Кошелёк, пропустивший
	// период, заявлял base+1 < p−1, а когорта периода base+1 сгорает по
	// правилу «перевод единицы периода P отвергается после границы P+1»
	// (recurring): каждый следующий выпуск мёртв при поступлении,
	// навсегда — отставание LastClaimedPeriod сбросить нечем (пишет
	// только этот обработчик и копия при миграции). Теперь заявляется
	// любой завершённый период mp.Period > LastClaimedPeriod (M3 выше)
	// суммой IssuanceAmount — один период, независимо от пропусков.
	if prof.IssuanceMode != wire.IssuanceQuotaReset && diff > prof.AccrualWindowPeriods {
		return reject("E_ACCRUAL_WINDOW")
	}
	// M6: amount = issuance × diff (10.1); quota_reset — всегда один
	// период (пропуски не накапливаются, M5 выше)
	want := prof.IssuanceAmount * diff
	if prof.IssuanceMode == wire.IssuanceQuotaReset {
		want = prof.IssuanceAmount
	}
	if mp.Amount != want {
		return reject("E_MINT_AMOUNT")
	}

	// эффекты (3.4): доля R из первых 12 выпусков (7.4). Для quota_reset
	// заявляется один период — цикл доли считается за один период
	// (mp.Period), а не за диапазон base+1..mp.Period (аудит п. 3.3).
	first := base + 1
	if prof.IssuanceMode == wire.IssuanceQuotaReset {
		first = mp.Period
	}
	shareTotal := uint64(0)
	for p := first; p <= mp.Period; p++ {
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
	w.LastTouchEpoch = l.tt.epoch
	return accept(tx.TxID())
}

// ---------------------------------------------------------------------------
// TRANSFER (spec/03 3.5)
// ---------------------------------------------------------------------------

func (l *Ledger) applyTransfer(tx *wire.Transaction) Result {
	tp := tx.Payload.(*wire.TransferPayload)
	w := l.State.Wallet(tx.Wallet)
	now := l.tt.now
	period := l.tt.period
	prof := l.State.Net.Profile

	switch {
	case tp.Send != nil:
		amount := tp.Send.Amount
		// T1: единица передаётся (раздел 4)
		if prof.TransferMode == wire.TransferModeSoulbound {
			return reject("E_TRANSFER_MODE")
		}
		// T2: restricted — только поставщику из списка (раздел 4);
		// peer-redeem — участник → участнику (погашение поставщику —
		// через redeem, 6.2; v9.10); членство поставщика — по
		// fingerprint (v9.11)
		if prof.TransferMode == wire.TransferModeRestricted {
			if !l.inProviders(tp.Send.Recipient) {
				return reject("E_TRANSFER_RECIPIENT")
			}
		}
		if prof.TransferMode == wire.TransferModePeerRedeem {
			rcpt := l.State.Wallet(tp.Send.Recipient)
			if rcpt == nil || rcpt.Terminal {
				return reject("E_TRANSFER_RECIPIENT")
			}
		}
		// T3: мораторий от register (7.6; статус не снимает)
		if now-w.RegisteredAt < prof.TransferMoratoriumDays*86400 {
			return reject("E_MORATORIUM")
		}
		fee := l.State.Net.Fee(amount, l.State.FeeRate)
		// T4/T8 (recurring): сгорание по периодной метке (v9.10, раздел 4)
		// — перевод единицы периода P отвергается после границы P+1;
		// отложенного пути нет (T6/T7 — только free-режим)
		if prof.Recurring {
			recipient := l.State.Wallet(tp.Send.Recipient)
			return l.recurringSend(tx, w, recipient, amount, fee, period)
		}
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
		} else {
			// получателя нет в реестре (адрес вне сети): единицы никому не
			// зачисляются — учитываются как сжигание, иначе INV-8
			// (сохранение предложения) нарушается тихо (аудит z-ai AUDIT-Z-02)
			l.State.TotalBurned += amount
		}
		if prof.FeeMode == wire.FeeModeBurn {
			l.State.TotalBurned += fee
		} else if prof.FeeMode == wire.FeeModePool {
			l.State.CommonFund += fee // общий фонд сети (раздел 4)
		}
		w.LastTouchEpoch = l.tt.epoch
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

// recurringSend — перевод/погашение в recurring-сети (v9.10, раздел 4):
// списание amount+fee из переводимых когорт (единица периода P переводима,
// пока текущий период ≤ P+1); получателю доли приходят с теми же метками
// периода, fee — по fee_mode. Баланс не сбрасывается: сгоревшие когорты
// остаются в кошельке, история публична и проверяема.
func (l *Ledger) recurringSend(tx *wire.Transaction, w, recipient *Wallet,
	amount, fee, period uint64) Result {
	if w.RecurringSpendable(period) < amount+fee {
		// остаток есть, но метки просрочены — сгоревшая единица (не E_BALANCE)
		if w.TotalBalance() >= amount+fee {
			return reject("E_BURNED")
		}
		return reject("E_BALANCE")
	}
	spent := w.RecurringSpend(amount+fee, period)
	// получателю — доли на сумму amount с метками периода отправителя
	var credit []Cohort
	remain := amount
	for _, c := range spent {
		if remain == 0 {
			break
		}
		take := c.Amount
		if take > remain {
			take = remain
		}
		credit = append(credit, Cohort{Period: c.Period, Amount: take})
		remain -= take
	}
	if recipient != nil {
		recipient.CreditCohorts(credit)
	} else {
		// получателя нет: зачисленные доли сгорают — единицы покинули
		// отправителя, INV-8 требует учёта (аудит z-ai AUDIT-Z-02)
		for _, c := range credit {
			l.State.TotalBurned += c.Amount
		}
	}
	// несписанное в получателя — комиссия (fee по построению)
	if prof := l.State.Net.Profile; fee > 0 {
		if prof.FeeMode == wire.FeeModeBurn {
			l.State.TotalBurned += fee
		} else if prof.FeeMode == wire.FeeModePool {
			l.State.CommonFund += fee
		}
	}
	w.LastTouchEpoch = l.tt.epoch
	return accept(tx.TxID())
}

// ---------------------------------------------------------------------------
// MIGRATE (spec/03 3.6)
// ---------------------------------------------------------------------------

func (l *Ledger) applyMigrate(tx *wire.Transaction) Result {
	mp := tx.Payload.(*wire.MigratePayload)
	now := l.tt.now
	epoch := l.tt.epoch
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
	period := l.tt.period
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
	w.FlaggedEpoch = l.tt.epoch
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
	now := l.tt.now
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
	now := l.tt.now

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
