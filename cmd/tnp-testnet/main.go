// tnp-testnet — утилита закрытой тестовой сети (этап 8, задача 1).
//
// Режимы:
//
//	generate  — genesis-пакет (genesis.bin), манифест (manifest.json),
//	            подписанный seed-список (seedlist.bin), приглашения
//	            (invite.txt: QR/ссылка для invite_only), таблица слотов
//	            челленджа посевного круга
//	slots     — слоты челленджа и их окна (адреса посевного круга или -address)
//	register  — REGISTER кошелька (синтетическая аттестация, усиленный PoW)
//	vouch     — VOUCH посевного кошелька за кандидата
//	mint      — MINT периода посевным кошельком (± -save/-no-submit для
//	            эквивокации)
//	publish   — публикация сохранённой транзакции (файл)
//	transfer  — TRANSFER (обычный или отложенный — свыше P)
//	cancel    — отмена отложенного перевода (X1–X4)
//	migrate   — MIGRATE propose (гардианы подписывают statements) или
//	            cancel старым ключом (-mode cancel)
//	evidence  — EVIDENCE на эквивокации (две ветки одного seq)
//	findlabels — метки кошельков со слотом челленджа в полосе (8.1)
//	state     — снимок состояния реестра узла-драйвера (8.1)
//	fleet     — супервизор 48 persistent headless-узлов (8.1)
//
// Параметры профиля — KNOWLEDGE 4/17 (база params.TestGenesis); сеть
// закрытая: network_access = invite_only, вход invite_only_vouch, ключ
// приглашения — в genesis-записи (в манифесте не публикуется, spec/02 2.1).
// Ключи посевного круга и кошельков детерминированы от меток — тестовая
// сеть, не боевые ключи; живая сеть генерирует ключи на устройствах.
package main

import (
	"context"
	"crypto/ecdsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/rumatakira74/equality-trust-networks/core/attest"
	"github.com/rumatakira74/equality-trust-networks/core/crypto"
	"github.com/rumatakira74/equality-trust-networks/core/ledger"
	"github.com/rumatakira74/equality-trust-networks/core/params"
	"github.com/rumatakira74/equality-trust-networks/core/wire"
	tnp "github.com/rumatakira74/equality-trust-networks/mobile"
	"github.com/rumatakira74/equality-trust-networks/net"
)

const usage = `tnp-testnet — утилита закрытой тестовой сети (этап 8)

Режимы:
  generate  -network-id ИМЯ [-out КАТ] [-genesis-time T] [-seeds V1,V2,V3]
            [-peer MULTIADDR]... [-invite-key-hex HEX]
  slots     -genesis ФАЙЛ [-address HEX] [-periods 1,2,3]
  register  -genesis Ф -wallet МЕТКА [-guardians HEX,HEX] [-android-id ID] [-peer M]...
  vouch     -genesis Ф -seed МЕТКА -candidate HEX [-peer M]...
  mint      -genesis Ф -seed МЕТКА -period N [-save Ф] [-no-submit] [-peer M]...
  publish   -genesis Ф -tx Ф [-peer M]...
  transfer  -genesis Ф -seed МЕТКА -to HEX -amount N [-peer M]...
  cancel    -genesis Ф -seed МЕТКА -pending-seq N [-peer M]...
  migrate   -genesis Ф -mode propose -new МЕТКА -old HEX -guardians V2,V3 [-peer M]...
  migrate   -genesis Ф -mode cancel -old МЕТКА -new HEX -propose-seq N [-peer M]...
  evidence  -genesis Ф -seed МЕТКА -txa Ф -txb Ф [-peer M]...
  findlabels -network-id ИМЯ -genesis-time T -kind seed|wallet -count N
            -period P [-band-hours 24] [-from N]
  state     -genesis Ф [-watch-seeds М] [-watch-wallets М] [-peer M]...
  fleet     -genesis Ф -out КАТ -nodes N [-seeds М,М] [-wallets М,М]
            [-base-port 4100] [-bin ПУТЬ] [-duration СЕК] [-poll СЕК]

Общие флаги tx-режимов: -peer (multiaddr пиров, повторяется),
-wait СЕК (после публикации, по умолчанию 2), -node-seed СТРОКА,
-db ПУТЬ (SQLite; по умолчанию — в памяти), -link-groups БУЛЕВ
(по умолчанию true; false для localhost — №35).`

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, usage)
		os.Exit(2)
	}
	var err error
	switch os.Args[1] {
	case "generate":
		err = generate(os.Args[2:])
	case "slots":
		err = slotsCmd(os.Args[2:])
	case "register":
		err = registerCmd(os.Args[2:])
	case "vouch":
		err = vouchCmd(os.Args[2:])
	case "mint":
		err = mintCmd(os.Args[2:])
	case "publish":
		err = publishCmd(os.Args[2:])
	case "transfer":
		err = transferCmd(os.Args[2:])
	case "cancel":
		err = cancelCmd(os.Args[2:])
	case "migrate":
		err = migrateCmd(os.Args[2:])
	case "evidence":
		err = evidenceCmd(os.Args[2:])
	case "findlabels":
		err = findlabelsCmd(os.Args[2:])
	case "state":
		err = stateCmd(os.Args[2:])
	case "fleet":
		err = fleetCmd(os.Args[2:])
	default:
		fmt.Fprintln(os.Stderr, usage)
		os.Exit(2)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "tnp-testnet: %v\n", err)
		os.Exit(1)
	}
}

// ---------------------------------------------------------------------------
// Ключи и адреса (тестовая сеть: детерминированы от меток)
// ---------------------------------------------------------------------------

