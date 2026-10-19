// Логика виртуального узла: применение транзакций (provisional + фиксация
// конфликта), голосование (опрос достижимых пиров), чекпоинты (подпись,
// сбор, финализация) и перестройка состояния при смене ветки (откаты).
package sim

import (
	"math/rand"

	"github.com/rumatakira74/equality-trust-networks/cons"
	"github.com/rumatakira74/equality-trust-networks/core/ledger"
	"github.com/rumatakira74/equality-trust-networks/core/wire"
)

// process — обработка доставленного сообщения.
func (n *Node) process(net *Network, m *msg) {
	switch m.kind {
	case msgTx:
		n.applyTx(net, m.tx)
	case msgShare:
		n.onShare(net, m.cp, m.share)
	case msgCheckpoint:
		n.onCheckpoint(net, m.cp)
	case msgChainGet:
		// отвечает только детерминированный предлагающий (иначе шторм
		// ответов: каждый отстающий узел получает копию от каждого)
		if len(n.Chain) > 0 && n.isProposer(net) {
			net.Broadcast(n.ID, &msg{from: n.ID, kind: msgChain, chain: n.Chain})
		}
	case msgChain:
		n.onChain(net, m.chain)
	}
}

// applyTx — provisional-применение транзакции (FirstWins ядра); вторая
// ветка конфликта запускает голосование (9.1).
func (n *Node) applyTx(net *Network, tx *wire.Transaction) {
	id := tx.TxID()
	if n.seen[string(id[:])] {
		return
	}
	n.seen[string(id[:])] = true
	n.allTxs = append(n.allTxs, tx)
	res := n.Ledger.Apply(tx)
	key := cons.ConflictKey(tx)
	switch res.Outcome {
	case ledger.Accepted:
		n.applied[string(id[:])] = true
		n.branchApplied[key] = id[:]
	case ledger.Conflict:
		// эквивокация доставлена: обе ветки в журнале (INV-1c),
		// выбор — за голосованием (9.1; вердиктов нет, 7.5)
		applied := n.branchApplied[key]
		if applied == nil {
			applied = n.Ledger.State.Wallet(tx.Wallet).TipTxID
		}
		n.Snowball.Observe(key, append([]byte(nil), applied...), id[:])
	default:
		// невалидная — не применяется (спам-фильтр сети — этап 4)
	}
}

// voteRound — один раунд голосования по каждому активному конфликту.
func (n *Node) voteRound(net *Network) {
	for _, st := range n.Snowball.States() {
		if st.Final {
			continue
		}
		before := st.Pref
		poller := &nodePoller{net: net, from: n.ID}
		finalized := n.Snowball.PollRound(st.Key, poller, net.rng)
		net.Metrics.VoteMessages += net.Params.SnowballK
		if !finalized {
			continue
		}
		// метрика финальности: первая финализация конфликта в сети
		if _, done := net.conflictFinalized[st.Key]; !done {
			net.conflictFinalized[st.Key] = net.RoundN
			net.Metrics.ConflictsResolved++
			if injected := net.conflictInjected[st.Key]; injected > 0 {
				net.Metrics.RoundsToFinality = append(net.Metrics.RoundsToFinality,
					net.RoundN-injected)
			}
		}
		// смена ветки: перестроить состояние из канонического журнала
		if string(st.Pref) != string(before) || string(st.Pref) != string(n.branchApplied[st.Key]) {
			n.rebuild(net)
		}
	}
}

