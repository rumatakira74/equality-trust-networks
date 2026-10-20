// fleet.go — режимы масштабной seed-симуляции (подэтап 8.1):
//
//	findlabels — поиск меток кошельков со слотом челленджа в заданной
//	            полосе периода (окно ±48 ч достижимо только для «поздних»
//	            слотов — №39; метки тестовой сети детерминированы, поэтому
//	            полоса ищется перебором до генерации сети)
//	state     — снимок состояния узла-драйвера после сверки (state_root,
//	            эпоха/период, active, seed_sunset, балансы наблюдаемых
//	            кошельков) — для метрик и проверки сходимости прогона
//	fleet     — супервизор 48 persistent headless-узлов (tnp-node):
//	            порты, кошельки, БД, статус-файлы, журналы; мониторинг
//	            RSS/связности/журнала в CSV; чистая остановка по сигналу
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
	"io"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	lcrypto "github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/peer"

	"github.com/rumatakira74/equality-trust-networks/core/crypto"
	"github.com/rumatakira74/equality-trust-networks/core/wire"
)

// ---------------------------------------------------------------------------
// findlabels — поиск меток под полосу слота челленджа
// ---------------------------------------------------------------------------

// findlabelsCmd ищет метки, у которых слот челленджа периода period лежит
// в последних bandHours часах периода. При genesis_time = T и окне ±48 ч
// (MintChallengeWindowHours) окно такого слота накрывает момент
// T + (period+1)*30 сут: заявляемый период завершён (M4), окно открыто
// (M7) — способ достижения окна прогона 8.1 (якорь слота не меняется, №39).
func findlabelsCmd(args []string) error {
	fs := flag.NewFlagSet("findlabels", flag.ContinueOnError)
	kind := fs.String("kind", "seed", "seed | wallet — домен деривации ключа")
	count := fs.Int("count", 20, "сколько меток найти")
	period := fs.Uint64("period", 2, "период слота челленджа")
	bandHours := fs.Uint64("band-hours", 24, "полоса: слот в последних N часах периода")
	networkIDName := fs.String("network-id", "", "имя сети (как в generate)")
	genesisTime := fs.Int64("genesis-time", 0, "genesis-время, unix (как в generate)")
	prefix := fs.String("prefix", "", "префикс меток (по умолчанию: seed→S, wallet→P)")
	from := fs.Int("from", 1, "начальный номер метки")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *networkIDName == "" || *genesisTime == 0 {
		return fmt.Errorf("нужны -network-id и -genesis-time (те же, что в generate)")
	}
	networkID := networkIDOf(*networkIDName)
	pfx := *prefix
	if pfx == "" {
		pfx = "S"
		if *kind == "wallet" {
			pfx = "P"
		}
	}
	keyOf := func(label string) *ecdsa.PrivateKey {
		if *kind == "wallet" {
			return walletKey(label)
		}
		return seedKey(label)
	}
	periodEnd := uint64(*genesisTime) + (*period+1)*30*86400
	lo := periodEnd - *bandHours*3600
	found := 0
	for n := *from; found < *count; n++ {
		label := fmt.Sprintf("%s%02d", pfx, n)
		addr := addrOf(keyOf(label), networkID)
		slot := challengeSlot(uint64(*genesisTime), addr, *period)
		if slot >= lo && slot < periodEnd {
			found++
			fmt.Printf("%s slot=%d (%s) до-конца-периода=%dс\n",
				label, slot, time.Unix(int64(slot), 0).UTC().Format(time.RFC3339), periodEnd-slot)
		}
	}
	return nil
}

// ---------------------------------------------------------------------------
// state — снимок состояния после сверки
// ---------------------------------------------------------------------------

