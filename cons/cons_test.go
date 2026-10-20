// Тесты пакета cons: чекпоинты (сборка/проверка цепочки/подписи), режимы
// и seed_sunset, Snowball-голосование, календарь периодов UTC,
// канонизация журнала и replay (INV-3 с правкой этапа 5).
package cons

import (
	"crypto/ecdsa"
	"math/rand"
	"testing"

	"github.com/rumatakira74/equality-trust-networks/core/crypto"
	"github.com/rumatakira74/equality-trust-networks/core/ledger"
	"github.com/rumatakira74/equality-trust-networks/core/params"
	"github.com/rumatakira74/equality-trust-networks/core/wire"
)

// consEnv — окружение: genesis с посевным кругом, ядро, ключи.
type consEnv struct {
	t       *testing.T
	gp      *wire.GenesisPackage
	keys    map[string]*ecdsa.PrivateKey
	addrs   map[string][]byte
	ledger  *ledger.Ledger
	clock   *ledger.FixedClock
	rootKey *ecdsa.PrivateKey
}

func newConsEnv(t *testing.T, opts ...func(*wire.ProfileParams)) *consEnv {
	t.Helper()
	gp := params.TestGenesis()
	prof := gp.Manifest.Profile
	prof.Pow = &wire.PowParams{
		TargetLight: 0x0010_0000_0000_0000, Argon2MemoryKiB: 8,
		Argon2Iterations: 1, Argon2Parallelism: 1, TargetHard: 0x0010_0000_0000_0000,
	}
	prof.FreshTransferWindowPeriods = 0
	prof.EpochSeconds = 86400
	for _, o := range opts {
		o(prof)
	}
	e := &consEnv{t: t, gp: gp, keys: make(map[string]*ecdsa.PrivateKey),
		addrs: make(map[string][]byte)}
	var circle []wire.SeedMember
	for _, id := range []string{"S1", "S2", "S3"} {
		priv := e.key(id)
		pub := crypto.MarshalPubKey(&priv.PublicKey)
		circle = append(circle, wire.SeedMember{
			Address: e.addr(id), PublicKey: pub,
		})
	}
	gp.Manifest.SeedCircle = circle
	gp.Genesis.GenesisTime = 1_700_000_000
	mh := gp.Manifest.ManifestHash()
	gp.Genesis.ManifestHash = mh[:]
	clock := &ledger.FixedClock{T: 1_700_000_000, E: 1, P: 1}
	l, err := ledger.New(gp, clock, ledger.Options{SkipMintChallengeWindow: true})
	if err != nil {
		t.Fatalf("genesis: %v", err)
	}
	e.ledger = l
	e.clock = clock
	return e
}

func (e *consEnv) key(id string) *ecdsa.PrivateKey {
	if k, ok := e.keys[id]; ok {
		return k
	}
	k := crypto.DeterministicKey("cons-test:" + id)
	e.keys[id] = k
	return k
}

func (e *consEnv) addr(id string) []byte {
	if a, ok := e.addrs[id]; ok {
		return a
	}
	pub := crypto.MarshalPubKey(&e.key(id).PublicKey)
	fp := crypto.Fingerprint(pub)
	a := crypto.DeriveAddress(fp, e.gp.Genesis.NetworkID)
	e.addrs[id] = a
	return a
}

// pubOf — ключи подписантов (кошельки посевного круга).
func (e *consEnv) pubOf(addr []byte) (*ecdsa.PublicKey, bool) {
	for id, a := range e.addrs {
		if string(a) == string(addr) {
			return &e.key(id).PublicKey, true
		}
	}
	return nil, false
}

