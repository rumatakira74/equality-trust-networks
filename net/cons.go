// Интеграция консенсуса этапа 5 в узел сети: голосование по конфликтам
// (протокол /tnp/<hex>/vote/1.0.0, Snowball из cons), чекпоинты (тема
// checkpoint/<network_id>: сборка, подпись участником, сбор долей,
// финализация и принятие), перестройка состояния при смене ветки,
// лёгкий узел проверяет Merkle-доказательство по чекпоинту (12.5).
package net

import (
	"context"
	"crypto/ecdsa"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"math/rand"
	"time"

	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"

	"github.com/rumatakira74/equality-trust-networks/cons"
	"github.com/rumatakira74/equality-trust-networks/core/crypto"
	"github.com/rumatakira74/equality-trust-networks/core/ledger"
	"github.com/rumatakira74/equality-trust-networks/core/wire"
)

// Сообщения протокола голосования и чекпоинтов (net/codec.go).
const (
	msgVoteQuery = 20 // запрос предпочтения по конфликту
	msgVoteResp  = 21 // ответ: ветка + вес голоса
	msgCpShare   = 22 // чекпоинт + доля подписи (тема checkpoint)
)

// votePreference — предпочтение узла по конфликту (для ответа пирам):
// локальное состояние Snowball; вес — кошелёк узла (возраст+аттестация).
func (n *Node) votePreference(key string) ([]byte, uint64, bool) {
	st := n.Snowball.State(key)
	if st == nil {
		return nil, 0, false
	}
	weight := uint64(1)
	// чтение реестра — под applyMu.RLock (data race — net/query.go, этап 7)
	n.applyMu.RLock()
	if w := n.wallet(n.Cfg.WalletAddr); w != nil && n.Ledger != nil {
		strict := n.Ledger.State.Net.Profile.DevicePolicy == wire.DevicePolicyStrict
		weight = cons.Weight(w, strict, n.Ledger.State.NetTime)
	}
	n.applyMu.RUnlock()
	return st.Pref, weight, true
}

// voteResponseHash — хеш подписываемого ответа голосования (долг этапа 5:
// вес в ответе заявлял отвечающий без подписи). Домен + network_id —
// защита от кросс-сетевого replay (D1).
func voteResponseHash(networkID []byte, key string, pref []byte, weight uint64) []byte {
	h := sha256.New()
	h.Write([]byte("tnp-vote/1"))
	h.Write(networkID)
	h.Write([]byte(key))
	h.Write(pref)
	var w [8]byte
	binary.BigEndian.PutUint64(w[:], weight)
	h.Write(w[:])
	return h.Sum(nil)
}

// serveVote — обработчик протокола голосования. Ответ подписан ключом
// кошелька узла (wallet + подпись над key|pref|weight); узел без кошелька
// отвечает пустым голосом — вес незаявленного голоса не учитывается.
func (n *Node) serveVote(s network.Stream) {
	defer s.Close()
	raw, err := readFrameStream(s)
	if err != nil {
		return
	}
	msgType, d, err := readFrame(raw)
	if err != nil || msgType != msgVoteQuery {
		return
	}
	key, err := d.str()
	if err != nil {
		return
	}
	pref, weight, ok := n.votePreference(key)
	var e enc
	e.str(key)
	if ok && n.Cfg.WalletKey != nil && len(n.Cfg.WalletAddr) == 32 {
		sig, err := crypto.SignDER(n.Cfg.WalletKey, voteResponseHash(n.networkID, key, pref, weight))
		if err == nil {
			e.blob(pref)
			e.u64(weight)
			e.blob(n.Cfg.WalletAddr)
			e.blob(sig)
			_, _ = s.Write(writeFrame(&e, msgVoteResp))
			return
		}
		n.logf("голосование: подпись ответа: %v", err)
	}
	// без кошелька или без подписи — голос не учитывается опрашивающим
	e.blob(nil)
	e.u64(0)
	e.blob(nil)
	e.blob(nil)
	_, _ = s.Write(writeFrame(&e, msgVoteResp))
}

