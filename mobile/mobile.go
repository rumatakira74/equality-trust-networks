// Пакет tnp — мост ядра платформы сетей доверия для Android (этап 6).
// Собирается в AAR через gomobile bind; Kotlin-приложение (android/)
// вызывает эти функции через JNI. Ключи кошелька в Android Keystore
// НЕэкспортируемы: мост отдаёт хеш на подпись и принимает подпись
// (ECDSA P-256, DER) — приватный ключ никогда не покидает Keystore.
//
// Ограничение gomobile: в экспортируемом API только int64/[]byte/string/
// bool/error и экспортируемые типы с методами; uint64 из ядра конвертируется
// в int64 (значения сети далеки от 2^63).
package tnp

import (
	"context"
	"crypto/ecdsa"
	"crypto/x509"
	"encoding/binary"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/rumatakira74/equality-trust-networks/core/attest"
	"github.com/rumatakira74/equality-trust-networks/core/crypto"
	"github.com/rumatakira74/equality-trust-networks/core/params"
	"github.com/rumatakira74/equality-trust-networks/core/wire"
	"github.com/rumatakira74/equality-trust-networks/net"
)

// Logger — приём журнала узла из Go (реализуется в Kotlin).
type Logger interface {
	Log(line string)
}

// ---------------------------------------------------------------------------
// Кошелёк и аттестация
// ---------------------------------------------------------------------------

// GenerateKey — мягкая ключевая пара ECDSA P-256 (тест/резерв; основной
// ключ — Android Keystore, не проходит через мост).
func GenerateKey() ([]byte, error) {
	k, err := crypto.GenerateKey()
	if err != nil {
		return nil, err
	}
	return crypto.MarshalPubKey(&k.PublicKey), nil
}

// AddressFromPublicKey — адрес кошелька из публичного ключа (Keystore
// отдаёт публичный ключ аттестованной пары).
func AddressFromPublicKey(pub, networkID []byte) ([]byte, error) {
	if len(pub) == 0 {
		return nil, errors.New("пустой публичный ключ")
	}
	fp := crypto.Fingerprint(pub)
	return crypto.DeriveAddress(fp, networkID), nil
}

// FingerprintOf — фингерпринт публичного ключа (7.2).
func FingerprintOf(pub []byte) []byte { return crypto.Fingerprint(pub) }

// AttestationChallengeFor — challenge для setAttestationChallenge (R4,
// вопрос №23: SHA-256(pub || protocol_id || network_id)); задаёт узел.
func AttestationChallengeFor(pub, networkID []byte) []byte {
	return attest.ChallengeFormula(pub, crypto.ProtocolID, networkID)
}

// Attestation — разобранная цепочка аттестации (для отображения и
// локальной проверки на устройстве; каноническая проверка — в ядре).
type Attestation struct {
	securityLevel     int
	verifiedBootState int
	deviceLocked      bool
	patchYear         int
	patchMonth        int
	challenge         []byte
}

func (a *Attestation) SecurityLevel() int     { return a.securityLevel }
func (a *Attestation) VerifiedBootState() int { return a.verifiedBootState }
func (a *Attestation) DeviceLocked() bool     { return a.deviceLocked }
func (a *Attestation) PatchYear() int         { return a.patchYear }
func (a *Attestation) PatchMonth() int        { return a.patchMonth }
func (a *Attestation) Challenge() []byte      { return a.challenge }

// AppendCert — добавляет DER-сертификат в блоб корней (формат: длина
// BE u32 + DER; тот же формат, что и цепочка аттестации).
func AppendCert(blob, certDER []byte) []byte {
	return attest.EncodeChain(append(append([]byte{}, blob...), certDER...))
}

// ParseAttestation — разбор и проверка цепочки аттестации (эмулятор —
// синтетические корни из core/attest; устройство — цепочка Google).
// minLevel: 0 Software, 1 TEE, 2 StrongBox (7.2).
func ParseAttestation(chain, roots []byte, minLevel, maxPatchAgeMonths int, nowSec int64) (*Attestation, error) {
	trusted, err := attest.DecodeChain(roots)
	if err != nil {
		return nil, fmt.Errorf("корни: %w", err)
	}
	c, err := attest.Verify(chain, trusted, minLevel, maxPatchAgeMonths,
		time.Unix(nowSec, 0).UTC())
	if err != nil {
		return nil, err
	}
	return &Attestation{
		securityLevel:     c.Extension.SecurityLevel,
		verifiedBootState: c.Extension.VerifiedBootState,
		deviceLocked:      c.Extension.DeviceLocked,
		patchYear:         c.Extension.PatchYear,
		patchMonth:        c.Extension.PatchMonth,
		challenge:         c.Challenge,
	}, nil
}

