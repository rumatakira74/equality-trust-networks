// Тесты recurring-единиц (v9.10, раздел 4, этап 12): сгорание по периодной
// метке (перевод единицы периода P отвергается после границы P+1),
// режим peer-redeem, replay между сетями (network_id в валидации всегда,
// включая глобальную константу "global").
package ledger

import (
	"bytes"
	"crypto/ecdsa"
	"testing"

	"github.com/rumatakira74/equality-trust-networks/core/crypto"
	"github.com/rumatakira74/equality-trust-networks/core/params"
	"github.com/rumatakira74/equality-trust-networks/core/wire"
)

// newEnvWithNetworkID — e2e-окружение на TestGenesis с заданным network_id
// (манифест и genesis-запись согласованы, manifest_hash пересчитан).
func newEnvWithNetworkID(t *testing.T, networkID []byte) *e2eEnv {
	t.Helper()
	gp := params.TestGenesis()
	gp.Manifest.NetworkID = networkID
	gp.Genesis.NetworkID = networkID
	gp.Manifest.Profile.Pow = &wire.PowParams{
		TargetLight: 0x0010_0000_0000_0000, Argon2MemoryKiB: 8, Argon2Iterations: 1,
		Argon2Parallelism: 1, TargetHard: 0x0010_0000_0000_0000,
	}
	wire.SetManifestHash(gp)
	clock := &FixedClock{T: 1_700_000_000, E: 1, P: 1}
	l, err := New(gp, clock, Options{SkipMintChallengeWindow: true})
	if err != nil {
		t.Fatalf("genesis: %v", err)
	}
	return &e2eEnv{t: t, ledger: l, clock: clock,
		keys: make(map[string]*ecdsa.PrivateKey), addrs: make(map[string][]byte)}
}

// newRecurringEnv — recurring-сеть профиля 8 (quota_reset + peer-redeem):
// поставщик V1, участники ALICE/BOB с квотой периода 10 (выпуск при
// текущем периоде 11, мораторий пройден).
func newRecurringEnv(t *testing.T) *e2eEnv {
	t.Helper()
	e := newEnvWithNetworkID(t, []byte("tnp-recurring-test"))
	p := &e.ledger.State.Net.Profile
	p.Unit = wire.UnitQuota
	p.ResourceID = "compute"
	p.IssuanceMode = wire.IssuanceQuotaReset
	p.AccrualWindowPeriods = 0
	p.Recurring = true
	p.TransferMode = wire.TransferModePeerRedeem
	p.FeeMode = wire.FeeModeNone
	p.DemurrageRateMicros = 0
	p.Modules = []uint32{wire.ModuleRedeem}
	// поставщик — V1 (та же деривация, что в тестах модулей)
	v1pub := crypto.MarshalPubKey(&e.key("V1").PublicKey)
	v1addr := crypto.DeriveAddress(crypto.Fingerprint(v1pub), e.ledger.State.Net.NetworkID)
	p.Providers = [][]byte{v1addr}

	e.clock.P = 11
	for _, id := range []string{"V1", "ALICE", "BOB"} {
		e.preStateWallet(id)
		// квота периода 10: last_claimed 9 (M3/M5/M6 — ровно один период)
		e.ledger.State.Wallet(e.addr(id)).LastClaimedPeriod = 9
	}
	for _, id := range []string{"V1", "ALICE", "BOB"} {
		mint := e.tx(id, wire.Mint, &wire.MintPayload{Period: 10, Amount: 1000}, true)
		e.requireAccepted(e.ledger.Apply(mint), "mint "+id)
	}
	return e
}