// stateCmd — подключиться, свериться, напечатать состояние реестра узла-драйвера.
func stateCmd(args []string) error {
	var c txFlags
	fs := flag.NewFlagSet("state", flag.ContinueOnError)
	addCommonFlags(fs, &c)
	watchSeeds := fs.String("watch-seeds", "", "метки посевных кошельков, через запятую")
	watchWallets := fs.String("watch-wallets", "", "метки кошельков-участников, через запятую")
	if err := fs.Parse(args); err != nil {
		return err
	}
	env, cleanup, err := startEnv(&c)
	if err != nil {
		return err
	}
	defer cleanup()

	st := env.node.Ledger.State
	active := uint64(0)
	for _, w := range st.Wallets {
		if w.Status == wire.StatusActive {
			active++
		}
	}
	out := map[string]any{
		"state_root":         fmt.Sprintf("%x", mustRoot(env)),
		"epoch":              st.Epoch,
		"period":             st.CurrentPeriod,
		"active_wallets":     active,
		"seed_rights_active": st.SeedRightsActive,
		"checkpoints":        env.node.CheckpointCount(),
		"journal_len":        env.node.Stats().JournalLen,
		"wallets":            map[string]any{},
	}
	wallets := out["wallets"].(map[string]any)
	add := func(label, domain string) {
		key := crypto.DeterministicKey(domain + label)
		addr := addrOf(key, env.gp.Genesis.NetworkID)
		w := st.Wallet(addr)
		if w == nil {
			wallets[label] = map[string]any{"address": hex.EncodeToString(addr), "status": "unknown"}
			return
		}
		wallets[label] = map[string]any{
			"address":     hex.EncodeToString(addr),
			"status":      w.Status.String(),
			"balance":     w.TotalBalance(),
			"liquid":      w.Liquid,
			"cohorts":     w.Cohorts,
			"seq":         w.Seq,
			"last_period": w.LastClaimedPeriod,
			"flagged":     w.Flagged,
		}
	}
	for _, l := range splitLabels(*watchSeeds) {
		add(l, "tnp-node-seed:")
	}
	for _, l := range splitLabels(*watchWallets) {
		add(l, "tnp-testnet-wallet:")
	}
	blob, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(blob))
	return nil
}

func splitLabels(s string) []string {
	var out []string
	for _, l := range strings.Split(s, ",") {
		if l = strings.TrimSpace(l); l != "" {
			out = append(out, l)
		}
	}
	return out
}

func mustRoot(env *txEnv) []byte {
	root, err := env.node.StateRoot()
	if err != nil {
		return nil
	}
	return root
}

// ---------------------------------------------------------------------------
// fleet — супервизор persistent headless-узлов
// ---------------------------------------------------------------------------

// fleetNode — описание узла флота (fleet.json).
type fleetNode struct {
	Index     int    `json:"index"`
	Key       string `json:"key"`
	Port      int    `json:"port"`
	Wallet    string `json:"wallet,omitempty"`
	Domain    string `json:"domain,omitempty"`
	PID       int    `json:"pid"`
	DB        string `json:"db"`
	Status    string `json:"status"`
	Log       string `json:"log"`
	Multiaddr string `json:"multiaddr"`
}

