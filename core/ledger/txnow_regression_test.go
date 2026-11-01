// Регресс-тесты фикса времени (v9.13, этап 15): txNow = timestamp_claim
// для всех часозависимых правил; сеттлемент монотонен от max(txNow);
// порядок подачи tx журнала не меняет итоговое состояние; слот
// челленджа — отдельный тест (mint_window_test). Основание —
// reports/stage12-testnet/FINDING-replay-live-divergence.txt
// (live/replay-расхождение mint у границы периода).
package ledger

import (
	"crypto/ecdsa"
	"fmt"
	"sort"
	"testing"

	"github.com/rumatakira74/equality-trust-networks/core/params"
	"github.com/rumatakira74/equality-trust-networks/core/wire"
)

// claimAt — момент внутри периода p (30-сут. конвенция тестового genesis).
func (e *e2eEnv) claimAt(p uint64) uint64 {
	return e.ledger.State.GenesisTime + p*30*86400 + 86400
}

// stateFingerprint — каноническая сводка состояния для сравнения путей
// применения (балансы, счётчики, границы сеттлемента).
func stateFingerprint(l *Ledger) string {
	fp := fmt.Sprintf("burned=%d minted=%d epoch=%d period=%d netTime=%d",
		l.State.TotalBurned, l.State.TotalMinted, l.State.Epoch,
		l.State.CurrentPeriod, l.State.NetTime)
	addrs := make([]string, 0, len(l.State.Wallets))
	for k := range l.State.Wallets {
		addrs = append(addrs, k)
	}
	sort.Strings(addrs)
	for _, k := range addrs {
		w := l.State.Wallets[k]
		fp += fmt.Sprintf("|%x:%d:%d:%d:%d:%d:%d", w.Address, w.Status,
			w.Liquid, w.sumCohorts(), w.Locked, w.Seq, w.LastClaimedPeriod)
	}
	return fp
}

