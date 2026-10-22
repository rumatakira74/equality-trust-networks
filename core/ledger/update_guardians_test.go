// Смена личных гардианов (11, решение владельца 21.10.2026): публикация
// update_guardians, валидации (active, flagged, №7, одна активная смена,
// лимит 150), окно 48 ч, отмена, вступление/истечение, миграция по
// старому списку в окне и по новому после вступления.
package ledger

import (
	"testing"

	"github.com/rumatakira74/equality-trust-networks/core/crypto"
	"github.com/rumatakira74/equality-trust-networks/core/wire"
)

// ugEnv — окружение тестов смены гардианов: посевные V1–V3 (active),
// кошелёк W (active) с личными гардианами V1–V3.
type ugEnv struct {
	*e2eEnv
}

func newUG(t *testing.T) *ugEnv {
	e := newE2E(t)
	for _, id := range []string{"V1", "V2", "V3", "G4"} {
		priv := e.key(id)
		pub := crypto.MarshalPubKey(&priv.PublicKey)
		fp := crypto.Fingerprint(pub)
		addr := e.addr(id)
		w := &Wallet{
			Address: addr, Fingerprint: fp, PublicKey: &priv.PublicKey,
			Status: wire.StatusActive, Seed: true,
			RegisteredAt:    e.clock.NetTime() - 400*86400,
			ActivatedAt:    e.clock.NetTime() - 400*86400,
			ActivatedPeriod: 1, LastClaimedPeriod: 10,
			LastTouchEpoch: 1, DelayedThreshold: 3000, Seq: 1,
		}
		w.TipTxID = make([]byte, 32)
		copy(w.TipTxID, []byte("seed:"+id))
		e.ledger.State.Wallets[string(addr)] = w
		e.ledger.State.FingerprintIndex[string(fp)] = addr
	}
	e.requireAccepted(e.register("W", "V1", "V2", "V3"), "register W")
	for _, v := range []string{"V1", "V2", "V3"} {
		res := e.ledger.Apply(e.tx(v, wire.Vouch, &wire.VouchPayload{
			Candidate: e.addr("W"), Epoch: e.clock.Epoch(),
		}, false))
		e.requireAccepted(res, "vouch "+v)
	}
	return &ugEnv{e}
}

// changeApprovals — подписи новых гардианов над хешем смены.
func (u *ugEnv) changeApprovals(nonce uint64, guardians ...string) []wire.GuardianApproval {
	var aps []wire.GuardianApproval
	stmt := &wire.GuardianChangeStatement{
		NetworkID: u.ledger.State.Net.NetworkID,
		Ward:      u.addr("W"),
		Nonce:     nonce,
	}
	for _, g := range guardians {
		stmt.NewGuardians = append(stmt.NewGuardians, u.addr(g))
	}
	sh := stmt.Hash()
	for _, g := range guardians {
		sig, err := crypto.SignDER(u.key(g), sh[:])
		if err != nil {
			u.t.Fatalf("guardian sig: %v", err)
		}
		aps = append(aps, wire.GuardianApproval{
			Guardian: u.addr(g), StatementSig: sig,
		})
	}
	return aps
}

func (u *ugEnv) propose(nonce uint64, guardians ...string) *wire.Transaction {
	return u.tx("W", wire.UpdateGuardians, &wire.UpdateGuardiansPayload{
		Propose: &wire.UpdateGuardiansPropose{
			NewGuardians: u.addrs(guardians),
			Approvals:    u.changeApprovals(nonce, guardians...),
			Nonce:        nonce,
		},
	}, false)
}

func (u *ugEnv) addrs(ids []string) [][]byte {
	out := make([][]byte, 0, len(ids))
	for _, id := range ids {
		out = append(out, u.addr(id))
	}
	return out
}

func (u *ugEnv) requireReject(res Result, what, code string) {
	u.t.Helper()
	if res.Outcome != Rejected || res.Code != code {
		u.t.Fatalf("%s: %v/%s, ожидается reject/%s", what, res.Outcome, res.Code, code)
	}
}