// rebuild — перестройка состояния из genesis + журнала: победившие ветки
// (локальная финальность голосования) применяются первыми (cons.Replay);
// вытесненные валидные транзакции — метрика отката.
func (n *Node) rebuild(net *Network) {
	// FinalityView: чекпоинты главнее; локальные победители голосования
	// дополняют только конфликты, которых чекпоинты ещё не покрывали
	chain := make([]*wire.Checkpoint, 0)
	chain = append(chain, n.Chain...)
	view := cons.NewFinalityView(chain)
	for _, st := range n.Snowball.States() {
		if !st.Final {
			continue
		}
		if !view.IsFinalized(st.A) && !view.IsFinalized(st.B) {
			view.MarkFinalized(st.Pref)
		}
	}
	before := make(map[string]bool)
	for k := range n.applied {
		before[k] = true
	}
	l, err := ledger.New(net.GP, &netClock{net: net}, ledger.Options{SkipMintChallengeWindow: true})
	if err != nil {
		panic(err)
	}
	// pre-state — исходное состояние кошельков на момент genesis (как при
	// создании узлов): replay применяется к genesis-состоянию, копировать
	// текущие (уже эволюционировавшие) кошельки нельзя — двойное применение
	net.prestateInto(l)
	if err := cons.Replay(l, n.allTxs, view); err != nil {
		// журнал с отклонениями (дубль после перестройки) — фиксируем
		// метрику и оставляем состояние (защита от зацикливания)
		net.Metrics.ReplayFailures++
		net.Metrics.LastVerifyError = "replay: " + err.Error()
		return
	}
	n.Ledger = l
	n.applied = make(map[string]bool)
	for _, tx := range l.AcceptedTxs() {
		id := tx.TxID()
		n.applied[string(id[:])] = true
	}
	// применённая ветка конфликта — та, что вошла в новое состояние;
	// локальное голосование выравнивается под финализацию (иначе следующая
	// сверка с чекпоинтом снова запустит перестройку)
	for _, st := range n.Snowball.States() {
		if n.applied[string(st.A)] {
			n.branchApplied[st.Key] = st.A
			st.Pref = st.A
		} else if n.applied[string(st.B)] {
			n.branchApplied[st.Key] = st.B
			st.Pref = st.B
		}
	}
	n.rebuilds++
	net.Metrics.Rebuilds++
	// откат: применённые ранее, но не вошедшие в новое состояние
	for k := range before {
		if !n.applied[k] {
			n.rolledBack++
			net.Metrics.RolledBackTxs++
		}
	}
}

// checkpointPhase — закрытие эпохи. Предлагающий детерминирован (минимальный
// номер среди онлайн-узлов): чекпоинт-контент детерминирован (журнал,
// время, участники), поэтому конкурирующих предложений не возникает;
// остальные узлы подписывают полученное предложение. Узел не подписывает
// при незавершённых конфликтах: состояние ещё не канонично.
func (n *Node) checkpointPhase(net *Network) {
	var prev *wire.Checkpoint
	if len(n.Chain) > 0 {
		prev = n.Chain[len(n.Chain)-1]
	}
	due := false
	if prev == nil {
		due = net.Time >= net.GP.Genesis.GenesisTime+net.Params.EpochSeconds
	} else {
		due = net.Time >= prev.NetworkTime+net.Params.EpochSeconds
	}
	if !due {
		n.collectPending(net)
		return
	}
	if !n.isProposer(net) {
		n.collectPending(net)
		return
	}
	// не строим чекпоинт при незавершённых конфликтах (чекпоинт фиксирует
	// сходимость; незавершённое голосование — не финализируем)
	for _, st := range n.Snowball.States() {
		if !st.Final {
			return
		}
	}
	epoch := uint64(1)
	var prevHash []byte
	if prev != nil {
		epoch = prev.Epoch + 1
		h := prev.CheckpointHash()
		prevHash = h[:]
	} else {
		prevHash = make([]byte, 32)
	}
	part, thresholdN, err := cons.Participants(
		net.GP.Manifest.Profile.CheckpointMode, n.Ledger, net.GP, net.Params, epoch, prevHash)
	if err != nil {
		return
	}
	// finalized_txid_list: принятые с прошлого чекпоинта
	journal := n.Ledger.AcceptedTxs()
	if n.journalAtCP > len(journal) {
		n.journalAtCP = 0
	}
	var finalized [][]byte
	for i := n.journalAtCP; i < len(journal); i++ {
		id := journal[i].TxID()
		finalized = append(finalized, id[:])
	}
	netTime := net.GP.Genesis.GenesisTime + epoch*net.Params.EpochSeconds
	if prev != nil {
		netTime = prev.NetworkTime + net.Params.EpochSeconds
	}
	cp, err := cons.Build(net.GP.Genesis.NetworkID, epoch, n.Ledger, prev,
		netTime, net.GP.Genesis.ProtocolVersion, part, thresholdN, finalized)
	if err != nil {
		return
	}
	n.onShare(net, cp, wire.SignatureShare{}) // регистрация предложения (без доли)
	// предложение рассылается всегда (предлагающий может не быть участником
	// выборки подписантов); участники подписывают полученное предложение
	net.Broadcast(n.ID, &msg{from: n.ID, kind: msgShare, cp: cp})
	// подпись своим кошельком, если узел — участник
	for _, p := range part {
		if string(p) == string(n.Wallet) {
			share, err := (&cons.Signer{Wallet: n.Wallet, Priv: n.Key}).Sign(cp)
			if err != nil {
				return
			}
			n.onShare(net, cp, share)
			net.Broadcast(n.ID, &msg{from: n.ID, kind: msgShare, share: share, cp: cp})
			break
		}
	}
	n.collectPending(net)
}

