// Тесты решений пользователя, закреплённых на этапе 5 (подэтап 5.0):
// №12 (повторное evidence — приостановка навсегда), №14 (vouch от
// flagged запрещён), №29 (награда гардиану 5% месячного выпуска
// подопечного, кулдаун 12 мес.), каузальный AcceptedTxs и конфликтные
// ветки в журнале (spec/04 INV-1c, правка этапа 5).
package ledger

import (
	"bytes"
	"testing"

	"github.com/rumatakira74/trust-networks-platform/core/crypto"
	"github.com/rumatakira74/trust-networks-platform/core/wire"
)

// seedVouchers — посевной круг V1..V3 как в e2e-сценарии (давние active,
// 10 выполненных периодов выпуска; для скорости — pre-state).
func (e *e2eEnv) seedVouchers() {
	for _, id := range []string{"V1", "V2", "V3"} {
		priv := e.key(id)
		pub := crypto.MarshalPubKey(&priv.PublicKey)
		fp := crypto.Fingerprint(pub)
		addr := e.addr(id)
		w := &Wallet{
			Address: addr, Fingerprint: fp, PublicKey: &priv.PublicKey,
			Status: wire.StatusActive, Seed: true,
			RegisteredAt:    e.clock.NetTime() - 400*86400,
			ActivatedAt:     e.clock.NetTime() - 400*86400,
			ActivatedPeriod: 1, LastClaimedPeriod: 10,
			LastTouchEpoch: 1, DelayedThreshold: 3000, Seq: 1,
		}
		w.TipTxID = make([]byte, 32)
		copy(w.TipTxID, []byte("seed:"+id))
		e.ledger.State.Wallets[string(addr)] = w
		e.ledger.State.FingerprintIndex[string(fp)] = addr
	}
}

// activate — K поручительств посевного круга -> active.
func (e *e2eEnv) activate(id string) {
	for _, v := range []string{"V1", "V2", "V3"} {
		res := e.ledger.Apply(e.tx(v, wire.Vouch, &wire.VouchPayload{
			Candidate: e.addr(id), Epoch: e.clock.Epoch(),
		}, false))
		e.requireAccepted(res, "vouch "+v)
	}
}

// mintFor — выпуск следующего завершённого периода (часы сдвигаются,
// чтобы заявляемый период был завершён; M3–M6).
func (e *e2eEnv) mintFor(id string) *wire.Transaction {
	w := e.ledger.State.Wallet(e.addr(id))
	period := w.LastClaimedPeriod + 1
	if period < w.ActivatedPeriod {
		period = w.ActivatedPeriod
	}
	if e.clock.P <= period {
		e.clock.P = period + 1
		e.clock.T += 31 * 86400
	}
	return e.tx(id, wire.Mint, &wire.MintPayload{
		Period: period, Amount: e.ledger.State.Net.Profile.IssuanceAmount,
	}, true)
}

// mature — созревание когорт: окно переводимости 3 периода (T8).
func (e *e2eEnv) mature() {
	e.clock.P += 3
	e.clock.T += 3 * 31 * 86400
	e.ledger.syncTimeForTest()
}

func (e *e2eEnv) transferFor(from, to string, amount uint64) *wire.Transaction {
	return e.tx(from, wire.Transfer, &wire.TransferPayload{
		Send: &wire.TransferSend{Recipient: e.addr(to), Amount: amount},
	}, false)
}

// equivocate — пара конфликтных веток: обе собираются на одном состоянии
// (один seq/prev, разное содержимое), затем применяются.
func (e *e2eEnv) equivocate(from, to1, to2 string, amount uint64) (t1, t2 *wire.Transaction) {
	t1 = e.transferFor(from, to1, amount)
	t2 = e.tx(from, wire.Transfer, &wire.TransferPayload{
		Send: &wire.TransferSend{Recipient: e.addr(to2), Amount: amount},
	}, false)
	e.requireAccepted(e.ledger.Apply(t1), "t1")
	if res := e.ledger.Apply(t2); res.Outcome != Conflict {
		e.t.Fatalf("конфликт: %v/%s", res.Outcome, res.Code)
	}
	return t1, t2
}

