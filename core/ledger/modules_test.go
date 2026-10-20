// Тесты модулей (раздел 6, этап 10): pool (6.3), redeem/receipt (6.2),
// contribution (6.4), quota_reset (раздел 4). Сценарии — spec/03 3.9,
// 3.11, 3.12; коды отказов — из таблиц P/D/K.
package ledger

import (
	"crypto/ecdsa"
	"testing"

	"github.com/rumatakira74/equality-trust-networks/core/crypto"
	"github.com/rumatakira74/equality-trust-networks/core/params"
	"github.com/rumatakira74/equality-trust-networks/core/wire"
)

// moduleEnv — окружение с включёнными модулями, поставщиком V1
// и ресурсом "water"; участники — pre-state active-кошельки с ликвидным
// балансом (мораторий пройден), без поручительств между собой.
type moduleEnv struct {
	*e2eEnv
}

func newModuleEnv(t *testing.T) *moduleEnv {
	t.Helper()
	gp := params.TestGenesis()
	p := gp.Manifest.Profile
	p.Modules = []uint32{wire.ModuleRedeem, wire.ModulePool, wire.ModuleContribution}
	p.ResourceID = "water"
	p.FreshTransferWindowPeriods = 0 // когорты сразу переводимы (тесты модулей)
	p.Pow = &wire.PowParams{
		TargetLight: 0x0010_0000_0000_0000, Argon2MemoryKiB: 8, Argon2Iterations: 1,
		Argon2Parallelism: 1, TargetHard: 0x0010_0000_0000_0000,
	}
	// поставщик — V1; адрес детерминирован от ключа (та же деривация e2e)
	v1 := crypto.DeterministicKey("e2e:V1")
	v1pub := crypto.MarshalPubKey(&v1.PublicKey)
	v1addr := crypto.DeriveAddress(crypto.Fingerprint(v1pub), gp.Genesis.NetworkID)
	p.Providers = [][]byte{v1addr}

	mh := gp.Manifest.ManifestHash()
	gp.Genesis.ManifestHash = mh[:]
	clock := &FixedClock{T: 1_700_000_000, E: 1, P: 1}
	l, err := New(gp, clock, Options{SkipMintChallengeWindow: true})
	if err != nil {
		t.Fatalf("genesis: %v", err)
	}
	e := &e2eEnv{t: t, ledger: l, clock: clock,
		keys: make(map[string]*ecdsa.PrivateKey), addrs: make(map[string][]byte)}
	for _, id := range []string{"V1", "V2", "V3", "ALICE", "BOB", "CAROL"} {
		e.preStateWallet(id)
	}
	// фонд участников — реальные выпуски (INV-8): период 11 при текущем 12
	e.clock.P = 12
	for _, id := range []string{"V1", "V2", "V3", "ALICE", "BOB", "CAROL"} {
		mint := e.tx(id, wire.Mint, &wire.MintPayload{Period: 11, Amount: 1000}, true)
		e.requireAccepted(e.ledger.Apply(mint), "mint "+id)
	}
	return &moduleEnv{e}
}

// preStateWallet — активный кошелёк с ликвидным балансом (мораторий
// пройден), вне графа поручительств.
func (e *e2eEnv) preStateWallet(id string) *Wallet {
	priv := e.key(id)
	pub := crypto.MarshalPubKey(&priv.PublicKey)
	fp := crypto.Fingerprint(pub)
	addr := e.addr(id)
	w := &Wallet{
		Address: addr, Fingerprint: fp, PublicKey: &priv.PublicKey,
		Status:          wire.StatusActive, Seed: true,
		RegisteredAt:    e.clock.NetTime() - 400*86400,
		ActivatedAt:     e.clock.NetTime() - 400*86400,
		ActivatedPeriod: 1, LastClaimedPeriod: 10,
		LastTouchEpoch: 1, DelayedThreshold: 3000, Seq: 1,
	}
	w.TipTxID = make([]byte, 32)
	copy(w.TipTxID, []byte("pre:"+id))
	e.ledger.State.Wallets[string(addr)] = w
	e.ledger.State.FingerprintIndex[string(fp)] = addr
	return w
}

