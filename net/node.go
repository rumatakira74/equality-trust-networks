// Узел сети: сборка хоста, DHT, gossip, синхронизации и bootstrap-каскада
// поверх core/ledger. Сеть только доставляет и запрашивает; валидация и
// применение — в ядре (этап 3).
package net

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/rumatakira74/equality-trust-networks/cons"

	"github.com/ipfs/go-cid"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/event"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/multiformats/go-multihash"

	dht "github.com/libp2p/go-libp2p-kad-dht"

	"github.com/rumatakira74/equality-trust-networks/core/crypto"
	"github.com/rumatakira74/equality-trust-networks/core/ledger"
	"github.com/rumatakira74/equality-trust-networks/core/smt"
	"github.com/rumatakira74/equality-trust-networks/core/store"
	"github.com/rumatakira74/equality-trust-networks/core/wire"
)

// maxPending — буфер внеочередных транзакций (спам: не переполнять очередь).
const maxPending = 4096

// Node — узел сети (полный или лёгкий, 12.2).
type Node struct {
	Cfg       *NodeConfig
	networkID []byte
	privKey   libp2pPrivKey
	pow       wire.PowParams

	Host    host.Host
	DHT     *dht.IpfsDHT
	PS      *pubsub.PubSub
	TxTopic *pubsub.Topic
	CpTopic *pubsub.Topic
	txSub   *pubsub.Subscription

	// Ledger — ядро реестра (полный узел; у лёгкого — nil до перехода).
	Ledger *ledger.Ledger
	Store  *store.Store
	Clock  *RealClock

	// Light — состояние лёгкого узла (12.5).
	Light *LightState

	Corroboration *Corroboration
	// Checkpoints — приём чекпоинтов (этап 5: проверка подписи).
	Checkpoints CheckpointHandler

	// Консенсус этапа 5 (голосование + чекпоинты).
	Snowball     *cons.Snowball
	Chain        []*wire.Checkpoint          // финализированные чекпоинты
	CpProposals  map[string]*wire.Checkpoint // hash -> предложение
	CpCollectors map[string]*cons.Collector  // hash -> сбор подписей
	Signer       *cons.Signer                // подписант (кошелёк узла)
	ConsParams   cons.Params
	consClock    *cons.Clock
	journalAtCP  int
	// Snapshots — снапшоты из чекпоинтов (интерфейс; этап 5).
	Snapshots SnapshotSource

	mdnsSvc     interface{ Close() error }
	mdnsNotifee *mdnsNotifee

	mu         sync.RWMutex
	journal    []*wire.Transaction                     // порядок вставки (INV-3)
	chainIndex map[string]map[uint64]*wire.Transaction // wallet -> seq -> tx
	headers    []checkpointHeaderMsg
	pending    map[string][]*wire.Transaction // wallet -> внеочередные
	pendingN   int
	seen       map[string]bool              // txid (дедуп)
	awaiting   map[string]*wire.Transaction // офлайн-канал: ожидает публикации
	published  map[string]bool              // опубликованные из awaiting
	spamCount  int64
	// syncAnchors — позиционные якоря сверки по пирам (№41): позиция в
	// нумерации журнала ПИРА + txid на границе; инкрементальное
	// продолжение без квадратичного трафика при росте журнала
	syncAnchors map[peer.ID]*syncAnchor

	// lastCpRepublish — момент последней перепубликации финализированного
	// чекпоинта (догоняющий механизм цепочки, блокер 8.1)
	lastCpRepublish time.Time

	// lastProposal — своё предложение чекпоинта текущей эпохи
	// (8.1: строится один раз на эпоху, доля переподписывается каждый тик)
	lastProposal      *wire.Checkpoint
	lastProposalEpoch uint64
	lastProposalHash  []byte

	relayReserved int

	// applyMu — сериализация Apply (ядро этапа 3 без мьютексов).
	// RWMutex — read-only запросы UI (net/query.go, этап 7; исключение
	// из правила 3 промпта этапа 7, авторизовано 19.10.2026):
	// Lock — везде Apply, RLock — геттеры состояния.
	applyMu sync.RWMutex

	ctx    context.Context
	cancel context.CancelFunc
	log    func(string, ...any)
}

// LightState — данные лёгкого узла (12.5): своя цепочка, свои входящие,
// заголовки чекпоинтов, доказательства баланса по запросу.
type LightState struct {
	OwnWallet []byte
	OwnChain  []*wire.Transaction
	Incoming  []*wire.Transaction
	Headers   []checkpointHeaderMsg
}

func (n *Node) logf(format string, args ...any) {
	if n.log != nil {
		n.log(fmt.Sprintf(format, args...))
	}
}