// nodeStatusJSON — статус-файл tnp-node (cmd/tnp-node writeStatus).
type nodeStatusJSON struct {
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

// fleetCmd — запуск и мониторинг флота. Супервизор переживает смерть
// отдельных узлов (прогон 1: потеря связи — kill извне, рестарт вручную
// теми же аргументами из fleet.json).
func fleetCmd(args []string) error {
	fs := flag.NewFlagSet("fleet", flag.ContinueOnError)
	genesis := fs.String("genesis", "", "путь к genesis.bin")
	out := fs.String("out", "fleet", "каталог артефактов флота")
	nodes := fs.Int("nodes", 48, "число headless-узлов")
	seeds := fs.String("seeds", "", "метки посевных кошельков, через запятую (узлы 1..N)")
	wallets := fs.String("wallets", "", "метки кошельков-участников, через запятую (следующие узлы)")
	basePort := fs.Int("base-port", 4100, "базовый UDP-порт (узел i: base+i)")
	bin := fs.String("bin", "./tnp-node", "бинарник tnp-node")
	duration := fs.Int64("duration", 0, "секунд работы (0 — до SIGTERM/SIGINT)")
	poll := fs.Int("poll", 5, "период мониторинга, сек")
	stagger := fs.Int("stagger", 300, "задержка между запусками узлов, мс")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *genesis == "" {
		return fmt.Errorf("нужен -genesis")
	}
	gp, err := loadGenesis(*genesis)
	if err != nil {
		return err
	}
	seedLabels := splitLabels(*seeds)
	walletLabels := splitLabels(*wallets)
	if len(seedLabels)+len(walletLabels) > *nodes {
		return fmt.Errorf("кошельков (%d) больше, чем узлов (%d)",
			len(seedLabels)+len(walletLabels), *nodes)
	}
	// посевные кошельки узлов обязаны быть в seed-круге genesis
	// (иначе не ACTIVE и не подписывают чекпоинты переходного режима)
	circle := map[string]bool{}
	for _, m := range gp.Manifest.SeedCircle {
		circle[hex.EncodeToString(m.Address)] = true
	}
	for _, l := range seedLabels {
		addr := addrOf(seedKey(l), gp.Genesis.NetworkID)
		if !circle[hex.EncodeToString(addr)] {
			return fmt.Errorf("кошелёк %s не в посевном круге genesis", l)
		}
	}
	for _, d := range []string{*out, *out + "/logs", *out + "/status", *out + "/db"} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			return err
		}
	}

	// раскладка: узел 1..len(seeds) — посевной круг, далее участники,
	// остаток — наблюдатели без кошелька
	type plan struct {
		key, wallet, domain string
	}
	plans := make([]plan, *nodes)
	for i := range plans {
		plans[i].key = fmt.Sprintf("fleet-%02d", i+1)
		if i < len(seedLabels) {
			plans[i].wallet = seedLabels[i]
			plans[i].domain = "seed"
		} else if j := i - len(seedLabels); j < len(walletLabels) {
			plans[i].wallet = walletLabels[j]
			plans[i].domain = "wallet"
		}
	} // multiaddr узла 1 (peer ID детерминирован от node-key) — bootstrap
	ma1, err := nodeMultiaddr(plans[0].key, *basePort+1)
	if err != nil {
		return fmt.Errorf("peer ID узла 1: %w", err)
	}

	fleet := make([]fleetNode, *nodes)
	logs := make([]*os.File, *nodes)
	logOff := make([]int64, *nodes)
	for i := range plans {
		port := *basePort + i + 1
		argv := []string{
			"-genesis-file", *genesis,
			"-node-key", plans[i].key,
			"-listen", fmt.Sprintf("/ip4/127.0.0.1/udp/%d/quic-v1", port),
			"-db", filepath.Join(*out, "db", fmt.Sprintf("node%02d.db", i+1)),
			"-link-groups=false",
			"-status-file", filepath.Join(*out, "status", fmt.Sprintf("node%02d.json", i+1)),
		}
		if i > 0 {
			argv = append(argv, "-peer", ma1)
			// второй ручной пир — перемешивание меша (каскад 12.3)
			if j := (i*7)%*nodes + 1; j != i+1 {
				ma, err := nodeMultiaddr(plans[j-1].key, *basePort+j)
				if err == nil {
					argv = append(argv, "-peer", ma)
				}
			}
		}
		if plans[i].wallet != "" {
			argv = append(argv, "-wallet", plans[i].wallet, "-wallet-domain", plans[i].domain)
		}
		logPath := filepath.Join(*out, "logs", fmt.Sprintf("node%02d.log", i+1))
		lf, err := os.Create(logPath)
		if err != nil {
			return err
		}
		logs[i] = lf
		cmd := exec.Command(*bin, argv...)
		cmd.Stdout = lf
		cmd.Stderr = lf
		if err := cmd.Start(); err != nil {
			return fmt.Errorf("узел %d: %w", i+1, err)
		}
		ma, err := nodeMultiaddr(plans[i].key, port)
		if err != nil {
			return err
		}
		fleet[i] = fleetNode{
			Index: i + 1, Key: plans[i].key, Port: port,
			Wallet: plans[i].wallet, Domain: plans[i].domain,
			PID:    cmd.Process.Pid,
			DB:     filepath.Join(*out, "db", fmt.Sprintf("node%02d.db", i+1)),
			Status: filepath.Join(*out, "status", fmt.Sprintf("node%02d.json", i+1)),
			Log:    logPath, Multiaddr: ma,
		}
		fmt.Fprintf(os.Stderr, "флот: узел %02d запущен (pid %d, порт %d, кошелёк %s)\n",
			i+1, cmd.Process.Pid, port, plans[i].wallet+spread(plans[i].domain))
		time.Sleep(time.Duration(*stagger) * time.Millisecond)
	}
	pids := make([]int, *nodes)
	for i := range fleet {
		pids[i] = fleet[i].PID
	}
	// fleet.json — для драйверов, рестартов и уборки
	blob, _ := json.MarshalIndent(map[string]any{"nodes": fleet, "bootstrap": ma1}, "", "  ")
	if err := os.WriteFile(filepath.Join(*out, "fleet.json"), blob, 0o644); err != nil {
		return err
	}
	fmt.Fprintf(os.Stderr, "флот: %d узлов, bootstrap %s\nfleet.json: %s\n",
		*nodes, ma1, filepath.Join(*out, "fleet.json"))

	// мониторинг: RSS + статус-файлы → CSV; события чекпоинтов из логов
	csvPath := filepath.Join(*out, "metrics.csv")
	csvFile, err := os.Create(csvPath)
	if err != nil {
		return err
	}
	defer csvFile.Close()
	w := csv.NewWriter(csvFile)
	_ = w.Write([]string{"ts", "node", "alive", "rss_kb", "peers", "journal_len",
		"checkpoints", "conflicts", "spam", "state_root12"})
	evPath := filepath.Join(*out, "events.csv")
	evFile, err := os.Create(evPath)
	if err != nil {
		return err
	}
	defer evFile.Close()
	ev := csv.NewWriter(evFile)
	_ = ev.Write([]string{"ts", "node", "event"})

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	deadline := time.Time{}
	if *duration > 0 {
		deadline = time.Now().Add(time.Duration(*duration) * time.Second)
	}
	tick := time.NewTicker(time.Duration(*poll) * time.Second)
	defer tick.Stop()
	for {
		select {
		case <-ctx.Done():
			fmt.Fprintln(os.Stderr, "флот: остановка по сигналу")
			shutdownFleet(pids)
			w.Flush()
			ev.Flush()
			writeFleetPIDs(*out, fleet)
			return nil
		case <-tick.C:
			now := time.Now().Unix()
			for i := range fleet {
				alive := processAlive(fleet[i].PID)
				rss := readRSSKB(fleet[i].PID)
				st := readNodeStatus(fleet[i].Status)
				root := ""
				if st != nil {
					root = st.StateRoot[:min(12, len(st.StateRoot))]
				}
				_ = w.Write([]string{
					strconv.FormatInt(now, 10), strconv.Itoa(i + 1), btoa(alive),
					strconv.Itoa(rss), itoa(st, func(s *nodeStatusJSON) int { return s.Peers }),
					itoa(st, func(s *nodeStatusJSON) int { return s.JournalLen }),
					itoa(st, func(s *nodeStatusJSON) int { return s.Checkpoints }),
					itoa(st, func(s *nodeStatusJSON) int { return s.Conflicts }),
					itoa64(st, func(s *nodeStatusJSON) int64 { return s.Spam }),
					root,
				})
				// события из журнала узла (чекпоинты, перестройки)
				for _, line := range readNewLines(fleet[i].Log, &logOff[i]) {
					if strings.Contains(line, "чекпоинт") ||
						strings.Contains(line, "перестройка") ||
						strings.Contains(line, "bootstrap") {
						_ = ev.Write([]string{strconv.FormatInt(now, 10), strconv.Itoa(i + 1), line})
					}
				}
			}
			w.Flush()
			ev.Flush()
			if !deadline.IsZero() && !time.Now().Before(deadline) {
				fmt.Fprintln(os.Stderr, "флот: время истекло")
				shutdownFleet(pids)
				w.Flush()
				ev.Flush()
				writeFleetPIDs(*out, fleet)
				return nil
			}
		}
	}
}

