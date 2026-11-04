// Тесты счётных лимитов разбора недоверенных сообщений (тема fix-d10-fuzz,
// FINDING-1 аудита fuzz-debts): предел срабатывает на мусоре сверх порога,
// валидные сообщения реальных размеров проходят (консенсусное поведение не
// меняется).
package net

import (
	"bytes"
	"testing"

	"github.com/rumatakira74/equality-trust-networks/core/wire"
)

// cpField — поле-блоб чекпоинта: тег (поле, wiretype 2) + длина + данные.
func cpField(field byte, data []byte) []byte {
	out := []byte{field<<3 | 2, byte(len(data))}
	return append(out, data...)
}

// TestDecodeCheckpointLimits — состав подписантов и finalized_txid_list:
// ровно предел проходит, предел+1 отвергается.
func TestDecodeCheckpointLimits(t *testing.T) {
	participant := cpField(9, bytes.Repeat([]byte{1}, 32))
	ok := bytes.Repeat(participant, maxCheckpointParticipants)
	if _, err := decodeCheckpoint(ok); err != nil {
		t.Fatalf("состав на пределе (%d) отвергнут: %v", maxCheckpointParticipants, err)
	}
	if _, err := decodeCheckpoint(append(ok, participant...)); err == nil {
		t.Fatalf("состав сверх предела (%d) принят", maxCheckpointParticipants+1)
	}

	finalized := cpField(12, bytes.Repeat([]byte{2}, 32))
	ok = bytes.Repeat(finalized, maxCheckpointFinalizedTxIDs)
	if _, err := decodeCheckpoint(ok); err != nil {
		t.Fatalf("finalized_txid на пределе (%d) отвергнут: %v", maxCheckpointFinalizedTxIDs, err)
	}
	if _, err := decodeCheckpoint(append(ok, finalized...)); err == nil {
		t.Fatalf("finalized_txid сверх предела (%d) принят", maxCheckpointFinalizedTxIDs+1)
	}
}

// TestDecodeCheckpointValid — валидный чекпоинт реального размера разбирается
// без потерь (регресс-маркер: поля не отбрасываются лимитами).
func TestDecodeCheckpointValid(t *testing.T) {
	cp := &wire.Checkpoint{
		NetworkID: []byte("net-1"), Epoch: 7,
		StateRoot:          bytes.Repeat([]byte{1}, 32),
		PrevCheckpointHash: bytes.Repeat([]byte{2}, 32),
		NetworkTime:        1000, ProtocolVersion: 1,
		ActiveWallets: 3, TotalSupply: 5000,
		Participants: [][]byte{bytes.Repeat([]byte{3}, 32), bytes.Repeat([]byte{4}, 32)},
		Threshold:    2,
		ThresholdSig: &wire.ThresholdSignature{Signatures: []wire.SignatureShare{
			{Signer: bytes.Repeat([]byte{3}, 32), Signature: []byte("sig")},
		}},
		FinalizedTxIDs: [][]byte{bytes.Repeat([]byte{5}, 32), bytes.Repeat([]byte{6}, 32)},
	}
	got, err := decodeCheckpoint(cp.Encode())
	if err != nil {
		t.Fatalf("валидный чекпоинт отвергнут: %v", err)
	}
	if got.Epoch != cp.Epoch || got.Threshold != cp.Threshold ||
		len(got.Participants) != len(cp.Participants) ||
		len(got.FinalizedTxIDs) != len(cp.FinalizedTxIDs) ||
		got.ThresholdSig == nil || len(got.ThresholdSig.Signatures) != 1 {
		t.Fatalf("чекпоинт разобран неполно: %+v", got)
	}
}

// TestDecodeThresholdSigLimit — подписи seed-списка: предел срабатывает в
// цикле, до накопления списка.
func TestDecodeThresholdSigLimit(t *testing.T) {
	share := (&wire.SignatureShare{
		Signer: bytes.Repeat([]byte{1}, 32), Signature: []byte("sig"),
	}).Encode()
	one := append([]byte{0x0a, byte(len(share))}, share...)

	ok := bytes.Repeat(one, maxSeedSignatures)
	if _, err := decodeThresholdSig(ok); err != nil {
		t.Fatalf("подписи на пределе (%d) отвергнуты: %v", maxSeedSignatures, err)
	}
	if _, err := decodeThresholdSig(append(ok, one...)); err == nil {
		t.Fatalf("подписи сверх предела (%d) приняты", maxSeedSignatures+1)
	}
}

