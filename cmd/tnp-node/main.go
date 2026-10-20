// tnp-node — headless-узел платформы сетей доверия (этап 4).
// Назначение: интеграционные тесты отдельными процессами и ручная проверка
// синхронизации на отдельных машинах (критерий готовности этапа 4;
// инструкция — reports/REPORT-STAGE-4.md).
//
// Флаги:
//
//	-level full|light      уровень узла (12.2; по умолчанию full)
//	-network-id            идентификатор сети (тестовый genesis)
//	-db                    путь к SQLite (по умолчанию — без персистентности)
//	-peer                  multiaddr пира (QR/ручной ввод; повторяется)
//	-relay                 multiaddr relay-узла (CGNAT; повторяется)
//	-no-announce           не анонсировать адреса (эмуляция CGNAT)
//	-force-public          форсировать публичную достижимость (relay-узел)
//	-own-wallet            адрес собственного кошелька (лёгкий узел)
//	-wallet                метка кошелька узла для подписи чекпоинтов
//	-wallet-domain         домен деривации кошелька: seed | wallet
//	-link-groups           несвязанность пиров по подсетям (№35);
//	                       false для флота на localhost (подэтап 8.1)
//	-status-file           файл JSON-статуса (периодически обновляется)
//	-demo-tx               опубликовать демонстрационные транзакции
//	                       (выпуск посевного кошелька + перевод)
//	-quiet                 не писать журнал в stderr
package main

import (
	"context"
	"crypto/ecdsa"
	"crypto/x509"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/rumatakira74/equality-trust-networks/core/attest"
	"github.com/rumatakira74/equality-trust-networks/core/crypto"
	"github.com/rumatakira74/equality-trust-networks/core/params"
	"github.com/rumatakira74/equality-trust-networks/core/store"
	"github.com/rumatakira74/equality-trust-networks/core/wire"
	tnp "github.com/rumatakira74/equality-trust-networks/mobile"
	"github.com/rumatakira74/equality-trust-networks/net"
)

