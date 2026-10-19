// Кодек служебных сообщений синхронизации и peer exchange (4.4).
// Формат кадра: [тип u8][длина u32 BE][тело]. Тела — компактное бинарное
// кодирование (varint/blobs); служебные сообщения не хешируются, поэтому
// canonical-требования spec/01 на них не распространяются.
package net

import (
	"encoding/binary"
	"errors"
	"fmt"
)

const (
	msgGetJournal     = 1  // запрос журнала (полная сверка, INV-3)
	msgJournal        = 2  // пачка транзакций журнала
	msgGetWalletRange = 3  // цепочка кошелька по диапазону seq
	msgTxs            = 4  // пачка транзакций (ответ)
	msgGetTip         = 5  // tip кошелька
	msgTip            = 6  // ответ tip
	msgGetHeaders     = 7  // заголовки чекпоинтов (после эпохи)
	msgHeaders        = 8  // ответ: заголовки
	msgGetProof       = 9  // Merkle-доказательство баланса (core/smt)
	msgProof          = 10 // ответ: доказательство
	msgGetIncoming    = 11 // входящие переводы на адрес (лёгкий узел)
	msgGetPeers       = 12 // peer exchange: запрос списка пиров
	msgPeers          = 13 // peer exchange: ответ
)

// errFrame — повреждённый кадр.
var errFrame = errors.New("net: повреждённый кадр")

// ---------------------------------------------------------------------------
// Запись
// ---------------------------------------------------------------------------

type enc struct{ b []byte }

func (e *enc) u64(v uint64) {
	var buf [binary.MaxVarintLen64]byte
	n := binary.PutUvarint(buf[:], v)
	e.b = append(e.b, buf[:n]...)
}

func (e *enc) blob(v []byte) {
	e.u64(uint64(len(v)))
	e.b = append(e.b, v...)
}

func (e *enc) str(v string) { e.blob([]byte(v)) }

// ---------------------------------------------------------------------------
// Чтение
// ---------------------------------------------------------------------------

type dec struct {
	b   []byte
	pos int
}

func (d *dec) u64() (uint64, error) {
	v, n := binary.Uvarint(d.b[d.pos:])
	if n <= 0 {
		return 0, errFrame
	}
	d.pos += n
	return v, nil
}

func (d *dec) blob() ([]byte, error) {
	n, err := d.u64()
	if err != nil {
		return nil, err
	}
	if uint64(len(d.b)-d.pos) < n {
		return nil, errFrame
	}
	v := d.b[d.pos : d.pos+int(n)]
	d.pos += int(n)
	return v, nil
}

