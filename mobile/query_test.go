// Unit-тесты расширения моста для UI (этап 7): конструкторы транзакций,
// ManifestJSON, JSON-снапшоты состояния (узел с малыми PoW, без сети).
package tnp

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/rumatakira74/equality-trust-networks/core/wire"
)

func testGenesis7(t *testing.T) *Genesis {
	t.Helper()
	// малые PoW (как в net-тестах; реальные — №9, этап 6)
	return SandboxGenesis("tnp-ui-test", 0, 3600,
		0x0010_0000_0000_0000, 0x0010_0000_0000_0000, 8, 1, 1)
}

func softRegister(t *testing.T, gen *Genesis) (addr []byte, pub, priv []byte) {
	t.Helper()
	pair, err := SoftKey()
	if err != nil {
		t.Fatal(err)
	}
	pub, priv = pair.Pub(), pair.PrivDER()
	addr, err = AddressFromPublicKey(pub, gen.NetworkID())
	if err != nil {
		t.Fatal(err)
	}
	return addr, pub, priv
}

// TestQueryRegisterGuardians — register с гардианами: разбор JSON,
// максимум 3, кодирование/разбор, гардианы в payload.
func TestQueryRegisterGuardians(t *testing.T) {
	gen := testGenesis7(t)
	addr, pub, _ := softRegister(t, gen)
	g1 := strings.Repeat("11", 32)
	g2 := strings.Repeat("22", 32)
	tx, err := NewRegisterGuardians(gen.NetworkID(), addr, pub, nil,
		[]byte("challenge"), "android-id",
		`["`+g1+`","`+g2+`"]`, gen.ProtocolVersion(), time.Now().Unix())
	if err != nil {
		t.Fatal(err)
	}
	tx.SolveLight(gen.PowTargetLight())
	rp, ok := tx.tx.Payload.(*wire.RegisterPayload)
	if !ok {
		t.Fatal("payload не RegisterPayload")
	}
	if len(rp.PersonalGuardians) != 2 {
		t.Fatalf("гардианов: %d, хочу 2", len(rp.PersonalGuardians))
	}
	if !bytes.Equal(rp.PersonalGuardians[0], mustHex(t, g1)) {
		t.Fatal("гардиан 1 не совпал")
	}
	// максимум 3 (11)
	if _, err := NewRegisterGuardians(gen.NetworkID(), addr, pub, nil, nil, "",
		`["`+g1+`","`+g2+`","`+g1+`","`+g2+`"]`, 1, 0); err == nil {
		t.Fatal("4 гардиана прошли")
	}
	// битый JSON
	if _, err := NewRegisterGuardians(gen.NetworkID(), addr, pub, nil, nil, "",
		`["zz"]`, 1, 0); err == nil {
		t.Fatal("битый hex прошёл")
	}
}