func seedKey(label string) *ecdsa.PrivateKey {
	return crypto.DeterministicKey("tnp-node-seed:" + label)
}
func walletKey(label string) *ecdsa.PrivateKey {
	return crypto.DeterministicKey("tnp-testnet-wallet:" + label)
}

func addrOf(key *ecdsa.PrivateKey, networkID []byte) []byte {
	pub := crypto.MarshalPubKey(&key.PublicKey)
	fp := crypto.Fingerprint(pub)
	return crypto.DeriveAddress(fp, networkID)
}

func hexAddr(s string) ([]byte, error) {
	b, err := hex.DecodeString(s)
	if err != nil || len(b) != 32 {
		return nil, fmt.Errorf("адрес: ожидается 32 байта в hex, получено %q", s)
	}
	return b, nil
}

// networkIDOf — 32-байтный network_id из имени (spec/02 2.1: 32 байта;
// имя — человекочитаемая метка, ID = SHA-256("tnp-network-id:" + имя)).
func networkIDOf(name string) []byte {
	h := sha256.Sum256([]byte("tnp-network-id:" + name))
	return h[:]
}

// ---------------------------------------------------------------------------
// Загрузка genesis и запуск узла
// ---------------------------------------------------------------------------

func loadGenesis(path string) (*wire.GenesisPackage, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	gp, err := wire.DecodeGenesisPackage(b)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	return gp, nil
}

// txEnv — окружение tx-режима: узел + genesis.
type txEnv struct {
	gp   *wire.GenesisPackage
	node *net.Node
	ctx  context.Context
}

// txFlags — общие флаги tx-режимов.
type txFlags struct {
	genesis    string
	wait       int
	nodeSeed   string
	db         string
	listen     string
	peers      []string
	linkGroups bool
}

func addCommonFlags(fs *flag.FlagSet, c *txFlags) {
	fs.StringVar(&c.genesis, "genesis", "", "путь к genesis.bin")
	fs.IntVar(&c.wait, "wait", 6, "секунд держать узел после публикации")
	fs.StringVar(&c.nodeSeed, "node-seed", "", "seed ключа узла (peer ID)")
	fs.StringVar(&c.db, "db", "", "путь к SQLite (пусто — в памяти)")
	fs.StringVar(&c.listen, "listen", "/ip4/0.0.0.0/udp/0/quic-v1,/ip4/0.0.0.0/tcp/0", "адреса прослушивания")
	fs.Var((*peerList)(&c.peers), "peer", "multiaddr пира (повторяется)")
	fs.BoolVar(&c.linkGroups, "link-groups", true, "несвязанность пиров по подсетям (№35); false для localhost")
}

func startEnv(c *txFlags) (*txEnv, func(), error) {
	if c.genesis == "" {
		return nil, nil, fmt.Errorf("нужен -genesis")
	}
	gp, err := loadGenesis(c.genesis)
	if err != nil {
		return nil, nil, err
	}
	nodeSeed := c.nodeSeed
	if nodeSeed == "" {
		nodeSeed = "tnp-testnet-node:" + strconv.FormatInt(time.Now().UnixNano(), 36)
	}
	// корни аттестации — та же деривация, что у приложения (mobile):
	// синтетический корень детерминирован от network_id
	mgen, err := tnp.LoadGenesis(gp.EncodeGenesisPackage())
	if err != nil {
		return nil, nil, err
	}
	var roots []*x509.Certificate
	if der := mgen.TrustedRootsDER(); len(der) > 0 {
		if root, err := x509.ParseCertificate(der); err == nil {
			roots = []*x509.Certificate{root}
		}
	}
	cfg := &net.NodeConfig{
		Genesis:      gp,
		NodeKey:      crypto.DeterministicKey(nodeSeed),
		Level:        net.LevelFull,
		DBPath:       c.db,
		ListenAddrs:  strings.Split(c.listen, ","),
		ManualAddrs:  c.peers,
		TrustedRoots: roots,
		// localhost: /24-классификатор (№35) схлопывает все узлы флота
		// в одну группу связи — несвязанность = уникальные Peer ID (8.1)
		LinkGroups: c.linkGroups,
		Log:        func(f string, a ...any) { fmt.Fprintf(os.Stderr, "узел: "+f+"\n", a...) },
	}
	node, err := net.NewNode(cfg)
	if err != nil {
		return nil, nil, fmt.Errorf("узел: %w", err)
	}
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	if err := node.Start(ctx); err != nil {
		stop()
		node.Close()
		return nil, nil, fmt.Errorf("запуск: %w", err)
	}
	// первичная сверка: узел обязан иметь журнал пиров ДО сборки
	// транзакций (tip/seq/prev из состояния; иначе эквивокация
	// получается случайной). Событие identify (net.syncLoop, 8.0.3)
	// сверяет сразу, здесь — детерминированный добор и пауза.
	time.Sleep(2 * time.Second)
	n := node.SyncAllPeers()
	if n > 0 {
		fmt.Fprintf(os.Stderr, "узел: первичная сверка: применено %d\n", n)
	}
	time.Sleep(1 * time.Second)
	wait := c.wait
	cleanup := func() {
		// держим узел живым ПОСЛЕ публикации: gossip-запись и сверка пиров
		// должны успеть до cancel/Close (закрытие сразу после SubmitTx
		// обрывало доставку — tx оставался только на узле драйвера)
		time.Sleep(time.Duration(wait) * time.Second)
		stop()
		node.Close()
	}
	fmt.Fprintf(os.Stderr, "узел: %s\n", node.Host.ID().String())
	return &txEnv{gp: gp, node: node, ctx: ctx}, cleanup, nil
}