func (n *Node) wallet(addr []byte) *ledger.Wallet {
	if n.Ledger == nil {
		return nil
	}
	return n.Ledger.State.Wallet(addr)
}

// NewNode — создание узла (без запуска сети).
func NewNode(cfg *NodeConfig) (*Node, error) {
	if cfg.Genesis == nil || cfg.Genesis.Manifest == nil || cfg.Genesis.Genesis == nil {
		return nil, errors.New("net: неполный genesis-пакет")
	}
	cfg.defaults()
	if cfg.NodeKey == nil {
		k, err := crypto.GenerateKey()
		if err != nil {
			return nil, err
		}
		cfg.NodeKey = k
	}
	priv, err := nodeKeyToLibp2p(cfg.NodeKey)
	if err != nil {
		return nil, fmt.Errorf("net: ключ узла: %w", err)
	}
	n := &Node{
		Cfg:           cfg,
		networkID:     append([]byte(nil), cfg.networkID()...),
		privKey:       priv,
		pow:           *cfg.Genesis.Manifest.Profile.Pow,
		chainIndex:    make(map[string]map[uint64]*wire.Transaction),
		pending:       make(map[string][]*wire.Transaction),
		seen:          make(map[string]bool),
		awaiting:      make(map[string]*wire.Transaction),
		published:     make(map[string]bool),
		syncAnchors:   make(map[peer.ID]*syncAnchor),
		Corroboration: NewCorroboration(cfg.CorroborationN),
		Snapshots:     NoSnapshots{},
		Snowball:      cons.NewSnowball(cons.DefaultParams()),
		CpProposals:   make(map[string]*wire.Checkpoint),
		CpCollectors:  make(map[string]*cons.Collector),
		ConsParams:    cons.FromProfile(cfg.Genesis.Manifest.Profile),
		log:           cfg.Log,
	}
	if cfg.ConsensusEpochSeconds > 0 {
		n.ConsParams.EpochSeconds = cfg.ConsensusEpochSeconds
		n.ConsParams.MaxTimeShift = cfg.ConsensusEpochSeconds / 4
	}
	if cfg.WalletKey != nil && len(cfg.WalletAddr) == 32 {
		n.Signer = &cons.Signer{Wallet: cfg.WalletAddr, Priv: cfg.WalletKey}
	}
	if n.log == nil {
		n.log = func(string, ...any) {}
	}
	if cfg.DBPath != "" {
		st, err := store.Open(cfg.DBPath)
		if err != nil {
			return nil, fmt.Errorf("net: store: %w", err)
		}
		n.Store = st
	}
	if cfg.Level == LevelFull {
		if err := n.initLedger(); err != nil {
			n.Close()
			return nil, err
		}
	} else {
		n.Light = &LightState{OwnWallet: append([]byte(nil), cfg.OwnWallet...)}
	}
	// оценка несвязанности eclipse-пиров по подсетям (№35, этап 6);
	// классификатор связывается с реальным узлом (Host появится в Start)
	if cfg.LinkGroups {
		n.Corroboration = NewCorroborationLinked(cfg.CorroborationN, n.peerLink)
	}
	return n, nil
}

// initLedger — ядро из genesis + replay журнала из store (INV-3).
func (n *Node) initLedger() error {
	gp := n.Cfg.Genesis
	clock := NewClock(gp.Genesis.GenesisTime, gp.Manifest.Profile.EpochSeconds)
	n.Clock = clock
	// Окно челленджа выпуска (M7, 7.6) проверяется при apply: долг этапа 7
	// закрыт (раньше SkipMintChallengeWindow=true — сетевой узёл не проверял
	// окно). Replay пинирует часы на timestamp_claim (cons.Replay), поэтому
	// пересчёт исторических выпусков проходит по моменту публикации.
	l, err := ledger.New(gp, clock, ledger.Options{})
	if err != nil {
		return fmt.Errorf("net: genesis: %w", err)
	}
	l.TrustedRoots = n.Cfg.TrustedRoots
	n.Ledger = l
	// replay журнала (INV-3, cons.Replay): часы пинируются на
	// timestamp_claim (C9 — на момент публикации); конфликтные ветки
	// (обе в журнале хранения, INV-1c) воспроизводят выбор финализации
	if n.Store != nil {
		txs, err := n.Store.LoadTxs()
		if err != nil {
			return fmt.Errorf("net: replay: %w", err)
		}
		if err := cons.Replay(l, txs, cons.NewFinalityView(nil)); err != nil {
			return fmt.Errorf("net: replay: %w", err)
		}
		for _, tx := range l.AcceptedTxs() {
			n.indexTx(tx)
		}
		// восстановление ряда предложения чекпоинтов (адаптивная
		// комиссия, 10.2, этап 10): та же цепочка пересчётов ставки
		epochs, supplies, err := n.Store.LoadCheckpointSupply()
		if err != nil {
			return fmt.Errorf("net: fee supply: %w", err)
		}
		for i := range epochs {
			l.OnCheckpointSupply(epochs[i], supplies[i])
		}
	}
	return nil
}

