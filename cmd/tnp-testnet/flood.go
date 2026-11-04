// flood.go — прогон с флудером (тема net-DoS A3): честный узел-жертва
// (отдельный процесс, отдельная БД), честный публикатор валидных tx и
// узел-флудер (raw gossipsub без валидатора — реалистичная модель
// атакующего, как спам-тест 4.11.3). Замеры: CPU/RSS жертвы
// (/proc/PID) и латентность приёма валидных tx (рост журнала жертвы
// по статус-файлу) в базовой фазе и под флудом; артефакты — CSV + JSON.
//
//	tnp-testnet flood -role victim -genesis Ф -db ПУТЬ -status-file Ф
//	                 [-listen MA] [-link-groups БУЛЕВ]
//	tnp-testnet flood -role driver -genesis Ф -out КАТ
//	                 [-duration СЕК] [-rate N/с] [-valid-every СЕК]
//	                 [-baseline СЕК] [-candidates N] [-victim-port N]
//
// Жертва и драйвер — один бинарь: драйвер запускает себя с -role
// victim (как fleet supervises tnp-node). Сеть — genesis утилиты
// generate: валидные tx — VOUCH посевного круга за зарегистрированных
// кандидатов (лёгкий PoW), мусор — MINT-конверты с корректным PoW-полем
// (argon2id + случайный nonce), битой подписью и неизвестным кошельком
// (до фикса A3 такой конверт запускал Argon2id на жертве до проверки
// подписи). Сравнение «до/после фикса»: тот же драйвер против жертвы,
// собранной из main (замер CPU/RSS/латентности), — отчёт темы.
package main

import (
	"context"
	"crypto/rand"
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
	"sync"
	"syscall"
	"time"

	"github.com/libp2p/go-libp2p"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/peer"
	ma "github.com/multiformats/go-multiaddr"

	"github.com/rumatakira74/equality-trust-networks/core/attest"
	"github.com/rumatakira74/equality-trust-networks/core/crypto"
	"github.com/rumatakira74/equality-trust-networks/core/wire"
	tnp "github.com/rumatakira74/equality-trust-networks/mobile"
	"github.com/rumatakira74/equality-trust-networks/net"
)

func floodCmd(args []string) error {
	fs := flag.NewFlagSet("flood", flag.ContinueOnError)
	role := fs.String("role", "driver", "driver | victim")
	genesis := fs.String("genesis", "", "путь к genesis.bin")
	db := fs.String("db", "", "SQLite жертвы (victim)")
	statusFile := fs.String("status-file", "", "файл JSON-статуса жертвы (victim)")
	listen := fs.String("listen", "/ip4/127.0.0.1/udp/0/quic-v1", "адреса прослушивания жертвы")
	linkGroups := fs.Bool("link-groups", false, "несвязанность eclipse-пиров по подсетям (№35); false для localhost")
	out := fs.String("out", "", "каталог артефактов (driver)")
	duration := fs.Int("duration", 60, "сек флуда")
	rate := fs.Int("rate", 50, "мусорных tx в секунду")
	validEvery := fs.Int("valid-every", 3, "валидная tx каждые N сек")
	baseline := fs.Int("baseline", 15, "сек базовой фазы без флуда")
	candidatesN := fs.Int("candidates", 12, "кандидатов для vouch")
	victimPort := fs.Int("victim-port", 4171, "UDP-порт жертвы (driver)")
	seeds := fs.String("seeds", "V1,V2,V3", "метки посевного круга (через запятую; driver)")
	seedPeriod := fs.Uint64("seed-period", 2, "период выпуска посевных кошельков до vouch (V3: CompletedMintPeriods >= 2)")
	victimBin := fs.String("victim-bin", "", "бинарь жертвы (по умолчанию — этот же бинарь; замер «до фикса» — жертва из main)")
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
		return floodVictim(*genesis, *db, *statusFile, *listen, *linkGroups)
	case "driver":
		if *out == "" {
			return fmt.Errorf("driver: нужен -out")
		}
		return floodDriver(*genesis, *out, *duration, *rate, *validEvery, *baseline, *candidatesN, *victimPort, *linkGroups, *seeds, *seedPeriod, *victimBin)
	}
	return fmt.Errorf("неизвестная роль: %s", *role)
}

// -----------------------------------------------------------------------
// victim — честный узел-жертва (отдельный процесс)
// -----------------------------------------------------------------------