// netPoller — опрос пиров по сети (cons.Poller): k случайных подключённых.
type netPoller struct {
	n   *Node
	ctx context.Context
}

func (p *netPoller) Sample(k int, rng *rand.Rand) []cons.PollablePeer {
	peers := p.n.connectedPeers()
	out := make([]cons.PollablePeer, 0, k)
	if len(peers) == 0 {
		return out
	}
	for i := 0; i < k; i++ {
		id := peers[rng.Intn(len(peers))]
		out = append(out, &remotePeer{n: p.n, id: id})
	}
	return out
}

// remotePeer — опрашиваемый пир сети.
type remotePeer struct {
	n  *Node
	id peer.ID
}

func (r *remotePeer) VotePreference(key string) ([]byte, uint64, bool) {
	var e enc
	e.str(key)
	ctx, cancel := context.WithTimeout(r.n.ctx, 5*time.Second)
	defer cancel()
	raw, err := r.n.requestOn(ctx, r.id, r.n.voteProtocol(), writeFrame(&e, msgVoteQuery))
	if err != nil {
		return nil, 0, false
	}
	msgType, d, err := readFrame(raw)
	if err != nil || msgType != msgVoteResp {
		return nil, 0, false
	}
	k2, err := d.str()
	if err != nil || k2 != key {
		return nil, 0, false
	}
	pref, err := d.blob()
	if err != nil || len(pref) == 0 {
		return nil, 0, false
	}
	weight, err := d.u64()
	if err != nil || weight == 0 {
		return nil, 0, false
	}
	// подпись ответа (долг этапа 5): wallet + подпись над key|pref|weight;
	// проверка на стороне опрашивающего — ключ кошелька из состояния.
	wallet, err := d.blob()
	if err != nil || len(wallet) != 32 {
		r.n.logf("голосование: ответ без кошелька от %s", r.id.ShortString())
		return nil, 0, false
	}
	sig, err := d.blob()
	if err != nil || len(sig) == 0 {
		r.n.logf("голосование: ответ без подписи от %s", r.id.ShortString())
		return nil, 0, false
	}
	pub, ok := r.n.pubKeyOf(wallet)
	if !ok {
		r.n.logf("голосование: кошелёк подписанта %s неизвестен", hex.EncodeToString(wallet)[:12])
		return nil, 0, false
	}
	if !crypto.VerifyDER(pub, voteResponseHash(r.n.networkID, key, pref, weight), sig) {
		r.n.logf("голосование: подпись ответа от %s не сошлась", r.id.ShortString())
		return nil, 0, false
	}
	// вес пересчитывается локально, если кошелёк известен состоянию:
	// подписанный вес — атрибуция, локальный — защита от завышения
	// (чтение реестра — под applyMu.RLock, net/query.go, этап 7)
	r.n.applyMu.RLock()
	if w := r.n.wallet(wallet); w != nil && r.n.Ledger != nil {
		strict := r.n.Ledger.State.Net.Profile.DevicePolicy == wire.DevicePolicyStrict
		if lw := cons.Weight(w, strict, r.n.Ledger.State.NetTime); lw > 0 {
			weight = lw
		}
	}
	r.n.applyMu.RUnlock()
	return pref, weight, true
}

// voteLoop — периодическое голосование по активным конфликтам (9.1);
// финальность β раундов; смена ветки — перестройка состояния.
func (n *Node) voteLoop(ctx context.Context) {
	t := pollTicker(n.Cfg.VoteInterval)
	defer t.Stop()
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			for _, st := range n.Snowball.States() {
				if st.Final {
					continue
				}
				finalized := n.Snowball.PollRound(st.Key, &netPoller{n: n, ctx: ctx}, rng)
				if !finalized {
					continue
				}
				n.logf("голосование: конфликт %s финализирован (ветка %s)",
					shortKey(st.Key), hex.EncodeToString(st.Pref)[:12])
				if string(st.Pref) != string(n.branchApplied(st.Key)) {
					n.rebuildToFinality()
				}
			}
		}
	}
}