// TestUpdateGuardiansPublishValidations — проверки при публикации:
// active, не flagged (№14), лимит №7, ≤ 3, подписи согласия, одна
// активная смена.
func TestUpdateGuardiansPublishValidations(t *testing.T) {
	u := newUG(t)

	// гардиан не active (registered без vouch)
	u.requireAccepted(u.register("REG"), "register REG")
	res := u.ledger.Apply(u.propose(1, "REG"))
	u.requireReject(res, "гардиан не active", "E_GUARDIAN_UPDATE")

	// flagged-гардиан (№14)
	v2 := u.ledger.State.Wallet(u.addr("V2"))
	v2.Flagged = true
	res = u.ledger.Apply(u.propose(2, "V2"))
	u.requireReject(res, "flagged-гардиан", "E_GUARDIAN_UPDATE")
	v2.Flagged = false

	// лимит №7: подопечных guardian_personal_max_wards
	prof := u.ledger.State.Net.Profile
	v3 := u.ledger.State.Wallet(u.addr("V3"))
	v3.Wards = prof.GuardianPersonalMaxWards
	res = u.ledger.Apply(u.propose(3, "V3"))
	u.requireReject(res, "лимит №7", "E_GUARDIAN_UPDATE")
	v3.Wards = 0

	// больше 3 гардианов
	res = u.ledger.Apply(u.tx("W", wire.UpdateGuardians, &wire.UpdateGuardiansPayload{
		Propose: &wire.UpdateGuardiansPropose{
			NewGuardians: [][]byte{u.addr("V1"), u.addr("V2"), u.addr("V3"), u.addr("V1")},
			Nonce:        4,
		},
	}, false))
	u.requireReject(res, "больше 3 гардианов", "E_GUARDIAN_UPDATE")

	// подпись согласия отсутствует/неверна: список V1, но подписи нет
	res = u.ledger.Apply(u.tx("W", wire.UpdateGuardians, &wire.UpdateGuardiansPayload{
		Propose: &wire.UpdateGuardiansPropose{
			NewGuardians: u.addrs([]string{"V1"}),
			Nonce:        5,
		},
	}, false))
	u.requireReject(res, "нет подписи согласия", "E_GUARDIAN_UPDATE")

	// валидная публикация
	res = u.ledger.Apply(u.propose(6, "V2", "V3"))
	u.requireAccepted(res, "валидная смена")

	// активная смена только одна
	res = u.ledger.Apply(u.propose(7, "V1", "V2"))
	u.requireReject(res, "вторая активная смена", "E_GUARDIAN_UPDATE_ACTIVE")

	// до истечения окна действует старый список
	w := u.ledger.State.Wallet(u.addr("W"))
	if len(w.PersonalGuardians) != 3 {
		t.Fatalf("старый список заменён до истечения окна: %d", len(w.PersonalGuardians))
	}
	if w.GuardianChanges != 0 {
		t.Fatalf("счётчик смен до вступления: %d", w.GuardianChanges)
	}
	if bad := u.ledger.CheckInvariants(); len(bad) > 0 {
		t.Fatalf("INV: %v", bad)
	}
}