// TestDecodeSignatureShareFieldLimit — повтор одиночных полей доли подписи.
func TestDecodeSignatureShareFieldLimit(t *testing.T) {
	signer := []byte{0x0a, 0x01, 0x07} // поле 1 (подписант), 1 байт
	sig := []byte{0x12, 0x01, 0x08}    // поле 2 (подпись), 1 байт

	ok := append(bytes.Repeat(signer, maxSeedShareFields-1), sig...)
	if _, err := decodeSignatureShare(ok); err != nil {
		t.Fatalf("доля на пределе полей (%d) отвергнута: %v", maxSeedShareFields, err)
	}
	if _, err := decodeSignatureShare(append(ok, signer...)); err == nil {
		t.Fatalf("доля сверх предела полей (%d) принята", maxSeedShareFields+1)
	}
}

// TestParseSeedListAddrsTotalLimit — общий объём адресов списка.
func TestParseSeedListAddrsTotalLimit(t *testing.T) {
	build := func(peers int) []byte {
		var e enc
		e.u64(1)
		e.blob([]byte("net-1"))
		e.u64(1000)
		e.u64(uint64(peers))
		for i := 0; i < peers; i++ {
			e.str("peer")
			e.u64(64)
			for j := 0; j < 64; j++ {
				e.str("a")
			}
		}
		return e.b
	}
	if _, err := parseSeedListPayload(build(maxSeedAddrsTotal / 64)); err != nil {
		t.Fatalf("список с %d адресами отвергнут: %v", maxSeedAddrsTotal, err)
	}
	if _, err := parseSeedListPayload(build(maxSeedAddrsTotal/64 + 1)); err == nil {
		t.Fatalf("список с %d адресами принят", maxSeedAddrsTotal+64)
	}
}

// TestVoteParseLimits — длины полей запроса и ответа голосования.
func TestVoteParseLimits(t *testing.T) {
	var e enc
	e.str(string(bytes.Repeat([]byte{'k'}, maxVoteKeyLen)))
	if _, err := parseVoteQuery(&dec{b: e.b}); err != nil {
		t.Fatalf("ключ на пределе отвергнут: %v", err)
	}
	e = enc{}
	e.str(string(bytes.Repeat([]byte{'k'}, maxVoteKeyLen+1)))
	if _, err := parseVoteQuery(&dec{b: e.b}); err == nil {
		t.Fatal("ключ сверх предела принят")
	}

	for _, tc := range []struct {
		name string
		pref int
		sig  int
	}{
		{"ветка", maxVotePrefLen + 1, 8},
		{"подпись", 32, maxVoteSigLen + 1},
	} {
		var r enc
		r.str("k")
		r.blob(bytes.Repeat([]byte{'p'}, tc.pref))
		r.u64(1)
		r.blob(bytes.Repeat([]byte{'w'}, 32))
		r.blob(bytes.Repeat([]byte{'s'}, tc.sig))
		if _, _, _, _, _, err := parseVoteResp(&dec{b: r.b}); err == nil {
			t.Fatalf("ответ голосования (%s) сверх предела принят", tc.name)
		}
	}

	var r enc
	r.str("k")
	r.blob(bytes.Repeat([]byte{'p'}, 32))
	r.u64(1)
	r.blob(bytes.Repeat([]byte{'w'}, 32))
	r.blob(bytes.Repeat([]byte{'s'}, 8))
	if _, _, _, _, _, err := parseVoteResp(&dec{b: r.b}); err != nil {
		t.Fatalf("валидный ответ голосования отвергнут: %v", err)
	}
}

// TestParseTipLimit — длина tip: 32 байта (txid) и пустой проходят, сверх
// предела — отказ.
func TestParseTipLimit(t *testing.T) {
	var e enc
	e.u64(3)
	e.blob(bytes.Repeat([]byte{1}, maxTipLen))
	if seq, tip, err := parseTip(&dec{b: e.b}); err != nil || seq != 3 || len(tip) != maxTipLen {
		t.Fatalf("tip на пределе: seq=%d len=%d err=%v", seq, len(tip), err)
	}

	e = enc{}
	e.u64(3)
	e.blob(bytes.Repeat([]byte{1}, maxTipLen+1))
	if _, _, err := parseTip(&dec{b: e.b}); err == nil {
		t.Fatal("tip сверх предела принят")
	}

	e = enc{}
	e.u64(3)
	e.blob(bytes.Repeat([]byte{1}, 32))
	if _, _, err := parseTip(&dec{b: e.b}); err != nil {
		t.Fatalf("валидный tip отвергнут: %v", err)
	}

	e = enc{}
	e.u64(0)
	e.blob(nil)
	if _, _, err := parseTip(&dec{b: e.b}); err != nil {
		t.Fatalf("пустой tip отвергнут: %v", err)
	}
}