// Start — запуск узла: хост, фильтр network_id, DHT, relay, протоколы,
// gossip, mDNS, bootstrap, DHT-анонс полного узла.
func (n *Node) Start(ctx context.Context) error {
	n.ctx, n.cancel = context.WithCancel(ctx)
	h, err := n.newNodeHost()
	if err != nil {
		return err
	}
	n.Host = h
	if err := n.watchNetworkFilter(n.ctx); err != nil {
		n.Close()
		return err
	}
	// протоколы узла (идентификация включает network_id)
	h.SetStreamHandler(n.syncProtocol(), n.serveSync)
	h.SetStreamHandler(n.pxProtocol(), n.servePX)
	h.SetStreamHandler(n.voteProtocol(), n.serveVote)

	// консенсусное время (9.2): кламп по последнему чекпоинту
	n.consClock = cons.NewClock(n.Cfg.Genesis.Genesis.GenesisTime,
		n.ConsParams.EpochSeconds, n.ConsParams.MaxTimeShift)

	// DHT (4.2): Kademlia; полный — server, лёгкий — client (12.2);
	// ключевое пространство разделено network_id (префикс протокола).
	mode := dht.Mode(dht.ModeClient)
	if n.Cfg.Level == LevelFull {
		mode = dht.Mode(dht.ModeServer)
	}
	kdht, err := dht.New(h,
		mode,
		dht.ProtocolPrefix(protocolPrefix(n.networkID)),
	)
	if err != nil {
		n.Close()
		return err
	}
	n.DHT = kdht

	// CGNAT (4.7): резерв слота на relay-узлах (circuit relay v2).
	if len(n.Cfg.RelayAddrs) > 0 {
		if err := n.reserveRelays(); err != nil {
			n.logf("relay: %v", err)
		}
	}

	// Gossip (4.3): полный узел ретранслирует (12.2); лёгкий участвует в
	// доставке своих данных, но не хранит реестр.
	if err := n.startGossip(n.ctx); err != nil {
		n.Close()
		return err
	}

	// mDNS (каскад, шаг 4).
	if err := n.startMDNS(); err != nil {
		n.logf("mdns: %v", err)
	}

	// Bootstrap-каскад (4.5).
	if err := n.bootstrap(n.ctx); err != nil {
		n.Close()
		return err
	}
	if err := kdht.Bootstrap(n.ctx); err != nil {
		n.logf("dht bootstrap: %v", err)
	}

	// Полный узел анонсирует сервис синхронизации в DHT (4.2: провайдеры
	// записей сети не видны другой сети — ключ включает network_id).
	if n.Cfg.Level == LevelFull {
		go n.provideSyncService()
		// Первичная синхронизация + периодическая сверка из genesis +
		// журнала (INV-3). Синхронизация со всех подключённых пиров —
		// источник corroboration (eclipse, 12.3).
		go n.syncLoop()
	}
	// Лёгкий узел: периодический опрос своих данных (12.5).
	if n.Cfg.Level == LevelLight {
		go n.lightPollLoop()
	}
	// Консенсус этапа 5: голосование по конфликтам + чекпоинты (полный
	// узел; лёгкий получает заголовки и доказательства из чекпоинтов)
	go n.voteLoop(n.ctx)
	if n.Cfg.Level == LevelFull {
		go n.cpLoop(n.ctx)
	}
	// Периодический peer exchange (каскад, шаг 7).
	go n.pxLoop()
	return nil
}

