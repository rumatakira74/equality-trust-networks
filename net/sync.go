// Синхронизация (4.4): req/res поверх streams. Полная сверка из genesis +
// журнала (INV-3; replay — пинирование часов на timestamp_claim каждой
// транзакции), цепочка кошелька по диапазону seq, входящие переводы,
// заголовки чекпоинтов и Merkle-доказательства баланса (core/smt) для
// лёгкого узла (12.5). Снапшоты из чекпоинтов — интерфейс (этап 5).
package net

import (
	"context"
	"crypto/sha256"
	"errors"
	"io"
	"time"

	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
	"github.com/rumatakira74/equality-trust-networks/core/ledger"
	"github.com/rumatakira74/equality-trust-networks/core/smt"
	"github.com/rumatakira74/equality-trust-networks/core/wire"
)

// readFrameStream — чтение одного кадра из потока (лимит 16 МиБ).
func readFrameStream(s network.Stream) ([]byte, error) {
	var hdr [5]byte
	if _, err := io.ReadFull(s, hdr[:]); err != nil {
		return nil, err
	}
	n := uint32(hdr[1])<<24 | uint32(hdr[2])<<16 | uint32(hdr[3])<<8 | uint32(hdr[4])
	if n > 16<<20 {
		return nil, errors.New("net: кадр слишком велик")
	}
	body := make([]byte, n)
	if _, err := io.ReadFull(s, body); err != nil {
		return nil, err
	}
	return append(hdr[:], body...), nil
}

// serveSync — обработчик протокола синхронизации (полный узел).
func (n *Node) serveSync(s network.Stream) {
	defer s.Close()
	raw, err := readFrameStream(s)
	if err != nil {
		return
	}
	msgType, d, err := readFrame(raw)
	if err != nil {
		return
	}
	switch msgType {
	case msgGetJournal:
		after, err := d.u64()
		if err != nil {
			return
		}
		// позиционный якорь (№41): txid на границе after−1 по нумерации
		// запрашивающего; старые клиенты поле не присылают — nil
		// (доверие позиции, прежняя семантика)
		anchor, _ := d.blob()
		n.sendJournal(s, after, anchor)
	case msgGetWalletRange:
		wallet, err := d.blob()
		if err != nil {
			return
		}
		from, err := d.u64()
		if err != nil {
			return
		}
		to, err := d.u64()
		if err != nil {
			return
		}
		n.sendWalletRange(s, wallet, from, to)
	case msgGetTip:
		wallet, err := d.blob()
		if err != nil {
			return
		}
		n.sendTip(s, wallet)
	case msgGetHeaders:
		after, err := d.u64()
		if err != nil {
			return
		}
		n.sendHeaders(s, after)
	case msgGetProof:
		addr, err := d.blob()
		if err != nil {
			return
		}
		n.sendProof(s, addr)
	case msgGetIncoming:
		addr, err := d.blob()
		if err != nil {
			return
		}
		n.sendIncoming(s, addr)
	default:
		return
	}
}

// encodeTxsMsg — сообщение с пачкой транзакций.
func encodeTxsMsg(msgType byte, txs []*wire.Transaction) []byte {
	var e enc
	e.u64(uint64(len(txs)))
	for _, tx := range txs {
		e.blob(tx.Encode())
	}
	return writeFrame(&e, msgType)
}

func decodeTxsMsg(d *dec) ([]*wire.Transaction, error) {
	n, err := d.u64()
	if err != nil {
		return nil, err
	}
	if n > 4096 {
		return nil, errors.New("net: слишком много транзакций в ответе")
	}
	out := make([]*wire.Transaction, 0, n)
	for i := uint64(0); i < n; i++ {
		raw, err := d.blob()
		if err != nil {
			return nil, err
		}
		tx, err := wire.DecodeTransaction(raw)
		if err != nil {
			return nil, err
		}
		out = append(out, tx)
	}
	return out, d.done()
}