// isProposer — детерминированный предлагающий: минимальный номер среди
// онлайн-узлов (в реальной сети предлагает любой — контент одинаков).
func (n *Node) isProposer(net *Network) bool {
	for _, other := range net.Nodes {
		if !other.Offline && other.Group == n.Group {
			return other.ID == n.ID
		}
	}
	return false
}

// collectPending — попытка финализации всех открытых сборов.
func (n *Node) collectPending(net *Network) {
	for h, col := range n.Collector {
		cp := n.Proposals[h]
		if cp == nil || cp.ThresholdSig != nil {
			continue
		}
		if sig := col.Ready(); sig != nil {
			cp.ThresholdSig = sig
			if err := cons.Verify(cp, n.lastCP(), net.GP.Genesis.GenesisTime,
				net.Params.EpochSeconds, net.Params.MaxTimeShift, net.pubOf); err != nil {
				net.Metrics.SignFailedNoQuorum++
				continue
			}
			n.adoptCheckpoint(net, cp)
			net.Broadcast(n.ID, &msg{from: n.ID, kind: msgCheckpoint, cp: cp})
		}
	}
}

func (n *Node) lastCP() *wire.Checkpoint {
	if len(n.Chain) == 0 {
		return nil
	}
	return n.Chain[len(n.Chain)-1]
}

// onShare — доля подписи чекпоинта (или регистрация предложения):
// проверить форму, добавить в сбор; участник подписывает полученное
// предложение и рассылает свою долю.
func (n *Node) onShare(net *Network, cp *wire.Checkpoint, share wire.SignatureShare) {
	if cp == nil {
		return
	}
	h := cp.CheckpointHash()
	if _, ok := n.Proposals[string(h[:])]; !ok {
		// предложение от другого узла: принять, если эпоха следующая;
		// отставший узел запрашивает цепочку у сети
		if cp.Epoch != n.lastCPEpoch()+1 {
			if cp.Epoch > n.lastCPEpoch()+1 && net.RoundN >= n.lastChainGetRound+2 {
				n.lastChainGetRound = net.RoundN
				net.Broadcast(n.ID, &msg{from: n.ID, kind: msgChainGet})
			}
			return
		}
		n.Proposals[string(h[:])] = cp
		n.Collector[string(h[:])] = cons.NewCollector(cp, net.pubOf)
		// участник подписывает чужое предложение (контент детерминирован)
		for _, p := range cp.Participants {
			if string(p) == string(n.Wallet) && !n.hasUnresolvedConflict() {
				myShare, err := (&cons.Signer{Wallet: n.Wallet, Priv: n.Key}).Sign(cp)
				if err != nil {
					return
				}
				n.Collector[string(h[:])].Add(myShare)
				net.Broadcast(n.ID, &msg{from: n.ID, kind: msgShare, share: myShare, cp: cp})
				break
			}
		}
	}
	if len(share.Signer) == 0 {
		return // регистрация предложения без доли
	}
	n.Collector[string(h[:])].Add(share)
}