// TestUpdateGuardiansWindowCancelEffect — окно 48 ч (пинирование часов),
// отмена в окне, вступление в силу, отмена после вступления невозможна.
func TestUpdateGuardiansWindowCancelEffect(t *testing.T) {
	u := newUG(t)
	t0 := u.clock.NetTime()

	// публикация; окно 48 ч
	res := u.ledger.Apply(u.propose(1, "V2", "V3"))
	u.requireAccepted(res, "смена опубликована")
	pu := u.ledger.State.PendingGuardianUpdates[0]
	if want := t0 + 48*3600; pu.EffectiveAt != want {
		t.Fatalf("окно: EffectiveAt %d, ожидается %d", pu.EffectiveAt, want)
	}

	// в окне — старый список
	w := u.ledger.State.Wallet(u.addr("W"))
	if len(w.PersonalGuardians) != 3 {
		t.Fatal("старый список не действует в окне")
	}

	// отмена в окне (подпись владельца — текущим ключом, C8)
	u.clock.T = t0 + 24 * 3600
	res = u.ledger.Apply(u.tx("W", wire.UpdateGuardians, &wire.UpdateGuardiansPayload{
		Cancel: &wire.UpdateGuardiansCancel{ProposeSeq: pu.ProposeSeq},
	}, false))
	u.requireAccepted(res, "отмена в окне")
	if !pu.Canceled {
		t.Fatal("смена не помечена отменённой")
	}
	// отменённая смена лимит не потребляет
	if w.GuardianChanges != 0 {
		t.Fatalf("отмена потребила лимит: %d", w.GuardianChanges)
	}

	// после отмены возможна новая смена
	res = u.ledger.Apply(u.propose(2, "V2", "G4"))
	u.requireAccepted(res, "повторная смена после отмены")

	// вступление через 48 ч: новый список, счётчик 1, wards
	u.clock.T += 49 * 3600
	u.ledger.syncTimeForTest()
	w = u.ledger.State.Wallet(u.addr("W"))
	if len(w.PersonalGuardians) != 2 ||
		!equalAddr(w.PersonalGuardians[0], u.addr("V2")) ||
		!equalAddr(w.PersonalGuardians[1], u.addr("G4")) {
		t.Fatal("новый список не вступил в силу")
	}
	if w.GuardianChanges != 1 {
		t.Fatalf("счётчик смен: %d, ожидается 1", w.GuardianChanges)
	}
	// G4 — новый гардиан: слот подопечного занят; V1/V3 ушли
	// (регистрационные гардианы wards не занимали — пол 0)
	if got := u.ledger.State.Wallet(u.addr("G4")).Wards; got != 1 {
		t.Fatalf("wards G4: %d, ожидается 1", got)
	}
	if got := u.ledger.State.Wallet(u.addr("V1")).Wards; got != 0 {
		t.Fatalf("wards V1 (ушёл): %d, ожидается 0", got)
	}

	// отмена после вступления невозможна
	res = u.ledger.Apply(u.tx("W", wire.UpdateGuardians, &wire.UpdateGuardiansPayload{
		Cancel: &wire.UpdateGuardiansCancel{ProposeSeq: pu.ProposeSeq + 1},
	}, false))
	u.requireReject(res, "отмена вступившей", "E_GUARDIAN_UPDATE_NOTFOUND")
	if bad := u.ledger.CheckInvariants(); len(bad) > 0 {
		t.Fatalf("INV: %v", bad)
	}
}

// TestUpdateGuardiansExpiryByWardLimit — если к моменту вступления лимит
// №7 у нового гардиана превышен, смена не вступает, помечается истёкшей.
func TestUpdateGuardiansExpiryByWardLimit(t *testing.T) {
	u := newUG(t)
	prof := u.ledger.State.Net.Profile

	// V2 на публикеции проходит (wards = max−1), к вступлению — превышен
	v2 := u.ledger.State.Wallet(u.addr("V2"))
	v2.Wards = prof.GuardianPersonalMaxWards - 1
	res := u.ledger.Apply(u.propose(1, "V2"))
	u.requireAccepted(res, "публикация на границе лимита")

	u.clock.T += 49 * 3600
	v2.Wards = prof.GuardianPersonalMaxWards // превышен к вступлению
	u.ledger.syncTimeForTest()

	pu := u.ledger.State.PendingGuardianUpdates[0]
	if !pu.Expired || pu.Effective {
		t.Fatal("смена не помечена истёкшей")
	}
	w := u.ledger.State.Wallet(u.addr("W"))
	if len(w.PersonalGuardians) != 3 {
		t.Fatal("истёкшая смена изменила список")
	}
	if w.GuardianChanges != 0 {
		t.Fatalf("истёкшая смена потребила лимит: %d", w.GuardianChanges)
	}
	// после истечения возможна новая смена
	res = u.ledger.Apply(u.propose(2, "V3"))
	u.requireAccepted(res, "смена после истёкшей")
}

