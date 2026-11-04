// Тесты фиксов аудита z-ai (REPORT-AUDIT-Z-AI, ветка audit-z-ai):
// AUDIT-Z-01 — детерминированный возврат донатов пула при дефиците;
// AUDIT-Z-02 — учёт сжигания при переводе/исполнении «в никуда» (INV-8);
// AUDIT-Z-03 — исполнение отложенного перевода при байте '|' в seq;
// AUDIT-Z-06 — запрет recurring + модули pool/contribution.
package ledger

import (
	"bytes"
	"crypto/ecdsa"
	"testing"

	"github.com/rumatakira74/equality-trust-networks/core/attest"
	"github.com/rumatakira74/equality-trust-networks/core/crypto"
	"github.com/rumatakira74/equality-trust-networks/core/params"
	"github.com/rumatakira74/equality-trust-networks/core/profiles"
	"github.com/rumatakira74/equality-trust-networks/core/wire"
)

// auditE2E — e2e-окружение с включёнными модулями (pool/contribution) и
// провайдером из параметров сети (для pool_create).
func auditE2E(t *testing.T, modules ...uint32) *e2eEnv {
	gp := e2eGenesis()
	gp.Manifest.Profile.Pow = &wire.PowParams{
		TargetLight: 0x0010_0000_0000_0000, Argon2MemoryKiB: 8, Argon2Iterations: 1,
		Argon2Parallelism: 1, TargetHard: 0x0010_0000_0000_0000,
	}
	gp.Manifest.Profile.Modules = modules
	// провайдер pool_create: существующий кошелёк + fingerprint в providers
	provKey := crypto.DeterministicKey("audit-z:provider")
	gp.Manifest.Profile.Providers = append(gp.Manifest.Profile.Providers,
		crypto.Fingerprint(crypto.MarshalPubKey(&provKey.PublicKey)))
	wire.SetManifestHash(gp)
	l, err := New(gp, Options{SkipMintChallengeWindow: true})
	if err != nil {
		t.Fatalf("genesis: %v", err)
	}
	env := &e2eEnv{
		t: t, ledger: l, now: e2eT0,
		keys:  make(map[string]*ecdsa.PrivateKey),
		addrs: make(map[string][]byte),
	}
	env.rootKey = crypto.DeterministicKey("e2e-root")
	cert, _, err := attest.SyntheticRoot("e2e", env.rootKey)
	if err != nil {
		t.Fatalf("root: %v", err)
	}
	env.rootCert = cert
	l.TrustedRoots = append(l.TrustedRoots, cert)
	// ключ провайдера подменяется на детерминированный из genesis: адрес
	// обязан соответствовать fingerprint'у в providers
	env.keys["provider"] = provKey
	delete(env.addrs, "provider")
	return env
}

// fund — white-box ввод переводимых средств (тестам не нужен полный путь
// mint; ввод учитывается в TotalMinted, чтобы INV-8 оставался осмысленным.
// Детали эмиссии покрывают e2e/mint-тесты).
func (e *e2eEnv) fund(id string, amount uint64) {
	w := e.ledger.State.Wallet(e.addr(id))
	w.Liquid += amount
	e.ledger.State.TotalMinted += amount
}

// totalBalance — сумма балансов всех кошельков и пулов (для
// INV-8-проверок; баланс пула — часть предложения).
func (e *e2eEnv) totalBalance() uint64 {
	var sum uint64
	for _, w := range e.ledger.State.Wallets {
		sum += w.TotalBalance()
	}
	for _, p := range e.ledger.State.Pools {
		sum += p.Balance
	}
	return sum
}