// onChain — синхронизация цепочки чекпоинтов (узел отстал после
// расщепления/офлайна): принять недостающие чекпоинты по порядку.
func (n *Node) onChain(net *Network, chain []*wire.Checkpoint) {
	for _, cp := range chain {
		if cp == nil || cp.Epoch != n.lastCPEpoch()+1 {
			continue
		}
		if err := cons.Verify(cp, n.lastCP(), net.GP.Genesis.GenesisTime,
			net.Params.EpochSeconds, net.Params.MaxTimeShift, net.pubOf); err != nil {
			continue
		}
		n.adoptCheckpoint(net, cp)
	}
	if n.needsRebuildFor(cons.NewFinalityView(n.Chain)) {
		n.rebuild(net)
	}
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

// onCheckpoint — финализированный чекпоинт от другого узла: проверить и принять.
func (n *Node) onCheckpoint(net *Network, cp *wire.Checkpoint) {
	if cp == nil {
		return
	}
	if cp.Epoch > n.lastCPEpoch()+1 {
		// отставший узел: нужен предыдущий чекпоинт для проверки цепочки
		// (не чаще раза в 2 раунда — иначе шторм запросов)
		if net.RoundN >= n.lastChainGetRound+2 {
			n.lastChainGetRound = net.RoundN
			net.Broadcast(n.ID, &msg{from: n.ID, kind: msgChainGet})
		}
		return
	}
	if err := cons.Verify(cp, n.lastCP(), net.GP.Genesis.GenesisTime,
		net.Params.EpochSeconds, net.Params.MaxTimeShift, net.pubOf); err != nil {
		return
	}
	if n.lastCPEpoch() >= cp.Epoch {
		return
	}
	n.adoptCheckpoint(net, cp)
	// чекпоинт фиксирует выбор веток: перестроиться, только если узел
	// держит ветку, противоречащую finalized_txid_list
	if n.needsRebuildFor(cons.NewFinalityView(n.Chain)) {
		n.rebuild(net)
	}
}

// needsRebuildFor — противоречит ли состояние узла финализированному
// выбору. Чекпоинт главнее локального голосования: при расщеплении
// группы голосуют независимо (Snowball финализируется в каждой), общий
// выбор даёт только чекпоинт (9.1: побеждает ветка, выбранная
// финализацией).
func (n *Node) needsRebuildFor(view *cons.FinalityView) bool {
	for _, st := range n.Snowball.States() {
		if !st.Final {
			continue // голосование ещё идёт — чекпоинт его не разрешал
		}
		alt := st.A
		if string(st.A) == string(st.Pref) {
			alt = st.B
		}
		switch {
		case view.IsFinalized(st.Pref):
			// победитель зафиксирован: узел обязан его держать
			if string(n.branchApplied[st.Key]) != string(st.Pref) {
				return true
			}
		case view.IsFinalized(alt):
			// локальная финализация была частной (расщепление), чекпоинт
			// выбрал другую ветку — перестроиться, если ещё не выровнены
			if string(n.branchApplied[st.Key]) != string(alt) {
				return true
			}
		default:
			// чекпоинт конфликт ещё не покрывал — локальный выбор действует
		}
	}
	return false
}

func (n *Node) lastCPEpoch() uint64 {
	if cp := n.lastCP(); cp != nil {
		return cp.Epoch
	}
	return 0
}

// adoptCheckpoint — принять финализированный чекпоинт (цепочка, метрики).
func (n *Node) adoptCheckpoint(net *Network, cp *wire.Checkpoint) {
	if n.lastCPEpoch() >= cp.Epoch {
		return
	}
	n.Chain = append(n.Chain, cp)
	n.journalAtCP = len(n.Ledger.AcceptedTxs())
	net.Metrics.Checkpoints++
	if b := len(cp.Encode()); b > net.Metrics.CheckpointBytes {
		net.Metrics.CheckpointBytes = b
	}
}

// VotePreference — ответ пира на опрос (cons.PollablePeer). Злонамеренные
// узлы всегда отвечают веткой атакующего (в сценариях — B).
func (n *Node) VotePreference(key string) ([]byte, uint64, bool) {
	if n.Offline {
		return nil, 0, false
	}
	st := n.Snowball.State(key)
	if st == nil {
		return nil, 0, false
	}
	pref := st.Pref
	if n.Malicious {
		pref = st.B // ветка атакующего (сценарии злонамеренного меньшинства)
	}
	w := n.Ledger.State.Wallet(n.Wallet)
	weight := cons.Weight(w, netStrictPolicy(n), n.Ledger.State.NetTime)
	return pref, weight, true
}

func netStrictPolicy(n *Node) bool {
	return n.Ledger.State.Net.Profile.DevicePolicy == wire.DevicePolicyStrict
}

// nodePoller — опрос пиров от имени узла from (cons.Poller): k случайных
// достижимых пиров (онлайн, та же группа разбиения).
type nodePoller struct {
	net  *Network
	from int
}

func (p *nodePoller) Sample(k int, rng *rand.Rand) []cons.PollablePeer {
	var reachable []*Node
	for _, n := range p.net.Nodes {
		if n.ID != p.from && p.net.reachable(p.from, n.ID) {
			reachable = append(reachable, n)
		}
	}
	out := make([]cons.PollablePeer, 0, k)
	if len(reachable) == 0 {
		return out
	}
	for i := 0; i < k; i++ {
		out = append(out, reachable[rng.Intn(len(reachable))])
	}
	return out
}