// tipOf — seq/prev цепочки кошелька из состояния узла.
func tipOf(env *txEnv, addr []byte) (uint64, []byte) {
	w := env.node.Ledger.State.Wallet(addr)
	if w == nil {
		return 0, nil
	}
	return w.Seq, append([]byte(nil), w.TipTxID...)
}

// buildTx — сборка конверта + PoW + подпись.
func buildTx(env *txEnv, key *ecdsa.PrivateKey, addr []byte, typ wire.TxType,
	payload wire.Payload, hard bool) (*wire.Transaction, error) {
	seq, prev := tipOf(env, addr)
	tx := &wire.Transaction{
		NetworkID:       env.gp.Genesis.NetworkID,
		Wallet:          addr,
		Seq:             seq + 1,
		Prev:            prev,
		Type:            typ,
		Payload:         payload,
		ProtocolVersion: env.gp.Genesis.ProtocolVersion,
		TimestampClaim:  uint64(time.Now().Unix()),
	}
	net.SolveTxPoW(tx, *env.gp.Manifest.Profile.Pow, hard)
	sh := tx.SigningHash()
	sig, err := crypto.SignDER(key, sh[:])
	if err != nil {
		return nil, err
	}
	tx.Signature = sig
	return tx, nil
}

func submit(env *txEnv, tx *wire.Transaction) error {
	res, err := env.node.SubmitTx(env.ctx, tx)
	if err != nil {
		return err
	}
	id := tx.TxID()
	fmt.Printf("tx %s: %s/%s seq=%d\n", hex.EncodeToString(id[:8]),
		res.Outcome, res.Code, tx.Seq)
	if res.Outcome != ledger.Accepted {
		return fmt.Errorf("отклонён: %s", res.Code)
	}
	return nil
}

// ---------------------------------------------------------------------------
// generate
// ---------------------------------------------------------------------------