// syncLoop — первичная синхронизация после bootstrap, затем периодическая
// сверка (узел офлайн -> подключается и догоняет, 4.11).
//
// Правка этапа 8 (долг этапа 7): первый проход по таймеру (500 мс) мог
// прийти до завершения identify пира — NewStream по протоколу sync падал,
// SyncAllPeers глотал ошибку (continue), и журнал не применялся до
// следующего тика (30 с) или принудительной сверки из приложения. Теперь
// сверка выполняется и по событию завершения identify каждого пира.
func (n *Node) syncLoop() {
	if sub, err := n.Host.EventBus().Subscribe(
		[]interface{}{new(event.EvtPeerIdentificationCompleted)}); err == nil {
		go func() {
			defer sub.Close()
			for {
				select {
				case <-n.ctx.Done():
					return
				case ev, ok := <-sub.Out():
					if !ok {
						return
					}
					if e, ok := ev.(event.EvtPeerIdentificationCompleted); ok {
						// чужие сети (agent != наш) отключаются
						// watchNetworkFilter — сверка с ними упадёт и просто попадёт в журнал
						go n.syncPeerLog(e.Peer)
					}
				}
			}
		}()
	}
	// дать bootstrap-подключениям устояться
	select {
	case <-n.ctx.Done():
		return
	case <-time.After(500 * time.Millisecond):
	}
	n.SyncAllPeers()
	t := pollTicker(30 * time.Second)
	defer t.Stop()
	for {
		select {
		case <-n.ctx.Done():
			return
		case <-t.C:
			n.SyncAllPeers()
		}
	}
}

// syncPeerLog — сверка с одним пиром с журналированием результата
// (молчаливое проглатывание ошибок — корень долга этапа 7).
func (n *Node) syncPeerLog(p peer.ID) {
	k, err := n.SyncFromPeer(n.ctx, p)
	if err != nil {
		n.logf("sync %s: %v", p.ShortString(), err)
		return
	}
	if k > 0 {
		n.logf("sync %s: применено транзакций: %d", p.ShortString(), k)
	}
}

// SyncAllPeers — сверка журнала со всеми подключёнными пирами.
func (n *Node) SyncAllPeers() int {
	applied := 0
	for _, p := range n.connectedPeers() {
		k, err := n.SyncFromPeer(n.ctx, p)
		if err != nil {
			continue
		}
		applied += k
	}
	return applied
}

// Close — остановка узла.
func (n *Node) Close() error {
	if n.cancel != nil {
		n.cancel()
	}
	if n.mdnsSvc != nil {
		_ = n.mdnsSvc.Close()
	}
	if n.DHT != nil {
		_ = n.DHT.Close()
	}
	if n.Host != nil {
		_ = n.Host.Close()
	}
	if n.Store != nil {
		_ = n.Store.Close()
	}
	return nil
}

// ---------------------------------------------------------------------------
// DHT (4.2)
// ---------------------------------------------------------------------------

// syncServiceKey — ключ DHT-записи сервиса синхронизации; включает
// network_id: провайдеры записей сети не видны другой сети.
func syncServiceKey(networkID []byte) (cid.Cid, error) {
	h := sha256.Sum256(append([]byte("tnp-sync/"), networkID...))
	mh, err := multihash.Encode(h[:], multihash.SHA2_256)
	if err != nil {
		return cid.Cid{}, err
	}
	return cid.NewCidV1(cid.Raw, mh), nil
}

func (n *Node) provideSyncService() {
	key, err := syncServiceKey(n.networkID)
	if err != nil {
		return
	}
	for {
		if n.ctx.Err() != nil {
			return
		}
		if err := n.DHT.Provide(n.ctx, key, true); err == nil {
			return
		}
		select {
		case <-n.ctx.Done():
			return
		case <-time.After(5 * time.Second):
		}
	}
}

// FindSyncProviders — поиск полных узлов через DHT (лёгкий узел; ключ
// namespace-ирован network_id — чужая сеть этих провайдеров не видит).
func (n *Node) FindSyncProviders(ctx context.Context) ([]peer.AddrInfo, error) {
	key, err := syncServiceKey(n.networkID)
	if err != nil {
		return nil, err
	}
	var out []peer.AddrInfo
	for p := range n.DHT.FindProvidersAsync(ctx, key, 8) {
		out = append(out, p)
	}
	return out, nil
}

// ---------------------------------------------------------------------------
// Приём и применение транзакций
// ---------------------------------------------------------------------------

// indexTx — индексы принятой транзакции (журнал INV-3, цепочка кошелька).
func (n *Node) indexTx(tx *wire.Transaction) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.journal = append(n.journal, tx)
	// txid журнала — в дедуп gossip (блокер 8.1, авторизовано владельцем):
	// после рестарта seen пуст, а журнал наполнен replay/sync — повторная
	// доставка известной транзакции не должна проходить cheapCheck
	// (Argon2id для register/mint) ради отказа E_DUP
	id := tx.TxID()
	n.seen[string(id[:])] = true
	chain, ok := n.chainIndex[string(tx.Wallet)]
	if !ok {
		chain = make(map[uint64]*wire.Transaction)
		n.chainIndex[string(tx.Wallet)] = chain
	}
	chain[tx.Seq] = tx
}

func (n *Node) journalSnapshot() []*wire.Transaction {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return append([]*wire.Transaction(nil), n.journal...)
}