// TestUpdateGuardiansLimit150 — не более 150 вступивших в силу смен;
// отменённые и истёкшие лимит не потребляют; при исчерпании — отказ.
func TestUpdateGuardiansLimit150(t *testing.T) {
	u := newUG(t)

	// отменённая и истёкшая смены лимит не потребляют
	res := u.ledger.Apply(u.propose(1, "V2"))
	u.requireAccepted(res, "смена к отмене")
	pu := u.ledger.State.PendingGuardianUpdates[0]
	res = u.ledger.Apply(u.tx("W", wire.UpdateGuardians, &wire.UpdateGuardiansPayload{
		Cancel: &wire.UpdateGuardiansCancel{ProposeSeq: pu.ProposeSeq},
	}, false))
	u.requireAccepted(res, "отмена")

	prof := u.ledger.State.Net.Profile
	v1 := u.ledger.State.Wallet(u.addr("V1"))
	v1.Wards = prof.GuardianPersonalMaxWards - 1 // публикация проходит
	res = u.ledger.Apply(u.propose(2, "V1"))
	u.requireAccepted(res, "смена к истечению")
	v1.Wards = prof.GuardianPersonalMaxWards // превышен к вступлению
	u.clock.T += 49 * 3600
	u.ledger.syncTimeForTest()
	if w := u.ledger.State.Wallet(u.addr("W")); w.GuardianChanges != 0 {
		t.Fatalf("лимит потреблён до вступивших смен: %d", w.GuardianChanges)
	}

	// 150 вступивших в силу смен
	nonce := uint64(100)
	for i := uint64(0); i < GuardianChangeLimit; i++ {
		res := u.ledger.Apply(u.propose(nonce, "V2", "V3"))
		if res.Outcome != Accepted {
			t.Fatalf("смена %d: %v/%s", i, res.Outcome, res.Code)
		}
		nonce++
		u.clock.T += 49 * 3600
		u.ledger.syncTimeForTest()
	}
	w := u.ledger.State.Wallet(u.addr("W"))
	if w.GuardianChanges != GuardianChangeLimit {
		t.Fatalf("счётчик: %d, ожидается %d", w.GuardianChanges, GuardianChangeLimit)
	}

	// исчерпание: отклонение
	res = u.ledger.Apply(u.propose(nonce, "V2", "V3"))
	u.requireReject(res, "исчерпание лимита", "E_GUARDIAN_CHANGE_LIMIT")
}

// TestMigrationGuardianListOldAndNew — подписи гардианов для миграции
// валидны по старому списку в окне смены и по новому после вступления.
func TestMigrationGuardianListOldAndNew(t *testing.T) {
	u := newUG(t)
	t0 := u.clock.NetTime()

	// смена: V1,V2,V3 -> V2,V3
	res := u.ledger.Apply(u.propose(1, "V2", "V3"))
	u.requireAccepted(res, "смена опубликована")

	u.requireAccepted(u.register("NEW"), "register NEW")

	// в окне: миграция с подписями по СТАРОМУ списку (V1, V2) проходит
	migrateApprovals := func(nonce uint64, gs ...string) []wire.GuardianApproval {
		var aps []wire.GuardianApproval
		for _, g := range gs {
			stmt := &wire.MigrateStatement{
				NetworkID:  u.ledger.State.Net.NetworkID,
				OldAddress: u.addr("W"), NewAddress: u.addr("NEW"),
				Nonce: nonce, Epoch: u.clock.Epoch(),
			}
			sh := stmt.Hash()
			sig, err := crypto.SignDER(u.key(g), sh[:])
			if err != nil {
				u.t.Fatalf("mig sig: %v", err)
			}
			aps = append(aps, wire.GuardianApproval{Guardian: u.addr(g), StatementSig: sig})
		}
		return aps
	}
	res = u.ledger.Apply(u.tx("NEW", wire.Migrate, &wire.MigratePayload{
		Propose: &wire.MigratePropose{
			OldAddress: u.addr("W"),
			Approvals:  migrateApprovals(1, "V1", "V2"),
			Nonce:      1,
		},
	}, false))
	u.requireAccepted(res, "миграция по старому списку в окне")

	// вступление смены
	u.clock.T = t0 + 49*3600
	u.ledger.syncTimeForTest()

	// после вступления: подписи по старому списку (V1, V2) не проходят —
	// V1 больше не гардиан (ни личный, ни публичный)
	res = u.ledger.Apply(u.tx("NEW", wire.Migrate, &wire.MigratePayload{
		Propose: &wire.MigratePropose{
			OldAddress: u.addr("W"),
			Approvals:  migrateApprovals(2, "V1", "V2"),
			Nonce:      2,
		},
	}, false))
	u.requireReject(res, "миграция по старому списку после вступления", "E_MIGRATE_GUARDIANS")

	// по НОВОМУ списку (V2, V3) — проходит
	res = u.ledger.Apply(u.tx("NEW", wire.Migrate, &wire.MigratePayload{
		Propose: &wire.MigratePropose{
			OldAddress: u.addr("W"),
			Approvals:  migrateApprovals(3, "V2", "V3"),
			Nonce:      3,
		},
	}, false))
	u.requireAccepted(res, "миграция по новому списку после вступления")
	if bad := u.ledger.CheckInvariants(); len(bad) > 0 {
		t.Fatalf("INV: %v", bad)
	}
}

func equalAddr(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