// TestDecision12RepeatEvidenceForever — первое evidence: приостановка 3 мес.;
// повторное: навсегда (решение №12).
func TestDecision12RepeatEvidenceForever(t *testing.T) {
	e := newE2E(t)
	e.seedVouchers()
	e.requireAccepted(e.register("ALICE"), "register")
	e.activate("ALICE")
	e.requireAccepted(e.ledger.Apply(e.mintFor("ALICE")), "mint")
	e.mature()

	// первая пара конфликтных веток + evidence
	t1, t2 := e.equivocate("ALICE", "V1", "V2", 10)
	ev := e.tx("V2", wire.Evidence, &wire.EvidencePayload{
		TxA: t1, TxB: t2, Kind: wire.EvidenceEquivocation,
	}, false)
	e.requireAccepted(e.ledger.Apply(ev), "evidence 1")
	w := e.ledger.State.Wallet(e.addr("ALICE"))
	period := e.clock.Period()
	if w.MintSuspendedUntilPeriod != period+3 {
		t.Fatalf("первое evidence: приостановка до %d, ожидается %d",
			w.MintSuspendedUntilPeriod, period+3)
	}

	// вторая пара + повторное evidence
	t3, t4 := e.equivocate("ALICE", "V3", "V1", 5)
	ev2 := e.tx("V3", wire.Evidence, &wire.EvidencePayload{
		TxA: t3, TxB: t4, Kind: wire.EvidenceEquivocation,
	}, false)
	e.requireAccepted(e.ledger.Apply(ev2), "evidence 2")
	if w.MintSuspendedUntilPeriod != ^uint64(0) {
		t.Fatalf("повторное evidence: приостановка %d, ожидается навсегда",
			w.MintSuspendedUntilPeriod)
	}
}

// TestDecision14VouchFromFlagged — vouch от flagged запрещён (решение №14);
// переводы остаются разрешёнными.
func TestDecision14VouchFromFlagged(t *testing.T) {
	e := newE2E(t)
	e.seedVouchers()
	e.requireAccepted(e.register("ALICE"), "register ALICE")
	e.requireAccepted(e.register("BOB"), "register BOB")
	e.requireAccepted(e.ledger.Apply(e.mintFor("V1")), "mint V1")
	e.mature()

	// помечаем V1 через evidence
	t1, t2 := e.equivocate("V1", "V2", "V3", 10)
	ev := e.tx("V2", wire.Evidence, &wire.EvidencePayload{
		TxA: t1, TxB: t2, Kind: wire.EvidenceEquivocation,
	}, false)
	e.requireAccepted(e.ledger.Apply(ev), "evidence")
	if !e.ledger.State.Wallet(e.addr("V1")).Flagged {
		t.Fatal("V1 не помечен")
	}

	// vouch от V1 (flagged) — отклоняется (решение №14)
	res := e.ledger.Apply(e.tx("V1", wire.Vouch, &wire.VouchPayload{
		Candidate: e.addr("ALICE"), Epoch: e.clock.Epoch(),
	}, false))
	if res.Outcome != Rejected || res.Code != "E_VOUCH_FLAGGED" {
		t.Fatalf("vouch от flagged: %v/%s, ожидается reject/E_VOUCH_FLAGGED", res.Outcome, res.Code)
	}
	// перевод от flagged — разрешён (штрафов нет, 7.5; решение №14)
	res = e.ledger.Apply(e.transferFor("V1", "ALICE", 5))
	if res.Outcome != Accepted {
		t.Fatalf("перевод от flagged отклонён: %v/%s", res.Outcome, res.Code)
	}
}

