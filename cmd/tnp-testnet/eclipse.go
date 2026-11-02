// eclipse.go — контрольный прогон F16 (аудит контура 1, п.5): двойная трата
// офлайн-канала через 3 соучастников в 3 несвязанных /24. По образцу A3-флуда
// (flood.go) и дизайну reports/audit-frontier-pass/eclipse-sybil-design.md §4.
//
//	tnp-testnet eclipse -role victim -genesis Ф -db ПУТЬ -status-file Ф
//	                    [-track-file Ф] [-listen MA] [-link-groups БУЛЕВ]
//	tnp-testnet eclipse -role relay -genesis Ф -listen MA -peer MA -tx ФАЙЛ
//	                    [-every-ms N] [-duration СЕК]
//	tnp-testnet eclipse -role driver -genesis Ф -out КАТ
//	                    [-victim-bin БИНАРЬ] [-link-groups БУЛЕВ]
//	                    [-attack-addrs IP,IP,IP] [-victim-port N]
//	                    [-seeds V1,V2,V3] [-seed-period N]
//	                    [-phase1 С] [-phase2 С] [-phase3 С]
//
// Жертва — отдельный процесс с отдельной БД (как в flood); 3 атакующих —
// raw gossipsub-реле (StrictNoSign, как флудер A3) на 127.0.0.1 / 127.0.1.1 /
// 127.0.2.1 — три разных /24 без контейнеров (весь 127/8 — local на Linux).
// Честная сеть — 3 узла драйвера с ключами посевного круга (подписанты
// чекпоинтов), подключены к жертве. Метрики жертвы — статус-файл 500 мс:
// LinkSpread(txid) -> (peers, groups), OfflineStatus(txid), финализирован ли
// txid, journal_len, state_root, чекпоинты. Драйвер пишет tx в файлы, реле
// публикуют их в теме tx/<network_id>.
//
// Фазы: setup (регистрация кошелька жертвы, выпуски посевного круга) →
// phase1 «честная латентность» (реле публикуют честную tx C; замер до
// «финализировано» на жертве) → phase2 «атака» (сначала конфликтная B —
// в честную сеть, затем реле публикуют двойную трату A 3 группами; статус A
// на жертве) → phase3 (ожидание чекпоинта, финальные статусы A/B).
// Двойная трата A проигрывает конфликт (B финализирована чекпоинтом).
// Замер «до/после фикса»: -victim-bin — бинарь жертвы из main+tooling.
package main

import (
	"context"
	"crypto/ecdsa"
	"crypto/x509"
	"encoding/csv"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/rumatakira74/equality-trust-networks/core/attest"
	"github.com/rumatakira74/equality-trust-networks/core/crypto"
	"github.com/rumatakira74/equality-trust-networks/core/wire"
	tnp "github.com/rumatakira74/equality-trust-networks/mobile"
	"github.com/rumatakira74/equality-trust-networks/net"
)

