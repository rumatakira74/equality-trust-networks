// Fuzz-цели на самописный protobuf-кодек (тема «Fuzz-долги»): недоверенные
// байты wire-сообщений пиров и genesis-пакетов не должны вызывать панику,
// зависание или неограниченное потребление памяти в decode-путях; ошибки
// декодирования возвращаются ошибкой. Seed-корпус: валидные векторы всех
// ветвей payload + граничные случаи (пустой ввод, усечение, переполнение
// varint, вложенная глубина, гигантские заявленные длины).
package wire

import (
	"bytes"
	"testing"
)

// fuzzTx — кодирование транзакции с заданным payload (все прочие поля
// из sampleTx: валидный вектор существующих тестов).
func fuzzTx(t TxType, pl Payload) []byte {
	tx := sampleTx()
	tx.Type = t
	tx.Payload = pl
	return tx.Encode()
}

func FuzzDecodeTransaction(f *testing.F) {
	// валидные векторы: каждая ветвь payload
	f.Add(fuzzTx(Register, &RegisterPayload{
		AttestedPublicKey: []byte("pk"), AttestationChain: []byte("chain"),
		AttestationChallenge: []byte("ch"), AndroidID: "id",
		PersonalGuardians: [][]byte{[]byte("g1"), []byte("g2")}, Synthetic: true,
	}))
	f.Add(fuzzTx(Vouch, &VouchPayload{Candidate: []byte("cand"), Epoch: 7}))
	f.Add(fuzzTx(Mint, &MintPayload{Period: 13, Amount: 1000}))
	f.Add(fuzzTx(Transfer, &TransferPayload{Send: &TransferSend{Recipient: []byte("r"), Amount: 5}}))
	f.Add(fuzzTx(Transfer, &TransferPayload{Cancel: &TransferCancel{PendingSeq: 2}}))
	f.Add(fuzzTx(Transfer, &TransferPayload{SetThreshold: &ThresholdUpdate{NewThreshold: 3000}}))
	f.Add(fuzzTx(Migrate, &MigratePayload{Propose: &MigratePropose{
		OldAddress: []byte("old"),
		Approvals:  []GuardianApproval{{Guardian: []byte("g"), StatementSig: []byte("sig")}},
		Nonce:      1,
	}}))
	f.Add(fuzzTx(Migrate, &MigratePayload{Cancel: &MigrateCancel{NewAddress: []byte("new"), ProposeSeq: 3}}))
	inner := sampleTx()
	inner.Type = Transfer
	f.Add(fuzzTx(Evidence, &EvidencePayload{TxA: inner, Kind: EvidenceEquivocation}))
	f.Add(fuzzTx(GuardianOffer, &GuardianOfferPayload{}))
	f.Add(fuzzTx(PoolCreate, &PoolCreatePayload{
		Recipient: []byte("r"), GoalAmount: 100, DeadlineEpoch: 9,
		ReleaseMode: PoolReleaseOnGoal, RefundMode: PoolRefundIfUnmet,
	}))
	f.Add(fuzzTx(PoolDonate, &PoolDonatePayload{PoolID: []byte("pool"), Amount: 7}))
	f.Add(fuzzTx(PoolClose, &PoolClosePayload{PoolID: []byte("pool")}))
	f.Add(fuzzTx(PoolClaim, &PoolClaimPayload{PoolID: []byte("pool")}))
	f.Add(fuzzTx(Redeem, &RedeemPayload{
		Provider: []byte("p"), ResourceID: "res", Amount: 3,
		Slot: &TimeSlot{Start: 1, End: 2},
	}))
	f.Add(fuzzTx(Receipt, &ReceiptPayload{
		RedeemTxID: []byte("tx"), Amount: 3, ResourceID: "res",
		Slot: &TimeSlot{Start: 1, End: 2},
	}))
	f.Add(fuzzTx(ContributionClaim, &ContributionClaimPayload{
		PoolID: []byte("pool"), DescriptionHash: []byte("h"), Size: 5, Period: 2,
	}))
	f.Add(fuzzTx(ContributionConfirm, &ContributionConfirmPayload{ClaimTxID: []byte("claim")}))
	f.Add(fuzzTx(UpdateGuardians, &UpdateGuardiansPayload{Propose: &UpdateGuardiansPropose{
		NewGuardians: [][]byte{[]byte("g1"), []byte("g2"), []byte("g3")},
		Approvals:    []GuardianApproval{{Guardian: []byte("g1"), StatementSig: []byte("s")}},
		Nonce:        4,
	}}))
	f.Add(fuzzTx(UpdateGuardians, &UpdateGuardiansPayload{Cancel: &UpdateGuardiansCancel{ProposeSeq: 6}}))
	// граничные: пустой ввод, усечение, переполнение varint,
	// гигантская заявленная длина, неизвестное поле
	f.Add([]byte(nil))
	f.Add([]byte{0x0a})
	f.Add(bytes.Repeat([]byte{0xff}, 16))
	f.Add([]byte{0x0a, 0xff, 0xff, 0xff, 0xff, 0x0f})
	f.Add(append(sampleTx().Encode(), 0xf8, 0x06, 0x01)) // unknown-поле 127
	// вложенная глубина: evidence в evidence в ... (20 уровней)
	deep := sampleTx()
	deep.Type = GuardianOffer
	deep.Payload = &GuardianOfferPayload{}
	for i := 0; i < 20; i++ {
		deep = &Transaction{
			NetworkID: []byte("net-1"), Wallet: []byte("w"), Seq: 1,
			Type: Evidence, TimestampClaim: 1, ProtocolVersion: 1,
			Payload: &EvidencePayload{TxA: deep, Kind: EvidenceKind(i % 3)},
		}
	}
	f.Add(deep.Encode())

	f.Fuzz(func(t *testing.T, b []byte) {
		tx, err := DecodeTransaction(b)
		if err != nil {
			if tx != nil {
				t.Fatalf("ошибка с ненулевым результатом: %v", err)
			}
			return
		}
		// успешный разбор: повторное кодирование и хеши без паники
		_ = tx.Encode()
		_ = tx.TxID()
		_ = tx.SigningHash()
	})
}