// countSpam — учёт отклонённой ретрансляции (спам-фильтр, 4.3).
func (n *Node) countSpam() {
	n.mu.Lock()
	n.spamCount++
	n.mu.Unlock()
}

// handleTxFromPeer — дедупликация по txid, corroboration (eclipse, 12.3),
// дешёвая проверка, применение в фоне.
func (n *Node) handleTxFromPeer(tx *wire.Transaction, from peer.ID) {
	txid := tx.TxID()
	n.Corroboration.Add(txid[:], from)
	n.mu.Lock()
	dup := n.seen[string(txid[:])]
	if !dup {
		n.seen[string(txid[:])] = true
	}
	n.mu.Unlock()
	if dup {
		return
	}
	n.logf("handleTx: tx %s type=%s seq=%d от %s", hex12(tx), tx.Type, tx.Seq, from.ShortString())
	if err := n.cheapCheck(tx); err != nil {
		n.countSpam()
		n.logf("спам от %s: %v", from.ShortString(), err)
		return
	}
	n.applyLive(tx)
}

// applyLive — применение транзакции (полный узел) или сохранение своих
// данных (лёгкий, 12.5). Полный apply — в фоне относительно сети.
func (n *Node) applyLive(tx *wire.Transaction) {
	if n.Cfg.Level == LevelLight {
		n.lightStore(tx)
		return
	}
	if n.Ledger == nil {
		return
	}
	n.applyMu.Lock()
	res := n.applyWithClock(tx, n.Clock)
	// postApply читает реестр (Snowball.Observe, isFutureSeq) — под тем же
	// локом, как в sync-цикле (data race — net/query.go, этап 7)
	n.postApply(tx, res)
	n.applyMu.Unlock()
	// повтор буфера внеочередных — строго вне applyMu (retryPending
	// берёт RLock; из-под Lock — self-deadlock)
	n.retryPending(tx.Wallet)
}

// ImportJournal — восстановление журнала из внешнего источника (БД):
// применяет транзакции с пинированием часов на timestamp_claim каждой —
// та же семантика, что сетевая сверка (SyncJournalFrom, net.ReplayClock).
// Введён как обход блокера 8.1-2 (авторизовано владельцем), когда replay
// из БД падал на vouch-ах прошлых эпох; с закрытием №40 (этап 9,
// пинирование Epoch/Period в cons.Replay) обычный рестарт из БД снова
// работает — ImportJournal остаётся путём восстановления (не удалять):
// импорт до старта сети, без гонки с gossip/сверками.
// Возвращает число применённых (accepted) транзакций.
func (n *Node) ImportJournal(txs []*wire.Transaction) int {
	rc := NewReplayClock(n.Cfg.Genesis.Genesis.GenesisTime,
		n.Cfg.Genesis.Manifest.Profile.EpochSeconds)
	applied := 0
	for _, tx := range txs {
		n.applyMu.Lock()
		rc.Pin(tx.TimestampClaim)
		res := n.applyWithClock(tx, rc)
		n.postApply(tx, res)
		n.applyMu.Unlock()
		n.retryPending(tx.Wallet)
		if res.Outcome == ledger.Accepted {
			applied++
		}
	}
	return applied
}

// applyWithClock — Apply с подменой часов (вызывающий держит applyMu).
// Используется при replay журнала: C9 проверяется на момент публикации.
func (n *Node) applyWithClock(tx *wire.Transaction, clock ledger.EpochSource) ledger.Result {
	old := n.Ledger.Clock
	n.Ledger.Clock = clock
	res := n.Ledger.Apply(tx)
	n.Ledger.Clock = old
	return res
}

// postApply — обработка результата применения.
func (n *Node) postApply(tx *wire.Transaction, res ledger.Result) {
	switch res.Outcome {
	case ledger.Accepted:
		n.commitTx(tx)
	case ledger.Conflict:
		// эквивокация: доставляются обе ветки, выбор за финализацией (9.1);
		// ветка остаётся в журнале хранения (spec/04 INV-1c) и в store
		n.logf("конфликтная ветка %s (seq=%d) доставлена", hex.EncodeToString(res.TxID[:])[:12], tx.Seq)
		if w := n.wallet(tx.Wallet); w != nil {
			n.Snowball.Observe(cons.ConflictKey(tx), append([]byte(nil), w.TipTxID...), res.TxID[:])
		}
		if n.Store != nil {
			_ = n.Store.SaveTx(tx, uint64(time.Now().Unix()))
		}
	case ledger.Rejected:
		if res.Code == "E_SEQ" && n.isFutureSeq(tx) {
			n.bufferPending(tx)
			return
		}
		n.logf("отклонена %s: %s", hex.EncodeToString(res.TxID[:])[:12], res.Code)
	}
}