// spread — суффикс описания кошелька узла.
func spread(domain string) string {
	if domain == "" {
		return " (наблюдатель)"
	}
	return " (" + domain + ")"
}

func btoa(b bool) string {
	if b {
		return "1"
	}
	return "0"
}

func itoa(st *nodeStatusJSON, f func(*nodeStatusJSON) int) string {
	if st == nil {
		return ""
	}
	return strconv.Itoa(f(st))
}

func itoa64(st *nodeStatusJSON, f func(*nodeStatusJSON) int64) string {
	if st == nil {
		return ""
	}
	return strconv.FormatInt(f(st), 10)
}

// nodeMultiaddr — полный multiaddr узла с детерминированным peer ID
// (деривация та же, что в cmd/tnp-node: tnp-node-key:<label>).
func nodeMultiaddr(nodeKey string, port int) (string, error) {
	key := crypto.DeterministicKey("tnp-node-key:" + nodeKey)
	der, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		return "", err
	}
	priv, err := lcrypto.UnmarshalECDSAPrivateKey(der)
	if err != nil {
		return "", err
	}
	id, err := peer.IDFromPrivateKey(priv)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("/ip4/127.0.0.1/udp/%d/quic-v1/p2p/%s", port, id.String()), nil
}

func processAlive(pid int) bool {
	err := syscall.Kill(pid, 0)
	return err == nil || err == syscall.EPERM
}