func generate(args []string) error {
	fs := flag.NewFlagSet("generate", flag.ContinueOnError)
	name := fs.String("network-id", "tnp-testnet-alpha", "имя сети (network_id = SHA-256 префикса+имени)")
	out := fs.String("out", "testnet", "каталог артефактов")
	genesisTime := fs.Int64("genesis-time", 0, "genesis-время, unix (0 = сейчас − 95 сут)")
	seeds := fs.String("seeds", "V1,V2,V3", "метки посевного круга, через запятую")
	inviteHex := fs.String("invite-key-hex", "", "ключ приглашения (hex; по умолчанию детерминирован от имени)")
	epochSeconds := fs.Uint64("epoch-seconds", 0, "epoch_seconds профиля (0 — 86400, №16); симуляция: 90/300")
	smallPow := fs.Bool("small-pow", false, "малые PoW-таргеты и Argon2 (тестовый оверрайд симуляции 8.1)")
	freshWindow := fs.Int("fresh-window", -1, "окно переводимости свежих выпусков (периодов; -1 — из KNOWLEDGE 4/17 = 3, №11); 0 — учебный оверрайд песочницы: за один прогон когорта не созревает")
	anchorEpochZero := fs.Bool("anchor-epoch-zero", false, "EpochZero = текущая эпоха от genesis; DeadlineEpoch seed_sunset = EpochZero+24 (закат через 24 эпохи от generate)")
	var peers peerList
	fs.Var(&peers, "peer", "multiaddr первого узла (повторяется)")
	if err := fs.Parse(args); err != nil {
		return err
	}

	networkID := networkIDOf(*name)
	gp := params.TestGenesis()
	gp.Genesis.NetworkID = networkID
	gp.Manifest.NetworkID = networkID
	gp.Manifest.DisplayName = "TNP Closed Testnet " + *name
	prof := gp.Manifest.Profile
	// закрытая сеть (4): доступ по приглашению, вход по ключу приглашения
	prof.NetworkAccess = wire.NetworkAccessInviteOnly
	prof.EntryMode = wire.EntryInviteOnlyVouch
	// прочие параметры — KNOWLEDGE 4/17 (значения params.TestGenesis):
	// выпуск 1000/мес, окно накопления 6, K=3, слоты 3/5/10, R=10% первых 12,
	// демередж 1%/3000, комиссия burn rate_min (заглушка этапа 3 — этап 10),
	// мораторий 14 сут, D=48 ч, P=3000/P_max=30000, окно челленджа ±48 ч,
	// допуск timestamp ±30 мин, эпоха 24 ч (№16), PoW №9, sandbox=false
	if *genesisTime == 0 {
		*genesisTime = time.Now().Truncate(24*time.Hour).Unix() - 95*86400
	}
	gp.Genesis.GenesisTime = uint64(*genesisTime)
	gp.Genesis.EpochZero = 0
	gp.Genesis.ProtocolVersion = 1
	// симуляция 8.1: эпоха короче суток (№16 — сетевые параметры теста);
	// малые PoW-таргеты (прогон без десятков секунд на транзакцию)
	if *epochSeconds > 0 {
		prof.EpochSeconds = *epochSeconds
	}
	if *smallPow {
		prof.Pow = &wire.PowParams{
			TargetLight: 0x0010_0000_0000_0000, TargetHard: 0x0010_0000_0000_0000,
			Argon2MemoryKiB: 8, Argon2Iterations: 1, Argon2Parallelism: 1,
		}
	}
	// учебный оверрайд окна переводимости (как в песочнице этапа 7):
	// когорта выпуска зреет 3 периода (№11) — в прогоне на живых часах
	// это 90 суток; полный путь за один прогон требует window=0
	if *freshWindow >= 0 {
		prof.FreshTransferWindowPeriods = uint64(*freshWindow)
	}
	// симуляция 8.1: якорь абсолютной эпохи. DeadlineEpoch seed_sunset (№15)
	// сравнивается с абсолютным номером эпохи (now-genesis)/epoch_seconds;
	// при genesis 91 сут назад и эпохе 5 мин ветка эпох сработала бы на
	// первой же транзакции. EpochZero = текущая эпоха, DeadlineEpoch =
	// EpochZero+24 — закат ровно через 24 эпохи от generate. Механизм №15
	// (порог active ИЛИ срок) не меняется; тестовая сеть — параметры
	// теста, живая сеть генерируется с дефолтами KNOWLEDGE 4/17.
	if *anchorEpochZero {
		if prof.EpochSeconds == 0 {
			prof.EpochSeconds = 86400
		}
		gp.Genesis.EpochZero = uint64(time.Now().Unix()-*genesisTime) / prof.EpochSeconds
		if ss := gp.Manifest.SeedSunset; ss != nil {
			ss.DeadlineEpoch = gp.Genesis.EpochZero + ss.DeadlineEpoch
		}
	}

	var circle []wire.SeedMember
	seedLabels := strings.Split(*seeds, ",")
	for _, label := range seedLabels {
		label = strings.TrimSpace(label)
		if label == "" {
			continue
		}
		key := seedKey(label)
		pub := crypto.MarshalPubKey(&key.PublicKey)
		circle = append(circle, wire.SeedMember{
			Address:   addrOf(key, networkID),
			PublicKey: pub,
		})
	}
	gp.Manifest.SeedCircle = circle
	inviteKey := []byte(*inviteHex)
	if *inviteHex == "" {
		h := sha256.Sum256([]byte("tnp-testnet-invite:" + *name))
		inviteKey = h[:]
	} else {
		b, err := hex.DecodeString(*inviteHex)
		if err != nil {
			return fmt.Errorf("-invite-key-hex: %w", err)
		}
		inviteKey = b
	}
	gp.Genesis.InviteKey = inviteKey
	mh := gp.Manifest.ManifestHash()
	gp.Genesis.ManifestHash = mh[:]

	if err := os.MkdirAll(*out, 0o755); err != nil {
		return err
	}
	// genesis.bin — пакет одним блобом (формат пары Encode/DecodeGenesisPackage)
	if err := os.WriteFile(*out+"/genesis.bin", gp.EncodeGenesisPackage(), 0o644); err != nil {
		return err
	}
	// manifest.json — человекочитаемый манифест
	mj, err := manifestJSON(gp, *name, seedLabels, inviteKey)
	if err != nil {
		return err
	}
	if err := os.WriteFile(*out+"/manifest.json", mj, 0o644); err != nil {
		return err
	}
	// seedlist.bin — подписанный seed-список (формат 12.3; в invite_only
	// bootstrap его не использует — артефакт формата)
	sl, err := seedlist(networkID, peers)
	if err != nil {
		return err
	}
	if err := os.WriteFile(*out+"/seedlist.bin", sl, 0o644); err != nil {
		return err
	}
	// invite.txt — приглашения (QR/ссылка): network_id, manifest_hash,
	// ключ приглашения, адреса первых узлов
	var inv strings.Builder
	fmt.Fprintf(&inv, "Приглашения в закрытую тестовую сеть %s\n", *name)
	fmt.Fprintf(&inv, "genesis-пакет: genesis.bin (передаётся вместе с приглашением)\n")
	fmt.Fprintf(&inv, "invite_key: %s\n", hex.EncodeToString(inviteKey))
	if len(peers) == 0 {
		fmt.Fprintf(&inv, "первый узел: адреса появятся после старта узлов —\n")
		fmt.Fprintf(&inv, "приглашение дополняется multiaddr первого узла\n")
	}
	for _, p := range []string(peers) {
		fmt.Fprintf(&inv, "tnp://invite?net=%s&genesis=%s&key=%s&peer=%s\n",
			hex.EncodeToString(networkID), hex.EncodeToString(mh[:]),
			hex.EncodeToString(inviteKey), p)
	}
	if err := os.WriteFile(*out+"/invite.txt", []byte(inv.String()), 0o644); err != nil {
		return err
	}

	fmt.Printf("сеть %s\n network_id: %s\n genesis_time: %d (%s)\n epoch_seconds: %d\n epoch_zero: %d\n seed_sunset: active>=%d или эпоха>=%d\n manifest_hash: %s\n invite_key: %s\n посевной круг: %d\n",
		*name, hex.EncodeToString(networkID), *genesisTime,
		time.Unix(*genesisTime, 0).UTC().Format(time.RFC3339),
		prof.EpochSeconds, gp.Genesis.EpochZero,
		gp.Manifest.SeedSunset.ActiveWallets, gp.Manifest.SeedSunset.DeadlineEpoch,
		hex.EncodeToString(mh[:]), hex.EncodeToString(inviteKey), len(circle))
	printSlots(gp, seedLabels, "", "1,2,3,4,5,6")
	fmt.Printf("артефакты: %s/{genesis.bin,manifest.json,seedlist.bin,invite.txt}\n", *out)
	return nil
}