// TestRecurringBurnBoundary — сгорание на границе периода: перевод единицы
// периода 10 принимается в периоде 11 (граница P+1 ещё не прошла) и
// отвергается E_BURNED в периоде 12; сброса баланса нет — остаток виден.
func TestRecurringBurnBoundary(t *testing.T) {
	e := newRecurringEnv(t)
	alice := e.ledger.State.Wallet(e.addr("ALICE"))
	bob := e.ledger.State.Wallet(e.addr("BOB"))

	// внутри окна: текущий период 11 = P+1 — принимается
	send := e.tx("ALICE", wire.Transfer,
		&wire.TransferPayload{Send: &wire.TransferSend{Recipient: bob.Address, Amount: 400}}, false)
	e.requireAccepted(e.ledger.Apply(send), "перевод внутри окна")

	// полученные единицы сохраняют метку периода: BOB переводит их дальше
	// в том же окне
	forward := e.tx("BOB", wire.Transfer,
		&wire.TransferPayload{Send: &wire.TransferSend{Recipient: alice.Address, Amount: 100}}, false)
	e.requireAccepted(e.ledger.Apply(forward), "перевод полученных единиц")

	// граница прошла: период 12 > P+1 — сгоревшая единица, E_BURNED
	e.clock.P = 12
	burned := e.tx("ALICE", wire.Transfer,
		&wire.TransferPayload{Send: &wire.TransferSend{Recipient: bob.Address, Amount: 10}}, false)
	if res := e.ledger.Apply(burned); res.Outcome != Rejected || res.Code != "E_BURNED" {
		t.Fatalf("перевод после границы P+1: want reject E_BURNED, got %v %s", res.Outcome, res.Code)
	}
	// не сброс баланса: когорты остались в кошельке (история проверяема)
	if alice.TotalBalance() == 0 {
		t.Fatalf("сгорание не сбрасывает баланс: остаток обязан остаться, got 0")
	}
	// BOB не может вывести полученные единицы периода 10 после границы
	burnedForward := e.tx("BOB", wire.Transfer,
		&wire.TransferPayload{Send: &wire.TransferSend{Recipient: alice.Address, Amount: 10}}, false)
	if res := e.ledger.Apply(burnedForward); res.Outcome != Rejected || res.Code != "E_BURNED" {
		t.Fatalf("перевод полученных единиц после границы: want reject E_BURNED, got %v %s", res.Outcome, res.Code)
	}
}

// TestRecurringNewPeriodUsable — новая квота нового периода переводима:
// сгорание старой метки не блокирует единицы нового периода; сгоревшая
// когорта остаётся в балансе (история проверяема, сброса нет).
func TestRecurringNewPeriodUsable(t *testing.T) {
	e := newRecurringEnv(t)
	alice := e.ledger.State.Wallet(e.addr("ALICE"))
	bob := e.ledger.State.Wallet(e.addr("BOB"))
	// период 12: квота периода 11 свежая, квота периода 10 сгорела
	e.clock.P = 12
	mint := e.tx("ALICE", wire.Mint, &wire.MintPayload{Period: 11, Amount: 1000}, true)
	e.requireAccepted(e.ledger.Apply(mint), "mint периода 11")
	send := e.tx("ALICE", wire.Transfer,
		&wire.TransferPayload{Send: &wire.TransferSend{Recipient: bob.Address, Amount: 1000}}, false)
	e.requireAccepted(e.ledger.Apply(send), "перевод свежей квоты")
	// списание шло из свежей когорты: переводимый остаток 0,
	// сгоревшая когорта периода 10 (1000) остаётся в балансе
	if got := alice.RecurringSpendable(12); got != 0 {
		t.Fatalf("после перевода свежей квоты переводимый остаток 0, got %d", got)
	}
	if got := alice.TotalBalance(); got != 1000 {
		t.Fatalf("сгоревшая когорта остаётся в балансе: want 1000, got %d", got)
	}
}

// TestRecurringBalanceVsBurned — E_BALANCE отличается от E_BURNED:
// запрос сверх остатка — E_BALANCE даже в recurring-сети.
func TestRecurringBalanceVsBurned(t *testing.T) {
	e := newRecurringEnv(t)
	bob := e.ledger.State.Wallet(e.addr("BOB"))
	over := e.tx("ALICE", wire.Transfer,
		&wire.TransferPayload{Send: &wire.TransferSend{Recipient: bob.Address, Amount: 5000}}, false)
	if res := e.ledger.Apply(over); res.Outcome != Rejected || res.Code != "E_BALANCE" {
		t.Fatalf("перевод сверх остатка: want reject E_BALANCE, got %v %s", res.Outcome, res.Code)
	}
}

