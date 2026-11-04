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

// FuzzNetCheckpoint — разбор чекпоинта темы gossip (FINDING-1 аудита
// fuzz-debts, D10): decodeCheckpoint с повторяющимися Participants (поле 9)
// и FinalizedTxIDs (поле 12) — счётные пределы вместо разворота заявленного
// счётчика в память.
func FuzzNetCheckpoint(f *testing.F) {
	cp := &wire.Checkpoint{
		NetworkID: []byte("net-1"), Epoch: 1,
		StateRoot:          bytes.Repeat([]byte{1}, 32),
		PrevCheckpointHash: bytes.Repeat([]byte{2}, 32),
		NetworkTime:        1000, ProtocolVersion: 1,
		ActiveWallets: 3, TotalSupply: 1000,
		Participants: [][]byte{bytes.Repeat([]byte{3}, 32), bytes.Repeat([]byte{4}, 32)},
		Threshold:    2,
		ThresholdSig: &wire.ThresholdSignature{Signatures: []wire.SignatureShare{
			{Signer: bytes.Repeat([]byte{3}, 32), Signature: []byte("sig")},
		}},
		FinalizedTxIDs: [][]byte{bytes.Repeat([]byte{5}, 32)},
	}
	f.Add(cp.Encode())
	// граничные: пусто, усечение, неизвестное поле, повторы поля 9
	f.Add([]byte(nil))
	f.Add([]byte{0x0a})
	f.Add([]byte{0x7a, 0x00})
	f.Add(bytes.Repeat([]byte{0x4a, 0x00}, 64))

	f.Fuzz(func(t *testing.T, b []byte) {
		_, _ = decodeCheckpoint(b)
	})
}

// FuzzSeedList — разбор seed-списка (FINDING-1 D10): набор подписей
// (decodeThresholdSig), доля подписи (decodeSignatureShare) и полезная
// нагрузка (parseSeedListPayload). Байты списка приходят из внешних
// каналов (12.3), кадра над ними нет — счётные пределы обязательны.
func FuzzSeedList(f *testing.F) {
	sl := &SeedList{
		NetworkID: []byte("net-1"), Timestamp: 1000,
		Peers: []peerAddrMsg{{ID: "peer-1", Addrs: []string{"/ip4/10.0.0.1/tcp/4001"}}},
	}
	sig := &wire.ThresholdSignature{Signatures: []wire.SignatureShare{
		{Signer: bytes.Repeat([]byte{1}, 32), Signature: []byte("sig")},
	}}
	var e enc
	e.blob(sl.payload())
	e.blob(sig.Encode())
	f.Add(e.b)
	// граничные: пусто, усечение, доля без полей, повторы поля 1
	f.Add([]byte(nil))
	f.Add([]byte{0x01})
	f.Add([]byte{0x0a, 0x00})
	f.Add(bytes.Repeat([]byte{0x0a, 0x00}, 64))

	f.Fuzz(func(t *testing.T, b []byte) {
		_, _ = decodeThresholdSig(b)
		_, _ = decodeSignatureShare(b)
		_, _ = parseSeedListPayload(b)
	})
}

// FuzzVote — протокол голосования (FINDING-1 D10): запрос (msgVoteQuery,
// ключ конфликта) и ответ (msgVoteResp: ключ, ветка, вес, кошелёк,
// подпись) — пределы длин полей до семантики Snowball.
func FuzzVote(f *testing.F) {
	wallet := bytes.Repeat([]byte{7}, 32)
	key := string(wallet) + "|" + string([]byte{0, 0, 0, 0, 0, 0, 0, 1})
	var q enc
	q.str(key)
	f.Add(writeFrame(&q, msgVoteQuery))
	var r enc
	r.str(key)
	r.blob(bytes.Repeat([]byte{8}, 32))
	r.u64(3)
	r.blob(wallet)
	r.blob([]byte("sig"))
	f.Add(writeFrame(&r, msgVoteResp))
	// граничные: пусто, кадр без тела, мусорные длины
	f.Add([]byte(nil))
	f.Add([]byte{msgVoteQuery})
	f.Add([]byte{msgVoteResp, 0x00, 0x00, 0x00, 0x00})
	f.Add(bytes.Repeat([]byte{0xff}, 32))

	f.Fuzz(func(t *testing.T, b []byte) {
		msgType, d, err := readFrame(b)
		if err != nil {
			return
		}
		switch msgType {
		case msgVoteQuery:
			_, _ = parseVoteQuery(d)
		case msgVoteResp:
			_, _, _, _, _, _ = parseVoteResp(d)
		}
	})
}

// FuzzNetTip — ответ tip (FINDING-1 D10, net/sync.go: msgTip): seq + txid
// с пределом длины tip; лёгкий узел разбирает ответ пира.
func FuzzNetTip(f *testing.F) {
	var e enc
	e.u64(7)
	e.blob(bytes.Repeat([]byte{9}, 32))
	f.Add(writeFrame(&e, msgTip))
	// граничные: пусто, пустое тело, tip без длины
	f.Add([]byte(nil))
	f.Add([]byte{msgTip, 0x00, 0x00, 0x00, 0x00})
	f.Add(writeFrame(&enc{}, msgTip))
	f.Add([]byte{msgTip, 0x00, 0x00, 0x00, 0x01, 0xff})

	f.Fuzz(func(t *testing.T, b []byte) {
		msgType, d, err := readFrame(b)
		if err != nil || msgType != msgTip {
			return
		}
		_, _, _ = parseTip(d)
	})
}
