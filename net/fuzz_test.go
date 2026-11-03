// Fuzz-цель на кадры служебных сообщений синхронизации и peer exchange
// (тема «Fuzz-долги»): readFrame + разбор тел всех типов сообщений —
// зеркально диспетчеру serveSync. Недоверенные байты пира не должны
// паниковать или зависать; повреждённые кадры — errFrame.
package net

import (
	"bytes"
	"testing"

	"github.com/rumatakira74/equality-trust-networks/core/wire"
)

func FuzzNetFrame(f *testing.F) {
	// валидные векторы: ответы всех декодируемых типов
	f.Add(encodeHeaders([]checkpointHeaderMsg{
		{Epoch: 1, Hash: []byte("h1"), StateRoot: []byte("r1")},
		{Epoch: 2, Hash: []byte("h2"), StateRoot: []byte("r2")},
	}))
	f.Add(encodeProof(&balanceProofMsg{
		Leaf: []byte("leaf"), LeafData: []byte("data"),
		Siblings: [][]byte{[]byte("s1"), []byte("s2")},
		Root:     []byte("root"), Seq: 7,
	}))
	f.Add(encodePeers([]peerAddrMsg{
		{ID: "peer-1", Addrs: []string{"/ip4/10.0.0.1/tcp/4001"}},
	}))
	tx := &wire.Transaction{
		NetworkID: []byte("net-1"), Wallet: []byte("w"), Seq: 1,
		Type: wire.Transfer, TimestampClaim: 1, ProtocolVersion: 1,
		Payload: &wire.TransferPayload{Send: &wire.TransferSend{Recipient: []byte("r"), Amount: 5}},
	}
	f.Add(encodeTxsMsg(msgTxs, []*wire.Transaction{tx}))
	f.Add(encodeTxsMsg(msgJournal, []*wire.Transaction{tx}))
	// валидные запросы (u64/blob-пути)
	var e enc
	e.u64(3)
	e.blob([]byte("anchor-32-bytes-xxxxxxxxxxxxxxxx"))
	f.Add(writeFrame(&e, msgGetJournal))
	var e2 enc
	e2.blob([]byte("wallet"))
	e2.u64(1)
	e2.u64(9)
	f.Add(writeFrame(&e2, msgGetWalletRange))
	var e3 enc
	e3.blob([]byte("addr"))
	f.Add(writeFrame(&e3, msgGetProof))
	var e4 enc
	e4.u64(5)
	f.Add(writeFrame(&e4, msgGetHeaders))
	f.Add(writeFrame(&enc{}, msgGetPeers))
	// граничные: короткий кадр, рассинхрон длины, пустое тело
	f.Add([]byte(nil))
	f.Add([]byte{msgHeaders})
	f.Add([]byte{msgHeaders, 0xff, 0x00, 0x00, 0x00})
	f.Add([]byte{msgHeaders, 0x00, 0x00, 0x00, 0x00})
	f.Add([]byte{msgPeers, 0x00, 0x00, 0x00, 0x01, 0xff})
	f.Add(bytes.Repeat([]byte{0xff}, 32))

	f.Fuzz(func(t *testing.T, b []byte) {
		msgType, d, err := readFrame(b)
		if err != nil {
			return
		}
		switch msgType {
		case msgHeaders:
			_, _ = decodeHeaders(d)
		case msgProof:
			_, _ = decodeProof(d)
		case msgPeers:
			_, _ = decodePeers(d)
		case msgTxs, msgJournal:
			_, _ = decodeTxsMsg(d)
		case msgGetJournal:
			_, _ = d.u64()
			_, _ = d.blob()
		case msgGetWalletRange:
			_, _ = d.blob()
			_, _ = d.u64()
			_, _ = d.u64()
		case msgGetTip, msgGetProof, msgGetIncoming:
			_, _ = d.blob()
		case msgGetHeaders:
			_, _ = d.u64()
		case msgGetPeers:
			_ = d.done()
		}
	})
}