func (d *dec) str() (string, error) {
	b, err := d.blob()
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// tag — protobuf-тег (номер поля + wiretype; для разбора wire-сообщений).
func (d *dec) tag() (field uint32, wiretype byte, err error) {
	v, err := d.u64()
	if err != nil {
		return 0, 0, err
	}
	return uint32(v >> 3), byte(v & 7), nil
}

func (d *dec) done() error {
	if d.pos != len(d.b) {
		return errFrame
	}
	return nil
}

// ---------------------------------------------------------------------------
// Кадр
// ---------------------------------------------------------------------------

func writeFrame(e *enc, msgType byte) []byte {
	out := make([]byte, 5, 5+len(e.b))
	out[0] = msgType
	binary.BigEndian.PutUint32(out[1:5], uint32(len(e.b)))
	return append(out, e.b...)
}

func readFrame(b []byte) (byte, *dec, error) {
	if len(b) < 5 {
		return 0, nil, errFrame
	}
	n := binary.BigEndian.Uint32(b[1:5])
	if uint64(len(b)-5) < uint64(n) {
		return 0, nil, errFrame
	}
	return b[0], &dec{b: b[5 : 5+int(n)]}, nil
}

// ---------------------------------------------------------------------------
// Сообщения
// ---------------------------------------------------------------------------

// checkpointHeaderMsg — заголовок чекпоинта (этап 5 подписывает; здесь —
// хранение и передача). StateRoot — для проверки доказательств лёгким
// узлом против финализированного чекпоинта (12.5).
type checkpointHeaderMsg struct {
	Epoch     uint64
	Hash      []byte
	StateRoot []byte
}

func encodeHeaders(hs []checkpointHeaderMsg) []byte {
	var e enc
	e.u64(uint64(len(hs)))
	for i := range hs {
		e.u64(hs[i].Epoch)
		e.blob(hs[i].Hash)
		e.blob(hs[i].StateRoot)
	}
	return writeFrame(&e, msgHeaders)
}

func decodeHeaders(d *dec) ([]checkpointHeaderMsg, error) {
	n, err := d.u64()
	if err != nil {
		return nil, err
	}
	if n > 4096 {
		return nil, fmt.Errorf("net: слишком много заголовков: %d", n)
	}
	out := make([]checkpointHeaderMsg, 0, n)
	for i := uint64(0); i < n; i++ {
		var h checkpointHeaderMsg
		if h.Epoch, err = d.u64(); err != nil {
			return nil, err
		}
		if h.Hash, err = d.blob(); err != nil {
			return nil, err
		}
		if h.StateRoot, err = d.blob(); err != nil {
			return nil, err
		}
		out = append(out, h)
	}
	return out, d.done()
}

// balanceProofMsg — Merkle-доказательство баланса (core/smt; 12.5).
type balanceProofMsg struct {
	// Leaf — хеш листа (участвует в доказательстве).
	Leaf []byte
	// LeafData — кодирование wire.StateLeaf (баланс, seq, статус; D7).
	LeafData []byte
	Siblings [][]byte
	Root     []byte
	// Seq — высота цепочки кошелька на момент доказательства (сверка лёгким
	// узлом своей цепочки).
	Seq uint64
}

func encodeProof(p *balanceProofMsg) []byte {
	var e enc
	e.blob(p.Leaf)
	e.blob(p.LeafData)
	e.u64(uint64(len(p.Siblings)))
	for _, s := range p.Siblings {
		e.blob(s)
	}
	e.blob(p.Root)
	e.u64(p.Seq)
	return writeFrame(&e, msgProof)
}

func decodeProof(d *dec) (*balanceProofMsg, error) {
	p := &balanceProofMsg{}
	var err error
	if p.Leaf, err = d.blob(); err != nil {
		return nil, err
	}
	if p.LeafData, err = d.blob(); err != nil {
		return nil, err
	}
	n, err := d.u64()
	if err != nil {
		return nil, err
	}
	if n > 4096 {
		return nil, fmt.Errorf("net: слишком много siblings: %d", n)
	}
	for i := uint64(0); i < n; i++ {
		s, err := d.blob()
		if err != nil {
			return nil, err
		}
		p.Siblings = append(p.Siblings, s)
	}
	if p.Root, err = d.blob(); err != nil {
		return nil, err
	}
	if p.Seq, err = d.u64(); err != nil {
		return nil, err
	}
	return p, d.done()
}

// peerAddrMsg — адрес пира для peer exchange.
type peerAddrMsg struct {
	ID    string
	Addrs []string
}

func encodePeers(ps []peerAddrMsg) []byte {
	var e enc
	e.u64(uint64(len(ps)))
	for i := range ps {
		e.str(ps[i].ID)
		e.u64(uint64(len(ps[i].Addrs)))
		for _, a := range ps[i].Addrs {
			e.str(a)
		}
	}
	return writeFrame(&e, msgPeers)
}

func decodePeers(d *dec) ([]peerAddrMsg, error) {
	n, err := d.u64()
	if err != nil {
		return nil, err
	}
	if n > 1024 {
		return nil, fmt.Errorf("net: слишком много пиров в px: %d", n)
	}
	out := make([]peerAddrMsg, 0, n)
	for i := uint64(0); i < n; i++ {
		var p peerAddrMsg
		if p.ID, err = d.str(); err != nil {
			return nil, err
		}
		na, err := d.u64()
		if err != nil {
			return nil, err
		}
		if na > 64 {
			return nil, fmt.Errorf("net: слишком много адресов: %d", na)
		}
		for j := uint64(0); j < na; j++ {
			a, err := d.str()
			if err != nil {
				return nil, err
			}
			p.Addrs = append(p.Addrs, a)
		}
		out = append(out, p)
	}
	return out, d.done()
}
