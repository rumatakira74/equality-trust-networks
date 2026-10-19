// Обязательный регресс: тестовые векторы этапа 0 (vectors/*.json).
// Каждый вектор даёт предсказанный исход: accept/reject/conflict + код OK/E_*
// и пост-эффекты (балансы, статусы, метки, слоты, pending-операции).
package ledger

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/rumatakira74/equality-trust-networks/core/wire"
)

var coreVectors = []string{
	"valid_register.json",
	"valid_vouch.json",
	"valid_mint.json",
	"valid_transfer.json",
	"valid_migrate.json",
	"valid_evidence.json",
	"valid_guardian_offer.json",
	"bad_equivocation.json",
	"bad_double_mint.json",
	"bad_moratorium.json",
	"bad_accrual_window.json",
	"flow_delayed_cancel.json",
}

func outcomeOf(s string) Outcome {
	switch s {
	case "accept":
		return Accepted
	case "reject":
		return Rejected
	case "conflict":
		return Conflict
	}
	return Outcome(-1)
}

func TestVectors(t *testing.T) {
	for _, f := range coreVectors {
		f := f
		t.Run(f, func(t *testing.T) {
			data, err := os.ReadFile(filepath.Join("..", "..", "vectors", f))
			if err != nil {
				t.Fatalf("чтение вектора: %v", err)
			}
			var vec vector
			if err := json.Unmarshal(data, &vec); err != nil {
				t.Fatalf("разбор вектора: %v", err)
			}
			h := newHarness(t, &vec)

			// конфликтная пара для EVIDENCE (known_conflicting_transactions)
			known := make(map[string]*wire.Transaction)
			if kc := vec.PreState.KnownConflicting; kc != nil {
				if kc.TxA != nil {
					known["tx_a"] = h.buildKnownTx("tx_a", kc.TxA)
				}
				if kc.TxB != nil {
					known["tx_b"] = h.buildKnownTx("tx_b", kc.TxB)
				}
			}

			for i := range vec.Transactions {
				vt := &vec.Transactions[i]
				tx := h.buildTx(vt, known)
				h.builtTxs[vt.ID] = tx
				// ссылки evidence могут указывать и на построенные транзакции
				for k, v := range h.builtTxs {
					if _, ok := known[k]; !ok {
						known[k] = v
					}
				}
				res := h.ledger.Apply(tx)
				exp := vec.Expected[i]
				if res.Outcome != outcomeOf(exp.Outcome) {
					t.Errorf("tx %s: исход %v, ожидается %s (код %s)",
						vt.ID, res.Outcome, exp.Outcome, res.Code)
					return
				}
				if res.Code != exp.Reason {
					t.Errorf("tx %s: код %q, ожидается %q", vt.ID, res.Code, exp.Reason)
					return
				}
				h.checkEffects(t, exp.Effects)
			}
		})
	}
}

// checkEffects — сверка пост-эффектов с состоянием.
func (h *harness) checkEffects(t *testing.T, raw json.RawMessage) {
	if len(raw) == 0 {
		return
	}
	var eff map[string]interface{}
	if err := json.Unmarshal(raw, &eff); err != nil {
		t.Fatalf("effects: %v", err)
	}
	for k, v := range eff {
		h.checkEffect(t, k, v)
	}
}

func (h *harness) walletByID(id string) *Wallet {
	return h.ledger.State.Wallet(h.addr(id))
}