// isFutureSeq — транзакция из будущего (пришла раньше предыдущей).
func (n *Node) isFutureSeq(tx *wire.Transaction) bool {
	w := n.wallet(tx.Wallet)
	return w != nil && tx.Seq > w.Seq+1
}

// bufferPending — буфер внеочередных (ограничен; спам не переполняет очередь).
func (n *Node) bufferPending(tx *wire.Transaction) {
	n.mu.Lock()
	defer n.mu.Unlock()
	if n.pendingN >= maxPending {
		// выбрасываем старейшую запись самого большого буфера
		var maxK string
		maxLen := 0
		for k, list := range n.pending {
			if len(list) > maxLen {
				maxLen, maxK = len(list), k
			}
		}
		if old := n.pending[maxK]; len(old) > 0 {
			n.pending[maxK] = old[1:]
			n.pendingN--
		}
	}
	n.pending[string(tx.Wallet)] = append(n.pending[string(tx.Wallet)], tx)
	n.pendingN++
}

// retryPending — повтор внеочередных после применения предыдущей.
func (n *Node) retryPending(wallet []byte) {
	n.mu.Lock()
	list := n.pending[string(wallet)]
	n.pending[string(wallet)] = nil
	n.pendingN -= len(list)
	n.mu.Unlock()
	// отбор готовых — чтение реестра под applyMu.RLock (net/query.go,
	// этап 7); applyLive берёт Lock — вне RLock
	n.applyMu.RLock()
	ready := make([]*wire.Transaction, 0, len(list))
	later := make([]*wire.Transaction, 0, len(list))
	for _, tx := range list {
		if n.isFutureSeq(tx) {
			later = append(later, tx)
		} else {
			ready = append(ready, tx)
		}
	}
	n.applyMu.RUnlock()
	for _, tx := range later {
		n.bufferPending(tx)
	}
	for _, tx := range ready {
		n.applyLive(tx)
	}
}

// commitTx — фиксация принятой транзакции: индексы, store.
// retryPending вызывающий делает САМ и только ВНЕ applyMu
// (retryPending берёт applyMu.RLock; из-под Lock — self-deadlock).
func (n *Node) commitTx(tx *wire.Transaction) {
	n.indexTx(tx)
	if n.Store != nil {
		_ = n.Store.SaveTx(tx, uint64(time.Now().Unix()))
		if tx.Type == wire.Register {
			if reg, ok := tx.Payload.(*wire.RegisterPayload); ok {
				fp := crypto.Fingerprint(reg.AttestedPublicKey)
				_ = n.Store.SaveFingerprint(fp, tx.Wallet)
			}
		}
	}
}

// SubmitTx — локальная публикация: полный apply, затем gossip. Конфликтные
// ветки публикуются (эквивокация доставляется, 4.3); отклонённые — нет.
func (n *Node) SubmitTx(ctx context.Context, tx *wire.Transaction) (ledger.Result, error) {
	res := ledger.Result{Outcome: ledger.Rejected, Code: "E_NO_LEDGER"}
	if n.Cfg.Level == LevelFull && n.Ledger != nil {
		n.applyMu.Lock()
		res = n.Ledger.Apply(tx)
		n.applyMu.Unlock()
		switch res.Outcome {
		case ledger.Accepted:
			n.commitTx(tx)
			// повтор буфера внеочередных — вне applyMu (см. commitTx)
			n.retryPending(tx.Wallet)
		case ledger.Conflict:
			// доставляем обе ветки (выбор — финализация, этап 5)
		case ledger.Rejected:
			return res, nil
		}
	} else if n.Cfg.Level == LevelLight {
		if err := n.lightCheckOwn(tx); err != nil {
			return res, nil
		}
		n.lightStore(tx)
		res = ledger.Result{Outcome: ledger.Accepted, Code: "OK", TxID: tx.TxID()}
	}
	if err := n.PublishTx(ctx, tx); err != nil {
		return res, err
	}
	return res, nil
}