// preStateRegistered — зарегистрированный кандидат (для реальных vouch).
func (e *e2eEnv) preStateRegistered(id string) *Wallet {
	priv := e.key(id)
	pub := crypto.MarshalPubKey(&priv.PublicKey)
	fp := crypto.Fingerprint(pub)
	addr := e.addr(id)
	w := &Wallet{
		Address: addr, Fingerprint: fp, PublicKey: &priv.PublicKey,
		Status:         wire.StatusRegistered,
		RegisteredAt:   e.clock.NetTime() - 400*86400,
		LastTouchEpoch: 1, DelayedThreshold: 3000, Seq: 1,
	}
	w.TipTxID = make([]byte, 32)
	copy(w.TipTxID, []byte("pre:"+id))
	e.ledger.State.Wallets[string(addr)] = w
	e.ledger.State.FingerprintIndex[string(fp)] = addr
	return w
}

func (m *moduleEnv) rejected(res Result, code, what string) {
	m.t.Helper()
	if res.Outcome != Rejected || res.Code != code {
		m.t.Fatalf("%s: %v/%s, ожидается reject/%s", what, res.Outcome, res.Code, code)
	}
}

// TestPoolLifecycle — пул release_on_goal: create → donate до цели →
// выпуск равными долями → повторный claim отклоняется (P1–P4).
func TestPoolLifecycle(t *testing.T) {
	m := newModuleEnv(t)
	e := m.e2eEnv
	// P1: create {получатель V1, цель 300, дедлайн, режимы}
	create := e.tx("ALICE", wire.PoolCreate, &wire.PoolCreatePayload{
		Recipient: e.addr("V1"), GoalAmount: 300, DeadlineEpoch: e.clock.Epoch() + 10,
		ReleaseMode: wire.PoolReleaseOnGoal, RefundMode: wire.PoolRefundIfUnmet,
	}, false)
	e.requireAccepted(e.ledger.Apply(create), "pool_create")
	poolID := create.TxID()

	// donate 100 — цель не достигнута, claim закрыт
	donate1 := e.tx("ALICE", wire.PoolDonate,
		&wire.PoolDonatePayload{PoolID: poolID[:], Amount: 100}, false)
	e.requireAccepted(e.ledger.Apply(donate1), "donate 1")
	claim0 := e.tx("ALICE", wire.PoolClaim, &wire.PoolClaimPayload{PoolID: poolID[:]}, false)
	m.rejected(e.ledger.Apply(claim0), "E_POOL_STATE", "claim до цели")

	// donate 200 — цель достигнута: выпуск открыт, доля = 300/5 = 60
	donate2 := e.tx("BOB", wire.PoolDonate,
		&wire.PoolDonatePayload{PoolID: poolID[:], Amount: 200}, false)
	e.requireAccepted(e.ledger.Apply(donate2), "donate 2")
	pool := e.ledger.State.Pools[string(poolID[:])]
	if !pool.Released || pool.Share != 50 {
		t.Fatalf("пул не выпущен или доля неверна: released=%v share=%d", pool.Released, pool.Share)
	}
	// P4: равные доли active-кошелькам; повтор — E_POOL_CLAIMED
	aliceBefore := e.ledger.State.Wallet(e.addr("ALICE")).Liquid
	claim1 := e.tx("ALICE", wire.PoolClaim, &wire.PoolClaimPayload{PoolID: poolID[:]}, false)
	e.requireAccepted(e.ledger.Apply(claim1), "claim ALICE")
	if got := e.ledger.State.Wallet(e.addr("ALICE")).Liquid - aliceBefore; got != 50 {
		t.Fatalf("доля ALICE: %d, ожидается 50", got)
	}
	claim2 := e.tx("ALICE", wire.PoolClaim, &wire.PoolClaimPayload{PoolID: poolID[:]}, false)
	m.rejected(e.ledger.Apply(claim2), "E_POOL_CLAIMED", "повторный claim")
	// P3: право закрытия — создатель (№26, предположение)
	close1 := e.tx("BOB", wire.PoolClose, &wire.PoolClosePayload{PoolID: poolID[:]}, false)
	m.rejected(e.ledger.Apply(close1), "E_POOL_STATE", "close не создателем")
	// выпущенный пул не закрывается
	close2 := e.tx("ALICE", wire.PoolClose, &wire.PoolClosePayload{PoolID: poolID[:]}, false)
	m.rejected(e.ledger.Apply(close2), "E_POOL_STATE", "close выпущенного")
}

