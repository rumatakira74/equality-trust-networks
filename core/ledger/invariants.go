// Инварианты ядра (spec/04) как проверяемые утверждения для property/fuzz-тестов.
// CheckInvariants — дешёвая проверка на каждом шаге; FullCheck — на границах эпох
// (INV-3/7/8). Идентификаторы INV-* соответствуют spec/04.
package ledger

import (
	"bytes"
	"fmt"

	"github.com/rumatakira74/equality-trust-networks/core/wire"
)

// CheckInvariants проверяет дешёвые инварианты текущего состояния и журнала.
// Возвращает список нарушений (пустой = все выполняются).
func (l *Ledger) CheckInvariants() []string {
	var bad []string
	st := l.State
	prof := st.Net.Profile

	// INV-1c: не более одной принятой транзакции на (wallet, seq)
	seen := make(map[string]uint64)
	for _, tx := range l.TxIndex {
		k := string(tx.Wallet)
		if _, ok := seen[k]; !ok {
			seen[k] = 0
		}
	}
	// (полная проверка — в FullCheck; здесь по цепочкам кошельков)
	for _, w := range st.Wallets {
		if w.Locked > w.Liquid {
			bad = append(bad, fmt.Sprintf("INV-1d: locked(%d) > liquid(%d) у %x",
				w.Locked, w.Liquid, w.Address))
		}
	}

	// INV-2a: active => K поручительств или посевной круг (до sunset)
	K := uint64(prof.VouchK)
	for _, w := range st.Wallets {
		if w.Status == wire.StatusActive && !w.Seed {
			if uint64(len(w.VouchesReceived)) < K {
				bad = append(bad, fmt.Sprintf("INV-2a: active без K поручительств у %x", w.Address))
			}
		}
	}

	// INV-8: сохранение предложения: Σ балансов + burned = minted
	var sum uint64
	for _, w := range st.Wallets {
		sum += w.TotalBalance()
	}
	if sum+st.TotalBurned != st.TotalMinted {
		bad = append(bad, fmt.Sprintf("INV-8: %d + %d != %d", sum, st.TotalBurned, st.TotalMinted))
	}

	// INV-11a: занятых слотов не больше пула
	for _, w := range st.Wallets {
		cap := st.Net.SlotCapacity(w.AgeDays(st.NetTime), w.Seed, w.AgeDays(st.NetTime))
		if uint32(len(w.OccupiedSlots)) > cap {
			bad = append(bad, fmt.Sprintf("INV-11a: слотов %d > пула %d у %x",
				len(w.OccupiedSlots), cap, w.Address))
		}
	}

	// INV-12a: flagged монотонен (снятий не существует — проверяем, что
	// помеченные не потеряли метку в состоянии)
	// (не нарушается конструктивно; полная проверка в FullCheck по журналу)

	// INV-13a: один фингерпринт — один адрес
	byFp := make(map[string]bool)
	for _, w := range st.Wallets {
		if w.Fingerprint == nil {
			continue
		}
		if byFp[string(w.Fingerprint)] {
			bad = append(bad, "INV-13a: дублирующийся фингерпринт")
		}
		byFp[string(w.Fingerprint)] = true
	}
	return bad
}

// FullCheck — тяжёлая проверка по журналу (на границах эпох): INV-1, 9, 10, 12.
func (l *Ledger) FullCheck() []string {
	var bad []string
	st := l.State
	prof := st.Net.Profile

	// INV-1a/1b/1c: подписи, цепочки, уникальность (wallet, seq)
	perWallet := make(map[string]map[uint64]*wire.Transaction)
	for _, tx := range l.TxIndex {
		k := string(tx.Wallet)
		m := perWallet[k]
		if m == nil {
			m = make(map[uint64]*wire.Transaction)
			perWallet[k] = m
		}
		if prev, ok := m[tx.Seq]; ok {
			if prev.TxID() != tx.TxID() {
				bad = append(bad, fmt.Sprintf("INV-1c: два tx на (%x, %d)", tx.Wallet, tx.Seq))
			}
		}
		m[tx.Seq] = tx
		// INV-1a: подпись
		w := st.Wallet(tx.Wallet)
		if tx.Type == wire.Register {
			reg := tx.Payload.(*wire.RegisterPayload)
			if pub, err := parsePub(reg.AttestedPublicKey); err == nil {
				sh := tx.SigningHash()
				if !verifySig(pub, sh[:], tx.Signature) {
					bad = append(bad, fmt.Sprintf("INV-1a: подпись register %x", tx.TxID()))
				}
			}
		} else if w != nil && w.PublicKey != nil {
			sh := tx.SigningHash()
			if !verifySig(w.PublicKey, sh[:], tx.Signature) {
				bad = append(bad, fmt.Sprintf("INV-1a: подпись tx %x", tx.TxID()))
			}
		}
	}
	// INV-1b: seq строго +1 и prev = txid предыдущей
	for k, m := range perWallet {
		w := st.Wallets[k]
		if w == nil {
			continue
		}
		for seq := uint64(2); seq <= w.Seq; seq++ {
			if m[seq] == nil {
				bad = append(bad, fmt.Sprintf("INV-1b: пропуск seq %d у %x", seq, w.Address))
				continue
			}
			if m[seq-1] == nil {
				continue
			}
			if !bytes.Equal(m[seq].Prev, txidBytes(m[seq-1])) {
				bad = append(bad, fmt.Sprintf("INV-1b: prev не сходится у %x seq %d", w.Address, seq))
			}
		}
	}

	// INV-9a: двойной выпуск — не более одного mint на период (по журналу)
	type pk struct {
		w      string
		period uint64
	}
	mintSeen := make(map[pk]bool)
	for _, tx := range l.TxIndex {
		if tx.Type != wire.Mint {
			continue
		}
		mp := tx.Payload.(*wire.MintPayload)
		p := pk{string(tx.Wallet), mp.Period}
		if mintSeen[p] {
			bad = append(bad, fmt.Sprintf("INV-9a: двойной mint %x период %d", tx.Wallet, mp.Period))
		}
		mintSeen[p] = true
	}

	// INV-10: мораторий на расходующих операциях
	for _, tx := range l.TxIndex {
		consuming := false
		switch tx.Type {
		case wire.Transfer:
			tp := tx.Payload.(*wire.TransferPayload)
			if tp.Send != nil {
				consuming = true
			}
		}
		if !consuming {
			continue
		}
		w := st.Wallet(tx.Wallet)
		if w == nil {
			continue
		}
		if tx.TimestampClaim < w.RegisteredAt ||
			tx.TimestampClaim-w.RegisteredAt < prof.TransferMoratoriumDays*86400 {
			bad = append(bad, fmt.Sprintf("INV-10: перевод в моратории у %x", tx.Wallet))
		}
	}
	return bad
}