// lightStore — лёгкий узел: своя цепочка, свои входящие (12.5).
func (n *Node) lightStore(tx *wire.Transaction) {
	if n.Light == nil {
		return
	}
	own := string(tx.Wallet) == string(n.Light.OwnWallet)
	incoming := false
	if !own {
		if pl, ok := tx.Payload.(*wire.TransferPayload); ok && pl.Send != nil &&
			string(pl.Send.Recipient) == string(n.Light.OwnWallet) {
			incoming = true
		}
	}
	if !own && !incoming {
		return
	}
	n.mu.Lock()
	defer n.mu.Unlock()
	if own {
		// дедуп по seq
		for _, t := range n.Light.OwnChain {
			if t.Seq == tx.Seq {
				return
			}
		}
		n.Light.OwnChain = append(n.Light.OwnChain, tx)
		if n.Store != nil {
			_ = n.Store.SaveTx(tx, uint64(time.Now().Unix()))
		}
	} else {
		id := tx.TxID()
		for _, t := range n.Light.Incoming {
			if t.TxID() == id {
				return
			}
		}
		n.Light.Incoming = append(n.Light.Incoming, tx)
	}
}

// lightCheckOwn — проверка seq/prev своей цепочки при публикации (12.4).
func (n *Node) lightCheckOwn(tx *wire.Transaction) error {
	if n.Light == nil || string(tx.Wallet) != string(n.Light.OwnWallet) {
		return nil
	}
	n.mu.RLock()
	chain := n.Light.OwnChain
	n.mu.RUnlock()
	var last *wire.Transaction
	for _, t := range chain {
		if last == nil || t.Seq > last.Seq {
			last = t
		}
	}
	if last == nil {
		if tx.Seq != 1 {
			return errors.New("net: первая транзакция обязана иметь seq=1")
		}
		return nil
	}
	if tx.Seq != last.Seq+1 {
		return fmt.Errorf("net: seq=%d, ожидается %d", tx.Seq, last.Seq+1)
	}
	lastID := last.TxID()
	if string(tx.Prev) != string(lastID[:]) {
		return errors.New("net: prev не сходится с tip своей цепочки")
	}
	return nil
}

// ---------------------------------------------------------------------------
// Лёгкий узел: опрос (12.5)
// ---------------------------------------------------------------------------

// lightPollLoop — периодический опрос входящих, заголовков и доказательств.
func (n *Node) lightPollLoop() {
	t := pollTicker(n.Cfg.PollInterval)
	defer t.Stop()
	for {
		select {
		case <-n.ctx.Done():
			return
		case <-t.C:
			_ = n.LightSync(n.ctx)
		}
	}
}

// LightSync — опрос подключённых пиров: своя цепочка, входящие, заголовки,
// Merkle-доказательство баланса (12.5).
func (n *Node) LightSync(ctx context.Context) error {
	if n.Light == nil {
		return errors.New("net: не лёгкий узел")
	}
	for _, p := range n.connectedPeers() {
		seq, _, err := n.GetTip(ctx, p, n.Light.OwnWallet)
		if err != nil {
			continue
		}
		n.mu.RLock()
		have := uint64(len(n.Light.OwnChain))
		n.mu.RUnlock()
		if seq > have {
			txs, err := n.GetWalletRange(ctx, p, n.Light.OwnWallet, have+1, seq)
			if err == nil {
				for _, tx := range txs {
					n.lightStore(tx)
				}
			}
		}
		if incoming, err := n.GetIncoming(ctx, p, n.Light.OwnWallet); err == nil {
			for _, tx := range incoming {
				n.lightStore(tx)
			}
		}
		if hs, err := n.GetHeaders(ctx, p, 0); err == nil && len(hs) > 0 {
			n.mu.Lock()
			n.Light.Headers = hs
			// заголовки (с корнем состояния) — и на уровне узла:
			// VerifyProofAgainstCheckpoint проверяет по последнему
			if len(hs) > len(n.headers) {
				n.headers = hs
			}
			n.mu.Unlock()
		}
	}
	return nil
}

// ---------------------------------------------------------------------------
// Офлайн-канал (12.4, 4.10)
// ---------------------------------------------------------------------------

// OnOfflineTx — транзакция, принятая по офлайн-каналу (Bluetooth): кладётся
// со статусом «ожидает публикации» (12.4). «Получено по Bluetooth» ≠ оплата.
func (n *Node) OnOfflineTx(tx *wire.Transaction) {
	id := tx.TxID()
	n.mu.Lock()
	if _, ok := n.awaiting[string(id[:])]; !ok {
		n.awaiting[string(id[:])] = tx
	}
	n.mu.Unlock()
}