func mustHex(t *testing.T, s string) []byte {
	t.Helper()
	b, err := hex.DecodeString(s)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

// TestQueryConstructors — vouch/cancel/threshold/offer/migrate: типы и
// payload после кодирования/разбора.
func TestQueryConstructors(t *testing.T) {
	gen := testGenesis7(t)
	netID := gen.NetworkID()
	w := bytes.Repeat([]byte{0xA1}, 32)
	prev := bytes.Repeat([]byte{0xB2}, 32)
	cand := bytes.Repeat([]byte{0xC3}, 32)

	v := NewVouch(netID, w, 2, prev, cand, 7, 1, time.Now().Unix())
	if v.Type() != "VOUCH" {
		t.Fatalf("vouch: тип %s", v.Type())
	}
	if vp, ok := v.tx.Payload.(*wire.VouchPayload); !ok || !bytes.Equal(vp.Candidate, cand) || vp.Epoch != 7 {
		t.Fatal("vouch payload не совпал")
	}

	c := NewTransferCancel(netID, w, 3, prev, 2, 1, time.Now().Unix())
	if cp, ok := c.tx.Payload.(*wire.TransferPayload); !ok || cp.Cancel == nil || cp.Cancel.PendingSeq != 2 {
		t.Fatal("cancel payload не совпал")
	}

	th := NewSetThreshold(netID, w, 4, prev, 500, 1, time.Now().Unix())
	if tp, ok := th.tx.Payload.(*wire.TransferPayload); !ok || tp.SetThreshold == nil || tp.SetThreshold.NewThreshold != 500 {
		t.Fatal("set_threshold payload не совпал")
	}

	off := NewGuardianOffer(netID, w, 5, prev, 1, time.Now().Unix())
	if _, ok := off.tx.Payload.(*wire.GuardianOfferPayload); !ok {
		t.Fatal("guardian_offer payload не совпал")
	}

	oldAddr := bytes.Repeat([]byte{0xD4}, 32)
	newAddr := bytes.Repeat([]byte{0xE5}, 32)
	apr := `[{"guardian":"` + hex.EncodeToString(w) + `","sig":"aabb"}]`
	mp, err := NewMigratePropose(netID, newAddr, 2, prev, oldAddr, apr, 9, 1, time.Now().Unix())
	if err != nil {
		t.Fatal(err)
	}
	if mpp, ok := mp.tx.Payload.(*wire.MigratePayload); !ok || mpp.Propose == nil ||
		!bytes.Equal(mpp.Propose.OldAddress, oldAddr) || mpp.Propose.Nonce != 9 ||
		len(mpp.Propose.Approvals) != 1 {
		t.Fatal("migrate propose payload не совпал")
	}

	mc := NewMigrateCancel(netID, oldAddr, 8, prev, newAddr, 2, 1, time.Now().Unix())
	if mcp, ok := mc.tx.Payload.(*wire.MigratePayload); !ok || mcp.Cancel == nil ||
		!bytes.Equal(mcp.Cancel.NewAddress, newAddr) || mcp.Cancel.ProposeSeq != 2 {
		t.Fatal("migrate cancel payload не совпал")
	}
}

// TestMigrateStatementHashBridge — хеш утверждения совпадает с ядром
// (wire.MigrateStatement.Hash): гардиан подписывает то же, что проверит
// applyMigrate.
func TestMigrateStatementHashBridge(t *testing.T) {
	netID := []byte("tnp-ui-test")
	oldA := bytes.Repeat([]byte{1}, 32)
	newA := bytes.Repeat([]byte{2}, 32)
	h := MigrateStatementHash(netID, oldA, newA, 42, 7)
	ref := (&wire.MigrateStatement{
		NetworkID: netID, OldAddress: oldA, NewAddress: newA, Nonce: 42, Epoch: 7,
	}).Hash()
	if !bytes.Equal(h, ref[:]) {
		t.Fatal("MigrateStatementHash не совпал с wire.MigrateStatement.Hash")
	}
}

// TestManifestJSON — манифест парсится и содержит параметры, которые
// приложение обязано читать из genesis (не хардкодить): K, слоты, P,
// D, демередж, комиссия, мораторий, гардианы, эпоха.
func TestManifestJSON(t *testing.T) {
	gen := testGenesis7(t)
	var m map[string]any
	if err := json.Unmarshal([]byte(gen.ManifestJSON()), &m); err != nil {
		t.Fatalf("ManifestJSON не парсится: %v", err)
	}
	for _, key := range []string{
		"vouch_k", "slot_tiers", "delayed_threshold_default", "delayed_threshold_max",
		"transfer_delay_hours", "demurrage_rate_micros", "fee_min", "rate_min",
		"transfer_moratorium_days", "accrual_window_periods", "epoch_seconds",
		"guardian_personal_threshold", "guardian_public_threshold",
		"migrate_delay_days", "mint_challenge_window_hours", "issuance_amount",
	} {
		if _, ok := m[key]; !ok {
			t.Errorf("ManifestJSON: нет ключа %s", key)
		}
	}
	if m["vouch_k"] != float64(3) {
		t.Errorf("vouch_k = %v, хочу 3", m["vouch_k"])
	}
	if m["transfer_delay_hours"] != float64(48) {
		t.Errorf("transfer_delay_hours = %v, хочу 48", m["transfer_delay_hours"])
	}
}

// TestQueryJSONSnapshots — узел с реестром: JSON-геттеры состояния
// (WalletInfo, VouchProgress, Slots, History, PendingTransfers,
// Migrations, MintWindow, FeeQuote, Registry, GuardianOffers).
func TestQueryJSONSnapshots(t *testing.T) {
	gen := testGenesis7(t)
	addr, pub, priv := softRegister(t, gen)
	node, err := StartNode(gen, "full", "", nil, "", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer node.Close()

	// register A: синтетическая цепочка песочницы (эмулятор, 6.5)
	if res := submitRegister(t, node, gen, addr, pub, priv, "android-A", "[]"); res != "" {
		t.Fatalf("register A: %s", res)
	}
	// register B с гардианом A (R6: гардианы — известные адреса, 11)
	addrB, pubB, privB := softRegister(t, gen)
	if res := submitRegister(t, node, gen, addrB, pubB, privB, "android-B",
		`["`+hex.EncodeToString(addr)+`"]`); res != "" {
		t.Fatalf("register B: %s", res)
	}
	addr = addrB

	// ManifestJSON через узел не нужен; проверяем снапшоты
	var wi map[string]any
	if err := json.Unmarshal([]byte(node.WalletInfoJSON(addr)), &wi); err != nil {
		t.Fatal(err)
	}
	if wi["exists"] != true || wi["status"] != "REGISTERED" {
		t.Fatalf("WalletInfo: %v", wi)
	}
	if wi["vouch_depth"] != float64(-1) {
		t.Errorf("vouch_depth = %v, хочу -1 (нет vouch-рёбер до B)", wi["vouch_depth"])
	}
	guardians, _ := wi["personal_guardians"].([]any)
	if len(guardians) != 1 {
		t.Errorf("гардианов в снапшоте: %d, хочу 1", len(guardians))
	}

	var vp map[string]any
	if err := json.Unmarshal([]byte(node.VouchProgressJSON(addr)), &vp); err != nil {
		t.Fatal(err)
	}
	if vp["received"] != float64(0) || vp["k"] != float64(3) {
		t.Fatalf("VouchProgress: %v", vp)
	}

	var slots map[string]any
	if err := json.Unmarshal([]byte(node.SlotsJSON(addr)), &slots); err != nil {
		t.Fatal(err)
	}
	if slots["exists"] != true {
		t.Fatalf("Slots: %v", slots)
	}

	var hist []map[string]any
	if err := json.Unmarshal([]byte(node.HistoryJSON(addr, 0)), &hist); err != nil {
		t.Fatal(err)
	}
	if len(hist) != 1 || hist[0]["type"] != "REGISTER" {
		t.Fatalf("History: %v", hist)
	}

	// отложенных нет, миграций нет — пустые списки
	var pts []map[string]any
	if err := json.Unmarshal([]byte(node.PendingTransfersJSON(addr)), &pts); err != nil {
		t.Fatal(err)
	}
	if len(pts) != 0 {
		t.Fatalf("PendingTransfers: %v", pts)
	}
	var pms []map[string]any
	if err := json.Unmarshal([]byte(node.PendingMigrationsJSON(addr)), &pms); err != nil {
		t.Fatal(err)
	}
	if len(pms) != 0 {
		t.Fatalf("PendingMigrations: %v", pms)
	}

	// окно челленджа: структура и период из genesis (65 суток → период 2)
	var mw map[string]any
	if err := json.Unmarshal([]byte(node.MintWindowJSON(addr)), &mw); err != nil {
		t.Fatal(err)
	}
	if _, ok := mw["open"]; !ok {
		t.Fatalf("MintWindow: %v", mw)
	}
	if mw["issuance_amount"] != float64(1000) {
		t.Errorf("issuance_amount = %v", mw["issuance_amount"])
	}

	// комиссия: fee = max(fee_min, rate × amount) (10.2);
	// стартовая ставка TestGenesis — 1% (RateMin=10000)
	if f := node.FeeQuote(1000); f != 10 {
		t.Errorf("FeeQuote(1000) = %d, хочу 10", f)
	}
	if f := node.FeeQuote(1); f != 1 { // ниже минимума
		t.Errorf("FeeQuote(1) = %d, хочу 1 (fee_min)", f)
	}

	// реестр: посевной круг (3) + A + B; публичных гардианов нет
	var reg2 []map[string]any
	if err := json.Unmarshal([]byte(node.RegistryJSON()), &reg2); err != nil {
		t.Fatal(err)
	}
	if len(reg2) != 5 {
		t.Fatalf("Registry: %d кошельков, хочу 5 (3 посевных + A + B)", len(reg2))
	}
	var offers []map[string]any
	if err := json.Unmarshal([]byte(node.GuardianOffersJSON()), &offers); err != nil {
		t.Fatal(err)
	}
	if len(offers) != 0 {
		t.Fatalf("GuardianOffers: %v", offers)
	}

	// часы сети
	var clock map[string]any
	if err := json.Unmarshal([]byte(node.ClockJSON()), &clock); err != nil {
		t.Fatal(err)
	}
	if clock["period"] == nil || clock["epoch"] == nil {
		t.Fatalf("Clock: %v", clock)
	}
}

// submitRegister — register через мост (синтетическая цепочка песочницы,
// усиленный PoW); "" — принят, иначе код отклонения.
func submitRegister(t *testing.T, node *Node, gen *Genesis, addr, pub, priv []byte,
	androidID, guardiansJSON string) string {
	t.Helper()
	challenge := AttestationChallengeFor(pub, gen.NetworkID())
	chain, err := SandboxAttestationChain(pub, gen.NetworkID(), challenge)
	if err != nil {
		t.Fatal(err)
	}
	reg, err := NewRegisterGuardians(gen.NetworkID(), addr, pub, chain, challenge,
		androidID, guardiansJSON, gen.ProtocolVersion(), time.Now().Unix())
	if err != nil {
		t.Fatal(err)
	}
	// register — усиленный PoW (7.6, C10/C11); лёгкий дал бы E_POW_HARD
	reg.SolveHard(gen.PowTargetHard(), gen.Argon2MemKiB(), gen.Argon2Iterations(), gen.Argon2Par())
	sig, err := SignWithKey(priv, reg.SigningHash())
	if err != nil {
		t.Fatal(err)
	}
	reg.SetSignature(sig)
	res, err := node.Submit(reg)
	if err != nil {
		t.Fatal(err)
	}
	if res.Outcome() != "accept" {
		return res.Outcome() + "/" + res.Code()
	}
	return ""
}

// TestMintWindowClaimedPeriod — окно челленджа в MintWindowJSON
// считается для ЗАЯВЛЯЕМОГО периода (находка №2 прогона stage11-prep):
// ядро (M7, №39) проверяет слот периода из Mint-транзакции, а мост
// передавал текущий — кнопка Mint была отключена при фактически
// открытом окне заявляемого периода.
func TestMintWindowClaimedPeriod(t *testing.T) {
	gen := testGenesis7(t)
	// реальное окно ±48 ч (7.6): SandboxGenesis в песочнице открывает
	// весь период (учебный оверрайд) — здесь проверяем именно слот №39
	gen.gp.Manifest.Profile.MintChallengeWindowHours = 48
	mh := gen.gp.Manifest.ManifestHash()
	gen.gp.Genesis.ManifestHash = mh[:]
	// genesis 60 суток назад: текущий период 2, заявляемый 1
	gen.gp.Genesis.GenesisTime = uint64(time.Now().Add(-60 * 24 * time.Hour).Unix())

	// слот челленджа (та же деривация, что core/ledger.hashThree):
	// slot(p) = genesis + p×30 сут + H(addr‖p) mod 30 сут; при genesis
	// = now − 60 сут: slot(1) − now = H1 mod 30сут − 30 сут,
	// slot(2) − now = H2 mod 30 сут
	const day = 24 * 3600
	slotOff := func(addr []byte, period uint64) uint64 {
		var pb [8]byte
		binary.BigEndian.PutUint64(pb[:], period)
		h := sha256.Sum256(append(append([]byte{}, addr...), pb[:]...))
		var out uint64
		for i := 0; i < 8; i++ {
			out = out<<8 | uint64(h[i])
		}
		return out % (30 * day)
	}
	// addrOpen: слот периода 1 в последних 24 ч периода (окно ±48 ч
	// открыто с запасом), слот периода 2 ≥ 96 ч от now (окно текущего
	// периода закрыто с запасом — старый код давал open=false)
	var addrOpen, pubOpen, privOpen []byte
	for i := 0; i < 10000 && addrOpen == nil; i++ {
		a, p, v := softRegister(t, gen)
		if slotOff(a, 1) >= uint64(30*day-24*3600) && slotOff(a, 2) >= uint64(96*3600) {
			addrOpen, pubOpen, privOpen = a, p, v
		}
	}
	if addrOpen == nil {
		t.Fatal("не подобран адрес с открытым окном заявляемого периода")
	}
	// addrClosed: слот периода 1 ≥ 96 ч от now — окно заявляемого
	// периода закрыто с запасом
	var addrClosed, pubClosed, privClosed []byte
	for i := 0; i < 10000 && addrClosed == nil; i++ {
		a, p, v := softRegister(t, gen)
		if slotOff(a, 1) <= uint64(30*day-96*3600) {
			addrClosed, pubClosed, privClosed = a, p, v
		}
	}
	if addrClosed == nil {
		t.Fatal("не подобран адрес с закрытым окном заявляемого периода")
	}

	node, err := StartNode(gen, "full", "", nil, "", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer node.Close()

	// register подтягивает сеттлемент-время к живым часам (syncTime при
	// Apply): QueryClock возвращает s.NetTime, без транзакций окно
	// считалось бы от несеттлевого времени
	if res := submitRegister(t, node, gen, addrOpen, pubOpen, privOpen,
		"android-mint-open", "[]"); res != "" {
		t.Fatalf("register (open): %s", res)
	}
	if res := submitRegister(t, node, gen, addrClosed, pubClosed, privClosed,
		"android-mint-closed", "[]"); res != "" {
		t.Fatalf("register (closed): %s", res)
	}

	// открытый случай: заявляемый период 1, окно открыто — несмотря на
	// закрытое окно текущего периода 2 (регрессия находки №2)
	var mwOpen map[string]any
	if err := json.Unmarshal([]byte(node.MintWindowJSON(addrOpen)), &mwOpen); err != nil {
		t.Fatal(err)
	}
	if mwOpen["claim_period"] != float64(1) {
		t.Fatalf("claim_period = %v, хочу 1", mwOpen["claim_period"])
	}
	if mwOpen["open"] != true {
		t.Errorf("окно заявляемого периода 1 закрыто (слот в ±24 ч), " +
			"хотя окно текущего периода 2 закрыто — регресс находки №2")
	}
	// сверка с ядром: open == CheckMintChallengeWindow(claim_period)
	now := uint64(mwOpen["now"].(float64))
	if want := node.n.QueryMintWindowOpen(addrOpen, 1, now); mwOpen["open"] != want {
		t.Errorf("open = %v, ядро для периода 1 говорит %v", mwOpen["open"], want)
	}

	// закрытый случай: слот заявляемого периода далеко от now
	var mwClosed map[string]any
	if err := json.Unmarshal([]byte(node.MintWindowJSON(addrClosed)), &mwClosed); err != nil {
		t.Fatal(err)
	}
	if mwClosed["claim_period"] != float64(1) {
		t.Fatalf("claim_period = %v, хочу 1", mwClosed["claim_period"])
	}
	if mwClosed["open"] != false {
		t.Errorf("окно заявляемого периода 1 открыто вне ±48 ч от слота")
	}
}
