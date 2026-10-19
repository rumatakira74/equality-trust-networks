// Пакет sim: детерминированная headless-симуляция реестра (9.1: «проверить
// до реализации» — сотни виртуальных узлов на одном компьютере, KNOWLEDGE
// 6.5). Поверх тех же core/ledger и cons; виртуальный транспорт вместо
// libp2p. Воспроизводима по seed; сценарии — sim/scenarios.go.
package sim

import (
	"crypto/ecdsa"
	"fmt"
	"math/rand"

	"github.com/rumatakira74/equality-trust-networks/cons"
	"github.com/rumatakira74/equality-trust-networks/core/crypto"
	"github.com/rumatakira74/equality-trust-networks/core/ledger"
	"github.com/rumatakira74/equality-trust-networks/core/params"
	"github.com/rumatakira74/equality-trust-networks/core/wire"
)

// Node — виртуальный узел: ядро, кошелёк, голосование, цепочка чекпоинтов.
type Node struct {
	ID        int
	Wallet    []byte
	Key       *ecdsa.PrivateKey
	Ledger    *ledger.Ledger
	Snowball  *cons.Snowball
	Chain     []*wire.Checkpoint          // финализированные чекпоинты
	Collector map[string]*cons.Collector  // checkpoint_hash -> сбор подписей
	Proposals map[string]*wire.Checkpoint // checkpoint_hash -> предложение
	Malicious bool
	Offline   bool
	Group     int // разбиение сети (partition): 0 — общая группа

	// allTxs — все увиденные транзакции в порядке прихода (журнал
	// хранения: обе ветки конфликтов; spec/04 INV-1c).
	allTxs        []*wire.Transaction
	seen          map[string]bool
	applied       map[string]bool   // provisional-принятые txid
	branchApplied map[string][]byte // конфликтный ключ -> txid применённой ветки
	journalAtCP   int               // длина журнала на последнем чекпоинте
	// lastChainGetRound — раунд последнего запроса цепочки (rate limit).
	lastChainGetRound int
	rebuilds          int // перестройки состояния (откаты веток)
	rolledBack        int // вытесненные валидные транзакции
}

// Network — виртуальная сеть: раунды, детерминированный rng.
type Network struct {
	Nodes   []*Node
	rng     *rand.Rand
	Params  cons.Params
	GP      *wire.GenesisPackage
	Keys    map[string]*ecdsa.PrivateKey
	Addrs   map[string][]byte
	Time    uint64 // виртуальное сетевое время (Unix-секунды)
	RoundN  int    // номер раунда
	queue   []*msg
	pubs    map[string]*ecdsa.PublicKey // addr -> ключ (узлы + посевной круг)
	Metrics Metrics
	// conflictInjected — раунд появления конфликта (для метрики финальности).
	conflictInjected map[string]int

	// conflictFinalized — конфликт -> раунд первой финализации.
	conflictFinalized map[string]int
}

// Metrics — метрики симуляции (reports/REPORT-STAGE-5.md).
type Metrics struct {
	Rounds             int
	ConflictsResolved  int
	RoundsToFinality   []int // раунды от инъекции конфликта до финальности
	Rebuilds           int
	RolledBackTxs      int
	Checkpoints        int
	CheckpointBytes    int
	VoteMessages       int
	SignFailedNoQuorum int // раундов без чекпоинта из-за отсутствия кворума
	LastVerifyError    string
	ReplayFailures     int
}

// msg — виртуальное сообщение (доставляется в следующем раунде).
type msg struct {
	from  int
	kind  int
	tx    *wire.Transaction
	share wire.SignatureShare
	cp    *wire.Checkpoint
	chain []*wire.Checkpoint
}

const (
	msgTx = iota
	msgShare
	msgCheckpoint
	msgChainGet
	msgChain
)