// TestTxNowVerdictsFromClaim — вердикт часозависимых правил не зависит
// от часов узла на момент apply: ядро вообще не читает часы (v9.13),
// точка времени валидации — txNow = timestamp_claim самой tx. Тест
// закрепляет claim-семантику по правилам: M2, M4 (кейс находки:
// сборка в периоде N, публикация в N+1), T3, V2, ActivatedPeriod,
// evidence-сроки, модули (дедлайн пула), recurring-граница P+1;
// одна и та же tx на двух независимых ядрах даёт один вердикт.
func TestTxNowVerdictsFromClaim(t *testing.T) {
	// --- M4 + находка 12-testnet: mint, собранный в периоде N (claim
	// в N) и опубликованный после границы (часы узла в N+1), валидируется
	// от claim: заявленный период N >= периода валидации N —
	// E_MINT_PERIOD на любом узле и в replay (старый live принял бы).
	mintCase := func(e *e2eEnv) Result {
		e.seedVouchers()
		e.requireAccepted(e.register("ALICE"), "register")
		e.activate("ALICE")  // активация в периоде 1 (claim T0)
		e.now = e.claimAt(2) // claim в периоде 2: граница 1 прошла
		return e.ledger.Apply(e.tx("ALICE", wire.Mint,
			&wire.MintPayload{Period: 2, Amount: 1000}, true))
	}
	resA := mintCase(newE2E(t))
	resB := mintCase(newE2E(t)) // другое ядро, другое реальное время apply
	if resA.Code != "E_MINT_PERIOD" || resA.Outcome != Rejected {
		t.Fatalf("mint заявленного периода валидации: %v/%s, ожидается reject/E_MINT_PERIOD",
			resA.Outcome, resA.Code)
	}
	if resA.Code != resB.Code || resA.Outcome != resB.Outcome {
		t.Fatalf("вердикт зависит от окружения: %v/%s vs %v/%s",
			resA.Outcome, resA.Code, resB.Outcome, resB.Code)
	}

	// заявленный период завершён (1 < 2) — принимается
	e := newE2E(t)
	e.seedVouchers()
	e.requireAccepted(e.register("ALICE"), "register")
	e.activate("ALICE")
	e.now = e.claimAt(2)
	e.requireAccepted(e.ledger.Apply(e.tx("ALICE", wire.Mint,
		&wire.MintPayload{Period: 1, Amount: 1000}, true)), "mint завершённого периода")

	// --- M2: приостановка выпуска от claim — граница от claim
	w := e.ledger.State.Wallet(e.addr("ALICE"))
	w.Flagged = true
	w.MintSuspendedUntilPeriod = e.ledger.periodAt(e.now) + 3
	e.now = e.claimAt(3) // период 3 < 2+3 — внутри приостановки
	if res := e.ledger.Apply(e.tx("ALICE", wire.Mint,
		&wire.MintPayload{Period: 2, Amount: 1000}, true)); res.Code != "E_FLAGGED_MINT" {
		t.Fatalf("mint в приостановке: %s, ожидается E_FLAGGED_MINT", res.Code)
	}

	// --- T3: мораторий от claim (14 сут от register)
	e2 := newE2E(t)
	e2.seedVouchers()
	e2.now = e2.claimAt(2)
	e2.requireAccepted(e2.register("FRESH"), "register FRESH")
	fw := e2.ledger.State.Wallet(e2.addr("FRESH"))
	fw.Liquid = 500                    // переводимые средства (мораторий — не баланс)
	e2.ledger.State.TotalMinted += 500 // INV-8
	if res := e2.ledger.Apply(e2.tx("FRESH", wire.Transfer, &wire.TransferPayload{
		Send: &wire.TransferSend{Recipient: e2.addr("V1"), Amount: 10},
	}, false)); res.Code != "E_MORATORIUM" {
		t.Fatalf("мораторий от claim: %s, ожидается E_MORATORIUM", res.Code)
	}
	// claim после моратория — принимается
	e2.now += 14 * 86400
	e2.requireAccepted(e2.ledger.Apply(e2.tx("FRESH", wire.Transfer, &wire.TransferPayload{
		Send: &wire.TransferSend{Recipient: e2.addr("V1"), Amount: 10},
	}, false)), "перевод после моратория")

	// --- V2: возраст поручителя от claim (90 сут)
	e3 := newE2E(t)
	e3.seedVouchers()
	e3.requireAccepted(e3.register("CAND"), "register CAND")
	v1 := e3.ledger.State.Wallet(e3.addr("V1"))
	v1.RegisteredAt = e3.now - 89*86400 // 89 < 90 суток на момент claim
	if res := e3.ledger.Apply(e3.tx("V1", wire.Vouch, &wire.VouchPayload{
		Candidate: e3.addr("CAND"), Epoch: e3.epoch(),
	}, false)); res.Code != "E_VOUCH_AGE" {
		t.Fatalf("vouch до возраста: %s, ожидается E_VOUCH_AGE", res.Code)
	}
	v1.RegisteredAt = e3.now - 90*86400
	e3.requireAccepted(e3.ledger.Apply(e3.tx("V1", wire.Vouch, &wire.VouchPayload{
		Candidate: e3.addr("CAND"), Epoch: e3.epoch(),
	}, false)), "vouch по достижении возраста")

	// --- ActivatedPeriod: период активации = periodAt(claim vouch-а)
	for _, v := range []string{"V2", "V3"} {
		e3.requireAccepted(e3.ledger.Apply(e3.tx(v, wire.Vouch, &wire.VouchPayload{
			Candidate: e3.addr("CAND"), Epoch: e3.epoch(),
		}, false)), "vouch "+v)
	}
	cand := e3.ledger.State.Wallet(e3.addr("CAND"))
	if cand.Status != wire.StatusActive {
		t.Fatalf("CAND не active: %s", cand.Status)
	}
	if want := e3.ledger.periodAt(e3.now); cand.ActivatedPeriod != want {
		t.Fatalf("ActivatedPeriod %d, ожидается periodAt(claim) = %d",
			cand.ActivatedPeriod, want)
	}

	// --- evidence: приостановка до periodAt(claim)+3
	e4 := newE2E(t)
	e4.seedVouchers()
	e4.requireAccepted(e4.register("MALLORY"), "register")
	e4.activate("MALLORY")
	e4.now = e4.claimAt(2)
	e4.requireAccepted(e4.ledger.Apply(e4.tx("MALLORY", wire.Mint,
		&wire.MintPayload{Period: 1, Amount: 1000}, true)), "mint")
	e4.mature()
	t1, t2 := e4.equivocate("MALLORY", "V1", "V2", 10)
	e4.requireAccepted(e4.ledger.Apply(e4.tx("V2", wire.Evidence, &wire.EvidencePayload{
		TxA: t1, TxB: t2, Kind: wire.EvidenceEquivocation,
	}, false)), "evidence")
	m := e4.ledger.State.Wallet(e4.addr("MALLORY"))
	if want := e4.ledger.periodAt(e4.now) + e4.ledger.State.Net.FlaggedMintSuspensionPeriods; m.MintSuspendedUntilPeriod != want {
		t.Fatalf("приостановка до %d, ожидается periodAt(claim)+3 = %d",
			m.MintSuspendedUntilPeriod, want)
	}

	// --- модули: дедлайн пула сравнивается с epochAt(claim)
	m5 := newModuleEnv(t)
	e5 := m5.e2eEnv
	cur := e5.ledger.epochAt(e5.now)
	if res := e5.ledger.Apply(e5.tx("ALICE", wire.PoolCreate, &wire.PoolCreatePayload{
		Recipient: e5.addr("V1"), GoalAmount: 300, DeadlineEpoch: cur,
		ReleaseMode: wire.PoolReleaseOnGoal, RefundMode: wire.PoolRefundIfUnmet,
	}, false)); res.Code != "E_POOL_PARAMS" {
		t.Fatalf("pool_create с дедлайном = эпохе claim: %s, ожидается E_POOL_PARAMS", res.Code)
	}
	e5.requireAccepted(e5.ledger.Apply(e5.tx("ALICE", wire.PoolCreate, &wire.PoolCreatePayload{
		Recipient: e5.addr("V1"), GoalAmount: 300, DeadlineEpoch: cur + 10,
		ReleaseMode: wire.PoolReleaseOnGoal, RefundMode: wire.PoolRefundIfUnmet,
	}, false)), "pool_create с дедлайном после эпохи claim")

	// --- recurring: граница P+1 от claim (сгорание меткой)
	e6 := newRecurringEnv(t)
	e6.now = e6.ledger.State.GenesisTime + 12*30*86400 + 86400 // период 12
	if res := e6.ledger.Apply(e6.tx("ALICE", wire.Transfer, &wire.TransferPayload{
		Send: &wire.TransferSend{Recipient: e6.addr("BOB"), Amount: 10},
	}, false)); res.Code != "E_BURNED" {
		t.Fatalf("recurring после границы P+1: %s, ожидается E_BURNED", res.Code)
	}
}