func eclipseCmd(args []string) error {
	fs := flag.NewFlagSet("eclipse", flag.ContinueOnError)
	role := fs.String("role", "driver", "driver | victim | relay")
	genesis := fs.String("genesis", "", "путь к genesis.bin")
	db := fs.String("db", "", "SQLite жертвы (victim)")
	statusFile := fs.String("status-file", "", "файл JSON-статуса жертвы (victim)")
	trackFile := fs.String("track-file", "", "файл с txid (hex по строке) для мониторинга (victim)")
	listen := fs.String("listen", "/ip4/127.0.0.1/udp/0/quic-v1", "адрес прослушивания (victim/relay)")
	linkGroups := fs.Bool("link-groups", true, "несвязанность eclipse-пиров по подсетям (№35)")
	peerMA := fs.String("peer", "", "multiaddr жертвы для подключения после submit (relay)")
	syncPeer := fs.String("sync-peer", "", "multiaddr честного узла для первичной сверки (relay)")
	txFile := fs.String("tx", "", "файл с wire.Transaction (relay)")
	attackDur := fs.Int("duration", 20, "длительность публикации tx, сек (relay)")
	out := fs.String("out", "", "каталог артефактов (driver)")
	victimBin := fs.String("victim-bin", "", "бинарь жертвы (по умолчанию — этот же бинарь; «до фикса» — из main+tooling)")
	victimPort := fs.Int("victim-port", 4181, "UDP-порт жертвы (driver)")
	seeds := fs.String("seeds", "V1,V2,V3", "метки посевного круга (через запятую; driver)")
	seedPeriod := fs.Uint64("seed-period", 2, "период выпуска посевных кошельков (driver)")
	attackAddrs := fs.String("attack-addrs", "127.0.0.1,127.0.1.1,127.0.2.1", "IP атакующих-реле (через запятую; один IP — негативный контроль одной подсети)")
	attackPort := fs.Int("attack-port", 4901, "базовый UDP-порт реле (driver)")
	phase1 := fs.Int("phase1", 55, "сек фазы честной латентности (driver)")
	phase2 := fs.Int("phase2", 55, "сек фазы атаки (driver)")
	phase3 := fs.Int("phase3", 70, "сек ожидания финализации конфликта (driver)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *genesis == "" {
		return fmt.Errorf("нужен -genesis")
	}
	switch *role {
	case "victim":
		if *db == "" || *statusFile == "" {
			return fmt.Errorf("victim: нужны -db и -status-file")
		}
		return eclipseVictim(*genesis, *db, *statusFile, *trackFile, *listen, *linkGroups)
	case "relay":
		if *peerMA == "" || *txFile == "" {
			return fmt.Errorf("relay: нужны -peer и -tx")
		}
		return eclipseRelay(*genesis, *listen, *peerMA, *syncPeer, *txFile, *attackDur)
	case "driver":
		if *out == "" {
			return fmt.Errorf("driver: нужен -out")
		}
		return eclipseDriver(*genesis, *out, *victimBin, *linkGroups, *victimPort,
			*seeds, *seedPeriod, *attackAddrs, *attackPort, *phase1, *phase2, *phase3)
	}
	return fmt.Errorf("неизвестная роль: %s", *role)
}

// -----------------------------------------------------------------------
// victim — честный узел-жертва (отдельный процесс, отдельная БД)
// -----------------------------------------------------------------------

// eclipseTracked — метрики одной txid на жертве.
type eclipseTracked struct {
	Peers     int    `json:"peers"`
	Groups    int    `json:"groups"`
	Status    string `json:"status"`
	Finalized bool   `json:"finalized"`
}

// eclipseVictimStatus — статус-файл жертвы для драйвера.
type eclipseVictimStatus struct {
	net.Stats
	Addrs       []string                  `json:"addrs"`
	Conns       []string                  `json:"conns"` // peer -> remote addr
	StateRoot   string                    `json:"state_root"`
	Checkpoints int                       `json:"checkpoints"`
	Tracked     map[string]eclipseTracked `json:"tracked"`
}

func eclipseVictim(genesisPath, dbPath, statusPath, trackPath, listen string, linkGroups bool) error {
	gp, err := loadGenesis(genesisPath)
	if err != nil {
		return err
	}
	rootKey := crypto.DeterministicKey("tnp-android-root:" + string(gp.Genesis.NetworkID))
	rootCert, _, err := attest.SyntheticRoot("android", rootKey)
	if err != nil {
		return fmt.Errorf("synthetic root: %w", err)
	}
	cfg := &net.NodeConfig{
		Genesis:             gp,
		NodeKey:             crypto.DeterministicKey("tnp-testnet-node:eclipse-victim"),
		Level:               net.LevelFull,
		DBPath:              dbPath,
		ListenAddrs:         strings.Split(listen, ","),
		TrustedRoots:        []*x509.Certificate{rootCert},
		AllowSyntheticRoots: true,
		LinkGroups:          linkGroups,
		Log:                 func(f string, a ...any) { fmt.Fprintf(os.Stderr, "жертва: "+f+"\n", a...) },
	}
	node, err := net.NewNode(cfg)
	if err != nil {
		return fmt.Errorf("узел: %w", err)
	}
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	if err := node.Start(ctx); err != nil {
		return fmt.Errorf("запуск: %w", err)
	}
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			_ = node.Close()
			return nil
		case <-ticker.C:
			writeEclipseStatus(statusPath, trackPath, node)
		}
	}
}