// branchApplied — txid применённой ветки конфликта (tip кошелька).
func (n *Node) branchApplied(key string) []byte {
	wallet := conflictWallet(key)
	// чтение реестра — под applyMu.RLock (net/query.go, этап 7)
	n.applyMu.RLock()
	w := n.wallet(wallet)
	var tip []byte
	if w != nil {
		tip = append([]byte(nil), w.TipTxID...)
	}
	n.applyMu.RUnlock()
	return tip
}

// conflictWallet — адрес кошелька из ключа конфликта (wallet|seq).
func conflictWallet(key string) []byte {
	for i := len(key) - 1; i >= 0; i-- {
		if key[i] == '|' {
			return []byte(key[:i])
		}
	}
	return []byte(key)
}

// rebuildToFinality — перестройка состояния под финализированный выбор:
// genesis + журнал (обе ветки) + finalized_txid_list (cons.Replay).
func (n *Node) rebuildToFinality() {
	if n.Ledger == nil {
		return
	}
	// журнал — под mu; конфликтные ветки — из реестра, под applyMu.RLock
	// (порядок applyMu → mu, как в postApply; net/query.go, этап 7)
	n.applyMu.RLock()
	n.mu.RLock()
	txs := append([]*wire.Transaction(nil), n.journal...)
	txs = append(txs, n.Ledger.ConflictBranches()...)
	n.mu.RUnlock()
	n.applyMu.RUnlock()
	view := n.finalityView()
	l, err := ledger.New(n.Cfg.Genesis, n.Clock, ledger.Options{SkipMintChallengeWindow: true})
	if err != nil {
		n.logf("перестройка: genesis: %v", err)
		return
	}
	l.TrustedRoots = n.Cfg.TrustedRoots
	if err := cons.Replay(l, txs, view); err != nil {
		n.logf("перестройка: replay: %v", err)
		return
	}
	n.applyMu.Lock()
	old := n.Ledger
	n.Ledger = l
	n.applyMu.Unlock()
	_ = old
	// индексы и store пересобираются из нового журнала
	n.mu.Lock()
	n.journal = append([]*wire.Transaction(nil), l.AcceptedTxs()...)
	n.chainIndex = make(map[string]map[uint64]*wire.Transaction)
	n.mu.Unlock()
	for _, tx := range l.AcceptedTxs() {
		n.indexTx(tx)
	}
	n.logf("перестройка: состояние пересчитано под финализацию")
}

// finalityView — финализированные txid (чекпоинты + локальные победители
// голосования по ещё не покрытым конфликтам).
func (n *Node) finalityView() *cons.FinalityView {
	n.mu.RLock()
	chain := append([]*wire.Checkpoint(nil), n.Chain...)
	n.mu.RUnlock()
	view := cons.NewFinalityView(chain)
	for _, st := range n.Snowball.States() {
		if !st.Final {
			continue
		}
		if !view.IsFinalized(st.A) && !view.IsFinalized(st.B) {
			view.MarkFinalized(st.Pref)
		}
	}
	return view
}

// ---------------------------------------------------------------------------
// Чекпоинты (тема checkpoint/<network_id>; сбор подписей N-из-M)
// ---------------------------------------------------------------------------

// cpLoop — закрытие эпох: сборка чекпоинта, подпись участником, публикация;
// сбор долей из темы; при пороге — Verify, принятие, store.
func (n *Node) cpLoop(ctx context.Context) {
	t := pollTicker(time.Second)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			n.cpTick(ctx)
		}
	}
}

