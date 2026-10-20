// Тестовая база этапа 4: genesis с посевным кругом, детерминированные ключи,
// сборка валидных транзакций (как e2e этапа 3), фабрика узлов.
// Порты — только localhost/эмерджентные (правило 5 промпта).
package net

import (
	"context"
	"crypto/ecdsa"
	"crypto/x509"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"math/rand"
	"path/filepath"
	"testing"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
	ma "github.com/multiformats/go-multiaddr"

	"github.com/rumatakira74/equality-trust-networks/core/attest"
	"github.com/rumatakira74/equality-trust-networks/core/crypto"
	"github.com/rumatakira74/equality-trust-networks/core/ledger"
	"github.com/rumatakira74/equality-trust-networks/core/params"
	"github.com/rumatakira74/equality-trust-networks/core/wire"
)

// testEnv — общее окружение сети (genesis, ключи, корень аттестации).
type testEnv struct {
	t           *testing.T
	gp          *wire.GenesisPackage
	rootKey     *ecdsa.PrivateKey
	rootCert    *x509.Certificate
	keys        map[string]*ecdsa.PrivateKey
	addrs       map[string][]byte
	genesisT    uint64
	epoch       uint64
	period      uint64
	now         uint64
	netAccess   wire.NetworkAccess
	nodeCounter int
}

// newTestEnv — окружение с genesis: посевной круг V1..V3, малые PoW-таргеты
// (вопрос №9: финализация замерами — этап 6), genesis за 65 суток до now
// (текущий период 2: посевной круг может выпустить завершённый период 1).
func newTestEnv(t *testing.T, opts ...func(*wire.ProfileParams)) *testEnv {
	t.Helper()
	now := uint64(time.Now().Unix())
	genesisT := now - 65*86400
	gp := params.TestGenesis()
	prof := gp.Manifest.Profile
	// малые таргеты для скорости тестов (реальные — вопрос №9, этап 6)
	prof.Pow = &wire.PowParams{
		TargetLight: 0x0010_0000_0000_0000, Argon2MemoryKiB: 8,
		Argon2Iterations: 1, Argon2Parallelism: 1, TargetHard: 0x0010_0000_0000_0000,
	}
	prof.FreshTransferWindowPeriods = 0 // мгновенная переводимость (тесты)
	// Окно челленджа выпуска (7.6) в сетевых тестах открыто весь период:
	// тесты проверяют транспорт/синхронизацию/консенсус и минтят в
	// произвольные моменты; слот челленджа при этом закрыт почти всегда
	// (№39: окно заявляемого периода достижимо только для «поздних»
	// слотов). Само окно покрыто тестами core/ledger и прогоном
	// тестовой сети (cmd/tnp-testnet, этап 8).
	prof.MintChallengeWindowHours = 0
	for _, o := range opts {
		o(prof)
	}
	env := &testEnv{
		t: t, gp: gp, keys: make(map[string]*ecdsa.PrivateKey),
		addrs: make(map[string][]byte), genesisT: genesisT,
		epoch:  (now - genesisT) / prof.EpochSeconds,
		period: (now - genesisT) / (30 * 86400),
		now:    now,
	}
	// посевной круг из манифеста: адреса и ключи (все узлы строят состояние
	// из одного genesis — сетевая консистентность)
	var circle []wire.SeedMember
	for _, id := range []string{"V1", "V2", "V3"} {
		priv := env.key(id)
		pub := crypto.MarshalPubKey(&priv.PublicKey)
		circle = append(circle, wire.SeedMember{
			Address: env.addr(id), PublicKey: pub,
		})
	}
	gp.Manifest.SeedCircle = circle
	gp.Genesis.GenesisTime = genesisT
	gp.Genesis.EpochZero = 0
	mh := gp.Manifest.ManifestHash()
	gp.Genesis.ManifestHash = mh[:]
	// синтетический корень аттестации (проверка R2; реальные корни — Google)
	env.rootKey = crypto.DeterministicKey("net-test-root")
	cert, _, err := attest.SyntheticRoot("net-test", env.rootKey)
	if err != nil {
		t.Fatalf("synthetic root: %v", err)
	}
	env.rootCert = cert
	return env
}

func (e *testEnv) key(id string) *ecdsa.PrivateKey {
	if k, ok := e.keys[id]; ok {
		return k
	}
	k := crypto.DeterministicKey("net-test:" + id)
	e.keys[id] = k
	return k
}

func (e *testEnv) addr(id string) []byte {
	if a, ok := e.addrs[id]; ok {
		return a
	}
	pub := crypto.MarshalPubKey(&e.key(id).PublicKey)
	fp := crypto.Fingerprint(pub)
	a := crypto.DeriveAddress(fp, e.gp.Genesis.NetworkID)
	e.addrs[id] = a
	return a
}