// ---------------------------------------------------------------------------
// Транзакции
// ---------------------------------------------------------------------------

// Tx — транзакция ядра (core/wire). Сборка → PoW → хеш на подпись →
// подпись (Keystore) → публикация.
type Tx struct{ tx *wire.Transaction }

func newTx(networkID, wallet []byte, seq int64, prev []byte, typ wire.TxType,
	payload wire.Payload, protocolVersion, ts int64) *Tx {
	return &Tx{tx: &wire.Transaction{
		NetworkID:       append([]byte(nil), networkID...),
		Wallet:          append([]byte(nil), wallet...),
		Seq:             uint64(seq),
		Prev:            append([]byte(nil), prev...),
		Type:            typ,
		Payload:         payload,
		ProtocolVersion: uint32(protocolVersion),
		TimestampClaim:  uint64(ts),
	}}
}

// NewRegister — register (9.3): аттестованный ключ, цепочка, challenge.
func NewRegister(networkID, wallet, pub, chain, challenge []byte, androidID string,
	protocolVersion, ts int64) *Tx {
	return newTx(networkID, wallet, 1, make([]byte, 32), wire.Register,
		&wire.RegisterPayload{
			AttestedPublicKey:    append([]byte(nil), pub...),
			AttestationChain:     append([]byte(nil), chain...),
			AttestationChallenge: append([]byte(nil), challenge...),
			AndroidID:            androidID,
		}, protocolVersion, ts)
}

// NewMint — выпуск периода (7.6, усиленный PoW Argon2id).
func NewMint(networkID, wallet []byte, seq int64, prev []byte, period, amount,
	protocolVersion, ts int64) *Tx {
	return newTx(networkID, wallet, seq, prev, wire.Mint,
		&wire.MintPayload{Period: uint64(period), Amount: uint64(amount)},
		protocolVersion, ts)
}

// NewTransfer — перевод (8.3; лёгкий PoW).
func NewTransfer(networkID, wallet []byte, seq int64, prev, recipient []byte,
	amount, protocolVersion, ts int64) *Tx {
	return newTx(networkID, wallet, seq, prev, wire.Transfer,
		&wire.TransferPayload{Send: &wire.TransferSend{
			Recipient: append([]byte(nil), recipient...), Amount: uint64(amount)}},
		protocolVersion, ts)
}

// Seq/Prev/Wallet/Type — поля транзакции (для UI).
func (t *Tx) Seq() int64     { return int64(t.tx.Seq) }
func (t *Tx) Prev() []byte   { return t.tx.Prev }
func (t *Tx) Wallet() []byte { return t.tx.Wallet }
func (t *Tx) Type() string   { return t.tx.Type.String() }

// SolveLight — лёгкий PoW (SHA-256, 7.6): nonce для register/transfer.
// Правка этапа 7 (латентный баг этапа 6): canonical-кодирование
// включает pow.nonce, поэтому nonce назначается В tx ДО вычисления
// preimage — иначе проверка на узле даёт E_POW_LIGHT.
func (t *Tx) SolveLight(target int64) {
	t.tx.Pow = &wire.PowProof{Algorithm: wire.PowLight}
	for n := uint64(0); ; n++ {
		var nb [8]byte
		binary.BigEndian.PutUint64(nb[:], n)
		t.tx.Pow.Nonce = nb[:]
		if crypto.VerifyLight(t.tx.SigningPreimage(), nb[:], uint64(target)) {
			return
		}
	}
}

// SolveHard — усиленный PoW (Argon2id, 7.6): выпуск. Nonce назначается
// в tx до вычисления preimage (см. SolveLight).
func (t *Tx) SolveHard(target, memKiB, iterations, parallelism int64) {
	t.tx.Pow = &wire.PowProof{Algorithm: wire.PowArgon2id}
	ap := crypto.Argon2Params{MemoryKiB: uint32(memKiB), Iterations: uint32(iterations),
		Parallelism: uint32(parallelism)}
	for n := uint64(0); ; n++ {
		var nb [8]byte
		binary.BigEndian.PutUint64(nb[:], n)
		t.tx.Pow.Nonce = nb[:]
		if crypto.VerifyHard(t.tx.SigningPreimage(), nb[:], uint64(target), ap) {
			return
		}
	}
}