// TestPoolDeadlineAndRefund — release_on_deadline открывается на границе
// эпохи после дедлайна; refund_if_unmet при закрытии возвращает донатам (P5).
func TestPoolDeadlineAndRefund(t *testing.T) {
	m := newModuleEnv(t)
	e := m.e2eEnv
	create := e.tx("ALICE", wire.PoolCreate, &wire.PoolCreatePayload{
		Recipient: e.addr("V1"), GoalAmount: 1_000_000, DeadlineEpoch: e.clock.Epoch() + 1,
		ReleaseMode: wire.PoolReleaseOnDeadline, RefundMode: wire.PoolRefundIfUnmet,
	}, false)
	e.requireAccepted(e.ledger.Apply(create), "pool_create")
	poolID := create.TxID()
	donate := e.tx("ALICE", wire.PoolDonate,
		&wire.PoolDonatePayload{PoolID: poolID[:], Amount: 50}, false)
	e.requireAccepted(e.ledger.Apply(donate), "donate")

	// граница эпохи: дедлайн прошёл — выпуск открыт, доля = 50/5 = 10
	e.clock.E += 2
	e.clock.T += 2 * 86400
	e.ledger.syncTimeForTest()
	pool := e.ledger.State.Pools[string(poolID[:])]
	if !pool.Released || pool.Share != 8 {
		t.Fatalf("deadline release: released=%v share=%d", pool.Released, pool.Share)
	}
	claim := e.tx("CAROL", wire.PoolClaim, &wire.PoolClaimPayload{PoolID: poolID[:]}, false)
	e.requireAccepted(e.ledger.Apply(claim), "claim после дедлайна")

	// отдельный пул с возвратом: цель не достигнута, close — донатам
	create2 := e.tx("BOB", wire.PoolCreate, &wire.PoolCreatePayload{
		Recipient: e.addr("V1"), GoalAmount: 1_000_000, DeadlineEpoch: e.clock.Epoch() + 10,
		ReleaseMode: wire.PoolReleaseOnGoal, RefundMode: wire.PoolRefundIfUnmet,
	}, false)
	e.requireAccepted(e.ledger.Apply(create2), "pool_create 2")
	pool2ID := create2.TxID()
	donate2 := e.tx("CAROL", wire.PoolDonate,
		&wire.PoolDonatePayload{PoolID: pool2ID[:], Amount: 77}, false)
	e.requireAccepted(e.ledger.Apply(donate2), "donate 2")
	carolBefore := e.ledger.State.Wallet(e.addr("CAROL")).Liquid
	close3 := e.tx("BOB", wire.PoolClose, &wire.PoolClosePayload{PoolID: pool2ID[:]}, false)
	e.requireAccepted(e.ledger.Apply(close3), "close с возвратом")
	if got := e.ledger.State.Wallet(e.addr("CAROL")).Liquid - carolBefore; got != 77 {
		t.Fatalf("возврат донату: %d, ожидается 77", got)
	}
	if p2 := e.ledger.State.Pools[string(pool2ID[:])]; !p2.Closed || p2.Balance != 0 {
		t.Fatalf("пул не закрыт или баланс не нулевой: closed=%v balance=%d", p2.Closed, p2.Balance)
	}
}