// TestCheckpointChainAndThreshold — сборка, подписи N-из-M, проверка
// цепочки и предела сдвига времени (spec/07 7.2.1).
func TestCheckpointChainAndThreshold(t *testing.T) {
	e := newConsEnv(t)
	p := DefaultParams()
	cp1, err := Build(e.gp.Genesis.NetworkID, 1, e.ledger, nil,
		e.gp.Genesis.GenesisTime+p.EpochSeconds, 1, nil, 0, nil)
	if err != nil {
		t.Fatal(err)
	}
	// participants/threshold для переходного режима (посевной круг)
	part, n, err := Participants(wire.CheckpointModeDecentralized, e.ledger, e.gp, p, 1, nil)
	if err != nil {
		t.Fatal(err)
	}
	cp1.Participants, cp1.Threshold = part, n
	if len(part) != 3 || n != 2 {
		t.Fatalf("посевной круг: M=%d N=%d, ожидается 3/2", len(part), n)
	}
	// подписи: 2 из 3
	col := NewCollector(cp1, e.pubOf)
	for _, id := range []string{"S1", "S2"} {
		share, err := (&Signer{Wallet: e.addr(id), Priv: e.key(id)}).Sign(cp1)
		if err != nil {
			t.Fatal(err)
		}
		if !col.Add(share) {
			t.Fatalf("подпись %s не принята", id)
		}
	}
	cp1.ThresholdSig = col.Ready()
	if cp1.ThresholdSig == nil {
		t.Fatal("порог не достигнут")
	}
	if err := Verify(cp1, nil, e.gp.Genesis.GenesisTime, p.EpochSeconds, p.MaxTimeShift, e.pubOf); err != nil {
		t.Fatalf("чекпоинт 1: %v", err)
	}
	// дубликат подписи не удваивает вес
	col2 := NewCollector(cp1, e.pubOf)
	share1, _ := (&Signer{Wallet: e.addr("S1"), Priv: e.key("S1")}).Sign(cp1)
	col2.Add(share1)
	col2.Add(share1)
	if col2.Count() != 1 {
		t.Fatalf("дубликат подписи учтён: %d", col2.Count())
	}

	// чекпоинт 2: неверный prev_hash отклоняется
	cp2, _ := Build(e.gp.Genesis.NetworkID, 2, e.ledger, cp1,
		cp1.NetworkTime+p.EpochSeconds, 1, part, n, nil)
	cp2.ThresholdSig = cp1.ThresholdSig
	cp2.PrevCheckpointHash = make([]byte, 32)
	if err := Verify(cp2, cp1, e.gp.Genesis.GenesisTime, p.EpochSeconds, p.MaxTimeShift, e.pubOf); err == nil {
		t.Fatal("битый prev_hash прошёл")
	}
	// сдвиг времени сверх предела отклоняется
	cp3, _ := Build(e.gp.Genesis.NetworkID, 2, e.ledger, cp1,
		cp1.NetworkTime+p.EpochSeconds+p.MaxTimeShift+1, 1, part, n, nil)
	cp3.ThresholdSig = cp1.ThresholdSig
	if err := Verify(cp3, cp1, e.gp.Genesis.GenesisTime, p.EpochSeconds, p.MaxTimeShift, e.pubOf); err == nil {
		t.Fatal("сдвиг времени сверх предела прошёл")
	}
	// корректная цепочка проходит
	cp4, _ := Build(e.gp.Genesis.NetworkID, 2, e.ledger, cp1,
		cp1.NetworkTime+p.EpochSeconds, 1, part, n, nil)
	col4 := NewCollector(cp4, e.pubOf)
	for _, id := range []string{"S2", "S3"} {
		share, _ := (&Signer{Wallet: e.addr(id), Priv: e.key(id)}).Sign(cp4)
		col4.Add(share)
	}
	cp4.ThresholdSig = col4.Ready()
	if err := Verify(cp4, cp1, e.gp.Genesis.GenesisTime, p.EpochSeconds, p.MaxTimeShift, e.pubOf); err != nil {
		t.Fatalf("чекпоинт 2: %v", err)
	}
}