// SigningHash — хеш для подписи (Keystore: SHA256withECDSA, выход DER).
// Подпись приватным ключом Keystore; ключ не покидает Keystore.
func (t *Tx) SigningHash() []byte {
	h := t.tx.SigningHash()
	return h[:]
}

// SetSignature — подпись собранной транзакции (DER, из Keystore).
func (t *Tx) SetSignature(sig []byte) { t.tx.Signature = append([]byte(nil), sig...) }

// VerifySignature — локальная проверка подписи публичным ключом.
func (t *Tx) VerifySignature(pub []byte) (bool, error) {
	pk, err := crypto.ParsePubKey(pub)
	if err != nil {
		return false, err
	}
	h := t.tx.SigningHash()
	return crypto.VerifyDER(pk, h[:], t.tx.Signature), nil
}

// Encode — каноническое protobuf-кодирование (публикация/Bluetooth).
func (t *Tx) Encode() []byte { return t.tx.Encode() }

// ID — txid (32 байта).
func (t *Tx) ID() []byte {
	id := t.tx.TxID()
	return id[:]
}

// DecodeTx — разбор кодированной транзакции (Bluetooth-приём, sync).
func DecodeTx(b []byte) (*Tx, error) {
	tx, err := wire.DecodeTransaction(b)
	if err != nil {
		return nil, err
	}
	return &Tx{tx: tx}, nil
}

// ---------------------------------------------------------------------------
// Genesis (песочница; реальные сети — genesis-файл, распространение —
// отдельный трек)
// ---------------------------------------------------------------------------

// Genesis — genesis-пакет сети (манифест + запись).
type Genesis struct{ gp *wire.GenesisPackage }

// SandboxGenesis — тестовый genesis с параметрами PoW (№9) и эпохи (№16).
// genesisTime — unix-секунды; 0 — 65 суток назад (текущий период 2).
//
// Этап 7 (песочница эмулятора, 6.5): добавлен посевной круг V1..V3 —
// детерминированные ключи (та же деривация, что в cmd/tnp-node), без
// него в песочнице нет ни одного active-кошелька и активация
// невозможна. Учебные оверрайды профиля: мораторий переводов и окно
// переводимости свежих выпусков выключены (демо полного пути за один
// сеанс), окно челленджа выпуска открыто весь период (в реальных сетях
// — ±48 ч, 7.6). Отложенные переводы в песочнице отключены ядром
// (T6/T7: !sandbox).
func SandboxGenesis(networkID string, genesisTime, epochSeconds,
	powTargetLight, powTargetHard, argon2MemKiB, argon2Iter, argon2Par int64) *Genesis {
	gp := params.TestGenesis()
	gp.Genesis.NetworkID = []byte(networkID)
	gp.Manifest.NetworkID = []byte(networkID)
	prof := gp.Manifest.Profile
	if epochSeconds > 0 {
		prof.EpochSeconds = uint64(epochSeconds)
	}
	prof.Sandbox = true
	prof.TransferMoratoriumDays = 0
	prof.FreshTransferWindowPeriods = 0
	prof.MintChallengeWindowHours = 24 * 31
	prof.Pow = &wire.PowParams{
		TargetLight: uint64(powTargetLight), TargetHard: uint64(powTargetHard),
		Argon2MemoryKiB: uint32(argon2MemKiB), Argon2Iterations: uint32(argon2Iter),
		Argon2Parallelism: uint32(argon2Par),
	}
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
	if genesisTime == 0 {
		genesisTime = time.Now().Add(-65 * 24 * time.Hour).Unix()
	}
	gp.Genesis.GenesisTime = uint64(genesisTime)
	mh := gp.Manifest.ManifestHash()
	gp.Genesis.ManifestHash = mh[:]
	return &Genesis{gp: gp}
}

// TrustedRootsDER — корни аттестации genesis (синтетические; для эмулятора).
func (g *Genesis) TrustedRootsDER() []byte {
	// синтетический корень детерминирован от network_id — одинаков у всех
	key := crypto.DeterministicKey("tnp-android-root:" + string(g.gp.Genesis.NetworkID))
	_, der, err := attest.SyntheticRoot("android", key)
	if err != nil {
		return nil
	}
	return der
}

// Параметры сети для приложения (Kotlin читает через мост).

// NetworkID — идентификатор сети.
func (g *Genesis) NetworkID() []byte { return g.gp.Genesis.NetworkID }