// floodVictimStatus — статус жертвы для драйвера (Stats + адреса).
type floodVictimStatus struct {
	net.Stats
	Addrs []string `json:"addrs"`
}

func floodVictim(genesisPath, dbPath, statusPath, listen string, linkGroups bool) error {
	gp, err := loadGenesis(genesisPath)
	if err != nil {
		return err
	}
	// корни аттестации — как у tx-драйверов (startEnv): тестовая сеть
	// регистрирует кошельки синтетическими цепочками при sandbox=false
	// (A2-обход AllowSyntheticRoots — тестовая утилита, не прод)
	rootKey := crypto.DeterministicKey("tnp-android-root:" + string(gp.Genesis.NetworkID))
	rootCert, _, err := attest.SyntheticRoot("android", rootKey)
	if err != nil {
		return fmt.Errorf("synthetic root: %w", err)
	}
	cfg := &net.NodeConfig{
		Genesis:             gp,
		NodeKey:             crypto.DeterministicKey("tnp-testnet-node:netdos-victim"),
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
	writeVictimStatus(statusPath, node)
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			_ = node.Close()
			return nil
		case <-ticker.C:
			writeVictimStatus(statusPath, node)
		}
	}
}

func writeVictimStatus(path string, node *net.Node) {
	st := floodVictimStatus{Stats: node.Stats()}
	for _, a := range node.Host.Addrs() {
		st.Addrs = append(st.Addrs, a.String())
	}
	blob, err := json.Marshal(st)
	if err != nil {
		return
	}
	_ = os.WriteFile(path+".tmp", blob, 0o644)
	_ = os.Rename(path+".tmp", path)
}

func readVictimStatus(path string) *floodVictimStatus {
	blob, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	st := &floodVictimStatus{}
	if err := json.Unmarshal(blob, st); err != nil {
		return nil
	}
	return st
}

// -----------------------------------------------------------------------
// driver — оркестрация прогона
// -----------------------------------------------------------------------

// floodSample — строка CSV мониторинга жертвы (1/с).
type floodSample struct {
	TS              string `json:"ts"`
	Phase           string `json:"phase"`
	CPUMs           int64  `json:"cpu_ms"`
	RSSKB           int    `json:"rss_kb"`
	JournalLen      int    `json:"journal_len"`
	SpamRejected    int64  `json:"spam_rejected"`
	ExpensiveChecks uint64 `json:"expensive_checks"`
	BudgetRejected  uint64 `json:"budget_rejected"`
}

// latencyRec — замер латентности одной валидной tx (публикация →
// применение жертвой; гранулярность статус-файла 500 мс).
type latencyRec struct {
	TS        string `json:"ts"`
	Phase     string `json:"phase"`
	Seq       int    `json:"seq"`
	Seed      string `json:"seed"`
	LatencyMs int64  `json:"latency_ms"`
}

// floodSummary — итог прогона (summary.json).
type floodSummary struct {
	NetworkID     string `json:"network_id"`
	Rate          int    `json:"rate_tx_per_s"`
	DurationSec   int    `json:"duration_sec"`
	BaselineSec   int    `json:"baseline_sec"`
	Candidates    int    `json:"candidates"`
	GarbageTotal  int64  `json:"garbage_total"`
	ValidTotal    int    `json:"valid_total"`
	CPUMsBase     int64  `json:"cpu_ms_baseline"`
	CPUMsFlood    int64  `json:"cpu_ms_flood"`
	RSSMaxKBBase  int    `json:"rss_max_kb_baseline"`
	RSSMaxKBFlood int    `json:"rss_max_kb_flood"`
	LatAvgMsBase  int64  `json:"latency_avg_ms_baseline"`
	LatAvgMsFlood int64  `json:"latency_avg_ms_flood"`
	LatMaxMsBase  int64  `json:"latency_max_ms_baseline"`
	LatMaxMsFlood int64  `json:"latency_max_ms_flood"`
	SpamRejected  int64  `json:"spam_rejected"`
	Expensive     uint64 `json:"expensive_checks"`
	BudgetRej     uint64 `json:"budget_rejected"`
	JournalLen    int    `json:"journal_len"`
}