func (h *harness) checkEffect(t *testing.T, k string, v interface{}) {
	// ключи вида "W_X.field"
	var walletID, field string
	if len(k) > 2 && k[0] == 'W' && k[1] == '_' {
		for i := 2; i < len(k); i++ {
			if k[i] == '.' {
				walletID, field = k[:i], k[i+1:]
				break
			}
		}
	}
	if walletID != "" {
		w := h.walletByID(walletID)
		if w == nil {
			t.Errorf("effects: кошелёк %s не найден", walletID)
			return
		}
		switch field {
		case "balance":
			if want, ok := v.(float64); ok {
				if w.TotalBalance() != uint64(want) {
					t.Errorf("effects: %s.balance = %d, ожидается %.0f", walletID, w.TotalBalance(), want)
				}
			}
		case "balance_locked":
			if want, ok := v.(float64); ok {
				if w.Locked != uint64(want) {
					t.Errorf("effects: %s.balance_locked = %d, ожидается %.0f", walletID, w.Locked, want)
				}
			}
		case "seq":
			if want, ok := v.(float64); ok {
				if w.Seq != uint64(want) {
					t.Errorf("effects: %s.seq = %d, ожидается %.0f", walletID, w.Seq, want)
				}
			}
		case "status":
			if want, ok := v.(string); ok {
				if w.Status.String() != want {
					t.Errorf("effects: %s.status = %s, ожидается %s", walletID, w.Status.String(), want)
				}
			}
		case "flags":
			if arr, ok := v.([]interface{}); ok {
				for _, f := range arr {
					if fs, _ := f.(string); fs == "FLAGGED" && !w.Flagged {
						t.Errorf("effects: %s.flagged не установлен", walletID)
					}
				}
			}
		case "last_mint_period":
			if want, ok := v.(float64); ok {
				if w.LastClaimedPeriod != uint64(want) {
					t.Errorf("effects: %s.last_mint_period = %d, ожидается %.0f", walletID, w.LastClaimedPeriod, want)
				}
			}
		case "slots_used":
			if want, ok := v.(float64); ok {
				if uint64(len(w.OccupiedSlots)) != uint64(want) {
					t.Errorf("effects: %s.slots_used = %d, ожидается %.0f", walletID, len(w.OccupiedSlots), want)
				}
			}
		case "mint_suspended_until_period":
			if want, ok := v.(float64); ok {
				if w.MintSuspendedUntilPeriod != uint64(want) {
					t.Errorf("effects: %s.mint_suspended_until_period = %d, ожидается %.0f",
						walletID, w.MintSuspendedUntilPeriod, want)
				}
			}
		case "public_guardian":
			if want, ok := v.(bool); ok {
				if w.PublicGuardian != want {
					t.Errorf("effects: %s.public_guardian = %v, ожидается %v", walletID, w.PublicGuardian, want)
				}
			}
		case "delayed_threshold":
			if want, ok := v.(float64); ok {
				if w.DelayedThreshold != uint64(want) {
					t.Errorf("effects: %s.delayed_threshold = %d, ожидается %.0f", walletID, w.DelayedThreshold, want)
				}
			}
		case "registered_at":
			if want, ok := v.(float64); ok {
				if w.RegisteredAt != uint64(want) {
					t.Errorf("effects: %s.registered_at = %d, ожидается %.0f", walletID, w.RegisteredAt, want)
				}
			}
		case "activators":
			if arr, ok := v.([]interface{}); ok {
				if len(arr) != len(w.Activators) {
					t.Errorf("effects: %s.activators: %d, ожидается %d", walletID, len(w.Activators), len(arr))
					return
				}
				for i, a := range arr {
					id, _ := a.(string)
					if string(w.Activators[i]) != string(h.addr(id)) {
						t.Errorf("effects: %s.activators[%d] не совпадает", walletID, i)
					}
				}
			}
		default:
			// прочие поля кошелька (модульные и пр.) — вне ядра, пропускаем
		}
		return
	}
	switch {
	case k == "burned":
		if want, ok := v.(float64); ok {
			if h.ledger.State.TotalBurned != uint64(want) {
				t.Errorf("effects: burned = %d, ожидается %.0f", h.ledger.State.TotalBurned, want)
			}
		}
	case len(k) > 2 && (k[0] == 'T') && len(k) > 3 && k[1] == '_':
		// "T1.state" / "T1.deadline": pending-перевод
		var txID, field string
		for i := 1; i < len(k); i++ {
			if k[i] == '.' {
				txID, field = k[:i], k[i+1:]
				break
			}
		}
		vt := h.findVecTx(txID)
		if vt == nil {
			t.Errorf("effects: транзакция %s не найдена", txID)
			return
		}
		pt := h.ledger.State.PendingTransfers[ptKey(h.addr(vt.Wallet), vt.Seq)]
		if pt == nil {
			t.Errorf("effects: pending-перевод %s не найден", txID)
			return
		}
		switch field {
		case "state":
			want, _ := v.(string)
			got := "PENDING"
			if pt.Canceled {
				got = "CANCELED"
			} else if pt.Executed {
				got = "EXECUTED"
			}
			if got != want {
				t.Errorf("effects: %s.state = %s, ожидается %s", txID, got, want)
			}
		case "deadline":
			if want, ok := v.(float64); ok {
				if pt.Deadline != uint64(want) {
					t.Errorf("effects: %s.deadline = %d, ожидается %.0f", txID, pt.Deadline, want)
				}
			}
		}
	case k == "migration":
		if m, ok := v.(map[string]interface{}); ok {
			state, _ := m["state"].(string)
			found := false
			for _, pm := range h.ledger.State.PendingMigrations {
				if !pm.Canceled && !pm.Executed {
					found = true
					if enb, ok := m["execute_not_before"].(float64); ok {
						if pm.ExecuteNotBefore != uint64(enb) {
							t.Errorf("effects: migration.execute_not_before = %d, ожидается %.0f",
								pm.ExecuteNotBefore, enb)
						}
					}
				}
			}
			if state == "PENDING" && !found {
				t.Errorf("effects: pending-миграция не найдена")
			}
		}
	default:
		// свободно-текстовые эффекты («ветка», «не меняется», ...) — пропускаем
		_ = fmt.Sprintf
	}
}

func (h *harness) findVecTx(id string) *vecTx {
	for i := range h.vec.Transactions {
		if h.vec.Transactions[i].ID == id {
			return &h.vec.Transactions[i]
		}
	}
	return nil
}