func writeEclipseStatus(path, trackPath string, node *net.Node) {
	st := eclipseVictimStatus{Stats: node.Stats(), Tracked: map[string]eclipseTracked{}}
	for _, a := range node.Host.Addrs() {
		st.Addrs = append(st.Addrs, a.String())
	}
	for _, c := range node.Host.Network().Conns() {
		st.Conns = append(st.Conns, fmt.Sprintf("%s -> %s", c.RemotePeer().ShortString(), c.RemoteMultiaddr().String()))
	}
	if r, err := node.StateRoot(); err == nil {
		st.StateRoot = hex.EncodeToString(r)
	}
	st.Checkpoints = node.CheckpointCount()
	if trackPath != "" {
		if blob, err := os.ReadFile(trackPath); err == nil {
			for _, line := range strings.Split(string(blob), "\n") {
				line = strings.TrimSpace(line)
				if line == "" {
					continue
				}
				b, err := hex.DecodeString(line)
				if err != nil || len(b) != 32 {
					continue
				}
				var id [32]byte
				copy(id[:], b)
				peers, groups, status, finalized := node.TxCorroboration(id)
				st.Tracked[line] = eclipseTracked{
					Peers: peers, Groups: groups,
					Status: status.String(), Finalized: finalized,
				}
			}
		}
	}
	blob, err := json.Marshal(st)
	if err != nil {
		return
	}
	_ = os.WriteFile(path+".tmp", blob, 0o644)
	_ = os.Rename(path+".tmp", path)
}

func readEclipseStatus(path string) *eclipseVictimStatus {
	blob, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	st := &eclipseVictimStatus{}
	if err := json.Unmarshal(blob, st); err != nil {
		return nil
	}
	return st
}

// -----------------------------------------------------------------------
// relay — атакующий: полный узел, кладёт tx-файл в свой журнал и держит
// соединение с жертвой; жертва сверяется с ним (SyncJournalFrom →
// Corroboration.Add от каждого реле). Source-адрес реле — его listen IP,
// поэтому группы связи у жертвы — по /24 (требуется unshare -Urn с
// /32-адресами на lo, см. driver.txt).
// -----------------------------------------------------------------------

func eclipseRelay(genesisPath, listen, peerMA, syncPeer, txFile string, duration int) error {
	gp, err := loadGenesis(genesisPath)
	if err != nil {
		return err
	}
	txBytes, err := os.ReadFile(txFile)
	if err != nil {
		return err
	}
	tx, err := wire.DecodeTransaction(txBytes)
	if err != nil {
		return fmt.Errorf("tx-файл: %w", err)
	}
	rootKey := crypto.DeterministicKey("tnp-android-root:" + string(gp.Genesis.NetworkID))
	rootCert, _, err := attest.SyntheticRoot("android", rootKey)
	if err != nil {
		return fmt.Errorf("synthetic root: %w", err)
	}
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	cfg := &net.NodeConfig{
		Genesis:             gp,
		NodeKey:             crypto.DeterministicKey("tnp-testnet-node:eclipse-relay:" + listen),
		Level:               net.LevelFull,
		ListenAddrs:         strings.Split(listen, ","),
		ManualAddrs:         manualOf(peerMA),
		TrustedRoots:        []*x509.Certificate{rootCert},
		AllowSyntheticRoots: true,
		LinkGroups:          false,
		Log:                 func(f string, a ...any) { fmt.Fprintf(os.Stderr, "реле: "+f+"\n", a...) },
	}
	node, err := net.NewNode(cfg)
	if err != nil {
		return fmt.Errorf("узел: %w", err)
	}
	if err := node.Start(ctx); err != nil {
		return fmt.Errorf("запуск: %w", err)
	}
	defer node.Close()
	txid := tx.TxID()
	fmt.Fprintf(os.Stderr, "реле: %s -> жертва %s, tx %s (%d байт) на %d с\n",
		node.Host.ID().ShortString(), peerMA, hex.EncodeToString(txid[:8]), len(txBytes), duration)
	// первичная сверка с жертвой (журнал: регистрации + выпуски — нужны для
	// приёма tx на seq 2), затем tx — в свой журнал; жертва забирает её сверкой
	// (identify при подключении + 30-с тик syncLoop). Dial только к жертве —
	// тогда source-адрес у жертвы из подсети listen-адреса реле
	wantJournal := 2 + 3 // 2 регистрации + 3 выпуска (см. driver)
	for i := 0; i < 40; i++ {
		if node.Stats().JournalLen >= wantJournal {
			break
		}
		time.Sleep(250 * time.Millisecond)
	}
	go func() {
		res, err := node.SubmitTx(ctx, tx)
		if err != nil {
			fmt.Fprintf(os.Stderr, "реле: submit err: %v\n", err)
			return
		}
		fmt.Fprintf(os.Stderr, "реле: tx %s: %s/%s (журнал %d)\n",
			hex.EncodeToString(txid[:8]), res.Outcome, res.Code, node.Stats().JournalLen)
	}()
	for i := 0; i < 20; i++ {
		if node.Stats().JournalLen > wantJournal {
			break
		}
		time.Sleep(250 * time.Millisecond)
	}
	fmt.Fprintf(os.Stderr, "реле: tx %s в журнале (журнал %d)\n",
		hex.EncodeToString(txid[:8]), node.Stats().JournalLen)
	// держим узел живым: жертва сверяется с каждым пиром по identify и по
	// 30-с тику syncLoop — Add(txid, реле) от каждого реле из своей /24
	end := time.Now().Add(time.Duration(duration) * time.Second)
	for time.Now().Before(end) {
		select {
		case <-ctx.Done():
			return nil
		case <-time.After(time.Second):
		}
	}
	return nil
}