func main() {
	level := flag.String("level", "full", "уровень узла: full | light")
	networkID := flag.String("network-id", "tnp-testnet-0001", "network_id тестовой сети")
	genesisFile := flag.String("genesis-file", "", "genesis.bin утилиты cmd/tnp-testnet (замена встроенному тестовому genesis)")
	nodeKey := flag.String("node-key", "", "seed ключа узла (Peer ID); по умолчанию — network-id+hostname")
	dbPath := flag.String("db", "", "путь к SQLite")
	statusFile := flag.String("status-file", "", "файл JSON-статуса")
	listen := flag.String("listen", "/ip4/0.0.0.0/udp/0/quic-v1,/ip4/0.0.0.0/tcp/0", "адреса прослушивания, через запятую")
	demoTx := flag.Bool("demo-tx", false, "опубликовать демонстрационные транзакции")
	noAnnounce := flag.Bool("no-announce", false, "не анонсировать адреса (CGNAT)")
	forcePublic := flag.Bool("force-public", false, "форсировать публичную достижимость (relay)")
	ownWallet := flag.String("own-wallet", "", "адрес собственного кошелька (light)")
	walletID := flag.String("wallet", "", "кошелёк узла для подписи чекпоинтов (метка посевного круга)")
	walletDomain := flag.String("wallet-domain", "seed", "деривация ключа кошелька: seed (tnp-node-seed:) | wallet (tnp-testnet-wallet:; участники утилиты tnp-testnet)")
	linkGroups := flag.Bool("link-groups", true, "несвязанность eclipse-пиров по подсетям (№35); false для localhost/флота — все пиры в одной /24")
	quiet := flag.Bool("quiet", false, "тихий режим")
	importDB := flag.String("import-db", "", "восстановить журнал из SQLite (БД с vouch-ами, недоступная для replay из-за №40); применять до старта сети")
	var peers, relays multiaddrList
	flag.Var(&peers, "peer", "multiaddr пира (повторяется)")
	flag.Var(&relays, "relay", "multiaddr relay-узла (повторяется)")
	flag.Parse()

	log := func(f string, a ...any) {
		if !*quiet {
			fmt.Fprintf(os.Stderr, "tnp-node: "+f+"\n", a...)
		}
	}

	gp := testGenesis(*networkID)
	var rootsDER []byte
	if *genesisFile != "" {
		// закрытая тестовая сеть (этап 8): genesis-пакет из файла утилиты
		// cmd/tnp-testnet; корни аттестации — та же синтетическая деривация
		// от network_id, что у приложения (mobile.TrustedRootsDER)
		b, err := os.ReadFile(*genesisFile)
		if err != nil {
			fatal("genesis-файл: %v", err)
		}
		decoded, err := wire.DecodeGenesisPackage(b)
		if err != nil {
			fatal("genesis-файл: %v", err)
		}
		gp = decoded
		mgen, err := tnp.LoadGenesis(b)
		if err != nil {
			fatal("genesis-файл: %v", err)
		}
		rootsDER = mgen.TrustedRootsDER()
		*networkID = string(gp.Genesis.NetworkID)
	}
	if *nodeKey == "" {
		hostname, _ := os.Hostname()
		*nodeKey = *networkID + "/" + hostname
	}
	cfg := &net.NodeConfig{
		Genesis:     gp,
		NodeKey:     crypto.DeterministicKey("tnp-node-key:" + *nodeKey),
		Level:       net.LevelFull,
		DBPath:      *dbPath,
		ListenAddrs: strings.Split(*listen, ","),
		ManualAddrs: peers,
		RelayAddrs:  relays,
		NoAnnounce:  *noAnnounce,
		ForcePublic: *forcePublic,
		// несвязанность eclipse-пиров по подсетям (№35, этап 6):
		// продакшен-дефолт — оценка включена (обоснование — отчёт этапа 6);
		// флот на localhost — false: /24-классификатор схлопывает все узлы
		// в одну группу связи (подэтап 8.1)
		LinkGroups: *linkGroups,
	}
	if *level == "light" {
		cfg.Level = net.LevelLight
		if *ownWallet == "" {
			fatal("лёгкому узлу нужен -own-wallet")
		}
		w, err := hex.DecodeString(*ownWallet)
		if err != nil || len(w) != 32 {
			fatal("-own-wallet: ожидается 32 байта в hex")
		}
		cfg.OwnWallet = w
	}
	// кошелёк узла: подпись чекпоинтов (этап 5) и вес голоса;
	// домен деривации: seed — посевной круг (как tx-драйверы tnp-testnet),
	// wallet — кошельки-участники tnp-testnet (tnp-testnet-wallet:),
	// чтобы узел-участник подписывал чекпоинты после seed_sunset
	if *walletID != "" {
		domain := "tnp-node-seed:"
		if *walletDomain == "wallet" {
			domain = "tnp-testnet-wallet:"
		}
		key := crypto.DeterministicKey(domain + *walletID)
		pub := crypto.MarshalPubKey(&key.PublicKey)
		fp := crypto.Fingerprint(pub)
		cfg.WalletKey = key
		cfg.WalletAddr = crypto.DeriveAddress(fp, []byte(*networkID))
	}
	// корни аттестации: синтетический корень (распространение реальных — этап 6);
	// для genesis-файла — деривация приложения (mobile), общая с cmd/tnp-testnet
	if rootsDER != nil {
		if cert, err := x509.ParseCertificate(rootsDER); err == nil {
			cfg.TrustedRoots = []*x509.Certificate{cert}
		}
	} else {
		rootKey := crypto.DeterministicKey("tnp-node-root:" + *networkID)
		cert, _, err := attest.SyntheticRoot("tnp-node", rootKey)
		if err != nil {
			fatal("synthetic root: %v", err)
		}
		cfg.TrustedRoots = []*x509.Certificate{cert}
	}
	cfg.Log = log

	node, err := net.NewNode(cfg)
	if err != nil {
		fatal("узел: %v", err)
	}
	// восстановление журнала из БД (блокер 8.1-2, авторизовано
	// владельцем): replay при старте (cons.Replay) делегирует эпоху
	// живым часам (№40) и падает на vouch-ах прошлых эпох
	// (E_VOUCH_EPOCH) — журнал применяется пинированным путём
	// (семантика сетевой сверки). Узел стартует с чистой/новой БД,
	// -import-db указывает на старую
	if *importDB != "" {
		st, err := store.Open(*importDB)
		if err != nil {
			fatal("import-db: %v", err)
		}
		txs, err := st.LoadTxs()
		_ = st.Close()
		if err != nil {
			fatal("import-db: %v", err)
		}
		applied := node.ImportJournal(txs)
		log("импорт журнала из %s: применено %d из %d", *importDB, applied, len(txs))
	}
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	if err := node.Start(ctx); err != nil {
		fatal("запуск: %v", err)
	}
	log("запущен: %s (уровень %s)", node.Host.ID().ShortString(), *level)
	for _, a := range node.Host.Addrs() {
		log("адрес: %s/p2p/%s", a, node.Host.ID())
	}

	if *demoTx {
		go demoTransactions(ctx, node, gp)
	}

	// статус-файл: для мультипроцессных тестов и ручной проверки
	if *statusFile != "" {
		go func() {
			t := time.NewTicker(500 * time.Millisecond)
			defer t.Stop()
			for {
				select {
				case <-ctx.Done():
					return
				case <-t.C:
					writeStatus(*statusFile, node)
				}
			}
		}()
	}

	<-ctx.Done()
	_ = node.Close()
}