// ProtocolVersion — версия протокола genesis.
func (g *Genesis) ProtocolVersion() int64 {
	return int64(g.gp.Genesis.ProtocolVersion)
}

// PowTargetLight — таргет лёгкого PoW (register/transfer).
func (g *Genesis) PowTargetLight() int64 { return int64(g.gp.Manifest.Profile.Pow.TargetLight) }

// PowTargetHard — таргет усиленного PoW (выпуск).
func (g *Genesis) PowTargetHard() int64 { return int64(g.gp.Manifest.Profile.Pow.TargetHard) }

// Argon2MemKiB / Argon2Iterations / Argon2Par — параметры Argon2id (№9).
func (g *Genesis) Argon2MemKiB() int64 {
	return int64(g.gp.Manifest.Profile.Pow.Argon2MemoryKiB)
}

func (g *Genesis) Argon2Iterations() int64 {
	return int64(g.gp.Manifest.Profile.Pow.Argon2Iterations)
}

func (g *Genesis) Argon2Par() int64 {
	return int64(g.gp.Manifest.Profile.Pow.Argon2Parallelism)
}

// EpochSeconds — длительность эпохи чекпоинта (№16: 24 ч).
func (g *Genesis) EpochSeconds() int64 {
	return int64(g.gp.Manifest.Profile.EpochSeconds)
}

// ---------------------------------------------------------------------------
// Узел сети (полный/лёгкий, 12.2)
// ---------------------------------------------------------------------------

// Node — узел сети в процессе приложения (foreground service).
type Node struct {
	n   *net.Node
	gen *Genesis
}

// StartNode — запуск узла. level: "full" или "light" (12.2); dbPath —
// SQLite ("" — без персистентности, только лёгкий); ownWallet — адрес
// своего кошелька (лёгкий узел); manualAddrs — multiaddr пиров через
// запятую (bootstrap); logger — журнал (nil — тихо).
func StartNode(gen *Genesis, level, dbPath string, ownWallet []byte,
	manualAddrs string, logger Logger) (*Node, error) {
	if gen == nil || gen.gp == nil {
		return nil, errors.New("genesis не задан")
	}
	cfg := &net.NodeConfig{
		Genesis:     gen.gp,
		DBPath:      dbPath,
		ListenAddrs: []string{"/ip4/0.0.0.0/udp/0/quic-v1", "/ip4/0.0.0.0/tcp/0"},
		// несвязанность eclipse-пиров по подсетям (№35): включена
		LinkGroups: true,
	}
	// корни аттестации из genesis (правка этапа 7 — латентный баг этапа 6:
	// без TrustedRoots узел отвергает любую цепочку, E_ATTEST). Песочница —
	// синтетический корень (детерминирован от network_id); реальные сети —
	// корни Google в APK (этап 8, чек-лист).
	if der := gen.TrustedRootsDER(); len(der) > 0 {
		if root, err := x509.ParseCertificate(der); err == nil {
			cfg.TrustedRoots = []*x509.Certificate{root}
		}
	}
	nd, err := startNodeCfg(cfg, level, ownWallet, manualAddrs, logger)
	if err != nil {
		return nil, err
	}
	nd.gen = gen
	return nd, nil
}

// StartNodeAddrs — запуск узла с явными адресами прослушивания
// (этап 7: host-узел песочницы для эмулятора на фиксированном порту).
func StartNodeAddrs(gen *Genesis, level, dbPath string, ownWallet []byte,
	manualAddrs, listenAddrs string, logger Logger) (*Node, error) {
	if gen == nil || gen.gp == nil {
		return nil, errors.New("genesis не задан")
	}
	cfg := &net.NodeConfig{
		Genesis:     gen.gp,
		DBPath:      dbPath,
		ListenAddrs: []string{"/ip4/0.0.0.0/udp/0/quic-v1", "/ip4/0.0.0.0/tcp/0"},
		LinkGroups:  true,
	}
	if listenAddrs != "" {
		cfg.ListenAddrs = strings.Split(listenAddrs, ",")
	}
	if der := gen.TrustedRootsDER(); len(der) > 0 {
		if root, err := x509.ParseCertificate(der); err == nil {
			cfg.TrustedRoots = []*x509.Certificate{root}
		}
	}
	nd, err := startNodeCfg(cfg, level, ownWallet, manualAddrs, logger)
	if err != nil {
		return nil, err
	}
	nd.gen = gen
	return nd, nil
}