// PublishAwaiting — публикация ожидающих транзакций (при появлении сети):
// полный apply проверяет seq/prev (12.4: быстрая находка эквивокации).
// Отклонённые навсегда (не E_SEQ-пробел) снимаются с публикации; пробел
// (транзакция из будущего) — остаётся ждать предыдущей.
func (n *Node) PublishAwaiting(ctx context.Context) {
	n.mu.Lock()
	txs := make([]*wire.Transaction, 0, len(n.awaiting))
	for _, tx := range n.awaiting {
		txs = append(txs, tx)
	}
	n.mu.Unlock()
	for _, tx := range txs {
		id := tx.TxID()
		res, err := n.SubmitTx(ctx, tx)
		if err != nil {
			continue // сеть недоступна — остаёмся в ожидании
		}
		n.mu.Lock()
		switch {
		case res.Outcome == ledger.Accepted || res.Outcome == ledger.Conflict:
			// доставлено (конфликт — эквивокация обнаружена; финализация —
			// этап 5, статус «не оплата» держится до подтверждения)
			delete(n.awaiting, string(id[:]))
			n.published[string(id[:])] = true
		case res.Code == "E_SEQ" || res.Code == "E_DUP":
			// пробел в цепочке или уже известна — ждём/считаем опубликованной
			if res.Code == "E_DUP" {
				delete(n.awaiting, string(id[:]))
				n.published[string(id[:])] = true
			}
		default:
			// невалидная — снята с ожидания, не опубликована
			delete(n.awaiting, string(id[:]))
		}
		n.mu.Unlock()
	}
}

// OfflineStatus — статус транзакции офлайн-канала (финализация — заглушка
// до этапа 5: «финализировано» = подтверждено N пирами, net/eclipse).
func (n *Node) OfflineStatus(txid [32]byte) OfflineStatus {
	n.mu.RLock()
	_, awaiting := n.awaiting[string(txid[:])]
	n.mu.RUnlock()
	if awaiting {
		return OfflineAwaiting
	}
	if !n.Corroboration.Confirmed(txid[:]) {
		return OfflinePublished
	}
	return OfflineFinalized
}

// ---------------------------------------------------------------------------
// Статистика и статус
// ---------------------------------------------------------------------------

// Stats — сводка узла (для тестов, cmd и отчёта).
type Stats struct {
	Level        string `json:"level"`
	PeerID       string `json:"peer_id"`
	Peers        int    `json:"peers"`
	JournalLen   int    `json:"journal_len"`
	SpamRejected int64  `json:"spam_rejected"`
	PendingN     int    `json:"pending"`
	AwaitingN    int    `json:"awaiting"`
	Conflicts    int    `json:"conflicts"`
}

func (n *Node) Stats() Stats {
	n.mu.RLock()
	defer n.mu.RUnlock()
	st := Stats{
		Level:        n.Cfg.Level.String(),
		PeerID:       "",
		Peers:        len(n.connectedPeers()),
		JournalLen:   len(n.journal),
		SpamRejected: n.spamCount,
		PendingN:     n.pendingN,
		AwaitingN:    len(n.awaiting),
	}
	if n.Host != nil {
		st.PeerID = n.Host.ID().String()
	}
	if n.Ledger != nil {
		st.Conflicts = len(n.Ledger.Conflicts)
	}
	return st
}

// smtTreeOf — SMT состояния реестра (лист = StateLeaf, D7; core/smt).
func smtTreeOf(l *ledger.Ledger) *smt.Tree {
	tree := smt.New()
	for _, w := range l.State.Wallets {
		leaf := stateLeafOf(w)
		var idx [32]byte
		copy(idx[:], w.Address)
		lh := sha256.Sum256(leaf.Encode())
		tree.SetLeaf(idx, lh)
	}
	return tree
}

// StateRoot — корень SMT состояния (полный узел; для сверки узлов).
// Чтение реестра — под applyMu.RLock: статус-файл/тесты/мост зовут без
// applyMu, фоновый applyLive пишет параллельно (race-аудит этапа 9).
func (n *Node) StateRoot() ([]byte, error) {
	if n.Ledger == nil {
		return nil, errors.New("net: нет ядра")
	}
	n.applyMu.RLock()
	tree := smtTreeOf(n.Ledger)
	n.applyMu.RUnlock()
	r := tree.Root()
	return r[:], nil
}

// ConflictsCount — число конфликтных веток (доставлены обе; этап 5 решит).
func (n *Node) ConflictsCount() int {
	if n.Ledger == nil {
		return 0
	}
	return len(n.Ledger.Conflicts)
}

// peerExchangeLoop — периодический peer exchange (шаг 7 каскада).
func (n *Node) pxLoop() {
	t := pollTicker(60 * time.Second)
	defer t.Stop()
	for {
		select {
		case <-n.ctx.Done():
			return
		case <-t.C:
			for _, p := range n.connectedPeers() {
				if ps, err := n.pxRequest(n.ctx, p); err == nil {
					for _, dp := range ps {
						if len(dp.info.Addrs) > 0 {
							_ = n.Host.Connect(n.ctx, dp.info)
						}
					}
				}
			}
		}
	}
}