func readRSSKB(pid int) int {
	b, err := os.ReadFile(fmt.Sprintf("/proc/%d/status", pid))
	if err != nil {
		return 0
	}
	for _, line := range strings.Split(string(b), "\n") {
		if strings.HasPrefix(line, "VmRSS:") {
			f := strings.Fields(line)
			if len(f) >= 2 {
				n, _ := strconv.Atoi(f[1])
				return n
			}
		}
	}
	return 0
}

func readNodeStatus(path string) *nodeStatusJSON {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	st := &nodeStatusJSON{}
	if json.Unmarshal(b, st) != nil {
		return nil
	}
	return st
}

// readNewLines — новые строки файла журнала с последнего смещения.
func readNewLines(path string, off *int64) []string {
	f, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer f.Close()
	if _, err := f.Seek(*off, io.SeekStart); err != nil {
		return nil
	}
	b, err := io.ReadAll(f)
	if err != nil {
		return nil
	}
	*off += int64(len(b))
	var out []string
	for _, l := range strings.Split(string(b), "\n") {
		if l = strings.TrimSpace(l); l != "" {
			out = append(out, l)
		}
	}
	return out
}

func shutdownFleet(pids []int) {
	for _, pid := range pids {
		if processAlive(pid) {
			_ = syscall.Kill(pid, syscall.SIGTERM)
		}
	}
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		alive := 0
		for _, pid := range pids {
			if processAlive(pid) {
				alive++
			}
		}
		if alive == 0 {
			return
		}
		time.Sleep(250 * time.Millisecond)
	}
	for _, pid := range pids {
		if processAlive(pid) {
			_ = syscall.Kill(pid, syscall.SIGKILL)
		}
	}
}

func writeFleetPIDs(out string, fleet []fleetNode) {
	// перезапись fleet.json с фактическими pid на момент остановки не нужна:
	// pid не меняются (супервизор единолично владеет процессами)
	_ = fleet
	_ = out
}

// min — минимум int.
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