// StartNodeAddrsKey — как StartNodeAddrs, но с детерминированным ключом
// узла из nodeKeySeed (этап 7, песочница): peer ID стабилен между
// запусками — приложение может хранить полный multiaddr с /p2p/<id>
// (AddrInfoFromP2pAddr требует ID в адресе). Пустой seed — случайный
// ключ. Только песочница/диагностика; реальные узлы — случайные ключи.
func StartNodeAddrsKey(gen *Genesis, level, dbPath string, ownWallet []byte,
	manualAddrs, listenAddrs, nodeKeySeed string, logger Logger) (*Node, error) {
	if nodeKeySeed != "" {
		key := crypto.DeterministicKey(nodeKeySeed)
		cfg := &net.NodeConfig{
			Genesis:     gen.gp,
			DBPath:      dbPath,
			ListenAddrs: strings.Split(listenAddrs, ","),
			LinkGroups:  true,
			NodeKey:     key,
		}
		if der := gen.TrustedRootsDER(); len(der) > 0 {
			if root, err := x509.ParseCertificate(der); err == nil {
				cfg.TrustedRoots = []*x509.Certificate{root}
			}
		}
		nd, err := startNodeCfg(cfg, level, ownWallet, manualAddrs, logger)
		if err != nil {
			return nil, err
		}
		nd.gen = gen
		return nd, nil
	}
	return StartNodeAddrs(gen, level, dbPath, ownWallet, manualAddrs, listenAddrs, logger)
}

// startNodeCfg — общая сборка и запуск по конфигу.
func startNodeCfg(cfg *net.NodeConfig, level string, ownWallet []byte,
	manualAddrs string, logger Logger) (*Node, error) {
	switch level {
	case "full":
		cfg.Level = net.LevelFull
	case "light":
		cfg.Level = net.LevelLight
		cfg.OwnWallet = append([]byte(nil), ownWallet...)
		if len(cfg.OwnWallet) == 0 {
			return nil, errors.New("лёгкому узлу нужен адрес кошелька")
		}
	default:
		return nil, fmt.Errorf("неизвестный уровень узла: %q", level)
	}
	if manualAddrs != "" {
		for _, a := range strings.Split(manualAddrs, ",") {
			a = strings.TrimSpace(a)
			if a != "" {
				cfg.ManualAddrs = append(cfg.ManualAddrs, a)
			}
		}
	}
	if logger != nil {
		cfg.Log = func(f string, a ...any) { logger.Log(fmt.Sprintf(f, a...)) }
	}
	n, err := net.NewNode(cfg)
	if err != nil {
		return nil, err
	}
	if err := n.Start(context.Background()); err != nil {
		n.Close()
		return nil, err
	}
	return &Node{n: n, gen: nil}, nil
}

// SubmitResult — результат публикации транзакции.
type SubmitResult struct {
	outcome string
	code    string
}

func (r *SubmitResult) Outcome() string { return r.outcome }
func (r *SubmitResult) Code() string    { return r.code }

// Submit — публикация подписанной транзакции (apply + gossip).
func (nd *Node) Submit(tx *Tx) (*SubmitResult, error) {
	if tx == nil {
		return &SubmitResult{"rejected", "E_ARGS"}, errors.New("пустая транзакция")
	}
	res, err := nd.n.SubmitTx(context.Background(), tx.tx)
	return &SubmitResult{res.Outcome.String(), res.Code}, err
}

// Tip — актуальный seq/prev кошелька по опросу пиров.
type Tip struct {
	seq  int64
	prev []byte
}

func (t *Tip) Seq() int64   { return t.seq }
func (t *Tip) Prev() []byte { return t.prev }

// TipFromPeers — актуальный seq кошелька от нескольких пиров перед
// подписью (12.4): опрашиваются подключённые пиры, требуется согласие
// minAgree одинаковых ответов (eclipse-защита); prev — tip- txid.
func (nd *Node) TipFromPeers(wallet []byte, minAgree int) (*Tip, error) {
	peers := nd.n.ConnectedPeers()
	if len(peers) == 0 {
		return nil, errors.New("нет подключённых пиров")
	}
	if minAgree <= 0 {
		minAgree = 2
	}
	type answer struct {
		seq  int64
		prev string
	}
	counts := map[answer]int{}
	for _, p := range peers {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		seq, prev, err := nd.n.GetTip(ctx, p, wallet)
		cancel()
		if err != nil {
			continue
		}
		a := answer{int64(seq), string(prev)}
		counts[a]++
		if counts[a] >= minAgree {
			return &Tip{seq: a.seq, prev: []byte(a.prev)}, nil
		}
	}
	return nil, fmt.Errorf("нет согласия %d пиров по seq (ответы: %d)",
		minAgree, len(counts))
}