// n.sendJournal — журнал с позиции after (нумерация журнала ЭТОГО узла;
// 0 — с начала). anchor — txid транзакции на позиции after−1 по нумерации
// ЗАПРАШИВАЮЩЕГО (граница его предыдущей сверки с нами, №41): не совпал с
// нашей журнальной позицией — истории разошлись (перестройка/другой
// порядок вставки), отдаём журнал с начала (выравнивание); пустой anchor —
// доверие позиции (прежняя семантика).
func (n *Node) sendJournal(s network.Stream, after uint64, anchor []byte) {
	n.mu.RLock()
	journal := n.journal
	n.mu.RUnlock()
	start := after
	if start > uint64(len(journal)) {
		// мы отстаём (в т.ч. перезапустились): на этой позиции подавать
		// нечего — отвечаем своей длиной, запрашивающий продолжит от неё
		start = uint64(len(journal))
	} else if after > 0 && len(anchor) == 32 {
		id := journal[after-1].TxID()
		if string(anchor) != string(id[:]) {
			start = 0 // якорь не сошёлся — выравнивание с начала (№41)
		}
	}
	end := start + uint64(n.Cfg.SyncBatch)
	if end > uint64(len(journal)) {
		end = uint64(len(journal))
	}
	batch := journal[start:end]
	var e enc
	e.u64(end) // следующая позиция
	e.u64(uint64(len(batch)))
	for _, tx := range batch {
		e.blob(tx.Encode())
	}
	_, _ = s.Write(writeFrame(&e, msgJournal))
}

func (n *Node) sendWalletRange(s network.Stream, wallet []byte, from, to uint64) {
	var txs []*wire.Transaction
	n.mu.RLock()
	chain := n.chainIndex[string(wallet)]
	for seq := from; seq <= to; seq++ {
		if tx, ok := chain[seq]; ok {
			txs = append(txs, tx)
		}
	}
	n.mu.RUnlock()
	_, _ = s.Write(encodeTxsMsg(msgTxs, txs))
}

func (n *Node) sendTip(s network.Stream, wallet []byte) {
	var e enc
	// чтение реестра — под applyMu.RLock (net/query.go, этап 7)
	n.applyMu.RLock()
	w := n.wallet(wallet)
	var seq uint64
	var tip []byte
	if w != nil {
		seq = w.Seq
		tip = append([]byte(nil), w.TipTxID...)
	}
	n.applyMu.RUnlock()
	if w != nil {
		e.u64(seq)
		e.blob(tip)
	} else {
		e.u64(0)
		e.blob(nil)
	}
	_, _ = s.Write(writeFrame(&e, msgTip))
}

func (n *Node) sendHeaders(s network.Stream, after uint64) {
	n.mu.RLock()
	hs := make([]checkpointHeaderMsg, 0)
	for _, h := range n.headers {
		if h.Epoch > after {
			hs = append(hs, h)
		}
	}
	n.mu.RUnlock()
	_, _ = s.Write(encodeHeaders(hs))
}

// stateLeafOf — лист SMT для кошелька (решение D7, spec/07 7.1.1).
func stateLeafOf(w *ledger.Wallet) *wire.StateLeaf {
	return &wire.StateLeaf{
		Seq:               w.Seq,
		Balance:           w.TotalBalance(),
		Status:            w.Status,
		Flags:             &wire.WalletFlags{Flagged: w.Flagged, Synthetic: w.Synthetic, Seed: w.Seed},
		RegisteredEpoch:   w.RegisteredEpoch,
		ActivatedEpoch:    w.ActivatedEpoch,
		LastClaimedPeriod: w.LastClaimedPeriod,
		DelayedThreshold:  w.DelayedThreshold,
	}
}