// TestRedeemReceipt — погашение с resource_id и подписанная квитанция
// (D1–D5).
func TestRedeemReceipt(t *testing.T) {
	m := newModuleEnv(t)
	e := m.e2eEnv
	v1 := e.ledger.State.Wallet(e.addr("V1"))

	// D2: чужой resource_id отклоняется
	bad1 := e.tx("ALICE", wire.Redeem, &wire.RedeemPayload{
		Provider: e.addr("V1"), ResourceID: "bread", Amount: 10,
	}, false)
	m.rejected(e.ledger.Apply(bad1), "E_REDEEM_RESOURCE", "чужой ресурс")
	// D1: не поставщик
	bad2 := e.tx("ALICE", wire.Redeem, &wire.RedeemPayload{
		Provider: e.addr("BOB"), ResourceID: "water", Amount: 10,
	}, false)
	m.rejected(e.ledger.Apply(bad2), "E_REDEEM_PROVIDER", "не поставщик")
	// D4: слот в прошлом
	bad3 := e.tx("ALICE", wire.Redeem, &wire.RedeemPayload{
		Provider: e.addr("V1"), ResourceID: "water", Amount: 10,
		Slot: &wire.TimeSlot{Start: e.clock.NetTime() - 7200, End: e.clock.NetTime() - 3600},
	}, false)
	m.rejected(e.ledger.Apply(bad3), "E_REDEEM_SLOT", "слот в прошлом")

	// корректное погашение: поставщик получает ровно amount
	redeem := e.tx("ALICE", wire.Redeem, &wire.RedeemPayload{
		Provider: e.addr("V1"), ResourceID: "water", Amount: 10,
	}, false)
	e.requireAccepted(e.ledger.Apply(redeem), "redeem")
	if v1.TotalBalance() != 1010 {
		t.Fatalf("баланс поставщика: %d", v1.TotalBalance())
	}
	redeemID := redeem.TxID()

	// D5: квитанция поставщиком — запись без изменения балансов
	receipt := e.tx("V1", wire.Receipt, &wire.ReceiptPayload{
		RedeemTxID: redeemID[:], Amount: 10, ResourceID: "water",
	}, false)
	e.requireAccepted(e.ledger.Apply(receipt), "receipt")
	if !e.ledger.State.Receipts[string(redeemID[:])] {
		t.Fatalf("квитанция не записана")
	}
	// повторная и чужая квитанции отклоняются
	receipt2 := e.tx("V1", wire.Receipt, &wire.ReceiptPayload{
		RedeemTxID: redeemID[:], Amount: 10, ResourceID: "water",
	}, false)
	m.rejected(e.ledger.Apply(receipt2), "E_RECEIPT_MISMATCH", "повторная квитанция")
	receipt3 := e.tx("BOB", wire.Receipt, &wire.ReceiptPayload{
		RedeemTxID: redeemID[:], Amount: 10, ResourceID: "water",
	}, false)
	m.rejected(e.ledger.Apply(receipt3), "E_RECEIPT_MISMATCH", "квитанция не поставщиком")
	receipt4 := e.tx("V1", wire.Receipt, &wire.ReceiptPayload{
		RedeemTxID: redeemID[:], Amount: 999, ResourceID: "water",
	}, false)
	m.rejected(e.ledger.Apply(receipt4), "E_RECEIPT_MISMATCH", "квитанция с другой суммой")
}