// demoTransactions — выпуск посевного кошелька V1 и перевод V2 (M3–M6, T-таблицы).
func demoTransactions(ctx context.Context, node *net.Node, gp *wire.GenesisPackage) {
	time.Sleep(2 * time.Second) // дать bootstrap устояться
	env := &demoEnv{gp: gp}
	mint := env.mintTx(node)
	if res, err := node.SubmitTx(ctx, mint); err != nil {
		env.logf("mint: %v", err)
		return
	} else if res.Outcome.String() != "accept" {
		env.logf("mint: %s", res.Code)
		return
	}
	env.logf("mint опубликован")
	tr := env.transferTx(node, 100)
	if res, err := node.SubmitTx(ctx, tr); err != nil {
		env.logf("transfer: %v", err)
	} else if res.Outcome.String() != "accept" {
		env.logf("transfer: %s", res.Code)
	} else {
		env.logf("transfer опубликован")
	}
}

type demoEnv struct{ gp *wire.GenesisPackage }

func (d *demoEnv) logf(f string, a ...any) { fmt.Fprintf(os.Stderr, "tnp-node demo: "+f+"\n", a...) }

func (d *demoEnv) key(id string) *ecdsa.PrivateKey {
	return crypto.DeterministicKey("tnp-node-seed:" + id)
}

func (d *demoEnv) addr(id string) []byte {
	pub := crypto.MarshalPubKey(&d.key(id).PublicKey)
	fp := crypto.Fingerprint(pub)
	return crypto.DeriveAddress(fp, d.gp.Genesis.NetworkID)
}

// mintTx — выпуск завершённого периода 1 посевным кошельком V1.
func (d *demoEnv) mintTx(node *net.Node) *wire.Transaction {
	return d.build(node, "V1", wire.Mint, &wire.MintPayload{Period: 1, Amount: 2000}, true)
}

// transferTx — перевод V1 -> V2.
func (d *demoEnv) transferTx(node *net.Node, amount uint64) *wire.Transaction {
	return d.build(node, "V1", wire.Transfer, &wire.TransferPayload{
		Send: &wire.TransferSend{Recipient: d.addr("V2"), Amount: amount},
	}, false)
}