// New — сеть из n узлов (детерминированно по seed). Кошельки узлов —
// pre-state active с детерминированным возрастом: консенсус тестируется
// на готовом социальном графе (механика активации — ядро этапа 3).
// Config — конфигурация симуляции (genesis собирается внутри: посевной
// круг — первые min(3, n) кошельков узлов).
type Config struct {
	Mode             wire.CheckpointMode
	SeedSunsetActive uint64 // дефолт 100 (№15); в сценариях — меньше
	SeedSunsetEpochs uint64 // дефолт 24
	EpochSeconds     uint64 // дефолт 24 ч (№16)
	GenesisTime      uint64 // дефолт — фиксированная точка (детерминизм)
}

// Option — настройка симуляции.
type Option func(*Config)

func WithMode(m wire.CheckpointMode) Option { return func(c *Config) { c.Mode = m } }

func WithSeedSunset(active, epochs uint64) Option {
	return func(c *Config) { c.SeedSunsetActive, c.SeedSunsetEpochs = active, epochs }
}

// New — сеть из n узлов (детерминированно по seed). Кошельки узлов —
// pre-state active с детерминированным возрастом и балансом: консенсус
// тестируется на готовом социальном графе (механика активации и выпуска —
// ядро этапа 3; здесь — финальность).
func New(seed int64, n int, opts ...Option) *Network {
	cfg := Config{
		Mode: wire.CheckpointModeDecentralized, SeedSunsetActive: 100,
		SeedSunsetEpochs: 24, EpochSeconds: 24 * 3600,
		GenesisTime: 1705315200, // 2024-01-15 12:00 UTC (детерминизм)
	}
	for _, o := range opts {
		o(&cfg)
	}
	net := &Network{
		rng: rand.New(rand.NewSource(seed)),
		Params: cons.Params{
			EpochSeconds: cfg.EpochSeconds, MaxTimeShift: cfg.EpochSeconds / 4,
			SnowballK: 10, SnowballAlpha: 0.7, SnowballBeta: 4,
			ValidatorsM: 21, SeedSunsetActive: cfg.SeedSunsetActive,
			SeedSunsetEpochs: cfg.SeedSunsetEpochs,
		},
		Keys:              make(map[string]*ecdsa.PrivateKey),
		Addrs:             make(map[string][]byte),
		pubs:              make(map[string]*ecdsa.PublicKey),
		conflictInjected:  make(map[string]int),
		conflictFinalized: make(map[string]int),
	}
	networkID := []byte("tnp-simnet-0001")
	for i := 0; i < n; i++ {
		id := fmt.Sprintf("node-%d", i)
		priv := crypto.DeterministicKey("sim:" + id)
		net.Keys[id] = priv
		net.Addrs[id] = crypto.DeriveAddress(
			crypto.Fingerprint(crypto.MarshalPubKey(&priv.PublicKey)), networkID)
		net.pubs[string(net.Addrs[id])] = &priv.PublicKey
	}
	// genesis: тестовый профиль, малые PoW, посевной круг — первые узлы
	gp := params.TestGenesis()
	gp.Genesis.NetworkID = networkID
	gp.Manifest.NetworkID = networkID
	prof := gp.Manifest.Profile
	prof.Pow = &wire.PowParams{
		TargetLight: 0x0010_0000_0000_0000, Argon2MemoryKiB: 8,
		Argon2Iterations: 1, Argon2Parallelism: 1, TargetHard: 0x0010_0000_0000_0000,
	}
	prof.FreshTransferWindowPeriods = 0
	prof.EpochSeconds = cfg.EpochSeconds
	prof.CheckpointMode = cfg.Mode
	prof.EclipseN = 3
	circleSize := 3
	if n < circleSize {
		circleSize = n
	}
	for i := 0; i < circleSize; i++ {
		id := fmt.Sprintf("node-%d", i)
		gp.Manifest.SeedCircle = append(gp.Manifest.SeedCircle, wire.SeedMember{
			Address: net.Addrs[id], PublicKey: crypto.MarshalPubKey(&net.Keys[id].PublicKey),
		})
	}
	gp.Manifest.SeedSunset = &wire.SeedSunset{
		UseActiveWallets: true, ActiveWallets: cfg.SeedSunsetActive, DeadlineEpoch: cfg.SeedSunsetEpochs,
	}
	gp.Genesis.GenesisTime = cfg.GenesisTime
	mh := gp.Manifest.ManifestHash()
	gp.Genesis.ManifestHash = mh[:]
	net.GP = gp
	net.Time = cfg.GenesisTime
	for i := 0; i < n; i++ {
		net.addNode(i)
	}
	// pre-state: каждый узел знает все кошельки (синхронизированный граф)
	for _, n1 := range net.Nodes {
		for _, n2 := range net.Nodes {
			w := n2.Ledger.State.Wallet(n2.Wallet)
			cp := *w
			n1.Ledger.State.Wallets[string(n2.Wallet)] = &cp
			n1.Ledger.State.FingerprintIndex[string(w.Fingerprint)] = n2.Wallet
		}
		n1.Ledger.State.TotalMinted = uint64(len(net.Nodes)) * 10000
	}
	return net
}