func (n *Node) cpTick(ctx context.Context) {
	if n.Ledger == nil {
		return
	}
	n.consClock.OnCheckpoint(n.lastCheckpoint())
	if !n.consClock.EpochDue() {
		return
	}
	if n.hasUnresolvedConflict() {
		return // чекпоинт фиксирует сходимость (9.1)
	}
	prev := n.lastCheckpoint()
	epoch := uint64(1)
	var prevHash []byte
	if prev != nil {
		epoch = prev.Epoch + 1
		h := prev.CheckpointHash()
		prevHash = h[:]
	} else {
		prevHash = make([]byte, 32)
	}
	// состав подписантов и сборка чекпоинта читают реестр — под
	// applyMu.RLock (data race — net/query.go, этап 7)
	n.applyMu.RLock()
	part, thresholdN, err := cons.Participants(
		n.Cfg.Genesis.Manifest.Profile.CheckpointMode, n.Ledger, n.Cfg.Genesis,
		n.ConsParams, epoch, prevHash)
	if err != nil {
		n.applyMu.RUnlock()
		return
	}
	// finalized_txid_list: принятые с прошлого чекпоинта
	n.mu.RLock()
	journal := append([]*wire.Transaction(nil), n.journal...)
	cpPos := n.journalAtCP
	n.mu.RUnlock()
	if cpPos > len(journal) {
		cpPos = 0
	}
	var finalized [][]byte
	for i := cpPos; i < len(journal); i++ {
		id := journal[i].TxID()
		finalized = append(finalized, id[:])
	}
	netTime := n.consClock.NetTime()
	cp, err := cons.Build(n.networkID, epoch, n.Ledger, prev, netTime,
		n.Cfg.Genesis.Genesis.ProtocolVersion, part, thresholdN, finalized)
	n.applyMu.RUnlock()
	if err != nil {
		return
	}
	n.onCpMessage(cp, wire.SignatureShare{}, true)
	// публикация предложения в тему (даже если узел не участник)
	if err := n.publishCp(ctx, cp, wire.SignatureShare{}); err != nil {
		n.logf("чекпоинт: публикация: %v", err)
	}
	// подпись участником
	for _, p := range part {
		if string(p) == string(n.Cfg.WalletAddr) && n.Signer != nil {
			share, err := n.Signer.Sign(cp)
			if err != nil {
				return
			}
			n.onCpMessage(cp, share, true)
			if err := n.publishCp(ctx, cp, share); err != nil {
				n.logf("чекпоинт: публикация доли: %v", err)
			}
			break
		}
	}
	n.collectCpPending(ctx)
}

// publishCp — чекпоинт + доля подписи в тему checkpoint/<network_id>.
func (n *Node) publishCp(ctx context.Context, cp *wire.Checkpoint, share wire.SignatureShare) error {
	if n.CpTopic == nil {
		return errors.New("тема чекпоинтов не подключена")
	}
	var e enc
	e.blob(cp.Encode())
	e.blob(share.Signer)
	e.blob(share.Signature)
	return n.CpTopic.Publish(ctx, writeFrame(&e, msgCpShare))
}