// sendProof — Merkle-доказательство баланса (12.5, core/smt).
func (n *Node) sendProof(s network.Stream, addr []byte) {
	if n.Ledger == nil {
		_, _ = s.Write(encodeProof(&balanceProofMsg{}))
		return
	}
	// чтение реестра — под applyMu.RLock (net/query.go, этап 7)
	n.applyMu.RLock()
	tree := smt.New()
	for _, w := range n.Ledger.State.Wallets {
		leaf := stateLeafOf(w)
		var idx [32]byte
		copy(idx[:], w.Address)
		lh := sha256.Sum256(leaf.Encode())
		tree.SetLeaf(idx, lh)
	}
	var idx [32]byte
	copy(idx[:], addr)
	siblings, leaf, ok := tree.Prove(idx)
	if !ok {
		n.applyMu.RUnlock()
		_, _ = s.Write(encodeProof(&balanceProofMsg{}))
		return
	}
	root := tree.Root()
	p := &balanceProofMsg{Leaf: leaf[:], Root: root[:], Seq: n.wallet(addr).Seq}
	if w := n.wallet(addr); w != nil {
		p.LeafData = stateLeafOf(w).Encode()
	}
	n.applyMu.RUnlock()
	for _, sb := range siblings {
		p.Siblings = append(p.Siblings, sb[:])
	}
	_, _ = s.Write(encodeProof(p))
}

// sendIncoming — входящие переводы на адрес (лёгкий узел, 12.5).
func (n *Node) sendIncoming(s network.Stream, addr []byte) {
	var txs []*wire.Transaction
	n.mu.RLock()
	for _, tx := range n.journal {
		if tx.Type != wire.Transfer {
			continue
		}
		if pl, ok := tx.Payload.(*wire.TransferPayload); ok && pl.Send != nil &&
			string(pl.Send.Recipient) == string(addr) {
			txs = append(txs, tx)
		}
	}
	n.mu.RUnlock()
	_, _ = s.Write(encodeTxsMsg(msgTxs, txs))
}

// ---------------------------------------------------------------------------
// Клиент
// ---------------------------------------------------------------------------

// request — запрос-ответ по протоколу синхронизации.
func (n *Node) request(ctx context.Context, p peer.ID, req []byte) ([]byte, error) {
	return n.requestOn(ctx, p, n.syncProtocol(), req)
}

// requestOn — запрос-ответ по указанному протоколу (sync, vote).
func (n *Node) requestOn(ctx context.Context, p peer.ID, proto protocol.ID, req []byte) ([]byte, error) {
	s, err := n.Host.NewStream(ctx, p, proto)
	if err != nil {
		return nil, err
	}
	defer s.Close()
	s.SetDeadline(time.Now().Add(30 * time.Second))
	if _, err := s.Write(req); err != nil {
		return nil, err
	}
	return readFrameStream(s)
}

// syncAnchor — позиционный якорь сверки с одним пиром (№41): позиция в
// нумерации журнала ПИРА и txid транзакции на позиции pos−1 (последняя
// полученная от этого пира). Продолжение сверки строго инкрементально:
// при совпадении якоря передаются только новые транзакции; расхождение
// истории (перестройка журнала у пира, другой порядок вставки) детектится
// несовпадением txid на границе — сверка выравнивается с начала.
type syncAnchor struct {
	pos  uint64
	txid []byte // nil — позиция без подтверждения (первая сверка/пир отстал)
}

// SyncFromPeer — инкрементальная сверка журнала с подключённым пиром:
// продолжение с позиционного якоря этого пира (№41). Первая сверка —
// с начала (заодно полный corroboration всего журнала этим пиром),
// далее — только новые транзакции.
func (n *Node) SyncFromPeer(ctx context.Context, p peer.ID) (int, error) {
	n.mu.Lock()
	a := n.syncAnchors[p]
	if a == nil {
		a = &syncAnchor{}
		n.syncAnchors[p] = a
	}
	pos, txid := a.pos, a.txid
	n.mu.Unlock()
	k, pos, txid, err := n.SyncJournalFrom(ctx, p, pos, txid)
	n.mu.Lock()
	a.pos, a.txid = pos, txid
	n.mu.Unlock()
	return k, err
}