// TestContributionFlow — заявка вклада, peer-подтверждения (не связанные
// поручительствами), лимит на человека в период, выплата из пула
// программы (K1–K6).
func TestContributionFlow(t *testing.T) {
	m := newModuleEnv(t)
	e := m.e2eEnv
	// пул программы: получатель V1, release_continuous
	poolCreate := e.tx("V1", wire.PoolCreate, &wire.PoolCreatePayload{
		Recipient: e.addr("V1"), GoalAmount: 0, DeadlineEpoch: e.clock.Epoch() + 100,
		ReleaseMode: wire.PoolReleaseContinuous, RefundMode: wire.PoolRefundNone,
	}, false)
	e.requireAccepted(e.ledger.Apply(poolCreate), "pool_create")
	poolID := poolCreate.TxID()
	fund := e.tx("V1", wire.PoolDonate,
		&wire.PoolDonatePayload{PoolID: poolID[:], Amount: 500}, false)
	e.requireAccepted(e.ledger.Apply(fund), "пополнение пула")

	// K1: заявка за пределами окна (период +2) отклоняется
	early := e.tx("ALICE", wire.ContributionClaim, &wire.ContributionClaimPayload{
		PoolID: poolID[:], DescriptionHash: []byte("desc-1"), Size: 100,
		Period: e.clock.Period() + 2,
	}, false)
	m.rejected(e.ledger.Apply(early), "E_CONTRIBUTION_PERIOD", "период вне окна")

	claim := e.tx("ALICE", wire.ContributionClaim, &wire.ContributionClaimPayload{
		PoolID: poolID[:], DescriptionHash: []byte("desc-1"), Size: 100,
		Period: e.clock.Period(),
	}, false)
	e.requireAccepted(e.ledger.Apply(claim), "contribution_claim")
	claimID := claim.TxID()

	// K5: дубль отклоняется; K6: N=2 подтверждений принимают заявку
	confirm1 := e.tx("BOB", wire.ContributionConfirm,
		&wire.ContributionConfirmPayload{ClaimTxID: claimID[:]}, false)
	e.requireAccepted(e.ledger.Apply(confirm1), "confirm BOB")
	confirm1b := e.tx("BOB", wire.ContributionConfirm,
		&wire.ContributionConfirmPayload{ClaimTxID: claimID[:]}, false)
	m.rejected(e.ledger.Apply(confirm1b), "E_CONTRIBUTION_DUP", "дубль подтверждения")

	aliceBefore := e.ledger.State.Wallet(e.addr("ALICE")).Liquid
	confirm2 := e.tx("CAROL", wire.ContributionConfirm,
		&wire.ContributionConfirmPayload{ClaimTxID: claimID[:]}, false)
	e.requireAccepted(e.ledger.Apply(confirm2), "confirm CAROL")
	if got := e.ledger.State.Wallet(e.addr("ALICE")).Liquid - aliceBefore; got != 100 {
		t.Fatalf("выплата вклада: %d, ожидается 100", got)
	}
	st := e.ledger.State.Claims[string(claimID[:])]
	if !st.Confirmed || !st.Paid {
		t.Fatalf("заявка не подтверждена/не выплачена")
	}
	// подтверждение принятой заявки отклоняется
	confirm3 := e.tx("V2", wire.ContributionConfirm,
		&wire.ContributionConfirmPayload{ClaimTxID: claimID[:]}, false)
	m.rejected(e.ledger.Apply(confirm3), "E_CONTRIBUTION_DUP", "подтверждение после выплаты")

	// K3: связанный поручительством подтверждать не может — реальное ребро:
	// DORA регистрируется и активируется тремя vouch (V1 среди них)
	e.preStateRegistered("DORA")
	for _, v := range []string{"V1", "V2", "V3"} {
		vouch := e.tx(v, wire.Vouch, &wire.VouchPayload{
			Candidate: e.addr("DORA"), Epoch: e.clock.Epoch(),
		}, false)
		e.requireAccepted(e.ledger.Apply(vouch), "vouch "+v+"→DORA")
	}
	claim2 := e.tx("DORA", wire.ContributionClaim, &wire.ContributionClaimPayload{
		PoolID: poolID[:], DescriptionHash: []byte("desc-2"), Size: 50,
		Period: e.clock.Period(),
	}, false)
	e.requireAccepted(e.ledger.Apply(claim2), "claim 2")
	claim2ID := claim2.TxID()
	confirmLinked := e.tx("V1", wire.ContributionConfirm,
		&wire.ContributionConfirmPayload{ClaimTxID: claim2ID[:]}, false)
	m.rejected(e.ledger.Apply(confirmLinked), "E_CONTRIBUTION_LINKED", "связанный подтверждает")
	confirmSelf := e.tx("DORA", wire.ContributionConfirm,
		&wire.ContributionConfirmPayload{ClaimTxID: claim2ID[:]}, false)
	m.rejected(e.ledger.Apply(confirmSelf), "E_CONTRIBUTION_LINKED", "самоподтверждение")

	// K4: лимит подтверждений на человека в период (5, предположение №13)
	for i := 0; i < ContributionConfirmLimit-1; i++ {
		c := e.tx("ALICE", wire.ContributionClaim, &wire.ContributionClaimPayload{
			PoolID: poolID[:], DescriptionHash: []byte("many"), Size: 1,
			Period: e.clock.Period(),
		}, false)
		e.requireAccepted(e.ledger.Apply(c), "claim filler")
		cID := c.TxID()
		cf := e.tx("BOB", wire.ContributionConfirm,
			&wire.ContributionConfirmPayload{ClaimTxID: cID[:]}, false)
		e.requireAccepted(e.ledger.Apply(cf), "confirm filler")
	}
	last := e.tx("ALICE", wire.ContributionClaim, &wire.ContributionClaimPayload{
		PoolID: poolID[:], DescriptionHash: []byte("over"), Size: 1,
		Period: e.clock.Period(),
	}, false)
	e.requireAccepted(e.ledger.Apply(last), "claim over")
	lastID := last.TxID()
	over := e.tx("BOB", wire.ContributionConfirm,
		&wire.ContributionConfirmPayload{ClaimTxID: lastID[:]}, false)
	m.rejected(e.ledger.Apply(over), "E_CONTRIBUTION_LIMIT", "лимит подтверждений")
}