// TestSettlementMonotoneFromMaxTxNow — сеттлемент монотонен от
// max(txNow) по журналу: демередж оседает по границам периодов от claim
// принятых tx; повторные применения идемпотентны (E_DUP, флаги
// Executed/Effective); tx с более ранним claim не откатывает сеттлемент;
// рестарт (пересчёт журнала с genesis) воспроизводит то же состояние.
func TestSettlementMonotoneFromMaxTxNow(t *testing.T) {
	gp := params.TestGenesis()
	gp.Genesis.GenesisTime = 1_700_000_000
	gp.Manifest.Profile.Pow = &wire.PowParams{
		TargetLight: 0x0010_0000_0000_0000, Argon2MemoryKiB: 8, Argon2Iterations: 1,
		Argon2Parallelism: 1, TargetHard: 0x0010_0000_0000_0000,
	}
	gp.Manifest.Profile.DemurrageFloor = 0 // демередж виден на малых балансах
	wire.SetManifestHash(gp)
	newCore := func() (*Ledger, *e2eEnv) {
		l, err := New(gp, Options{SkipMintChallengeWindow: true})
		if err != nil {
			t.Fatalf("genesis: %v", err)
		}
		e := &e2eEnv{t: t, ledger: l, now: gp.Genesis.GenesisTime,
			keys: make(map[string]*ecdsa.PrivateKey), addrs: make(map[string][]byte)}
		for _, id := range []string{"V1", "V2", "V3"} {
			e.preStateWallet(id)
			e.ledger.State.Wallet(e.addr(id)).Liquid = 10_000
			e.ledger.State.TotalMinted += 10_000
		}
		return l, e
	}
	l1, e := newCore()

	ps := uint64(30 * 86400)
	claim := func(p uint64) uint64 { return gp.Genesis.GenesisTime + p*ps + 86400 }
	// журнал: переводы с растущими claim через границы периодов 2–4
	// (демередж оседает на ненулевых балансах)
	steps := []struct {
		from, to string
		amount   uint64
		claim    uint64
	}{
		{"V1", "V2", 100, claim(2)},
		{"V2", "V3", 200, claim(3)},
		{"V3", "V1", 300, claim(4)},
	}
	var built []*wire.Transaction
	burned := uint64(0)
	for i, s := range steps {
		e.now = s.claim
		tx := e.tx(s.from, wire.Transfer, &wire.TransferPayload{
			Send: &wire.TransferSend{Recipient: e.addr(s.to), Amount: s.amount},
		}, false)
		built = append(built, tx)
		if res := e.ledger.Apply(tx); res.Outcome != Accepted {
			t.Fatalf("шаг %d: %v/%s", i, res.Outcome, res.Code)
		}
		if e.ledger.State.TotalBurned < burned {
			t.Fatalf("шаг %d: демередж откатился: %d < %d", i, e.ledger.State.TotalBurned, burned)
		}
		burned = e.ledger.State.TotalBurned
		if e.ledger.State.NetTime != s.claim {
			t.Fatalf("шаг %d: NetTime %d, ожидается max(txNow) = %d",
				i, e.ledger.State.NetTime, s.claim)
		}
	}
	if burned == 0 {
		t.Fatal("демередж не осел по границам периодов")
	}
	fp := stateFingerprint(l1)
	curPeriod, netTime := e.ledger.State.CurrentPeriod, e.ledger.State.NetTime

	// повторное применение тех же tx идемпотентно (E_DUP, сеттлемент на месте)
	for _, tx := range built {
		if res := e.ledger.Apply(tx); res.Code != "E_DUP" {
			t.Fatalf("повторное применение: %s, ожидается E_DUP", res.Code)
		}
	}
	if e.ledger.State.TotalBurned != burned || e.ledger.State.CurrentPeriod != curPeriod {
		t.Fatal("повторное применение изменило сеттлемент")
	}
	if got := stateFingerprint(l1); got != fp {
		t.Fatal("повторное применение изменило состояние")
	}

	// tx с более ранним claim не откатывает сеттлемент (монотонность):
	// повторная подача первой tx журнала (claim периода 2)
	if res := e.ledger.Apply(built[0]); res.Code != "E_DUP" {
		t.Fatalf("ранний claim: %s, ожидается E_DUP", res.Code)
	}
	if e.ledger.State.NetTime != netTime || e.ledger.State.CurrentPeriod != curPeriod {
		t.Fatal("сеттлемент откатился от более раннего claim")
	}

	// рестарт: пересчёт того же журнала с genesis — то же состояние
	l2, _ := newCore()
	for _, tx := range e.ledger.AcceptedTxs() {
		if res := l2.Apply(tx); res.Outcome != Accepted {
			t.Fatalf("рестарт: tx %s: %v/%s", tx.TxID(), res.Outcome, res.Code)
		}
	}
	if l2.State.TotalBurned != burned {
		t.Fatalf("рестарт: демередж %d, ожидается %d", l2.State.TotalBurned, burned)
	}
	if got := stateFingerprint(l2); got != fp {
		t.Fatal("рестарт: состояние не воспроизводится из журнала")
	}
}