// -----------------------------------------------------------------------
// driver — оркестрация прогона
// -----------------------------------------------------------------------

// eclipseSample — строка CSV мониторинга.
type eclipseSample struct {
	TS          string `json:"ts"`
	Phase       string `json:"phase"`
	JournalLen  int    `json:"journal_len"`
	Checkpoints int    `json:"checkpoints"`
	StateRoot   string `json:"state_root"`
	A           string `json:"a"` // peers/groups/status/finalized
	B           string `json:"b"`
	C           string `json:"c"`
}

func trackStr(t eclipseTracked) string {
	return fmt.Sprintf("%d/%d/%s/f=%v", t.Peers, t.Groups, t.Status, t.Finalized)
}

// eclipseSummary — итог прогона (summary.json).
type eclipseSummary struct {
	NetworkID          string `json:"network_id"`
	LinkGroups         bool   `json:"link_groups"`
	AttackAddrs        string `json:"attack_addrs"`
	VictimBin          string `json:"victim_bin"`
	HonestLatMs        int64  `json:"honest_c_latency_ms"` // -1: не финализирована за phase1
	HonestCGroups      int    `json:"honest_c_groups"`
	AttackAGroups      int    `json:"attack_a_groups_max"`
	AttackAStatus      string `json:"attack_a_status_end_phase2"`
	AttackAPreFinality string `json:"attack_a_status_at_3groups_pre_finality"`
	FinalAStatus       string `json:"final_a_status"`
	FinalAFinal        bool   `json:"final_a_finalized"`
	FinalBStatus       string `json:"final_b_status"`
	FinalBFinal        bool   `json:"final_b_finalized"`
	Checkpoints        int    `json:"checkpoints"`
	JournalLen         int    `json:"journal_len"`
	StateRoot          string `json:"state_root"`
	AttackSucceeded    bool   `json:"attack_a_finalized_pre_finality"` // A стала finalized до финальности (баг)
}