// onCpMessage — предложение/доля чекпоинта: регистрация, сбор, подпись
// участником полученного предложения (контент детерминирован).
//
// Чтения состояния реестра (hasUnresolvedConflict, pubKeyOf через
// Collector.Add) вынесены ЗА n.mu: порядок блокировок всегда
// applyMu → mu (postApply), обратный порядок (mu → applyMu.RLock)
// дал бы AB-BA (data race — net/query.go, этап 7).
func (n *Node) onCpMessage(cp *wire.Checkpoint, share wire.SignatureShare, local bool) {
	if cp == nil {
		return
	}
	h := cp.CheckpointHash()
	// конфликт и подпись участника — до мьютекса (читают реестр)
	signIncoming := false
	if !local && n.Signer != nil && !n.hasUnresolvedConflict() {
		for _, p := range cp.Participants {
			if string(p) == string(n.Cfg.WalletAddr) {
				signIncoming = true
				break
			}
		}
	}
	n.mu.Lock()
	isNew := false
	if _, ok := n.CpProposals[string(h[:])]; !ok {
		// эпоха последнего чекпоинта — под уже взятым локом
		// (lastCPEpock использует RLock — нельзя при живом Lock)
		lastEpoch := uint64(0)
		if len(n.Chain) > 0 {
			lastEpoch = n.Chain[len(n.Chain)-1].Epoch
		}
		if cp.Epoch != lastEpoch+1 {
			n.mu.Unlock()
			return
		}
		n.CpProposals[string(h[:])] = cp
		n.CpCollectors[string(h[:])] = cons.NewCollector(cp, n.pubKeyOf)
		isNew = true
	}
	col := n.CpCollectors[string(h[:])]
	n.mu.Unlock()
	// подпись участником полученного предложения — после снятия мьютекса
	// (только для нового предложения, как до выноса из-под лок)
	if isNew && signIncoming {
		if myShare, err := n.Signer.Sign(cp); err == nil {
			if col != nil {
				col.Add(myShare)
			}
			_ = n.publishCp(n.ctx, cp, myShare)
		}
	}
	if len(share.Signer) > 0 && col != nil {
		col.Add(share)
	}
}

// collectCpPending — финализация сборов с достигнутым порогом.
func (n *Node) collectCpPending(ctx context.Context) {
	n.mu.Lock()
	var ready []*wire.Checkpoint
	for h, col := range n.CpCollectors {
		cp := n.CpProposals[h]
		if cp == nil || cp.ThresholdSig != nil {
			continue
		}
		if sig := col.Ready(); sig != nil {
			cp.ThresholdSig = sig
			ready = append(ready, cp)
		}
	}
	n.mu.Unlock()
	for _, cp := range ready {
		if err := cons.Verify(cp, n.lastCheckpoint(), n.Cfg.Genesis.Genesis.GenesisTime,
			n.ConsParams.EpochSeconds, n.ConsParams.MaxTimeShift, n.pubKeyOf); err != nil {
			n.logf("чекпоинт %d: проверка: %v", cp.Epoch, err)
			continue
		}
		if n.adoptCheckpoint(cp) {
			n.logf("чекпоинт эпохи %d финализирован (%d подписей)", cp.Epoch, len(cp.ThresholdSig.Signatures))
			_ = n.publishCp(ctx, cp, wire.SignatureShare{})
		}
	}
}

// adoptCheckpoint — принять финализированный чекпоинт: цепочка, store,
// заголовки для лёгких узлов; перестройка при противоречии.
func (n *Node) adoptCheckpoint(cp *wire.Checkpoint) bool {
	n.mu.Lock()
	// эпоха последнего чекпоинта — под уже взятым локом (lastCPEpoch
	// берёт RLock — реентерабельность запрещена)
	lastEpoch := uint64(0)
	if len(n.Chain) > 0 {
		lastEpoch = n.Chain[len(n.Chain)-1].Epoch
	}
	if cp.Epoch <= lastEpoch {
		n.mu.Unlock()
		return false
	}
	n.Chain = append(n.Chain, cp)
	n.journalAtCP = len(n.journal)
	// заголовки для лёгких узлов (12.5): эпоха, хеш, корень состояния
	n.headers = append(n.headers, checkpointHeaderMsg{
		Epoch: cp.Epoch, Hash: cpHashOf(cp), StateRoot: cp.StateRoot,
	})
	n.mu.Unlock()
	if n.Store != nil {
		_ = n.Store.SaveCheckpointHeader(cp.Epoch, cpHashOf(cp))
	}
	n.consClock.OnCheckpoint(cp)
	// противоречит ли состояние финализированному выбору — перестроиться
	if n.Ledger != nil && n.needsRebuild() {
		n.rebuildToFinality()
	}
	return true
}