// manifestJSON — человекочитаемый манифест тестовой сети.
func manifestJSON(gp *wire.GenesisPackage, name string, seedLabels []string, inviteKey []byte) ([]byte, error) {
	p := gp.Manifest.Profile
	type seedRow struct {
		Label     string `json:"label"`
		Address   string `json:"address"`
		PublicKey string `json:"public_key"`
	}
	seeds := make([]seedRow, 0, len(gp.Manifest.SeedCircle))
	for i, m := range gp.Manifest.SeedCircle {
		label := ""
		if i < len(seedLabels) {
			label = seedLabels[i]
		}
		seeds = append(seeds, seedRow{Label: label,
			Address: hex.EncodeToString(m.Address), PublicKey: hex.EncodeToString(m.PublicKey)})
	}
	out := map[string]any{
		"name":                          name,
		"network_id":                    hex.EncodeToString(gp.Genesis.NetworkID),
		"display_name":                  gp.Manifest.DisplayName,
		"manifest_hash":                 hex.EncodeToString(gp.Genesis.ManifestHash),
		"genesis_time":                  gp.Genesis.GenesisTime,
		"epoch_zero":                    gp.Genesis.EpochZero,
		"protocol_version":              gp.Genesis.ProtocolVersion,
		"network_access":                "invite_only",
		"entry_mode":                    "invite_only_vouch",
		"invite_key":                    hex.EncodeToString(inviteKey),
		"invite_key_note":               "тестовая сеть: детерминирован от имени; живая — случайный, вне репозитория",
		"seed_circle":                   seeds,
		"seed_circle_note":              "ключи детерминированы от меток (тестовая сеть); рекомендация ≥ 300 — 7.7",
		"issuance_amount":               p.IssuanceAmount,
		"issuance_period_months":        p.IssuancePeriodMonths,
		"accrual_window_periods":        p.AccrualWindowPeriods,
		"fresh_transfer_window_periods": p.FreshTransferWindowPeriods,
		"vouch_k":                       p.VouchK,
		"voucher_min_age_days":          p.VoucherMinAgeDays,
		"voucher_min_mint_periods":      p.VoucherMinMintPeriods,
		"slot_tiers":                    p.SlotTiers,
		"seed_slot_count":               p.SeedSlotCount,
		"voucher_reward_micros":         p.VoucherRewardMicros,
		"voucher_reward_periods":        p.VoucherRewardPeriods,
		"fee_mode":                      "burn",
		"fee_min":                       p.FeeMin,
		"rate_min_micros":               p.RateMin,
		"rate_max_micros":               p.RateMax,
		"demurrage_rate_micros":         p.DemurrageRateMicros,
		"demurrage_floor":               p.DemurrageFloor,
		"transfer_moratorium_days":      p.TransferMoratoriumDays,
		"transfer_delay_hours":          p.TransferDelayHours,
		"delayed_threshold":             p.DelayedThresholdDefault,
		"delayed_threshold_max":         p.DelayedThresholdMax,
		"guardian_personal_threshold":   p.GuardianPersonalThreshold,
		"guardian_personal_count":       p.GuardianPersonalCount,
		"guardian_public_threshold":     p.GuardianPublicThreshold,
		"guardian_min_age_days":         p.GuardianMinAgeDays,
		"guardian_reward_micros":        p.GuardianReward,
		"migrate_delay_days":            p.MigrateDelayDays,
		"epoch_seconds":                 p.EpochSeconds,
		"timestamp_tolerance_seconds":   p.TimestampToleranceSeconds,
		"mint_challenge_window_hours":   p.MintChallengeWindowHours,
		"checkpoint_mode":               "decentralized",
		"validators_m":                  p.ValidatorsM,
		"eclipse_n":                     p.EclipseN,
		"pow_target_light":              p.Pow.TargetLight,
		"pow_target_hard":               p.Pow.TargetHard,
		"argon2_mem_kib":                p.Pow.Argon2MemoryKiB,
		"argon2_iterations":             p.Pow.Argon2Iterations,
		"device_policy":                 "strict",
		"signing_auth":                  "biometric_strong",
		"sandbox":                       p.Sandbox,
		"seed_sunset_active_wallets":    gp.Manifest.SeedSunset.ActiveWallets,
		"seed_sunset_deadline_epoch":    gp.Manifest.SeedSunset.DeadlineEpoch,
	}
	return json.MarshalIndent(out, "", "  ")
}

// seedlist — подписанный seed-список (формат 12.3; дублирует кодирование
// net.SeedList — конструктор в пакете net не экспортирован, правило 3).
func seedlist(networkID []byte, peers []string) ([]byte, error) {
	var payload []byte
	payload = appendUvarint(payload, 1) // версия формата
	payload = appendBlob(payload, networkID)
	payload = appendUvarint(payload, uint64(time.Now().Unix()))
	payload = appendUvarint(payload, uint64(len(peers)))
	for _, p := range []string(peers) {
		ma := strings.Split(p, "/p2p/")
		if len(ma) != 2 {
			return nil, fmt.Errorf("peer %q: ожидается /p2p/<id>", p)
		}
		payload = appendBlob(payload, []byte(ma[1]))
		payload = appendUvarint(payload, 1)
		payload = appendBlob(payload, []byte(ma[0]))
	}
	h := sha256.Sum256(payload)
	sig := &wire.ThresholdSignature{}
	for i := 1; i <= 3; i++ {
		key := crypto.DeterministicKey(fmt.Sprintf("tnp-testnet-seedlist-key:%d", i))
		s, err := ecdsa.SignASN1(nil, key, h[:]) //nolint:staticcheck // rand nil = crypto/rand
		if err != nil {
			return nil, err
		}
		pub := crypto.MarshalPubKey(&key.PublicKey)
		id := sha256.Sum256(pub)
		sig.Signatures = append(sig.Signatures, wire.SignatureShare{Signer: id[:], Signature: s})
	}
	var out []byte
	out = appendBlob(out, payload)
	out = appendBlob(out, sig.Encode())
	return out, nil
}