func (net *Network) addNode(i int) {
	id := fmt.Sprintf("node-%d", i)
	addr := net.Addrs[id]
	l, err := ledger.New(net.GP, &netClock{net: net}, ledger.Options{SkipMintChallengeWindow: true})
	if err != nil {
		panic(err)
	}
	age := uint64(30 + (i*37)%700)
	w := &ledger.Wallet{
		Address: addr, PublicKey: &net.Keys[id].PublicKey,
		Fingerprint:    crypto.Fingerprint(crypto.MarshalPubKey(&net.Keys[id].PublicKey)),
		Status:         wire.StatusActive,
		RegisteredAt:   net.GP.Genesis.GenesisTime - age*86400,
		ActivatedAt:    net.GP.Genesis.GenesisTime - age*86400,
		ActivatedEpoch: 1, ActivatedPeriod: 1, LastClaimedPeriod: 0,
		LastTouchEpoch: 1, DelayedThreshold: 3000, Liquid: 10000,
	}
	l.State.Wallets[string(addr)] = w
	l.State.FingerprintIndex[string(w.Fingerprint)] = addr
	net.Nodes = append(net.Nodes, &Node{
		ID: i, Wallet: addr, Key: net.Keys[id], Ledger: l,
		Snowball:      cons.NewSnowball(net.Params),
		Collector:     make(map[string]*cons.Collector),
		Proposals:     make(map[string]*wire.Checkpoint),
		seen:          make(map[string]bool),
		applied:       make(map[string]bool),
		branchApplied: make(map[string][]byte),
	})
}

// prestateInto — исходный (genesis-времени) pre-state всех кошельков
// узлов в свежее ядро: детерминированные возраст и баланс (как addNode).
func (net *Network) prestateInto(l *ledger.Ledger) {
	for i, n := range net.Nodes {
		id := fmt.Sprintf("node-%d", i)
		age := uint64(30 + (i*37)%700)
		w := &ledger.Wallet{
			Address:        n.Wallet,
			PublicKey:      &net.Keys[id].PublicKey,
			Fingerprint:    crypto.Fingerprint(crypto.MarshalPubKey(&net.Keys[id].PublicKey)),
			Status:         wire.StatusActive,
			RegisteredAt:   net.GP.Genesis.GenesisTime - age*86400,
			ActivatedAt:    net.GP.Genesis.GenesisTime - age*86400,
			ActivatedEpoch: 1, ActivatedPeriod: 1, LastClaimedPeriod: 0,
			LastTouchEpoch: 1, DelayedThreshold: 3000, Liquid: 10000,
		}
		l.State.Wallets[string(n.Wallet)] = w
		l.State.FingerprintIndex[string(w.Fingerprint)] = n.Wallet
	}
	l.State.TotalMinted = uint64(len(net.Nodes)) * 10000
}