// TestAcceptedTxOrderDemurrageConvergence — подача tx журнала в различном
// порядке даёт одно итоговое состояние: claims всех tx в одном периоде,
// сеттлемент к max(txNow) оседает на первой принятой tx (граница периода
// — до любых эффектов), дальнейший порядок эффектов на итоговое
// состояние не влияет. Каузальный порядок цепочки каждого кошелька (seq)
// сохраняется; перемешиваются блоки разных кошельков.
func TestAcceptedTxOrderDemurrageConvergence(t *testing.T) {
	build := func() (*Ledger, *e2eEnv, []*wire.Transaction) {
		gp := params.TestGenesis()
		gp.Genesis.GenesisTime = 1_700_000_000
		gp.Manifest.Profile.Pow = &wire.PowParams{
			TargetLight: 0x0010_0000_0000_0000, Argon2MemoryKiB: 8, Argon2Iterations: 1,
			Argon2Parallelism: 1, TargetHard: 0x0010_0000_0000_0000,
		}
		gp.Manifest.Profile.DemurrageFloor = 0
		gp.Manifest.Profile.FreshTransferWindowPeriods = 0 // когорты сразу переводимы
		wire.SetManifestHash(gp)
		l, err := New(gp, Options{SkipMintChallengeWindow: true})
		if err != nil {
			t.Fatalf("genesis: %v", err)
		}
		e := &e2eEnv{t: t, ledger: l, now: 1_700_000_000,
			keys: make(map[string]*ecdsa.PrivateKey), addrs: make(map[string][]byte)}
		for _, id := range []string{"V1", "V2", "V3"} {
			w := e.preStateWallet(id)
			w.LastClaimedPeriod = 0 // квота периода 1 заявляема (M3)
			w.ActivatedPeriod = 1
		}
		// все claim в периоде 2 (граница периода 1 пересечена; демередж
		// границы 2 оседает от первой принятой tx — до любых эффектов)
		e.now = gp.Genesis.GenesisTime + 2*30*86400 + 86400
		var txs []*wire.Transaction
		for _, id := range []string{"V1", "V2", "V3"} {
			mint := e.tx(id, wire.Mint, &wire.MintPayload{Period: 1, Amount: 1000}, true)
			mintID := mint.TxID()
			txs = append(txs, mint)
			// перевод следует за mint в цепочке кошелька (seq+1, prev = mint)
			w := e.ledger.State.Wallet(e.addr(id))
			send := &wire.Transaction{
				NetworkID:       e.ledger.State.Net.NetworkID,
				Wallet:          e.addr(id),
				Seq:             w.Seq + 2,
				Prev:            mintID[:],
				Type:            wire.Transfer,
				Payload:         &wire.TransferPayload{Send: &wire.TransferSend{Recipient: e.addr("V2"), Amount: 100}},
				ProtocolVersion: e.ledger.State.Net.ProtocolVersion,
				TimestampClaim:  e.now,
			}
			e.solveAndSign(send, id, false)
			txs = append(txs, send)
		}
		return l, e, txs
	}

	// блоки кошельков [mint_i, transfer_i]; порядок блоков — любой
	blocks := [][]int{{0, 1}, {2, 3}, {4, 5}}
	orders := [][]int{{0, 1, 2}, {0, 2, 1}, {1, 0, 2}, {1, 2, 0}, {2, 0, 1}, {2, 1, 0}}
	var want string
	for p, order := range orders {
		l, _, txs := build()
		for _, b := range order {
			for _, i := range blocks[b] {
				if res := l.Apply(txs[i]); res.Outcome != Accepted {
					t.Fatalf("перестановка %d, tx %d: %v/%s", p, i, res.Outcome, res.Code)
				}
			}
		}
		fp := stateFingerprint(l)
		if p == 0 {
			want = fp
			continue
		}
		if fp != want {
			t.Fatalf("перестановка %d: итоговое состояние разошлось", p)
		}
	}
}