func appendUvarint(b []byte, v uint64) []byte {
	var buf [binary.MaxVarintLen64]byte
	n := binary.PutUvarint(buf[:], v)
	return append(b, buf[:n]...)
}

func appendBlob(b, v []byte) []byte {
	b = appendUvarint(b, uint64(len(v)))
	return append(b, v...)
}

// ---------------------------------------------------------------------------
// slots
// ---------------------------------------------------------------------------

// challengeSlot — слот челленджа (7.6). Дублирует mintChallengeSlot
// (core/ledger, не экспортирован — правило 3): genesis_time + period×30 сут +
// SHA-256(addr‖period) mod 30 сут.
func challengeSlot(genesisTime uint64, addr []byte, period uint64) uint64 {
	var pb [8]byte
	binary.BigEndian.PutUint64(pb[:], period)
	h := sha256.Sum256(append(append([]byte{}, addr...), pb[:]...))
	return genesisTime + period*30*86400 + binary.BigEndian.Uint64(h[:8])%(30*86400)
}

func slotsCmd(args []string) error {
	fs := flag.NewFlagSet("slots", flag.ContinueOnError)
	genesisPath := fs.String("genesis", "", "путь к genesis.bin")
	address := fs.String("address", "", "доп. адрес (hex)")
	periods := fs.String("periods", "1,2,3,4,5,6", "периоды, через запятую")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *genesisPath == "" {
		return fmt.Errorf("нужен -genesis")
	}
	gp, err := loadGenesis(*genesisPath)
	if err != nil {
		return err
	}
	labels := make([]string, 0, len(gp.Manifest.SeedCircle))
	for range gp.Manifest.SeedCircle {
		labels = append(labels, "")
	}
	// метки посевного круга неизвестны из genesis — вывод по индексу
	_ = labels
	printSlots(gp, nil, *address, *periods)
	return nil
}

func printSlots(gp *wire.GenesisPackage, seedLabels []string, extraAddr string, periodsStr string) {
	periods := parsePeriods(periodsStr)
	win := gp.Manifest.Profile.MintChallengeWindowHours * 3600
	fmt.Printf("слоты челленджа (окно ±%d ч; достижимо из периода P+1 только для поздних слотов, №39):\n",
		gp.Manifest.Profile.MintChallengeWindowHours)
	type row struct {
		label string
		addr  []byte
	}
	rows := make([]row, 0, len(gp.Manifest.SeedCircle)+1)
	for i, m := range gp.Manifest.SeedCircle {
		label := fmt.Sprintf("seed#%d", i)
		if i < len(seedLabels) {
			label = seedLabels[i]
		}
		rows = append(rows, row{label: label, addr: m.Address})
	}
	if extraAddr != "" {
		if a, err := hexAddr(extraAddr); err == nil {
			rows = append(rows, row{label: "address", addr: a})
		} else {
			fmt.Fprintf(os.Stderr, "адрес игнорирован: %v\n", err)
		}
	}
	for _, r := range rows {
		for _, p := range periods {
			slot := challengeSlot(gp.Genesis.GenesisTime, r.addr, p)
			start := gp.Genesis.GenesisTime + (p+1)*30*86400
			reach := ""
			if win > 0 && slot+win >= start {
				lo := slot - win
				if lo < start {
					lo = start
				}
				reach = fmt.Sprintf(" ДОСТИЖИМО с %s по %s",
					time.Unix(int64(lo), 0).UTC().Format(time.RFC3339),
					time.Unix(int64(slot+win), 0).UTC().Format(time.RFC3339))
			}
			fmt.Printf(" %s период %d: слот %s (unix %d)%s\n",
				r.label, p, time.Unix(int64(slot), 0).UTC().Format(time.RFC3339), slot, reach)
		}
	}
}

func parsePeriods(s string) []uint64 {
	var out []uint64
	for _, part := range strings.Split(s, ",") {
		if v, err := strconv.ParseUint(strings.TrimSpace(part), 10, 64); err == nil {
			out = append(out, v)
		}
	}
	return out
}

// ---------------------------------------------------------------------------
// tx-режимы
// ---------------------------------------------------------------------------

func registerCmd(args []string) error {
	var c txFlags
	fs := flag.NewFlagSet("register", flag.ContinueOnError)
	addCommonFlags(fs, &c)
	wallet := fs.String("wallet", "", "метка кошелька (ключ детерминирован)")
	guardians := fs.String("guardians", "", "личные гардианы, hex через запятую")
	androidID := fs.String("android-id", "", "android_id устройства")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *wallet == "" {
		return fmt.Errorf("нужен -wallet")
	}
	env, cleanup, err := startEnv(&c)
	if err != nil {
		return err
	}
	defer cleanup()

	key := walletKey(*wallet)
	pub := crypto.MarshalPubKey(&key.PublicKey)
	fp := crypto.Fingerprint(pub)
	addr := crypto.DeriveAddress(fp, env.gp.Genesis.NetworkID)
	challenge := attest.ChallengeFormula(pub, crypto.ProtocolID, env.gp.Genesis.NetworkID)
	chain, err := tnp.SandboxAttestationChain(pub, env.gp.Genesis.NetworkID, challenge)
	if err != nil {
		return err
	}
	var personal [][]byte
	if *guardians != "" {
		for _, g := range strings.Split(*guardians, ",") {
			a, err := hexAddr(strings.TrimSpace(g))
			if err != nil {
				return err
			}
			personal = append(personal, a)
		}
	}
	payload := &wire.RegisterPayload{
		AttestedPublicKey:    pub,
		AttestationChain:     chain,
		AttestationChallenge: challenge,
		AndroidID:            *androidID,
		PersonalGuardians:    personal,
	}
	tx, err := buildTx(env, key, addr, wire.Register, payload, true)
	if err != nil {
		return err
	}
	fmt.Printf("register: кошелёк %s\n", hex.EncodeToString(addr))
	return submit(env, tx)
}