// nodeOpts — конфиг узла в окружении.
type nodeOpts struct {
	level       NodeLevel
	db          string // "" — без SQLite
	manual      []string
	relay       []string
	noAnnounce  bool
	forcePublic bool
	walletID    string // кошелёк узла: подпись чекпоинтов, вес голоса
	consEpoch   uint64 // короткие эпохи для тестов (0 — из genesis)
	ownWallet   []byte
	inviteOnly  bool
	minPeers    int
	corroN      int
	linkGroups  bool // оценка несвязанности по подсетям (№35, этап 6)
	log         bool
}

// newNode — узел в окружении (эмерджентные localhost-порты).
func (e *testEnv) newNode(o nodeOpts) *Node {
	e.t.Helper()
	cfg := &NodeConfig{
		Genesis:        e.gp,
		NodeKey:        crypto.DeterministicKey("net-node:" + fmt.Sprint(rand.Int63())),
		Level:          o.level,
		DBPath:         o.db,
		ListenAddrs:    []string{"/ip4/127.0.0.1/udp/0/quic-v1", "/ip4/127.0.0.1/tcp/0"},
		ManualAddrs:    o.manual,
		RelayAddrs:     o.relay,
		NoAnnounce:     o.noAnnounce,
		ForcePublic:    o.forcePublic,
		OwnWallet:      o.ownWallet,
		InviteOnly:     o.inviteOnly,
		MinPeers:       o.minPeers,
		CorroborationN: o.corroN,
		LinkGroups:     o.linkGroups,
		TrustedRoots:   []*x509.Certificate{e.rootCert},
		PollInterval:   500 * time.Millisecond,
	}
	if o.walletID != "" {
		cfg.WalletKey = e.key(o.walletID)
		cfg.WalletAddr = e.addr(o.walletID)
	}
	cfg.ConsensusEpochSeconds = o.consEpoch
	cfg.VoteInterval = 300 * time.Millisecond
	if o.log {
		nodeNo := e.nodeCounter
		cfg.Log = func(f string, a ...any) { e.t.Logf("[узел %d] "+f, append([]any{nodeNo}, a...)...) }
	}
	e.nodeCounter++
	n, err := NewNode(cfg)
	if err != nil {
		e.t.Fatalf("newNode: %v", err)
	}
	return n
}

// start — запуск узла.
func (e *testEnv) start(n *Node) {
	e.t.Helper()
	if err := n.Start(context.Background()); err != nil {
		e.t.Fatalf("start: %v", err)
	}
	e.t.Cleanup(func() { _ = n.Close() })
}

// addrInfo — адрес узла для ручного подключения (каскад, шаг 6).
func addrInfoOf(n *Node) (peer.AddrInfo, error) {
	if len(n.Host.Addrs()) == 0 {
		return peer.AddrInfo{}, fmt.Errorf("нет адресов у %s", n.Host.ID().ShortString())
	}
	return peer.AddrInfo{ID: n.Host.ID(), Addrs: n.Host.Addrs()}, nil
}

// manualAddrs — адреса узлов для конфигурации (QR/ручной ввод).
func manualAddrs(t *testing.T, nodes ...*Node) []string {
	t.Helper()
	var out []string
	for _, n := range nodes {
		ai, err := addrInfoOf(n)
		if err != nil {
			t.Fatal(err)
		}
		s, err := peer.AddrInfoToP2pAddrs(&ai)
		if err != nil {
			t.Fatal(err)
		}
		for _, a := range s {
			out = append(out, a.String())
		}
	}
	return out
}

// hexEncode — hex network_id (тема gossip).
func hexEncode(b []byte) string { return hex.EncodeToString(b) }

// maNew — multiaddr из строки.
func maNew(a string) ma.Multiaddr {
	m, err := ma.NewMultiaddr(a)
	if err != nil {
		panic(err)
	}
	return m
}

// firstAddr — первый анонсируемый адрес узла.
func firstAddr(n *Node) ma.Multiaddr {
	addrs := n.Host.Addrs()
	if len(addrs) == 0 {
		panic("нет адресов")
	}
	return addrs[0]
}

// addrInfoOfMA — AddrInfo из составного multiaddr (relay-circuit).
func addrInfoOfMA(m ma.Multiaddr) peer.AddrInfo {
	ai, err := peer.AddrInfoFromP2pAddr(m)
	if err != nil {
		panic(err)
	}
	return *ai
}

// waitPeers — ждать, пока у узла не будет min подключений.
func waitPeers(t *testing.T, n *Node, min int, timeout time.Duration) bool {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if len(n.Host.Network().Peers()) >= min {
			return true
		}
		time.Sleep(100 * time.Millisecond)
	}
	return false
}

// waitJournal — ждать длину журнала узла.
func waitJournal(t *testing.T, n *Node, want int, timeout time.Duration) bool {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if n.Stats().JournalLen >= want {
			return true
		}
		time.Sleep(100 * time.Millisecond)
	}
	return false
}

// ---------------------------------------------------------------------------
// Сборка транзакций (как e2e этапа 3, но часы — реального узла)
// ---------------------------------------------------------------------------