// TestPeerRedeem — режим peer-redeem: перевод участнику принимается,
// неизвестному получателю — E_TRANSFER_RECIPIENT; погашение поставщиком
// через redeem внутри окна принимается, после границы — E_BURNED.
func TestPeerRedeem(t *testing.T) {
	e := newRecurringEnv(t)
	bob := e.ledger.State.Wallet(e.addr("BOB"))

	// участник → участнику
	send := e.tx("ALICE", wire.Transfer,
		&wire.TransferPayload{Send: &wire.TransferSend{Recipient: bob.Address, Amount: 50}}, false)
	e.requireAccepted(e.ledger.Apply(send), "peer-redeem: участнику")

	// получатель не зарегистрирован — E_TRANSFER_RECIPIENT
	unknown := crypto.DeriveAddress(crypto.Fingerprint(crypto.MarshalPubKey(
		&crypto.DeterministicKey("e2e:UNKNOWN").PublicKey)), e.ledger.State.Net.NetworkID)
	bad := e.tx("ALICE", wire.Transfer,
		&wire.TransferPayload{Send: &wire.TransferSend{Recipient: unknown, Amount: 10}}, false)
	if res := e.ledger.Apply(bad); res.Outcome != Rejected || res.Code != "E_TRANSFER_RECIPIENT" {
		t.Fatalf("peer-redeem: неизвестному: want E_TRANSFER_RECIPIENT, got %v %s", res.Outcome, res.Code)
	}

	// участник → поставщик через redeem (внутри окна)
	redeem := e.tx("ALICE", wire.Redeem, &wire.RedeemPayload{
		Provider: e.addr("V1"), ResourceID: "compute", Amount: 200}, false)
	e.requireAccepted(e.ledger.Apply(redeem), "redeem поставщику")
	v1 := e.ledger.State.Wallet(e.addr("V1"))
	if got := v1.TotalBalance(); got != 1200 { // 1000 своей квоты + 200 погашено
		t.Fatalf("поставщик: want 1200, got %d", got)
	}

	// после границы P+1 погашение сгоревшей единицы — E_BURNED
	e.clock.P = 12
	burned := e.tx("ALICE", wire.Redeem, &wire.RedeemPayload{
		Provider: e.addr("V1"), ResourceID: "compute", Amount: 10}, false)
	if res := e.ledger.Apply(burned); res.Outcome != Rejected || res.Code != "E_BURNED" {
		t.Fatalf("redeem после границы: want E_BURNED, got %v %s", res.Outcome, res.Code)
	}
}

// TestReplayBetweenNetworks — replay между сетями: транзакция сети A
// отвергается узлом сети B (E_NET); network_id = "global" участвует
// в валидации так же (константа — не пустой идентификатор).
func TestReplayBetweenNetworks(t *testing.T) {
	// сеть A — recurring-сеть; сеть B — с другим network_id
	ea := newRecurringEnv(t)
	eb := newEnvWithNetworkID(t, []byte("tnp-other-network"))
	if bytes.Equal(ea.ledger.State.Net.NetworkID, eb.ledger.State.Net.NetworkID) {
		t.Fatalf("network_id сетей обязаны различаться")
	}
	bob := ea.ledger.State.Wallet(ea.addr("BOB"))
	tx := ea.tx("ALICE", wire.Transfer,
		&wire.TransferPayload{Send: &wire.TransferSend{Recipient: bob.Address, Amount: 10}}, false)
	// узел сети B: та же транзакция (подписана в сети A) — E_NET
	if res := eb.ledger.Apply(tx); res.Outcome != Rejected || res.Code != "E_NET" {
		t.Fatalf("replay между сетями: want E_NET, got %v %s", res.Outcome, res.Code)
	}

	// глобальная сеть: network_id = "global" участвует в валидации так же
	eg := newEnvWithNetworkID(t, wire.GlobalNetworkID)
	eg.clock.P = 11
	eg.ledger.State.Net.Profile.FreshTransferWindowPeriods = 0 // когорты сразу переводимы
	for _, id := range []string{"ALICE", "BOB"} {
		eg.preStateWallet(id)
		eg.ledger.State.Wallet(eg.addr(id)).LastClaimedPeriod = 9
	}
	for _, id := range []string{"ALICE", "BOB"} {
		mint := eg.tx(id, wire.Mint, &wire.MintPayload{Period: 10, Amount: 1000}, true)
		eg.requireAccepted(eg.ledger.Apply(mint), "mint "+id)
	}
	// транзакция с чужим network_id — E_NET
	foreign := eg.tx("ALICE", wire.Transfer,
		&wire.TransferPayload{Send: &wire.TransferSend{Recipient: eg.addr("BOB"), Amount: 1}}, false)
	foreign.NetworkID = ea.ledger.State.Net.NetworkID
	if res := eg.ledger.Apply(foreign); res.Outcome != Rejected || res.Code != "E_NET" {
		t.Fatalf("глобальная сеть, чужой network_id: want E_NET, got %v %s", res.Outcome, res.Code)
	}
	// своя транзакция с network_id = "global" проходит C1 как любая другая
	good := eg.tx("ALICE", wire.Transfer,
		&wire.TransferPayload{Send: &wire.TransferSend{Recipient: eg.addr("BOB"), Amount: 1}}, false)
	eg.requireAccepted(eg.ledger.Apply(good), "global: своя транзакция")
}
