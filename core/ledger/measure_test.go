// Замеры для отчёта этапа 3: размер транзакции против ориентира ~250 байт
// (12.5), скорость apply/верификации, размер БД после сквозного сценария.
package ledger

import (
	"encoding/binary"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/rumatakira74/trust-networks-platform/core/crypto"
	"github.com/rumatakira74/trust-networks-platform/core/store"
	"github.com/rumatakira74/trust-networks-platform/core/wire"
)

func TestMeasurements(t *testing.T) {
	e := newE2E(t)
	// посевной круг
	for _, id := range []string{"V1", "V2", "V3"} {
		priv := e.key(id)
		pub := crypto.MarshalPubKey(&priv.PublicKey)
		fp := crypto.Fingerprint(pub)
		addr := e.addr(id)
		w := &Wallet{
			Address: addr, Fingerprint: fp, PublicKey: &priv.PublicKey,
			Status: wire.StatusActive, Seed: true,
			RegisteredAt:    e.clock.NetTime() - 400*86400,
			ActivatedAt:     e.clock.NetTime() - 400*86400,
			ActivatedPeriod: 1, LastClaimedPeriod: 10,
			LastTouchEpoch: 1, DelayedThreshold: 3000, Seq: 1,
		}
		w.TipTxID = make([]byte, 32)
		copy(w.TipTxID, []byte("seed:"+id))
		e.ledger.State.Wallets[string(addr)] = w
		e.ledger.State.FingerprintIndex[string(fp)] = addr
		e.ledger.State.TotalMinted += 1_000_000
		e.ledger.State.Wallet(e.addr("V1")).Liquid += 1_000_000
	}

	// размер типовой transfer-транзакции
	tx := e.tx("V1", wire.Transfer, &wire.TransferPayload{
		Send: &wire.TransferSend{Recipient: e.addr("V2"), Amount: 500},
	}, false)
	t.Logf("размер transfer-транзакции: %d байт (ориентир ~250, 12.5)", len(tx.Encode()))

	// скорость apply: сначала собираем последовательность (PoW-солв + подпись
	// вне замера), затем применяем с замером только Apply
	const N = 500
	prebuilt := make([]*wire.Transaction, 0, N)
	for i := 0; i < N; i++ {
		tx := e.tx("V1", wire.Transfer, &wire.TransferPayload{
			Send: &wire.TransferSend{Recipient: e.addr("V2"), Amount: 1},
		}, false)
		prebuilt = append(prebuilt, tx)
		// применяем без замера, чтобы обновить tip для следующей сборки
		if res := e.ledger.Apply(tx); res.Outcome != Accepted {
			t.Fatalf("apply %d: %s", i, res.Code)
		}
	}
	// отдельное ядро для чистого замера Apply
	e2 := newE2E(t)
	for _, id := range []string{"V1", "V2", "V3"} {
		priv := e2.key(id)
		pub := crypto.MarshalPubKey(&priv.PublicKey)
		fp := crypto.Fingerprint(pub)
		addr := e2.addr(id)
		w := &Wallet{
			Address: addr, Fingerprint: fp, PublicKey: &priv.PublicKey,
			Status: wire.StatusActive, Seed: true,
			RegisteredAt:    e2.clock.NetTime() - 400*86400,
			ActivatedAt:     e2.clock.NetTime() - 400*86400,
			ActivatedPeriod: 1, LastClaimedPeriod: 10,
			LastTouchEpoch: 1, DelayedThreshold: 3000, Seq: 1,
		}
		w.TipTxID = make([]byte, 32)
		copy(w.TipTxID, []byte("seed:"+id))
		e2.ledger.State.Wallets[string(addr)] = w
		e2.ledger.State.FingerprintIndex[string(fp)] = addr
	}
	e2.ledger.State.TotalMinted += 1_000_000
	e2.ledger.State.Wallet(e2.addr("V1")).Liquid += 1_000_000
	// пересобрать ту же последовательность на чистом ядре и замерить Apply
	seq2 := make([]*wire.Transaction, 0, N)
	for i := 0; i < N; i++ {
		tx := e2.tx("V1", wire.Transfer, &wire.TransferPayload{
			Send: &wire.TransferSend{Recipient: e2.addr("V2"), Amount: 1},
		}, false)
		seq2 = append(seq2, tx)
		if res := e2.ledger.Apply(tx); res.Outcome != Accepted {
			t.Fatalf("e2 apply %d: %s", i, res.Code)
		}
	}
	// третье ядро: применяем готовую последовательность, замер только Apply
	e3 := newE2E(t)
	for _, id := range []string{"V1", "V2", "V3"} {
		priv := e3.key(id)
		pub := crypto.MarshalPubKey(&priv.PublicKey)
		fp := crypto.Fingerprint(pub)
		addr := e3.addr(id)
		w := &Wallet{
			Address: addr, Fingerprint: fp, PublicKey: &priv.PublicKey,
			Status: wire.StatusActive, Seed: true,
			RegisteredAt:    e3.clock.NetTime() - 400*86400,
			ActivatedAt:     e3.clock.NetTime() - 400*86400,
			ActivatedPeriod: 1, LastClaimedPeriod: 10,
			LastTouchEpoch: 1, DelayedThreshold: 3000, Seq: 1,
		}
		w.TipTxID = make([]byte, 32)
		copy(w.TipTxID, []byte("seed:"+id))
		e3.ledger.State.Wallets[string(addr)] = w
		e3.ledger.State.FingerprintIndex[string(fp)] = addr
	}
	e3.ledger.State.TotalMinted += 1_000_000
	e3.ledger.State.Wallet(e3.addr("V1")).Liquid += 1_000_000
	// последовательность e3 строится от tip "seed:V1" — та же, что seq2
	start := time.Now()
	for i, tx := range seq2 {
		_ = i
		// prev цепочки e3 совпадает с seq2 (тот же детерминированный tip)
		if res := e3.ledger.Apply(tx); res.Outcome != Accepted {
			t.Fatalf("e3 apply %d: %s", i, res.Code)
		}
	}
	elapsed := time.Since(start)
	t.Logf("Apply (валидация+эффекты, подпись+PoW уже решены): %d tx за %v = %.1f мкс/tx",
		N, elapsed, float64(elapsed.Microseconds())/N)

	// скорость верификации подписи отдельно
	sh := tx.SigningHash()
	start = time.Now()
	for i := 0; i < 1000; i++ {
		if !crypto.VerifyDER(e.ledger.State.Wallet(e.addr("V1")).PublicKey, sh[:], tx.Signature) {
			t.Fatal("verify")
		}
	}
	t.Logf("верификация ECDSA P-256: %.1f мкс/подпись",
		float64(time.Since(start).Microseconds())/1000)

	// лёгкий PoW: верификация
	pre := tx.SigningPreimage()
	start = time.Now()
	for i := 0; i < 1000; i++ {
		crypto.VerifyLight(pre, tx.Pow.Nonce, e.ledger.State.Net.Pow.TargetLight)
	}
	t.Logf("верификация лёгкого PoW: %.2f мкс",
		float64(time.Since(start).Microseconds())/1000)

	// БД после сценария
	dbPath := filepath.Join(t.TempDir(), "measure.db")
	st, err := store.Open(dbPath)
	if err != nil {
		t.Fatalf("store: %v", err)
	}
	defer st.Close()
	for _, tx := range e.ledger.AcceptedTxs() {
		if err := st.SaveTx(tx, 0); err != nil {
			t.Fatalf("save: %v", err)
		}
	}
	fi, _ := os.Stat(dbPath)
	t.Logf("размер БД после %d транзакций: %d байт (%.1f байт/tx)",
		len(e.ledger.AcceptedTxs()), fi.Size(),
		float64(fi.Size())/float64(len(e.ledger.AcceptedTxs())))

	_ = binary.BigEndian
}