// TestAuditZPoolRefundDeterministicOnDeficit (AUDIT-Z-01): при
// Balance < ΣDonations возврат идёт в порядке сортировки адресов доноров,
// итог одинаков у любых узлов.
func TestAuditZPoolRefundDeterministicOnDeficit(t *testing.T) {
	e := auditE2E(t, wire.ModulePool, wire.ModuleContribution)
	if res := e.register("provider"); res.Outcome != Accepted {
		t.Fatalf("provider: %s", res.Code)
	}
	for _, id := range []string{"a", "b"} {
		if res := e.register(id); res.Outcome != Accepted {
			t.Fatalf("%s: %s", id, res.Code)
		}
		e.fund(id, 500)
	}
	// мораторий 14 дней — двигаем время
	e.now += 15 * 86400

	create := e.tx("a", wire.PoolCreate, &wire.PoolCreatePayload{
		Recipient:     e.addr("provider"),
		GoalAmount:    500,
		DeadlineEpoch: e.epoch() + 10,
		ReleaseMode:   wire.PoolReleaseOnDeadline,
		RefundMode:    wire.PoolRefundIfUnmet,
	}, false)
	if res := e.ledger.Apply(create); res.Outcome != Accepted {
		t.Fatalf("pool_create: %s", res.Code)
	}
	poolID := create.TxID()
	for _, tc := range []struct {
		id     string
		amount uint64
	}{{"a", 60}, {"b", 40}} {
		donate := e.tx(tc.id, wire.PoolDonate, &wire.PoolDonatePayload{
			PoolID: poolID[:], Amount: tc.amount,
		}, false)
		if res := e.ledger.Apply(donate); res.Outcome != Accepted {
			t.Fatalf("donate %s: %s", tc.id, res.Code)
		}
	}
	// дефицит: пул опустошён contribution-выплатой (здесь — напрямую,
	// тест фиксирует порядок возврата, не путь опустошения)
	pool := e.ledger.State.Pools[string(poolID[:])]
	pool.Balance = 70
	sumBefore := e.totalBalance()

	closeTx := e.tx("a", wire.PoolClose, &wire.PoolClosePayload{PoolID: poolID[:]}, false)
	if res := e.ledger.Apply(closeTx); res.Outcome != Accepted {
		t.Fatalf("pool_close: %s", res.Code)
	}
	// итог обязан соответствовать детерминированному порядку (сортировка
	// адресов доноров): первый по адресу получает свой донат целиком,
	// второй — остаток. Учтена комиссия доната (fee_min = 1, burn).
	first, second := "a", "b"
	if bytes.Compare(e.addr("b"), e.addr("a")) < 0 {
		first, second = "b", "a"
	}
	if got := e.ledger.State.Wallet(e.addr(first)).Liquid; got != 499 {
		t.Fatalf("%s: liquid %d, ожидается 499 (500 − донат 61 + возврат 60)", first, got)
	}
	if got := e.ledger.State.Wallet(e.addr(second)).Liquid; got != 469 {
		t.Fatalf("%s: liquid %d, ожидается 469 (500 − донат 41 + усечённый возврат 10)", second, got)
	}
	if pool.Balance != 0 {
		t.Fatalf("остаток пула %d, ожидается 0", pool.Balance)
	}
	if e.totalBalance() != sumBefore { // средства не исчезли
		t.Fatalf("сумма балансов изменилась: %d → %d", sumBefore, e.totalBalance())
	}
}

// TestAuditZTransferToUnknownAddressBurns (AUDIT-Z-02): перевод «в
// никуда» учитывается как сжигание — INV-8 остаётся целым.
func TestAuditZTransferToUnknownAddressBurns(t *testing.T) {
	e := newE2E(t)
	if res := e.register("a"); res.Outcome != Accepted {
		t.Fatalf("register: %s", res.Code)
	}
	e.fund("a", 1000)
	e.now += 15 * 86400 // мораторий

	burnedBefore := e.ledger.State.TotalBurned
	unknown := make([]byte, 32) // валидный формат адреса, кошелька нет
	for i := range unknown {
		unknown[i] = byte(0xA0 + i%16)
	}
	tx := e.tx("a", wire.Transfer, &wire.TransferPayload{Send: &wire.TransferSend{
		Recipient: unknown, Amount: 100,
	}}, false)
	if res := e.ledger.Apply(tx); res.Outcome != Accepted {
		t.Fatalf("transfer: %s", res.Code)
	}
	if e.ledger.State.TotalBurned <= burnedBefore {
		t.Fatalf("TotalBurned %d не вырос — единицы уничтожены без учёта", e.ledger.State.TotalBurned)
	}
	if bad := e.ledger.CheckInvariants(); len(bad) > 0 {
		t.Fatalf("INV-8 нарушен: %v", bad)
	}
}