func vouchCmd(args []string) error {
	var c txFlags
	fs := flag.NewFlagSet("vouch", flag.ContinueOnError)
	addCommonFlags(fs, &c)
	seed := fs.String("seed", "", "метка поручителя (посевной круг)")
	candidate := fs.String("candidate", "", "адрес кандидата (hex)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *seed == "" || *candidate == "" {
		return fmt.Errorf("нужны -seed и -candidate")
	}
	cand, err := hexAddr(*candidate)
	if err != nil {
		return err
	}
	env, cleanup, err := startEnv(&c)
	if err != nil {
		return err
	}
	defer cleanup()
	key := seedKey(*seed)
	addr := addrOf(key, env.gp.Genesis.NetworkID)
	payload := &wire.VouchPayload{Candidate: cand, Epoch: env.node.Clock.Epoch()}
	tx, err := buildTx(env, key, addr, wire.Vouch, payload, false)
	if err != nil {
		return err
	}
	fmt.Printf("vouch: %s → %s (эпоха %d)\n", *seed, *candidate, payload.Epoch)
	return submit(env, tx)
}

func mintCmd(args []string) error {
	var c txFlags
	fs := flag.NewFlagSet("mint", flag.ContinueOnError)
	addCommonFlags(fs, &c)
	seed := fs.String("seed", "", "метка посевного кошелька")
	period := fs.Uint64("period", 0, "заявляемый период")
	keyDomain := fs.String("key-domain", "seed", "домен деривации ключа: seed (tnp-node-seed:) | wallet (tnp-testnet-wallet:)")
	save := fs.String("save", "", "сохранить транзакцию в файл")
	noSubmit := fs.Bool("no-submit", false, "не публиковать (для эквивокации)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *seed == "" || *period == 0 {
		return fmt.Errorf("нужны -seed и -period")
	}
	env, cleanup, err := startEnv(&c)
	if err != nil {
		return err
	}
	defer cleanup()

	key := seedKey(*seed)
	if *keyDomain == "wallet" {
		key = walletKey(*seed)
	}
	addr := addrOf(key, env.gp.Genesis.NetworkID)
	w := env.node.Ledger.State.Wallet(addr)
	if w == nil {
		return fmt.Errorf("кошелёк %s не в реестре", *seed)
	}
	// amount = эмиссия × diff (M6; конвенция ядра: base = last_claimed,
	// для первого выпуска — activated_period − 1, uint64-перенос для посева)
	base := w.LastClaimedPeriod
	if base == 0 {
		base = w.ActivatedPeriod - 1
	}
	diff := *period - base
	amount := env.gp.Manifest.Profile.IssuanceAmount * diff
	payload := &wire.MintPayload{Period: *period, Amount: amount}
	tx, err := buildTx(env, key, addr, wire.Mint, payload, true)
	if err != nil {
		return err
	}
	if *save != "" {
		if err := os.WriteFile(*save, tx.Encode(), 0o644); err != nil {
			return err
		}
	}
	if *noSubmit {
		fmt.Printf("mint: собран, не опубликован (seq=%d amount=%d)\n", tx.Seq, amount)
		return nil
	}
	return submit(env, tx)
}

func publishCmd(args []string) error {
	var c txFlags
	fs := flag.NewFlagSet("publish", flag.ContinueOnError)
	addCommonFlags(fs, &c)
	txF := fs.String("tx", "", "файл транзакции")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *txF == "" {
		return fmt.Errorf("нужен -tx")
	}
	env, cleanup, err := startEnv(&c)
	if err != nil {
		return err
	}
	defer cleanup()
	tx, err := readTx(*txF)
	if err != nil {
		return err
	}
	return submit(env, tx)
}

func transferCmd(args []string) error {
	var c txFlags
	fs := flag.NewFlagSet("transfer", flag.ContinueOnError)
	addCommonFlags(fs, &c)
	seed := fs.String("seed", "", "метка отправителя")
	to := fs.String("to", "", "адрес получателя (hex)")
	amount := fs.Uint64("amount", 0, "сумма")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *seed == "" || *to == "" || *amount == 0 {
		return fmt.Errorf("нужны -seed, -to, -amount")
	}
	env, cleanup, err := startEnv(&c)
	if err != nil {
		return err
	}
	defer cleanup()
	rcpt, err := hexAddr(*to)
	if err != nil {
		return err
	}
	key := seedKey(*seed)
	addr := addrOf(key, env.gp.Genesis.NetworkID)
	fee := env.node.Ledger.State.Net.Fee(*amount)
	payload := &wire.TransferPayload{Send: &wire.TransferSend{Recipient: rcpt, Amount: *amount}}
	tx, err := buildTx(env, key, addr, wire.Transfer, payload, false)
	if err != nil {
		return err
	}
	fmt.Printf("transfer: %d → %s, комиссия %d (сжигается)\n", *amount, *to, fee)
	return submit(env, tx)
}