func floodDriver(genesisPath, outDir string, duration, rate, validEvery, baseline, candidatesN, victimPort int, linkGroups bool, seedsStr string, seedPeriod uint64, victimBin string) error {
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
	_ = os.Remove(statusPath)
	_ = os.Remove(dbPath)

	// 1. жертва — отдельный процесс (CPU/RSS меряем по /proc/PID)
	victimLog, err := os.Create(filepath.Join(outDir, "victim.log"))
	if err != nil {
		return err
	}
	defer victimLog.Close()
	victim := exec.Command(self, "flood", "-role", "victim",
		"-genesis", genesisPath, "-db", dbPath, "-status-file", statusPath,
		"-listen", fmt.Sprintf("/ip4/127.0.0.1/udp/%d/quic-v1", victimPort),
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
	fmt.Fprintf(os.Stderr, "флуд: жертва запущена (pid %d)\n", victim.Process.Pid)

	// ждём адреса жертвы
	var vst *floodVictimStatus
	deadline := time.Now().Add(20 * time.Second)
	for time.Now().Before(deadline) {
		if vst = readVictimStatus(statusPath); vst != nil && len(vst.Addrs) > 0 {
			break
		}
		time.Sleep(200 * time.Millisecond)
	}
	if vst == nil || len(vst.Addrs) == 0 {
		return fmt.Errorf("жертва не поднялась (статус-файл пуст)")
	}
	victimMA := fmt.Sprintf("%s/p2p/%s", vst.Addrs[0], vst.PeerID)
	fmt.Fprintf(os.Stderr, "флуд: жертва %s\n", victimMA)

	// 2. честный публикатор B — в процессе драйвера
	rootKey := crypto.DeterministicKey("tnp-android-root:" + string(gp.Genesis.NetworkID))
	rootCert, _, err := attest.SyntheticRoot("android", rootKey)
	if err != nil {
		return err
	}
	bcfg := &net.NodeConfig{
		Genesis:             gp,
		NodeKey:             crypto.DeterministicKey("tnp-testnet-node:netdos-driver"),
		Level:               net.LevelFull,
		ListenAddrs:         []string{"/ip4/127.0.0.1/udp/0/quic-v1"},
		ManualAddrs:         []string{victimMA},
		TrustedRoots:        []*x509.Certificate{rootCert},
		AllowSyntheticRoots: true,
		LinkGroups:          linkGroups,
		Log:                 func(f string, a ...any) { fmt.Fprintf(os.Stderr, "драйвер: "+f+"\n", a...) },
	}
	bnode, err := net.NewNode(bcfg)
	if err != nil {
		return err
	}
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	if err := bnode.Start(ctx); err != nil {
		return err
	}
	defer bnode.Close()
	time.Sleep(2 * time.Second)
	bnode.SyncAllPeers()
	time.Sleep(1 * time.Second)

	// 3. регистрация кандидатов (валидные VOUCH пойдут за них)
	type cand struct {
		label string
		addr  []byte
	}
	var cands []cand
	for i := 0; i < candidatesN; i++ {
		label := fmt.Sprintf("NETDOS-P%02d", i+1)
		key := walletKey(label)
		pub := crypto.MarshalPubKey(&key.PublicKey)
		addr := crypto.DeriveAddress(crypto.Fingerprint(pub), gp.Genesis.NetworkID)
		challenge := attest.ChallengeFormula(pub, crypto.ProtocolID, gp.Genesis.NetworkID)
		chain, err := tnp.SandboxAttestationChain(pub, gp.Genesis.NetworkID, challenge)
		if err != nil {
			return err
		}
		env := &txEnv{gp: gp, node: bnode, ctx: ctx}
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
		cands = append(cands, cand{label: label, addr: addr})
	}
	// ждём: жертва применяет регистрации
	deadline = time.Now().Add(30 * time.Second)
	for time.Now().Before(deadline) {
		if st := readVictimStatus(statusPath); st != nil && st.JournalLen >= candidatesN {
			break
		}
		time.Sleep(200 * time.Millisecond)
	}
	if st := readVictimStatus(statusPath); st == nil || st.JournalLen < candidatesN {
		return fmt.Errorf("жертва не догнала регистрации: %+v", st)
	}
	fmt.Fprintf(os.Stderr, "флуд: %d кандидатов зарегистрированы\n", candidatesN)

	// 3b. выпуск посевных кошельков (V3: CompletedMintPeriods >= 2
	// для vouch): период seedPeriod одной tx (M6: amount = эмиссия ×
	// diff; окно челленджа — метки найдены findlabels, генезис
	// подобран так, что окно периода открыто)
	seedLabels := splitLabels(seedsStr)
	for _, sl := range seedLabels {
		key := seedKey(sl)
		addr := addrOf(key, gp.Genesis.NetworkID)
		env := &txEnv{gp: gp, node: bnode, ctx: ctx}
		amount := mintAmount(gp.Manifest.Profile, 0, seedPeriod)
		tx, err := buildTx(env, key, addr, wire.Mint,
			&wire.MintPayload{Period: seedPeriod, Amount: amount}, true)
		if err != nil {
			return err
		}
		if err := submit(env, tx); err != nil {
			return fmt.Errorf("mint %s: %w", sl, err)
		}
	}
	// ждём: жертва применяет регистрации + выпуски
	wantBase := candidatesN + len(seedLabels)
	deadline = time.Now().Add(30 * time.Second)
	for time.Now().Before(deadline) {
		if st := readVictimStatus(statusPath); st != nil && st.JournalLen >= wantBase {
			break
		}
		time.Sleep(200 * time.Millisecond)
	}
	if st := readVictimStatus(statusPath); st == nil || st.JournalLen < wantBase {
		return fmt.Errorf("жертва не догнала регистрации/выпуски: %+v", st)
	}
	fmt.Fprintf(os.Stderr, "флуд: %d выпусков посевного круга применены\n", len(seedLabels))

	// 4. флудер — raw gossipsub без валидатора
	evilHost, err := libp2p.New(
		libp2p.ListenAddrStrings("/ip4/127.0.0.1/udp/0/quic-v1"),
		libp2p.UserAgent("tnp/"+hex.EncodeToString(gp.Genesis.NetworkID)),
	)
	if err != nil {
		return err
	}
	defer evilHost.Close()
	evilPS, err := pubsub.NewGossipSub(ctx, evilHost,
		pubsub.WithMessageSignaturePolicy(pubsub.StrictNoSign),
		pubsub.WithMaxMessageSize(64*1024+4096),
	)
	if err != nil {
		return err
	}
	topicName := "tx/" + hex.EncodeToString(gp.Genesis.NetworkID)
	evilTopic, err := evilPS.Join(topicName)
	if err != nil {
		return err
	}
	ai, err := addrInfoOfMA(victimMA)
	if err != nil {
		return err
	}
	if err := evilHost.Connect(ctx, ai); err != nil {
		return fmt.Errorf("флудер не подключился к жертве: %w", err)
	}
	time.Sleep(500 * time.Millisecond)
	fmt.Fprintf(os.Stderr, "флуд: флудер %s подключён\n", evilHost.ID().ShortString())

	// 5. мониторинг жертвы: CPU/RSS + статус, CSV каждую секунду
	var samples []floodSample
	cpuBase := readProcCPUMs(victim.Process.Pid)
	var mu sync.Mutex
	phase := "baseline"
	stopMon := make(chan struct{})
	monWG := sync.WaitGroup{}
	monWG.Add(1)
	go func() {
		defer monWG.Done()
		tick := time.NewTicker(time.Second)
		defer tick.Stop()
		for {
			select {
			case <-stopMon:
				return
			case <-tick.C:
				pid := victim.Process.Pid
				s := floodSample{
					TS:    time.Now().UTC().Format(time.RFC3339),
					Phase: phase,
					CPUMs: readProcCPUMs(pid) - cpuBase,
					RSSKB: readRSSKB(pid),
				}
				if v := readVictimStatus(statusPath); v != nil {
					s.JournalLen = v.JournalLen
					s.SpamRejected = v.SpamRejected
					s.ExpensiveChecks = v.ExpensiveChecks
					s.BudgetRejected = v.BudgetRejected
				}
				mu.Lock()
				samples = append(samples, s)
				mu.Unlock()
			}
		}
	}()

	// 6. валидные tx: VOUCH посевного круга (лёгкий PoW), замер
	// латентности приёма жертвой (рост журнала по статус-файлу)
	var latencies []latencyRec
	vouchIdx := 0
	publishValid := func() error {
		env := &txEnv{gp: gp, node: bnode, ctx: ctx}
		// пары без повторов (V7) и без переподписи активированных
		// (V6: K=3 vouch активируют кандидата): кандидату — 3 разных
		// посевных, посевные вращаются (слоты посевного не исчерпываются)
		c := cands[vouchIdx/3]
		seed := seedLabels[(vouchIdx/3+vouchIdx%3)%len(seedLabels)]
		key := seedKey(seed)
		addr := addrOf(key, gp.Genesis.NetworkID)
		payload := &wire.VouchPayload{Candidate: c.addr, Epoch: bnode.Clock.Epoch()}
		tx, err := buildTx(env, key, addr, wire.Vouch, payload, false)
		if err != nil {
			return err
		}
		start := time.Now()
		if err := submit(env, tx); err != nil {
			return fmt.Errorf("vouch %s→%s: %w", seed, c.label, err)
		}
		vouchIdx++
		want := wantBase + vouchIdx
		for time.Since(start) < 20*time.Second {
			if v := readVictimStatus(statusPath); v != nil && v.JournalLen >= want {
				break
			}
			time.Sleep(100 * time.Millisecond)
		}
		lat := time.Since(start).Milliseconds()
		mu.Lock()
		latencies = append(latencies, latencyRec{
			TS: time.Now().UTC().Format(time.RFC3339), Phase: phase,
			Seq: vouchIdx, Seed: seed, LatencyMs: lat,
		})
		mu.Unlock()
		fmt.Fprintf(os.Stderr, "флуд: vouch #%d (%s) латентность %d мс\n", vouchIdx, phase, lat)
		return nil
	}

	// базовая фаза: валидные tx без флуда
	baselineStart := time.Now()
	for time.Since(baselineStart) < time.Duration(baseline)*time.Second {
		if err := publishValid(); err != nil {
			return err
		}
		time.Sleep(time.Duration(validEvery) * time.Second)
	}

	// фаза флуда: мусор с корректным PoW-полем и битой подписью
	phase = "flood"
	fmt.Fprintf(os.Stderr, "флуд: старт флуда %d tx/с на %d с\n", rate, duration)
	garbage := int64(0)
	floodWG := sync.WaitGroup{}
	floodWG.Add(1)
	go func() {
		defer floodWG.Done()
		interval := time.Second / time.Duration(rate)
		tick := time.NewTicker(interval)
		defer tick.Stop()
		end := time.Now().Add(time.Duration(duration) * time.Second)
		for time.Now().Before(end) {
			select {
			case <-ctx.Done():
				return
			case <-tick.C:
			}
			_ = evilTopic.Publish(ctx, garbageMintTx(gp).Encode())
			garbage++
		}
	}()
	floodStart := time.Now()
	for time.Since(floodStart) < time.Duration(duration)*time.Second {
		if err := publishValid(); err != nil {
			return err
		}
		time.Sleep(time.Duration(validEvery) * time.Second)
	}
	floodWG.Wait()
	close(stopMon)
	monWG.Wait()

	// 7. артефакты: samples.csv, latency.csv, summary.json
	mu.Lock()
	defer mu.Unlock()
	if err := writeSamplesCSV(filepath.Join(outDir, "samples.csv"), samples); err != nil {
		return err
	}
	if err := writeLatencyCSV(filepath.Join(outDir, "latency.csv"), latencies); err != nil {
		return err
	}
	sum := floodSummary{
		NetworkID:    hex.EncodeToString(gp.Genesis.NetworkID),
		Rate:         rate,
		DurationSec:  duration,
		BaselineSec:  baseline,
		Candidates:   candidatesN,
		GarbageTotal: garbage,
		ValidTotal:   len(latencies),
	}
	for _, s := range samples {
		if s.Phase == "baseline" {
			if s.CPUMs > sum.CPUMsBase {
				sum.CPUMsBase = s.CPUMs
			}
			if s.RSSKB > sum.RSSMaxKBBase {
				sum.RSSMaxKBBase = s.RSSKB
			}
		} else {
			if s.CPUMs > sum.CPUMsFlood {
				sum.CPUMsFlood = s.CPUMs
			}
			if s.RSSKB > sum.RSSMaxKBFlood {
				sum.RSSMaxKBFlood = s.RSSKB
			}
			if s.SpamRejected > sum.SpamRejected {
				sum.SpamRejected = s.SpamRejected
			}
			if s.ExpensiveChecks > sum.Expensive {
				sum.Expensive = s.ExpensiveChecks
			}
			if s.BudgetRejected > sum.BudgetRej {
				sum.BudgetRej = s.BudgetRejected
			}
			if s.JournalLen > sum.JournalLen {
				sum.JournalLen = s.JournalLen
			}
		}
	}
	for _, l := range latencies {
		if l.Phase == "baseline" {
			sum.LatAvgMsBase += l.LatencyMs
			if l.LatencyMs > sum.LatMaxMsBase {
				sum.LatMaxMsBase = l.LatencyMs
			}
		} else {
			sum.LatAvgMsFlood += l.LatencyMs
			if l.LatencyMs > sum.LatMaxMsFlood {
				sum.LatMaxMsFlood = l.LatencyMs
			}
		}
	}
	if n := int64(countPhase(latencies, "baseline")); n > 0 {
		sum.LatAvgMsBase /= n
	}
	if n := int64(countPhase(latencies, "flood")); n > 0 {
		sum.LatAvgMsFlood /= n
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

func countPhase(recs []latencyRec, phase string) int {
	n := 0
	for _, r := range recs {
		if r.Phase == phase {
			n++
		}
	}
	return n
}

// garbageMintTx — мусорная tx флудера (A3): MINT-конверт с корректным
// PoW-полем (argon2id + случайный nonce), битой подписью и случайным
// неизвестным кошельком. До фикса: cheapCheck запускал Argon2id до
// проверки подписи; после: дешёвый отказ (неизвестный кошелёк/
// подпись), Argon2id не выполняется.
func garbageMintTx(gp *wire.GenesisPackage) *wire.Transaction {
	var wallet [32]byte
	_, _ = rand.Read(wallet[:])
	var nonce [8]byte
	_, _ = rand.Read(nonce[:])
	prev := make([]byte, 32)
	return &wire.Transaction{
		NetworkID:       gp.Genesis.NetworkID,
		Wallet:          wallet[:],
		Seq:             1,
		Prev:            prev,
		Type:            wire.Mint,
		Payload:         &wire.MintPayload{Period: 1, Amount: 1000},
		ProtocolVersion: gp.Genesis.ProtocolVersion,
		TimestampClaim:  uint64(time.Now().Unix()),
		Pow:             &wire.PowProof{Algorithm: wire.PowArgon2id, Nonce: nonce[:]},
		Signature:       []byte{0x30, 0x06, 0x02, 0x01, 0x01, 0x02, 0x01, 0x01},
	}
}

// readProcCPUMs — CPU процесса (utime+stime) из /proc/PID/stat, мс.
func readProcCPUMs(pid int) int64 {
	blob, err := os.ReadFile(fmt.Sprintf("/proc/%d/stat", pid))
	if err != nil {
		return 0
	}
	s := string(blob)
	// поля 14/15 (utime/stime): после «) » идут поля с 3-го (state)
	i := strings.LastIndex(s, ")")
	if i < 0 || i+2 >= len(s) {
		return 0
	}
	fields := strings.Fields(s[i+2:])
	if len(fields) < 13 {
		return 0
	}
	utime, _ := strconv.ParseInt(fields[11], 10, 64)
	stime, _ := strconv.ParseInt(fields[12], 10, 64)
	// CLK_TCK = 100 (Linux userspace API)
	return (utime + stime) * 10
}

func addrInfoOfMA(s string) (peer.AddrInfo, error) {
	m, err := ma.NewMultiaddr(s)
	if err != nil {
		return peer.AddrInfo{}, err
	}
	ai, err := peer.AddrInfoFromP2pAddr(m)
	if err != nil {
		return peer.AddrInfo{}, err
	}
	return *ai, nil
}

func writeSamplesCSV(path string, samples []floodSample) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	w := csv.NewWriter(f)
	defer w.Flush()
	_ = w.Write([]string{"ts", "phase", "cpu_ms", "rss_kb", "journal_len",
		"spam_rejected", "expensive_checks", "budget_rejected"})
	for _, s := range samples {
		_ = w.Write([]string{
			s.TS, s.Phase, strconv.FormatInt(s.CPUMs, 10), strconv.Itoa(s.RSSKB),
			strconv.Itoa(s.JournalLen), strconv.FormatInt(s.SpamRejected, 10),
			strconv.FormatUint(s.ExpensiveChecks, 10), strconv.FormatUint(s.BudgetRejected, 10),
		})
	}
	return nil
}

func writeLatencyCSV(path string, recs []latencyRec) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	w := csv.NewWriter(f)
	defer w.Flush()
	_ = w.Write([]string{"ts", "phase", "seq", "seed", "latency_ms"})
	for _, r := range recs {
		_ = w.Write([]string{r.TS, r.Phase, strconv.Itoa(r.Seq), r.Seed, strconv.FormatInt(r.LatencyMs, 10)})
	}
	return nil
}