// TestSeedSunsetModes — переходный режим (посевной круг) до sunset;
// после (active ≥ порога) — режим из манифеста.
func TestSeedSunsetModes(t *testing.T) {
	e := newConsEnv(t)
	p := DefaultParams()
	// до sunset: посевной круг (3 кошелька, N=2)
	part, _, err := Participants(wire.CheckpointModeDecentralized, e.ledger, e.gp, p, 1, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(part) != 3 {
		t.Fatalf("переходный режим: M=%d", len(part))
	}
	// sunset по эпохам (24): после — decentralized из active-кошельков
	// (в тесте active только посевной круг — выборка совпадает)
	part2, _, err := Participants(wire.CheckpointModeDecentralized, e.ledger, e.gp, p, 100, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(part2) != 3 {
		t.Fatalf("после sunset по эпохам: M=%d — ожидался decentralized", len(part2))
	}
	// members_quorum: все active, N = 2/3
	e.ledger.State.SeedRightsActive = false
	part3, n3, err := Participants(wire.CheckpointModeMembersQuorum, e.ledger, e.gp, p, 1, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(part3) != 3 || n3 != 2 {
		t.Fatalf("members_quorum: M=%d N=%d", len(part3), n3)
	}
}

// TestPeriodOfEpoch — календарь UTC (9.2): «1-е число месяца» = первая
// эпоха, начавшаяся после 00:00 UTC 1-го.
func TestPeriodOfEpoch(t *testing.T) {
	// genesis 15 января 2024 12:00 UTC
	g := uint64(1705315200)
	cases := []struct {
		epochStart uint64
		want       uint64
		what       string
	}{
		{g, 0, "та же точка"},
		{g + 10*86400, 0, "до 1 февраля"},
		{1706745600, 1, "1 февраля 00:00 UTC"}, // 2024-02-01
		{1706745600 + 3600, 1, "после 1 февраля"},
		{1709251200, 2, "1 марта 00:00 UTC"}, // 2024-03-01
		{1735689600, 12, "1 января 2025"},    // 2025-01-01
	}
	for _, c := range cases {
		if got := PeriodOfEpoch(g, c.epochStart); got != c.want {
			t.Errorf("%s: период %d, ожидается %d", c.what, got, c.want)
		}
	}
}

// TestSnowballDeterministic — голосование сходится к ветке большинства и
// детерминировано по seed.
func TestSnowballDeterministic(t *testing.T) {
	p := DefaultParams()
	sb := NewSnowball(p)
	key := "w|2"
	a, b := []byte("branch-A"), []byte("branch-B")
	sb.Observe(key, a, b)

	// 10 пиров: 7 за A (вес 1), 3 за B — кворум 0,6 достигается
	peers := make([]testPeer, 0, 10)
	for i := 0; i < 7; i++ {
		peers = append(peers, testPeer{pref: a, weight: 1})
	}
	for i := 0; i < 3; i++ {
		peers = append(peers, testPeer{pref: b, weight: 1})
	}
	rng := newTestRng(42)
	tp := &testPoller{peers: peers}
	finalizedAt := -1
	for round := 1; round <= 20; round++ {
		if sb.PollRound(key, tp, rng) {
			finalizedAt = round
			break
		}
	}
	if finalizedAt != p.SnowballBeta {
		t.Fatalf("финальность на раунде %d, ожидается %d (β)", finalizedAt, p.SnowballBeta)
	}
	st := sb.states[key]
	if string(st.Pref) != string(a) {
		t.Fatal("победила не ветка большинства")
	}
}

type testPeer struct {
	pref   []byte
	weight uint64
}

func (p testPeer) VotePreference(string) ([]byte, uint64, bool) {
	return p.pref, p.weight, true
}

type testPoller struct {
	peers []testPeer
}

// Sample — все пиры (выборка не меньше k).
func (t *testPoller) Sample(k int, rng *rand.Rand) []PollablePeer {
	out := make([]PollablePeer, 0, len(t.peers))
	for _, p := range t.peers {
		out = append(out, p)
	}
	return out
}

// newTestRng — детерминированный источник.
func newTestRng(seed int64) *rand.Rand {
	return rand.New(rand.NewSource(seed))
}

// TestCheckpointSkipGap — №42 (D), решение владельца, этап 9: пропуск
// застрявшей эпохи — гэп нумерации в цепочке разрешён (пропущенная эпоха
// закрыта пустой, маркер — разрыв), регрессия нумерации запрещена; окно
// времени (9.2) — на чекпоинт, без масштабирования на пропуски.
func TestCheckpointSkipGap(t *testing.T) {
	e := newConsEnv(t)
	p := DefaultParams()
	cp1, err := Build(e.gp.Genesis.NetworkID, 1, e.ledger, nil,
		e.gp.Genesis.GenesisTime+p.EpochSeconds, 1, nil, 0, nil)
	if err != nil {
		t.Fatal(err)
	}
	part, n, err := Participants(wire.CheckpointModeDecentralized, e.ledger, e.gp, p, 1, nil)
	if err != nil {
		t.Fatal(err)
	}
	cp1.Participants, cp1.Threshold = part, n
	col := NewCollector(cp1, e.pubOf)
	for _, id := range []string{"S1", "S2"} {
		share, _ := (&Signer{Wallet: e.addr(id), Priv: e.key(id)}).Sign(cp1)
		col.Add(share)
	}
	cp1.ThresholdSig = col.Ready()
	if err := Verify(cp1, nil, e.gp.Genesis.GenesisTime, p.EpochSeconds, p.MaxTimeShift, e.pubOf); err != nil {
		t.Fatalf("чекпоинт 1: %v", err)
	}

	// чекпоинт эпохи 3 поверх эпохи 1 (эпоха 2 застряла — закрыта пустой)
	cp3, err := Build(e.gp.Genesis.NetworkID, 3, e.ledger, cp1,
		cp1.NetworkTime+p.EpochSeconds, 1, part, n, nil)
	if err != nil {
		t.Fatal(err)
	}
	col3 := NewCollector(cp3, e.pubOf)
	for _, id := range []string{"S1", "S2"} {
		share, _ := (&Signer{Wallet: e.addr(id), Priv: e.key(id)}).Sign(cp3)
		col3.Add(share)
	}
	cp3.ThresholdSig = col3.Ready()
	if err := Verify(cp3, cp1, e.gp.Genesis.GenesisTime, p.EpochSeconds, p.MaxTimeShift, e.pubOf); err != nil {
		t.Fatalf("пропуск застрявшей эпохи отклонён: %v", err)
	}

	// регрессия нумерации (эпоха 2 после эпохи 3) запрещена: Build
	// отклоняет, Verify (если бы пропустили) — тоже
	if cpBack, err := Build(e.gp.Genesis.NetworkID, 2, e.ledger, cp3,
		cp3.NetworkTime+p.EpochSeconds, 1, part, n, nil); err == nil && cpBack != nil {
		cpBack.ThresholdSig = cp3.ThresholdSig
		if Verify(cpBack, cp3, e.gp.Genesis.GenesisTime, p.EpochSeconds, p.MaxTimeShift, e.pubOf) == nil {
			t.Fatal("регрессия нумерации чекпоинтов прошла")
		}
		t.Fatal("Build принял регрессию нумерации")
	}
}