// TestDecision29GuardianReward — публичный гардиан при миграции получает
// 5% месячного выпуска подопечного (решение №29).
func TestDecision29GuardianReward(t *testing.T) {
	e := newE2E(t)
	e.seedVouchers()
	e.requireAccepted(e.register("ALICE"), "register ALICE")
	e.activate("ALICE")
	e.requireAccepted(e.ledger.Apply(e.mintFor("ALICE")), "mint ALICE")
	// V1..V3 — публичные гардианы (возраст посевного круга > 12 мес.)
	for _, g := range []string{"V1", "V2", "V3"} {
		e.requireAccepted(e.ledger.Apply(e.tx(g, wire.GuardianOffer,
			&wire.GuardianOfferPayload{}, false)), "guardian offer "+g)
	}

	supplyBefore := e.ledger.State.TotalMinted
	e.requireAccepted(e.register("NEW"), "register NEW")
	pro := &wire.MigratePropose{OldAddress: e.addr("ALICE"), Nonce: 1}
	for _, g := range []string{"V1", "V2", "V3"} {
		stmt := &wire.MigrateStatement{
			NetworkID:  e.ledger.State.Net.NetworkID,
			OldAddress: e.addr("ALICE"), NewAddress: e.addr("NEW"),
			Nonce: 1, Epoch: e.clock.Epoch(),
		}
		sh := stmt.Hash()
		sig, err := crypto.SignDER(e.key(g), sh[:])
		if err != nil {
			t.Fatalf("guardian sig: %v", err)
		}
		pro.Approvals = append(pro.Approvals, wire.GuardianApproval{
			Guardian: e.addr(g), StatementSig: sig,
		})
	}
	e.requireAccepted(e.ledger.Apply(e.tx("NEW", wire.Migrate,
		&wire.MigratePayload{Propose: pro}, false)), "migrate propose")
	e.clock.T += 15 * 86400
	e.ledger.syncTimeForTest()

	// награда: 5% × issuance (1000) = 50 каждому из трёх публичных
	want := e.ledger.State.Net.Profile.IssuanceAmount *
		e.ledger.State.Net.Profile.GuardianReward / 1_000_000
	if want != 50 {
		t.Fatalf("расчёт награды: %d, ожидается 50", want)
	}
	for _, g := range []string{"V1", "V2", "V3"} {
		gw := e.ledger.State.Wallet(e.addr(g))
		if gw.Wards == 0 {
			t.Fatalf("%s не получил подопечного", g)
		}
	}
	if got := e.ledger.State.TotalMinted - supplyBefore; got != 3*want {
		t.Fatalf("эмиссия наград: %d, ожидается %d", got, 3*want)
	}
	// кулдаун зафиксирован по паре гардиан|подопечный (12 мес., №29)
	if len(e.ledger.State.GuardianLastReward) != 3 {
		t.Fatalf("кулдаун-записей: %d, ожидается 3", len(e.ledger.State.GuardianLastReward))
	}
	// INV-8: награда — новая эмиссия, инвариант цел
	if bad := e.ledger.CheckInvariants(); len(bad) > 0 {
		t.Fatalf("INV: %v", bad)
	}
}

// TestCausalJournalAndConflictBranches — AcceptedTxs каузален (порядок
// принятия); конфликтные ветки — в ConflictBranches (spec/04 INV-1c).
func TestCausalJournalAndConflictBranches(t *testing.T) {
	e := newE2E(t)
	e.seedVouchers()
	e.requireAccepted(e.register("ALICE"), "register")
	e.activate("ALICE")
	e.requireAccepted(e.ledger.Apply(e.mintFor("ALICE")), "mint")
	e.mature()
	_, t2 := e.equivocate("ALICE", "V1", "V2", 10)
	e.requireAccepted(e.ledger.Apply(e.transferFor("ALICE", "V3", 5)), "t3")

	journal := e.ledger.AcceptedTxs()
	// каузальный порядок принятия: register, vouch x3, mint, t1, t3
	want := []wire.TxType{wire.Register, wire.Vouch, wire.Vouch, wire.Vouch,
		wire.Mint, wire.Transfer, wire.Transfer}
	if len(journal) != len(want) {
		t.Fatalf("журнал: %d транзакций, ожидается %d", len(journal), len(want))
	}
	for i, w := range want {
		if journal[i].Type != w {
			t.Fatalf("журнал[%d].type = %s, ожидается %s", i, journal[i].Type, w)
		}
	}
	// конфликтная ветка — полная транзакция в отдельном списке
	branches := e.ledger.ConflictBranches()
	if len(branches) != 1 || !bytes.Equal(branches[0].Wallet, e.addr("ALICE")) {
		t.Fatalf("конфликтные ветки: %d", len(branches))
	}
	if branches[0].Seq != t2.Seq {
		t.Fatalf("ветка seq = %d, ожидается %d", branches[0].Seq, t2.Seq)
	}
}