func FuzzDecodeGenesisPackage(f *testing.F) {
	f.Add(testGenesisPackage().EncodeGenesisPackage())
	// манифест без профиля (неполный пакет)
	f.Add(appendMessageField(nil, 2, testGenesisPackage().Genesis.Encode()))
	// пустой и граничные
	f.Add([]byte(nil))
	f.Add([]byte{0x0a})
	f.Add(bytes.Repeat([]byte{0xff}, 16))
	f.Add([]byte{0x0a, 0xff, 0xff, 0xff, 0xff, 0x0f})

	f.Fuzz(func(t *testing.T, b []byte) {
		gp, err := DecodeGenesisPackage(b)
		if err != nil {
			if gp != nil {
				t.Fatalf("ошибка с ненулевым результатом: %v", err)
			}
			return
		}
		_ = gp.EncodeGenesisPackage()
	})
}

func FuzzDecodeStateLeaf(f *testing.F) {
	leaf := &StateLeaf{
		Seq: 3, Balance: 42, Status: StatusActive,
		Flags:           &WalletFlags{Flagged: true, Synthetic: true, Seed: true},
		RegisteredEpoch: 5, ActivatedEpoch: 6, LastClaimedPeriod: 7,
		DelayedThreshold: 3000, ThresholdChangeEpoch: 9,
	}
	f.Add(leaf.Encode())
	f.Add((&StateLeaf{Seq: 1}).Encode())
	f.Add((&WalletFlags{Flagged: true}).Encode())
	f.Add((&PowProof{Algorithm: PowArgon2id, Nonce: []byte{1, 2, 3}}).Encode())
	f.Add([]byte(nil))
	f.Add([]byte{0x08})
	f.Add(bytes.Repeat([]byte{0xff}, 16))
	f.Add([]byte{0x22, 0xff, 0xff, 0xff, 0xff, 0x0f}) // гигантский flags

	f.Fuzz(func(t *testing.T, b []byte) {
		if l, err := DecodeStateLeaf(b); err == nil {
			_ = l.Encode()
		}
		if fl, err := DecodeWalletFlags(b); err == nil {
			_ = fl.Encode()
		}
		if p, err := DecodePowProof(b); err == nil {
			_ = p.Encode()
		}
	})
}