// netClock — консенсусные часы (ledger.EpochSource): время виртуальное,
// эпоха/период — по последнему финализированному чекпоинту сети.
type netClock struct {
	net *Network
}

func (c *netClock) NetTime() uint64 { return c.net.Time }

func (c *netClock) Epoch() uint64 {
	for _, n := range c.net.Nodes {
		if len(n.Chain) > 0 {
			return n.Chain[len(n.Chain)-1].Epoch
		}
	}
	return 0
}

func (c *netClock) Period() uint64 {
	for _, n := range c.net.Nodes {
		if len(n.Chain) > 0 {
			cp := n.Chain[len(n.Chain)-1]
			return cons.PeriodOfEpoch(c.net.GP.Genesis.GenesisTime, cp.NetworkTime)
		}
	}
	return cons.PeriodOfEpoch(c.net.GP.Genesis.GenesisTime, c.net.GP.Genesis.GenesisTime)
}

// pubOf — ключ подписанта (узел или посевной круг).
func (net *Network) pubOf(addr []byte) (*ecdsa.PublicKey, bool) {
	pub, ok := net.pubs[string(addr)]
	return pub, ok
}

// SetPartition — разбиение: узел группы g видит только свою группу.
func (net *Network) SetPartition(groups map[int]int) {
	for id, g := range groups {
		net.Nodes[id].Group = g
	}
}

// Heal — снять разбиение.
func (net *Network) Heal() {
	for _, n := range net.Nodes {
		n.Group = 0
	}
}

func (net *Network) reachable(from, to int) bool {
	a, b := net.Nodes[from], net.Nodes[to]
	if a.Offline || b.Offline {
		return false
	}
	return a.Group == b.Group
}

// Broadcast — поставить сообщение в очередь (доставка в следующем раунде
// всем достижимым; между группами/офлайн теряется).
func (net *Network) Broadcast(from int, m *msg) {
	net.queue = append(net.queue, m)
}

func (net *Network) deliverAll() {
	q := net.queue
	net.queue = nil
	for _, m := range q {
		for _, n := range net.Nodes {
			if n.ID != m.from && net.reachable(m.from, n.ID) {
				n.process(net, m)
			}
		}
	}
}

// Round — один раунд: доставка, голосование, чекпоинты; время += epoch/4.
func (net *Network) Round() {
	net.RoundN++
	net.deliverAll()
	for _, n := range net.Nodes {
		if n.Offline {
			continue
		}
		n.voteRound(net)
		n.checkpointPhase(net)
	}
	net.Time += net.Params.EpochSeconds / 4
}

// Run — прогон rounds раундов.
func (net *Network) Run(rounds int) {
	for i := 0; i < rounds; i++ {
		net.Round()
	}
	net.Metrics.Rounds = net.RoundN
}

// InjectTx — инъекция транзакции от узла from (как будто опубликована).
func (net *Network) InjectTx(from int, tx *wire.Transaction) {
	net.Nodes[from].applyTx(net, tx)
	net.Broadcast(from, &msg{from: from, kind: msgTx, tx: tx})
}

// InjectConflict — инъекция пары конфликтных веток: A — одним узлам,
// B — другим (порядок прихода разный; A/B — также предпочтение
// честных/атакующего в сценариях).
func (net *Network) InjectConflict(a, b *wire.Transaction, firstForA []int) {
	key := cons.ConflictKey(a)
	net.conflictInjected[key] = net.RoundN
	inA := make(map[int]bool)
	for _, i := range firstForA {
		inA[i] = true
	}
	for _, n := range net.Nodes {
		first, second := b, a
		if inA[n.ID] {
			first, second = a, b
		}
		if n.Offline {
			continue
		}
		n.applyTx(net, first)
		n.applyTx(net, second)
		net.Broadcast(n.ID, &msg{from: n.ID, kind: msgTx, tx: first})
		net.Broadcast(n.ID, &msg{from: n.ID, kind: msgTx, tx: second})
	}
}