func eclipseDriver(genesisPath, outDir, victimBin string, linkGroups bool, victimPort int,
	seedsStr string, seedPeriod uint64, attackAddrs string, attackPortBase, phase1, phase2, phase3 int) error {
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return err
	}
	gp, err := loadGenesis(genesisPath)
	if err != nil {
		return err
	}
	self, err := os.Executable()
	if err != nil {
		return err
	}
	if victimBin != "" {
		self = victimBin
	}
	statusPath := filepath.Join(outDir, "victim-status.json")
	dbPath := filepath.Join(outDir, "victim.db")
	trackPath := filepath.Join(outDir, "track.txt")
	_ = os.Remove(statusPath)
	_ = os.Remove(dbPath)

	// 1. жертва — отдельный процесс; слушает на 127.0.0.1 и на каждом IP
	// атакующих — тогда source-адрес подключения реле совпадает с его IP и
	// жертва видит три разных /24 (иначе ядро выбирает source 127.0.0.1)
	attackIPs := splitLabels(attackAddrs)
	listenIPs := []string{"127.0.0.1"}
	for _, ip := range attackIPs {
		found := false
		for _, l := range listenIPs {
			if l == ip {
				found = true
			}
		}
		if !found {
			listenIPs = append(listenIPs, ip)
		}
	}
	var listenParts []string
	for _, ip := range listenIPs {
		listenParts = append(listenParts, fmt.Sprintf("/ip4/%s/udp/%d/quic-v1", ip, victimPort))
	}
	victimListen := strings.Join(listenParts, ",")
	victimLog, err := os.Create(filepath.Join(outDir, "victim.log"))
	if err != nil {
		return err
	}
	defer victimLog.Close()
	victim := exec.Command(self, "eclipse", "-role", "victim",
		"-genesis", genesisPath, "-db", dbPath, "-status-file", statusPath,
		"-track-file", trackPath,
		"-listen", victimListen,
		"-link-groups", strconv.FormatBool(linkGroups))
	victim.Stderr = victimLog
	if err := victim.Start(); err != nil {
		return fmt.Errorf("запуск жертвы: %w", err)
	}
	defer func() {
		_ = victim.Process.Signal(syscall.SIGTERM)
		done := make(chan error, 1)
		go func() { done <- victim.Wait() }()
		select {
		case <-done:
		case <-time.After(5 * time.Second):
			_ = victim.Process.Kill()
			<-done
		}
	}()
	fmt.Fprintf(os.Stderr, "eclipse: жертва запущена (pid %d)\n", victim.Process.Pid)

	var vst *eclipseVictimStatus
	deadline := time.Now().Add(20 * time.Second)
	for time.Now().Before(deadline) {
		if vst = readEclipseStatus(statusPath); vst != nil && len(vst.Addrs) > 0 {
			break
		}
		time.Sleep(200 * time.Millisecond)
	}
	if vst == nil || len(vst.Addrs) == 0 {
		return fmt.Errorf("жертва не поднялась")
	}
	// адреса жертвы по IP: ip -> multiaddr (для реле — свой IP, чтобы
	// source-адрес у жертвы был из нужной подсети)
	victimByIP := map[string]string{}
	victimMA := ""
	for _, a := range vst.Addrs {
		ma := fmt.Sprintf("%s/p2p/%s", a, vst.PeerID)
		if victimMA == "" {
			victimMA = ma
		}
		if strings.HasPrefix(a, "/ip4/") {
			ip := strings.Split(strings.TrimPrefix(a, "/ip4/"), "/")[0]
			victimByIP[ip] = ma
		}
	}
	fmt.Fprintf(os.Stderr, "eclipse: жертва %s (addrs %v)\n", victimMA, vst.Addrs)

	// 2. честная сеть: 3 узла с ключами посевного круга (подписанты чекпоинтов)
	rootKey := crypto.DeterministicKey("tnp-android-root:" + string(gp.Genesis.NetworkID))
	rootCert, _, err := attest.SyntheticRoot("android", rootKey)
	if err != nil {
		return err
	}
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	seedLabels := splitLabels(seedsStr)
	if len(seedLabels) < 2 {
		return fmt.Errorf("нужно минимум 2 метки посевного круга, дано %d", len(seedLabels))
	}
	// 2 честных узла (подписи чекпоинтов 2-из-3): вместе с 3 реле и жертвой
	// на теме 6 пиров = D mesh, жертва в mesh каждого реле и получает
	// сообщения напрямую (иначе ReceivedFrom — форвардер, группы схлопываются)
	honest := make([]*net.Node, 2)
	for i := 0; i < 2; i++ {
		key := seedKey(seedLabels[i])
		cfg := &net.NodeConfig{
			Genesis:             gp,
			NodeKey:             crypto.DeterministicKey(fmt.Sprintf("tnp-testnet-node:eclipse-h%d", i)),
			Level:               net.LevelFull,
			ListenAddrs:         []string{"/ip4/127.0.0.1/udp/0/quic-v1"},
			ManualAddrs:         []string{victimMA},
			TrustedRoots:        []*x509.Certificate{rootCert},
			AllowSyntheticRoots: true,
			LinkGroups:          false, // localhost
			WalletKey:           key,
			WalletAddr:          addrOf(key, gp.Genesis.NetworkID),
			Log:                 func(f string, a ...any) { fmt.Fprintf(os.Stderr, "честный: "+f+"\n", a...) },
		}
		if i > 0 {
			cfg.ManualAddrs = append(cfg.ManualAddrs, honestAddrs(honest[:i])...)
		}
		n, err := net.NewNode(cfg)
		if err != nil {
			return err
		}
		if err := n.Start(ctx); err != nil {
			return fmt.Errorf("честный узел %d: %w", i, err)
		}
		defer n.Close()
		honest[i] = n
	}
	time.Sleep(2 * time.Second)
	for _, n := range honest {
		n.SyncAllPeers()
	}
	time.Sleep(1 * time.Second)
	fmt.Fprintf(os.Stderr, "eclipse: честная сеть из 2 узлов поднята\n")

	// 3. сетап: регистрация кошельков жертвы V и второго получателя V2
	// (поручительства ниже — vouch от посевного круга, выпуски не нужны)
	env := &txEnv{gp: gp, node: honest[0], ctx: ctx}
	vKey := walletKey("ECL-V")
	vPub := crypto.MarshalPubKey(&vKey.PublicKey)
	vAddr := crypto.DeriveAddress(crypto.Fingerprint(vPub), gp.Genesis.NetworkID)
	v2Addr := addrOf(walletKey("ECL-W2"), gp.Genesis.NetworkID)
	register := func(key *ecdsa.PrivateKey, addr []byte, label string) error {
		pub := crypto.MarshalPubKey(&key.PublicKey)
		challenge := attest.ChallengeFormula(pub, crypto.ProtocolID, gp.Genesis.NetworkID)
		chain, err := tnp.SandboxAttestationChain(pub, gp.Genesis.NetworkID, challenge)
		if err != nil {
			return err
		}
		tx, err := buildTx(env, key, addr, wire.Register, &wire.RegisterPayload{
			AttestedPublicKey:    pub,
			AttestationChain:     chain,
			AttestationChallenge: challenge,
		}, true)
		if err != nil {
			return err
		}
		if err := submit(env, tx); err != nil {
			return fmt.Errorf("register %s: %w", label, err)
		}
		return nil
	}
	if err := register(vKey, vAddr, "V"); err != nil {
		return err
	}
	if err := register(walletKey("ECL-W2"), v2Addr, "V2"); err != nil {
		return err
	}
	wantBase := 2
	for _, sl := range seedLabels {
		key := seedKey(sl)
		addr := addrOf(key, gp.Genesis.NetworkID)
		amount := gp.Manifest.Profile.IssuanceAmount * seedPeriod
		tx, err := buildTx(env, key, addr, wire.Mint,
			&wire.MintPayload{Period: seedPeriod, Amount: amount}, true)
		if err != nil {
			return err
		}
		if err := submit(env, tx); err != nil {
			return fmt.Errorf("mint %s: %w", sl, err)
		}
		wantBase++
	}
	deadline = time.Now().Add(60 * time.Second)
	for time.Now().Before(deadline) {
		if st := readEclipseStatus(statusPath); st != nil && st.JournalLen >= wantBase {
			break
		}
		time.Sleep(200 * time.Millisecond)
	}
	if st := readEclipseStatus(statusPath); st == nil || st.JournalLen < wantBase {
		return fmt.Errorf("жертва не догнала сетап: %+v", st)
	}
	fmt.Fprintf(os.Stderr, "eclipse: сетап применён жертвой (журнал %d)\n", wantBase)

	// 4. переводы (seq 2 после выпуска): A (W->V) — «офлайн-платёж» жертве,
	// B (W->V2) — конфликтная двойная трата (тот же seq/prev), C (S15->V) — честная
	wKey := seedKey(seedLabels[0])
	wAddr := addrOf(wKey, gp.Genesis.NetworkID)
	txA, err := buildTx(env, wKey, wAddr, wire.Transfer, &wire.TransferPayload{
		Send: &wire.TransferSend{Recipient: vAddr, Amount: 100},
	}, false)
	if err != nil {
		return err
	}
	txB, err := buildTx(env, wKey, wAddr, wire.Transfer, &wire.TransferPayload{
		Send: &wire.TransferSend{Recipient: v2Addr, Amount: 100},
	}, false)
	if err != nil {
		return err
	}
	if txA.Seq != txB.Seq || string(txA.Prev) != string(txB.Prev) {
		return fmt.Errorf("A и B не конфликтуют по seq/prev")
	}
	v3Key := seedKey(seedLabels[1])
	txC, err := buildTx(env, v3Key, addrOf(v3Key, gp.Genesis.NetworkID), wire.Transfer, &wire.TransferPayload{
		Send: &wire.TransferSend{Recipient: vAddr, Amount: 1},
	}, false)
	if err != nil {
		return err
	}
	idA, idB, idC := txA.TxID(), txB.TxID(), txC.TxID()
	hexA, hexB, hexC := hex.EncodeToString(idA[:]), hex.EncodeToString(idB[:]), hex.EncodeToString(idC[:])
	if err := os.WriteFile(filepath.Join(outDir, "txA.bin"), txA.Encode(), 0o644); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(outDir, "txB.bin"), txB.Encode(), 0o644); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(outDir, "txC.bin"), txC.Encode(), 0o644); err != nil {
		return err
	}
	if err := os.WriteFile(trackPath, []byte(hexA+"\n"+hexB+"\n"+hexC+"\n"), 0o644); err != nil {
		return err
	}
	fmt.Fprintf(os.Stderr, "eclipse: tx A=%s B=%s C=%s\n", hexA[:12], hexB[:12], hexC[:12])

	// монитор: CSV каждую 500 мс
	var samples []eclipseSample
	phase := "setup"
	stopMon := make(chan struct{})
	monDone := make(chan struct{})
	go func() {
		defer close(monDone)
		tick := time.NewTicker(500 * time.Millisecond)
		defer tick.Stop()
		for {
			select {
			case <-stopMon:
				return
			case <-tick.C:
			}
			st := readEclipseStatus(statusPath)
			if st == nil {
				continue
			}
			s := eclipseSample{
				TS: time.Now().UTC().Format(time.RFC3339), Phase: phase,
				JournalLen: st.JournalLen, Checkpoints: st.Checkpoints, StateRoot: st.StateRoot,
			}
			if t, ok := st.Tracked[hexA]; ok {
				s.A = trackStr(t)
			}
			if t, ok := st.Tracked[hexB]; ok {
				s.B = trackStr(t)
			}
			if t, ok := st.Tracked[hexC]; ok {
				s.C = trackStr(t)
			}
			samples = append(samples, s)
		}
	}()
	defer func() { close(stopMon); <-monDone }()

	// startRelays — 3 реле-процесса, публикующих данный tx-файл; реле i
	// подключается к адресу жертвы на своём IP (source = свой IP -> своя /24)
	h1MA := ""
	for _, a := range honest[0].Host.Addrs() {
		h1MA = fmt.Sprintf("%s/p2p/%s", a.String(), honest[0].Host.ID().String())
		break
	}
	startRelays := func(txPath string, dur int) ([]*exec.Cmd, error) {
		var procs []*exec.Cmd
		for i, ip := range attackIPs {
			peer := victimMA
			if ma, ok := victimByIP[ip]; ok {
				peer = ma
			}
			logF, err := os.Create(filepath.Join(outDir, fmt.Sprintf("relay%d.log", i+1)))
			if err != nil {
				return nil, err
			}
			cmd := exec.Command(self, "eclipse", "-role", "relay",
				"-genesis", genesisPath,
				"-listen", fmt.Sprintf("/ip4/%s/udp/%d/quic-v1", ip, attackPortBase+i),
				"-peer", peer, "-sync-peer", h1MA, "-tx", txPath,
				"-duration", strconv.Itoa(dur))
			cmd.Stderr = logF
			if err := cmd.Start(); err != nil {
				return nil, err
			}
			procs = append(procs, cmd)
		}
		fmt.Fprintf(os.Stderr, "eclipse: реле x%d (%s) публикуют %s на %d с\n",
			len(procs), attackAddrs, filepath.Base(txPath), dur)
		return procs, nil
	}
	waitProcs := func(procs []*exec.Cmd) {
		for _, p := range procs {
			done := make(chan error, 1)
			go func(p *exec.Cmd) { done <- p.Wait() }(p)
			select {
			case <-done:
			case <-time.After(10 * time.Second):
				_ = p.Process.Kill()
				<-done
			}
		}
	}

	// 5. phase1 — честная латентность: реле публикуют C, замер до finalized
	phase = "honest"
	cStart := time.Now()
	relays, err := startRelays(filepath.Join(outDir, "txC.bin"), phase1)
	if err != nil {
		return err
	}
	latMs := int64(-1)
	cGroups := 0
	deadline = time.Now().Add(time.Duration(phase1) * time.Second)
	for time.Now().Before(deadline) {
		if st := readEclipseStatus(statusPath); st != nil {
			if t, ok := st.Tracked[hexC]; ok {
				cGroups = t.Groups
				if t.Status == "finalized" && latMs < 0 {
					latMs = time.Since(cStart).Milliseconds()
					fmt.Fprintf(os.Stderr, "eclipse: C финализирована на жертве за %d мс (групп %d)\n", latMs, t.Groups)
					break
				}
			}
		}
		time.Sleep(250 * time.Millisecond)
	}
	waitProcs(relays)
	fmt.Fprintf(os.Stderr, "eclipse: phase1 завершена: C latency=%d мс groups=%d\n", latMs, cGroups)

	// 6. phase2 — атака: реле (3 /24) доставляют двойную трату A жертве;
	// пока конфликт B ещё не опубликован, у жертвы «до финальности» —
	// фиксируем, достигается ли «финализировано» через 3 соучастников
	phase = "attack"
	attackAStatus := ""
	attackAGroups := 0
	relays, err = startRelays(filepath.Join(outDir, "txA.bin"), phase2)
	if err != nil {
		return err
	}
	attackAPreFinality := "" // статус A при 3 группах ДО финальности (окно атаки)
	deadline = time.Now().Add(time.Duration(phase2) * time.Second)
	for time.Now().Before(deadline) {
		if st := readEclipseStatus(statusPath); st != nil {
			if t, ok := st.Tracked[hexA]; ok {
				if t.Groups > attackAGroups {
					attackAGroups = t.Groups
				}
				attackAStatus = t.Status
				if t.Groups >= 3 && !t.Finalized && attackAPreFinality == "" {
					attackAPreFinality = t.Status
				}
			}
		}
		time.Sleep(250 * time.Millisecond)
	}
	fmt.Fprintf(os.Stderr, "eclipse: атака: A группы=%d статус=%q, при 3 группах до финальности=%q\n",
		attackAGroups, attackAStatus, attackAPreFinality)
	// затем конфликтная B — в честную сеть (B побеждает финализацией)
	for _, n := range honest {
		// B — конфликтная ветка: accept или conflict оба доставляют её в сеть
		res, err := n.SubmitTx(ctx, txB)
		if err != nil || (res.Outcome.String() != "accept" && res.Outcome.String() != "conflict") {
			return fmt.Errorf("B на честном узле: %v/%s %w", res.Outcome, res.Code, err)
		}
		fmt.Fprintf(os.Stderr, "eclipse: B на честном узле %s: %s/%s\n",
			n.Host.ID().ShortString(), res.Outcome, res.Code)
	}
	fmt.Fprintf(os.Stderr, "eclipse: B опубликована в честной сети\n")
	waitProcs(relays)

	// 7. phase3 — ждём чекпоинт(ы): B финализирована, A — нет
	phase = "conflict"
	deadline = time.Now().Add(time.Duration(phase3) * time.Second)
	for time.Now().Before(deadline) {
		if st := readEclipseStatus(statusPath); st != nil {
			if st.Checkpoints >= 1 {
				if t, ok := st.Tracked[hexB]; ok && t.Finalized {
					break
				}
			}
		}
		time.Sleep(500 * time.Millisecond)
	}

	// 8. артефакты
	final := readEclipseStatus(statusPath)
	sum := eclipseSummary{
		NetworkID:          hex.EncodeToString(gp.Genesis.NetworkID),
		LinkGroups:         linkGroups,
		AttackAddrs:        attackAddrs,
		VictimBin:          victimBin,
		HonestLatMs:        latMs,
		HonestCGroups:      cGroups,
		AttackAGroups:      attackAGroups,
		AttackAStatus:      attackAStatus,
		AttackAPreFinality: attackAPreFinality,
	}
	if final != nil {
		sum.Checkpoints = final.Checkpoints
		sum.JournalLen = final.JournalLen
		sum.StateRoot = final.StateRoot
		if t, ok := final.Tracked[hexA]; ok {
			sum.FinalAStatus = t.Status
			sum.FinalAFinal = t.Finalized
		}
		if t, ok := final.Tracked[hexB]; ok {
			sum.FinalBStatus = t.Status
			sum.FinalBFinal = t.Finalized
		}
	}
	// успех атаки (баг F16): A получила finalized при 3 группах ДО финальности
	sum.AttackSucceeded = attackAPreFinality == "finalized"

	if err := writeEclipseCSV(filepath.Join(outDir, "samples.csv"), samples); err != nil {
		return err
	}
	sumBlob, err := json.MarshalIndent(sum, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(outDir, "summary.json"), sumBlob, 0o644); err != nil {
		return err
	}
	fmt.Println(string(sumBlob))
	return nil
}

func manualOf(ma string) []string {
	if ma == "" {
		return nil
	}
	return []string{ma}
}

func honestAddrs(nodes []*net.Node) []string {
	var out []string
	for _, n := range nodes {
		for _, a := range n.Host.Addrs() {
			out = append(out, fmt.Sprintf("%s/p2p/%s", a.String(), n.Host.ID().String()))
			break
		}
	}
	return out
}

func writeEclipseCSV(path string, samples []eclipseSample) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	w := csv.NewWriter(f)
	defer w.Flush()
	_ = w.Write([]string{"ts", "phase", "journal_len", "checkpoints", "state_root", "a", "b", "c"})
	for _, s := range samples {
		_ = w.Write([]string{s.TS, s.Phase, strconv.Itoa(s.JournalLen),
			strconv.Itoa(s.Checkpoints), s.StateRoot, s.A, s.B, s.C})
	}
	return nil
}