// LightSync — лёгкий узел: заголовки чекпоинтов и свои входящие (12.5).
func (nd *Node) LightSync() error {
	return nd.n.LightSync(context.Background())
}

// Header — заголовок финализированного чекпоинта (12.5).
type Header struct {
	epoch     int64
	hash      []byte
	stateRoot []byte
}

func (h *Header) Epoch() int64      { return h.epoch }
func (h *Header) Hash() []byte      { return h.hash }
func (h *Header) StateRoot() []byte { return h.stateRoot }

// LatestHeader — последний финализированный чекпоинт (nil — ещё нет).
func (nd *Node) LatestHeader() (*Header, error) {
	h, ok := nd.n.LatestHeader()
	if !ok {
		return nil, nil
	}
	return &Header{epoch: int64(h.Epoch), hash: h.Hash, stateRoot: h.StateRoot}, nil
}

// BalanceWithProof — баланс по Merkle-доказательству против корня
// ФИНАЛИЗИРОВАННОГО чекпоинта (12.5): лёгкий узел не доверяет состоянию
// полного узла, только пороговой подписи чекпоинта. Отрицательный
// баланс невозможен в сети; -1 = доказательства нет.
func (nd *Node) BalanceWithProof(addr []byte) (int64, error) {
	b, ok, err := nd.n.BalanceWithProof(context.Background(), addr)
	if err != nil {
		return -1, err
	}
	if !ok {
		return -1, errors.New("доказательство не сходится с чекпоинтом")
	}
	return b, nil
}

// OnOfflineTx — транзакция, принятая по Bluetooth (12.4): статус
// «ожидает публикации»; проверка seq/prev — при публикации.
func (nd *Node) OnOfflineTx(tx *Tx) {
	if tx != nil {
		nd.n.OnOfflineTx(tx.tx)
	}
}

// PublishAwaiting — публикация ожидающих (появилась сеть).
func (nd *Node) PublishAwaiting() {
	nd.n.PublishAwaiting(context.Background())
}

// OfflineStatus — статус офлайн-транзакции: awaiting_publication /
// published / finalized (12.4: «получено по Bluetooth» ≠ оплата).
func (nd *Node) OfflineStatus(txid []byte) string {
	var id [32]byte
	copy(id[:], txid)
	return nd.n.OfflineStatus(id).String()
}

// StatsText — сводка узла для UI (JSON).
func (nd *Node) StatsText() string {
	s := nd.n.Stats()
	return fmt.Sprintf(`{"level":%q,"peer_id":%q,"peers":%d,"journal_len":%d,`+
		`"spam_rejected":%d,"pending":%d,"awaiting":%d,"conflicts":%d}`,
		s.Level, s.PeerID, s.Peers, s.JournalLen, s.SpamRejected, s.PendingN,
		s.AwaitingN, s.Conflicts)
}

// Close — остановка узла.
func (nd *Node) Close() error { return nd.n.Close() }

// ---------------------------------------------------------------------------
// Служебное для тестов моста (unit-тесты Go; на устройстве подписывает
// Keystore — эти функции только для тестов без Keystore)
// ---------------------------------------------------------------------------

// SoftKeyPair — мягкая ключевая пара для тестов (pub + DER приватного).
type SoftKeyPair struct {
	pub     []byte
	privDER []byte
}

func (p *SoftKeyPair) Pub() []byte     { return p.pub }
func (p *SoftKeyPair) PrivDER() []byte { return p.privDER }

// SoftKey — мягкая ключевая пара для тестов (на устройстве подписывает
// Keystore; это — только тесты моста без Keystore).
func SoftKey() (*SoftKeyPair, error) {
	k, err := crypto.GenerateKey()
	if err != nil {
		return nil, err
	}
	return &SoftKeyPair{crypto.MarshalPubKey(&k.PublicKey), marshalPriv(k)}, nil
}

// SignWithKey — подпись мягким ключом (DER), тесты.
func SignWithKey(privDER, hash []byte) ([]byte, error) {
	k, err := x509.ParseECPrivateKey(privDER)
	if err != nil {
		return nil, err
	}
	return crypto.SignDER(k, hash)
}

func marshalPriv(k *ecdsa.PrivateKey) []byte {
	b, err := x509.MarshalECPrivateKey(k)
	if err != nil {
		return nil
	}
	return b
}