func cancelCmd(args []string) error {
	var c txFlags
	fs := flag.NewFlagSet("cancel", flag.ContinueOnError)
	addCommonFlags(fs, &c)
	seed := fs.String("seed", "", "метка отправителя")
	pendingSeq := fs.Uint64("pending-seq", 0, "seq отложенного перевода")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *seed == "" || *pendingSeq == 0 {
		return fmt.Errorf("нужны -seed и -pending-seq")
	}
	env, cleanup, err := startEnv(&c)
	if err != nil {
		return err
	}
	defer cleanup()
	key := seedKey(*seed)
	addr := addrOf(key, env.gp.Genesis.NetworkID)
	payload := &wire.TransferPayload{Cancel: &wire.TransferCancel{PendingSeq: *pendingSeq}}
	tx, err := buildTx(env, key, addr, wire.Transfer, payload, false)
	if err != nil {
		return err
	}
	return submit(env, tx)
}

func migrateCmd(args []string) error {
	var c txFlags
	fs := flag.NewFlagSet("migrate", flag.ContinueOnError)
	addCommonFlags(fs, &c)
	mode := fs.String("mode", "propose", "propose | cancel")
	newW := fs.String("new", "", "propose: метка нового кошелька; cancel: адрес нового (hex)")
	old := fs.String("old", "", "propose: адрес старого (hex); cancel: метка старого")
	guardians := fs.String("guardians", "", "метки гардианов через запятую (propose)")
	proposeSeq := fs.Uint64("propose-seq", 0, "seq propose-транзакции (cancel)")
	nonce := fs.Uint64("nonce", 0, "nonce (propose; 0 = время)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	env, cleanup, err := startEnv(&c)
	if err != nil {
		return err
	}
	defer cleanup()
	epoch := env.node.Clock.Epoch()

	switch *mode {
	case "propose":
		if *newW == "" || *old == "" {
			return fmt.Errorf("нужны -new и -old")
		}
		oldAddr, err := hexAddr(*old)
		if err != nil {
			return err
		}
		key := walletKey(*newW)
		newAddr := addrOf(key, env.gp.Genesis.NetworkID)
		if *nonce == 0 {
			*nonce = uint64(time.Now().Unix())
		}
		var approvals []wire.GuardianApproval
		for _, g := range strings.Split(*guardians, ",") {
			g = strings.TrimSpace(g)
			if g == "" {
				continue
			}
			gk := seedKey(g)
			stmt := &wire.MigrateStatement{
				NetworkID:  env.gp.Genesis.NetworkID,
				OldAddress: oldAddr,
				NewAddress: newAddr,
				Nonce:      *nonce,
				Epoch:      epoch,
			}
			sh := stmt.Hash()
			sig, err := crypto.SignDER(gk, sh[:])
			if err != nil {
				return err
			}
			approvals = append(approvals, wire.GuardianApproval{
				Guardian:     addrOf(gk, env.gp.Genesis.NetworkID),
				StatementSig: sig,
			})
		}
		payload := &wire.MigratePayload{Propose: &wire.MigratePropose{
			OldAddress: oldAddr, Nonce: *nonce, Approvals: approvals,
		}}
		tx, err := buildTx(env, key, newAddr, wire.Migrate, payload, false)
		if err != nil {
			return err
		}
		fmt.Printf("migrate propose: %s → %s, гардианов %d\n",
			hex.EncodeToString(oldAddr[:8]), hex.EncodeToString(newAddr[:8]), len(approvals))
		return submit(env, tx)
	case "cancel":
		if *old == "" || *newW == "" || *proposeSeq == 0 {
			return fmt.Errorf("нужны -old, -new, -propose-seq")
		}
		newAddr, err := hexAddr(*newW)
		if err != nil {
			return err
		}
		key := walletKey(*old)
		oldAddr := addrOf(key, env.gp.Genesis.NetworkID)
		payload := &wire.MigratePayload{Cancel: &wire.MigrateCancel{
			NewAddress: newAddr, ProposeSeq: *proposeSeq,
		}}
		tx, err := buildTx(env, key, oldAddr, wire.Migrate, payload, false)
		if err != nil {
			return err
		}
		return submit(env, tx)
	}
	return fmt.Errorf("неизвестный режим %q", *mode)
}

func evidenceCmd(args []string) error {
	var c txFlags
	fs := flag.NewFlagSet("evidence", flag.ContinueOnError)
	addCommonFlags(fs, &c)
	seed := fs.String("seed", "", "метка подателя (посевной круг)")
	txa := fs.String("txa", "", "файл ветки A")
	txb := fs.String("txb", "", "файл ветки B")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *seed == "" || *txa == "" || *txb == "" {
		return fmt.Errorf("нужны -seed, -txa, -txb")
	}
	env, cleanup, err := startEnv(&c)
	if err != nil {
		return err
	}
	defer cleanup()
	ta, err := readTx(*txa)
	if err != nil {
		return err
	}
	tb, err := readTx(*txb)
	if err != nil {
		return err
	}
	key := seedKey(*seed)
	addr := addrOf(key, env.gp.Genesis.NetworkID)
	payload := &wire.EvidencePayload{TxA: ta, TxB: tb, Kind: wire.EvidenceEquivocation}
	tx, err := buildTx(env, key, addr, wire.Evidence, payload, false)
	if err != nil {
		return err
	}
	fmt.Printf("evidence: эквивокация %s seq=%d\n", hex.EncodeToString(ta.Wallet[:8]), ta.Seq)
	return submit(env, tx)
}

func readTx(path string) (*wire.Transaction, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return wire.DecodeTransaction(b)
}

// peerList — повторяемый флаг -peer.
type peerList []string

func (p *peerList) String() string     { return "multiaddr" }
func (p *peerList) Set(v string) error { *p = append(*p, v); return nil }