// SyncJournalFrom — сверка из genesis + журнала (INV-3) с позиции after —
// позиционный якорь (№41): after — позиция в журнале ПИРА, anchor — txid
// транзакции на позиции after−1 (граница предыдущей сверки; nil — доверие
// позиции, 0 — с начала). Якорь не сошёлся (пир перестроил журнал или
// порядок вставки разошёлся) — пир отвечает с начала, сверка
// выравнивается (дубликаты отсеиваются дедупом дёшево, вне cheapCheck);
// сошёлся — продолжение строго инкрементально, трафик пропорционален
// только новым транзакциям. Каждый ответ пира подтверждает доставленные
// транзакции (corroboration, eclipse 12.3). Транзакции применяются с
// пинированием часов на timestamp_claim (replay), статус «provisional»
// до подтверждения N пирами.
// Возвращает число обработанных транзакций и обновлённый якорь.
func (n *Node) SyncJournalFrom(ctx context.Context, p peer.ID, after uint64, anchor []byte) (int, uint64, []byte, error) {
	if n.Ledger == nil {
		return 0, after, anchor, errors.New("net: синхронизация журнала — только для полного узла")
	}
	applied := 0
	// Replay (INV-3): часы пинируются на timestamp_claim каждой транзакции —
	// C9 (±30 мин) проверяется на момент публикации, а не пересчёта.
	rc := NewReplayClock(n.Cfg.Genesis.Genesis.GenesisTime, n.Cfg.Genesis.Manifest.Profile.EpochSeconds)
	for {
		var e enc
		e.u64(after)
		e.blob(anchor)
		raw, err := n.request(ctx, p, writeFrame(&e, msgGetJournal))
		if err != nil {
			return applied, after, anchor, err
		}
		msgType, d, err := readFrame(raw)
		if err != nil || msgType != msgJournal {
			return applied, after, anchor, errors.New("net: некорректный ответ журнала")
		}
		next, err := d.u64()
		if err != nil {
			return applied, after, anchor, err
		}
		txs, err := decodeTxsMsg(d)
		if err != nil {
			return applied, after, anchor, err
		}
		if len(txs) > 0 {
			touched := make([][]byte, 0, len(txs))
			n.applyMu.Lock()
			for _, tx := range txs {
				// corroboration: журнал доставлен этим пиром (eclipse, 12.3)
				id := tx.TxID()
				n.Corroboration.Add(id[:], p)
				rc.Pin(tx.TimestampClaim)
				res := n.applyWithClock(tx, rc)
				n.postApply(tx, res)
				touched = append(touched, tx.Wallet)
				applied++
			}
			n.applyMu.Unlock()
			// повтор буфера внеочередных — строго вне applyMu
			// (retryPending берёт RLock; из-под Lock — self-deadlock)
			for _, w := range touched {
				n.retryPending(w)
			}
			// якорь продолжения — последняя полученная от пира транзакция
			// (его нумерация): следующая сверка пойдёт строго дальше
			id := txs[len(txs)-1].TxID()
			anchor = append([]byte(nil), id[:]...)
			after = next
			continue
		}
		// пустой ответ: пир исчерпан (next == after — мы в его конце) либо
		// отстал/перезапустился (next < after) — продолжим от его границы,
		// якорь восстановится первым непустым ответом
		if next < after {
			after, anchor = next, nil
		}
		return applied, after, anchor, nil
	}
}

// GetWalletRange — цепочка кошелька по диапазону seq (клиент).
func (n *Node) GetWalletRange(ctx context.Context, p peer.ID, wallet []byte, from, to uint64) ([]*wire.Transaction, error) {
	var e enc
	e.blob(wallet)
	e.u64(from)
	e.u64(to)
	raw, err := n.request(ctx, p, writeFrame(&e, msgGetWalletRange))
	if err != nil {
		return nil, err
	}
	msgType, d, err := readFrame(raw)
	if err != nil || msgType != msgTxs {
		return nil, errors.New("net: некорректный ответ диапазона")
	}
	return decodeTxsMsg(d)
}

// GetTip — tip кошелька (клиент).
func (n *Node) GetTip(ctx context.Context, p peer.ID, wallet []byte) (uint64, []byte, error) {
	var e enc
	e.blob(wallet)
	raw, err := n.request(ctx, p, writeFrame(&e, msgGetTip))
	if err != nil {
		return 0, nil, err
	}
	msgType, d, err := readFrame(raw)
	if err != nil || msgType != msgTip {
		return 0, nil, errors.New("net: некорректный ответ tip")
	}
	seq, err := d.u64()
	if err != nil {
		return 0, nil, err
	}
	tip, err := d.blob()
	return seq, tip, err
}