func (n *Node) lastCheckpoint() *wire.Checkpoint {
	n.mu.RLock()
	defer n.mu.RUnlock()
	if len(n.Chain) == 0 {
		return nil
	}
	return n.Chain[len(n.Chain)-1]
}

func (n *Node) lastCPEpoch() uint64 {
	if cp := n.lastCheckpoint(); cp != nil {
		return cp.Epoch
	}
	return 0
}

// hasUnresolvedConflict — есть ли незавершённое голосование.
func (n *Node) hasUnresolvedConflict() bool {
	for _, st := range n.Snowball.States() {
		if !st.Final {
			return true
		}
	}
	return false
}

// needsRebuild — противоречит ли состояние узла финализированному выбору
// (чекпоинт главнее локального голосования; 9.1).
func (n *Node) needsRebuild() bool {
	view := cons.NewFinalityView(n.chainSnapshot())
	for _, st := range n.Snowball.States() {
		if !st.Final {
			continue
		}
		alt := st.A
		if string(st.A) == string(st.Pref) {
			alt = st.B
		}
		// чтение реестра — под applyMu.RLock (net/query.go, этап 7)
		n.applyMu.RLock()
		w := n.wallet(conflictWallet(st.Key))
		tip := []byte(nil)
		if w != nil {
			tip = append([]byte(nil), w.TipTxID...)
		}
		n.applyMu.RUnlock()
		switch {
		case view.IsFinalized(st.Pref):
			if string(tip) != string(st.Pref) {
				return true
			}
		case view.IsFinalized(alt):
			if string(tip) != string(alt) {
				return true
			}
		}
	}
	return false
}

func (n *Node) chainSnapshot() []*wire.Checkpoint {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return append([]*wire.Checkpoint(nil), n.Chain...)
}

func cpHashOf(cp *wire.Checkpoint) []byte {
	h := cp.CheckpointHash()
	return h[:]
}

// pubKeyOf — ключ подписанта чекпоинта (кошелёк из состояния или
// ключ узла из конфигурации). Чтение реестра — под applyMu.RLock
// (net/query.go, этап 7); вызывается вне mu (Collector.Add).
func (n *Node) pubKeyOf(addr []byte) (*ecdsa.PublicKey, bool) {
	n.applyMu.RLock()
	w := n.wallet(addr)
	var pub *ecdsa.PublicKey
	if w != nil {
		pub = w.PublicKey
	}
	n.applyMu.RUnlock()
	if pub != nil {
		return pub, true
	}
	if string(addr) == string(n.Cfg.WalletAddr) && n.Cfg.WalletKey != nil {
		return &n.Cfg.WalletKey.PublicKey, true
	}
	return nil, false
}

// VerifyProofAgainstCheckpoint — лёгкий узел (12.5): Merkle-доказательство
// баланса проверяется против корня состояния ФИНАЛИЗИРОВАННОГО чекпоинта
// (не текущего состояния полного узла): подпись порогом N-из-M — гарантия
// того, что корень не подделан одиночным узлом.
func (n *Node) VerifyProofAgainstCheckpoint(addr []byte, p *balanceProofMsg, h checkpointHeaderMsg) bool {
	return VerifyBalanceProof(addr, p) && string(p.Root) == string(h.StateRoot)
}

// LatestHeader — последний принятый заголовок чекпоинта (лёгкий узел).
func (n *Node) LatestHeader() (checkpointHeaderMsg, bool) {
	n.mu.RLock()
	defer n.mu.RUnlock()
	if len(n.headers) == 0 {
		return checkpointHeaderMsg{}, false
	}
	return n.headers[len(n.headers)-1], true
}

// shortKey — короткое представление ключа конфликта.
func shortKey(key string) string {
	wallet := conflictWallet(key)
	return hex.EncodeToString(wallet)[:8] + "|" + key[len(key)-8:]
}

// CheckpointCount — число финализированных чекпоинтов узла (статус).
func (n *Node) CheckpointCount() int {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return len(n.Chain)
}