// register — REGISTER с синтетической аттестацией (R2).
func (e *testEnv) register(n *Node, id string) *wire.Transaction {
	e.t.Helper()
	priv := e.key(id)
	pub := crypto.MarshalPubKey(&priv.PublicKey)
	challenge := attest.ChallengeFormula(pub, crypto.ProtocolID, e.gp.Genesis.NetworkID)
	ext := attest.Extension{
		SecurityLevel: attest.LevelTEE, VerifiedBootState: attest.BootVerified,
		DeviceLocked: true, PatchYear: 2026, PatchMonth: 10,
	}
	leaf, err := attest.BuildLeaf(&priv.PublicKey, challenge, ext, e.rootKey, e.rootCert, 42)
	if err != nil {
		e.t.Fatalf("leaf: %v", err)
	}
	rp := &wire.RegisterPayload{
		AttestedPublicKey:    pub,
		AttestationChain:     attest.EncodeChain(leaf),
		AttestationChallenge: challenge,
	}
	tx := &wire.Transaction{
		NetworkID:       e.gp.Genesis.NetworkID,
		Wallet:          e.addr(id),
		Seq:             1,
		Type:            wire.Register,
		Payload:         rp,
		ProtocolVersion: e.gp.Genesis.ProtocolVersion,
		TimestampClaim:  uint64(time.Now().Unix()),
	}
	e.solveAndSign(tx, id, true)
	return tx
}

// tx — транзакция с корректным seq/prev по состоянию узла n.
// Чтение кошелька — под applyMu.RLock: фоновый gossip применяет
// транзакции параллельно с тестом (race-аудит этапа 9).
func (e *testEnv) tx(n *Node, id string, typ wire.TxType, payload wire.Payload, hard bool) *wire.Transaction {
	e.t.Helper()
	n.applyMu.RLock()
	w := n.Ledger.State.Wallet(e.addr(id))
	if w == nil {
		n.applyMu.RUnlock()
		e.t.Fatalf("кошелёк %s не зарегистрирован", id)
	}
	seq := w.Seq + 1
	prev := append([]byte(nil), w.TipTxID...)
	n.applyMu.RUnlock()
	tx := &wire.Transaction{
		NetworkID:       e.gp.Genesis.NetworkID,
		Wallet:          e.addr(id),
		Seq:             seq,
		Prev:            prev,
		Type:            typ,
		Payload:         payload,
		ProtocolVersion: e.gp.Genesis.ProtocolVersion,
		TimestampClaim:  uint64(time.Now().Unix()),
	}
	e.solveAndSign(tx, id, hard)
	return tx
}

// solveAndSign — PoW + подпись (как e2e этапа 3).
func (e *testEnv) solveAndSign(tx *wire.Transaction, id string, hard bool) {
	e.t.Helper()
	net := e.gp.Manifest.Profile
	if hard {
		tx.Pow = &wire.PowProof{Algorithm: wire.PowArgon2id}
		ap := crypto.Argon2Params{MemoryKiB: net.Pow.Argon2MemoryKiB,
			Iterations: net.Pow.Argon2Iterations, Parallelism: net.Pow.Argon2Parallelism}
		for n := uint64(0); ; n++ {
			var nb [8]byte
			binary.BigEndian.PutUint64(nb[:], n)
			tx.Pow.Nonce = nb[:]
			if crypto.VerifyHard(tx.SigningPreimage(), nb[:], net.Pow.TargetHard, ap) {
				break
			}
		}
	} else {
		tx.Pow = &wire.PowProof{Algorithm: wire.PowLight}
		for n := uint64(0); ; n++ {
			var nb [8]byte
			binary.BigEndian.PutUint64(nb[:], n)
			tx.Pow.Nonce = nb[:]
			if crypto.VerifyLight(tx.SigningPreimage(), nb[:], net.Pow.TargetLight) {
				break
			}
		}
	}
	sh := tx.SigningHash()
	sig, err := crypto.SignDER(e.key(id), sh[:])
	if err != nil {
		e.t.Fatalf("sign: %v", err)
	}
	tx.Signature = sig
}

// mintTx — выпуск завершённого периода 1 (для посевного круга; M3–M6).
func (e *testEnv) mintTx(n *Node, id string) *wire.Transaction {
	return e.tx(n, id, wire.Mint, &wire.MintPayload{Period: 1, Amount: 2000}, true)
}

// transferTx — немедленный перевод (сумма ≤ P; 8.3).
func (e *testEnv) transferTx(n *Node, from, to string, amount uint64) *wire.Transaction {
	return e.tx(n, from, wire.Transfer, &wire.TransferPayload{
		Send: &wire.TransferSend{Recipient: e.addr(to), Amount: amount},
	}, false)
}

// dbPath — временная база узла.
func dbPath(t *testing.T, name string) string {
	t.Helper()
	return filepath.Join(t.TempDir(), name)
}

var _ = ledger.FirstWins{}