// TestAuditZWalletByKeyPipeByteInSeq (AUDIT-Z-03): ключ отложенного
// перевода с байтом '|' в seq (seq = 124) разбирается по фиксированному
// смещению — исполнение по сроку работает, средства не застревают.
func TestAuditZWalletByKeyPipeByteInSeq(t *testing.T) {
	e := newE2E(t)
	if res := e.register("a"); res.Outcome != Accepted {
		t.Fatalf("register a: %s", res.Code)
	}
	if res := e.register("b"); res.Outcome != Accepted {
		t.Fatalf("register b: %s", res.Code)
	}
	e.fund("a", 5000)
	e.now += 15 * 86400

	// seq = 124: младший BE-байт = 0x7C == '|'
	seq := uint64(124)
	key := ptKey(e.addr("a"), seq)
	e.ledger.State.PendingTransfers[key] = &PendingTransfer{
		Seq: seq, Recipient: append([]byte(nil), e.addr("b")...),
		Amount: 100, Fee: 1, CreatedAt: e.now,
		Deadline: e.now + 48*3600,
	}
	if got := e.ledger.State.WalletByKey(key); got != e.ledger.State.Wallet(e.addr("a")) {
		t.Fatalf("WalletByKey вернул не отправителя (байт '|' в seq)")
	}
	e.ledger.State.Wallet(e.addr("a")).Locked += 101

	e.now += 49 * 3600
	e.settle()
	pt := e.ledger.State.PendingTransfers[key]
	if !pt.Executed {
		t.Fatalf("отложенный перевод не исполнен (sender не найден?)")
	}
	if w := e.ledger.State.Wallet(e.addr("a")); w.Locked != 0 {
		t.Fatalf("Locked отправителя %d, ожидается 0", w.Locked)
	}
	if w := e.ledger.State.Wallet(e.addr("b")); w.Liquid != 100 {
		t.Fatalf("получатель: liquid %d, ожидается 100", w.Liquid)
	}
	if bad := e.ledger.CheckInvariants(); len(bad) > 0 {
		t.Fatalf("инварианты нарушены: %v", bad)
	}
}

// TestAuditZRecurringPoolModuleRejected (AUDIT-Z-06): recurring-профиль
// с модулем pool/contribution отвергается валидацией и ledger.New.
func TestAuditZRecurringPoolModuleRejected(t *testing.T) {
	mk := func(modules []uint32) *wire.GenesisPackage {
		gp := params.TestGenesis()
		p := gp.Manifest.Profile
		p.Recurring = true
		p.Unit = wire.UnitQuota
		p.ResourceID = "res-1"
		p.TransferMode = wire.TransferModeRestricted
		p.Modules = modules
		return gp
	}
	for _, modules := range [][]uint32{{wire.ModulePool}, {wire.ModuleContribution}, {wire.ModuleRedeem, wire.ModulePool}} {
		if bad := profiles.Validate(mk(modules).Manifest.Profile); len(bad) == 0 {
			t.Fatalf("profiles.Validate пропустил recurring + модули %v", modules)
		}
		gp := mk(modules)
		wire.SetManifestHash(gp)
		if _, err := New(gp, Options{}); err == nil {
			t.Fatalf("ledger.New пропустил recurring + модули %v", modules)
		}
	}
	// redeem-модуль один — валидно
	if bad := profiles.Validate(mk([]uint32{wire.ModuleRedeem}).Manifest.Profile); len(bad) != 0 {
		t.Fatalf("recurring + redeem отклонён: %v", bad)
	}
}