// TestQuotaReset — квота сгорает в конце периода: выпуск ровно одного
// периода, неиспользованная когорта сжигается на границе.
func TestQuotaReset(t *testing.T) {
	gp := params.TestGenesis()
	p := gp.Manifest.Profile
	p.IssuanceMode = wire.IssuanceQuotaReset
	p.AccrualWindowPeriods = 0
	p.Pow = &wire.PowParams{
		TargetLight: 0x0010_0000_0000_0000, Argon2MemoryKiB: 8, Argon2Iterations: 1,
		Argon2Parallelism: 1, TargetHard: 0x0010_0000_0000_0000,
	}
	mh := gp.Manifest.ManifestHash()
	gp.Genesis.ManifestHash = mh[:]
	clock := &FixedClock{T: 1_700_000_000, E: 1, P: 2}
	l, err := New(gp, clock, Options{SkipMintChallengeWindow: true})
	if err != nil {
		t.Fatalf("genesis: %v", err)
	}
	e := &e2eEnv{t: t, ledger: l, clock: clock,
		keys: make(map[string]*ecdsa.PrivateKey), addrs: make(map[string][]byte)}
	w := e.preStateWallet("ALICE")
	w.ActivatedPeriod = 1
	w.LastClaimedPeriod = 0

	mrej := func(res Result, code, what string) {
		if res.Outcome != Rejected || res.Code != code {
			t.Fatalf("%s: %v/%s, ожидается reject/%s", what, res.Outcome, res.Code, code)
		}
	}
	// квота одного периода
	mint := e.tx("ALICE", wire.Mint, &wire.MintPayload{Period: 1, Amount: 1000}, true)
	e.requireAccepted(l.Apply(mint), "mint квоты")
	if w.TotalBalance() != 1000 {
		t.Fatalf("квота не начислена: %d", w.TotalBalance())
	}
	// накопления нет: пропуск периода и заявка двух сразу — отказ
	clock.P = 5
	clock.T += 3 * 31 * 86400
	l.syncTimeForTest()
	multi := e.tx("ALICE", wire.Mint, &wire.MintPayload{Period: 4, Amount: 2000}, true)
	mrej(l.Apply(multi), "E_ACCRUAL_WINDOW", "мульти-периодная квота")
	// граница периода: неиспользованная квота сгорела (период 1 < 5)
	if w.TotalBalance() != 0 {
		t.Fatalf("квота не сгорела: %d", w.TotalBalance())
	}
	if l.State.TotalBurned != 1000 {
		t.Fatalf("сгоревшее не учтено: %d", l.State.TotalBurned)
	}
}

// TestModuleGating — операции выключенного модуля отклоняются (E_MODULE).
func TestModuleGating(t *testing.T) {
	gp := params.TestGenesis()
	gp.Manifest.Profile.Pow = &wire.PowParams{
		TargetLight: 0x0010_0000_0000_0000, Argon2MemoryKiB: 8, Argon2Iterations: 1,
		Argon2Parallelism: 1, TargetHard: 0x0010_0000_0000_0000,
	}
	mh := gp.Manifest.ManifestHash()
	gp.Genesis.ManifestHash = mh[:]
	clock := &FixedClock{T: 1_700_000_000, E: 1, P: 1}
	l, err := New(gp, clock, Options{SkipMintChallengeWindow: true})
	if err != nil {
		t.Fatalf("genesis: %v", err)
	}
	e := &e2eEnv{t: t, ledger: l, clock: clock,
		keys: make(map[string]*ecdsa.PrivateKey), addrs: make(map[string][]byte)}
	e.preStateWallet("ALICE")
	create := e.tx("ALICE", wire.PoolCreate, &wire.PoolCreatePayload{
		Recipient: e.addr("ALICE"), GoalAmount: 1, DeadlineEpoch: 99,
		ReleaseMode: wire.PoolReleaseOnGoal, RefundMode: wire.PoolRefundNone,
	}, false)
	res := l.Apply(create)
	if res.Outcome != Rejected || res.Code != "E_MODULE" {
		t.Fatalf("выключенный модуль: %v/%s", res.Outcome, res.Code)
	}
}