// GetHeaders — заголовки чекпоинтов после эпохи (клиент; этап 5 подписывает).
func (n *Node) GetHeaders(ctx context.Context, p peer.ID, afterEpoch uint64) ([]checkpointHeaderMsg, error) {
	var e enc
	e.u64(afterEpoch)
	raw, err := n.request(ctx, p, writeFrame(&e, msgGetHeaders))
	if err != nil {
		return nil, err
	}
	msgType, d, err := readFrame(raw)
	if err != nil || msgType != msgHeaders {
		return nil, errors.New("net: некорректный ответ заголовков")
	}
	return decodeHeaders(d)
}

// GetBalanceProof — Merkle-доказательство баланса (клиент, core/smt).
func (n *Node) GetBalanceProof(ctx context.Context, p peer.ID, addr []byte) (*balanceProofMsg, error) {
	var e enc
	e.blob(addr)
	raw, err := n.request(ctx, p, writeFrame(&e, msgGetProof))
	if err != nil {
		return nil, err
	}
	msgType, d, err := readFrame(raw)
	if err != nil || msgType != msgProof {
		return nil, errors.New("net: некорректный ответ доказательства")
	}
	return decodeProof(d)
}

// GetIncoming — входящие переводы на адрес (клиент лёгкого узла, 12.5).
func (n *Node) GetIncoming(ctx context.Context, p peer.ID, addr []byte) ([]*wire.Transaction, error) {
	var e enc
	e.blob(addr)
	raw, err := n.request(ctx, p, writeFrame(&e, msgGetIncoming))
	if err != nil {
		return nil, err
	}
	msgType, d, err := readFrame(raw)
	if err != nil || msgType != msgTxs {
		return nil, errors.New("net: некорректный ответ входящих")
	}
	return decodeTxsMsg(d)
}

// BalanceWithProof — баланс адреса по Merkle-доказательству от
// подключённого пира против корня ФИНАЛИЗИРОВАННОГО чекпоинта (12.5;
// мост Android, этап 6). Лёгкий узел не доверяет состоянию полного
// узла — только пороговой подписи чекпоинта.
func (n *Node) BalanceWithProof(ctx context.Context, addr []byte) (int64, bool, error) {
	peers := n.connectedPeers()
	if len(peers) == 0 {
		return 0, false, errors.New("net: нет подключённых пиров")
	}
	h, ok := n.LatestHeader()
	if !ok {
		return 0, false, errors.New("net: нет финализированных чекпоинтов")
	}
	var lastErr error
	for _, p := range peers {
		proof, err := n.GetBalanceProof(ctx, p, addr)
		if err != nil {
			lastErr = err
			continue
		}
		if !n.VerifyProofAgainstCheckpoint(addr, proof, h) {
			lastErr = errors.New("net: доказательство не сходится с корнем чекпоинта")
			continue
		}
		leaf, err := wire.DecodeStateLeaf(proof.LeafData)
		if err != nil {
			lastErr = err
			continue
		}
		return int64(leaf.Balance), true, nil
	}
	return 0, false, lastErr
}

// VerifyBalanceProof — проверка Merkle-доказательства (лёгкий узел).
func VerifyBalanceProof(addr []byte, p *balanceProofMsg) bool {
	if len(p.Leaf) != 32 || len(p.Root) != 32 || len(p.Siblings) != smt.Depth {
		return false
	}
	var idx [32]byte
	copy(idx[:], addr)
	var leaf [32]byte
	copy(leaf[:], p.Leaf)
	var root [32]byte
	copy(root[:], p.Root)
	siblings := make([][32]byte, len(p.Siblings))
	for i, s := range p.Siblings {
		if len(s) != 32 {
			return false
		}
		copy(siblings[i][:], s)
	}
	return smt.VerifyProof(idx, leaf, siblings, root)
}