func (d *demoEnv) build(node *net.Node, id string, typ wire.TxType, payload wire.Payload, hard bool) *wire.Transaction {
	w := node.Ledger.State.Wallet(d.addr(id))
	tx := &wire.Transaction{
		NetworkID:       d.gp.Genesis.NetworkID,
		Wallet:          d.addr(id),
		Seq:             w.Seq + 1,
		Prev:            append([]byte(nil), w.TipTxID...),
		Type:            typ,
		Payload:         payload,
		ProtocolVersion: d.gp.Genesis.ProtocolVersion,
		TimestampClaim:  uint64(time.Now().Unix()),
	}
	net.SolveTxPoW(tx, *d.gp.Manifest.Profile.Pow, hard)
	sh := tx.SigningHash()
	sig, err := crypto.SignDER(d.key(id), sh[:])
	if err != nil {
		d.logf("sign: %v", err)
	}
	tx.Signature = sig
	return tx
}

// testGenesis — детерминированный тестовый genesis (одинаков на всех машинах):
// посевной круг V1..V3, genesis за 65 суток (текущий период 2).
func testGenesis(networkID string) *wire.GenesisPackage {
	gp := params.TestGenesis()
	gp.Genesis.NetworkID = []byte(networkID)
	gp.Manifest.NetworkID = []byte(networkID)
	prof := gp.Manifest.Profile
	prof.Pow = &wire.PowParams{
		TargetLight: 0x0010_0000_0000_0000, Argon2MemoryKiB: 8,
		Argon2Iterations: 1, Argon2Parallelism: 1, TargetHard: 0x0010_0000_0000_0000,
	}
	prof.FreshTransferWindowPeriods = 0
	// окно челленджа в headless-демо открыто весь период (этап 8: сетевые
	// узлы проверяют окно при apply; демо-минт tnp-node происходит в
	// произвольный момент — слот почти всегда вне окна, №39). Реальное
	// окно — закрытая тестовая сеть (cmd/tnp-testnet).
	prof.MintChallengeWindowHours = 0
	genesisT := uint64(time.Now().Truncate(24*time.Hour).Unix()) - 65*86400
	var circle []wire.SeedMember
	for _, id := range []string{"V1", "V2", "V3"} {
		priv := crypto.DeterministicKey("tnp-node-seed:" + id)
		pub := crypto.MarshalPubKey(&priv.PublicKey)
		fp := crypto.Fingerprint(pub)
		circle = append(circle, wire.SeedMember{
			Address: crypto.DeriveAddress(fp, []byte(networkID)), PublicKey: pub,
		})
	}
	gp.Manifest.SeedCircle = circle
	gp.Genesis.GenesisTime = genesisT
	gp.Genesis.EpochZero = 0
	mh := gp.Manifest.ManifestHash()
	gp.Genesis.ManifestHash = mh[:]
	return gp
}

type nodeStatus struct {
	Level       string   `json:"level"`
	PeerID      string   `json:"peer_id"`
	Addrs       []string `json:"addrs"`
	Peers       int      `json:"peers"`
	JournalLen  int      `json:"journal_len"`
	StateRoot   string   `json:"state_root"`
	Conflicts   int      `json:"conflicts"`
	Spam        int64    `json:"spam_rejected"`
	Checkpoints int      `json:"checkpoints"`
}

func writeStatus(path string, node *net.Node) {
	st := node.Stats()
	root, _ := node.StateRoot()
	s := nodeStatus{
		Level: st.Level, PeerID: st.PeerID, Peers: st.Peers,
		JournalLen: st.JournalLen, Conflicts: st.Conflicts, Spam: st.SpamRejected,
		StateRoot:   fmt.Sprintf("%x", root),
		Checkpoints: node.CheckpointCount(),
	}
	for _, a := range node.Host.Addrs() {
		s.Addrs = append(s.Addrs, a.String())
	}
	blob, err := json.Marshal(s)
	if err != nil {
		return
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, blob, 0o644); err != nil {
		return
	}
	_ = os.Rename(tmp, path)
}

func fatal(f string, a ...any) {
	fmt.Fprintf(os.Stderr, "tnp-node: "+f+"\n", a...)
	os.Exit(1)
}

// multiaddrList — повторяемый строковый флаг.
type multiaddrList []string

func (m *multiaddrList) String() string { return fmt.Sprint(*m) }
func (m *multiaddrList) Set(v string) error {
	*m = append(*m, v)
	return nil
}
