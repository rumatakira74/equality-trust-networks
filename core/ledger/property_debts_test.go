// Property-тесты темы «Fuzz-долги»: пробелы относительно property_test.go —
// рандомизированный поток с ВКЛЮЧЁННЫМ демереджем (базовый тест выключает
// его), консистентность индексов состояния/журнала после каждого шага и
// детерминированная проверка правил демереджа (формула, floor, идемпотентность).
package ledger

import (
	"math/rand"
	"testing"

	"github.com/rumatakira74/equality-trust-networks/core/wire"
)

// checkIndexConsistency — консистентность индексов ядра после шага:
// TxIndex соответствует цепочкам кошельков, ключи TxIndex — txid,
// фингерпринт-индекс сходится с кошельками.
func checkIndexConsistency(t *testing.T, l *Ledger, step int) {
	t.Helper()
	st := l.State
	// ключ TxIndex — txid значения; каждый tx относится к известному кошельку
	perWallet := make(map[string]map[uint64]bool)
	for key, tx := range l.TxIndex {
		id := tx.TxID()
		if key != string(id[:]) {
			t.Fatalf("шаг %d: ключ TxIndex не txid", step)
		}
		if st.Wallet(tx.Wallet) == nil {
			t.Fatalf("шаг %d: tx %x неизвестного кошелька", step, tx.Wallet)
		}
		m := perWallet[string(tx.Wallet)]
		if m == nil {
			m = make(map[uint64]bool)
			perWallet[string(tx.Wallet)] = m
		}
		if m[tx.Seq] {
			t.Fatalf("шаг %d: два tx на (%x, %d) в TxIndex", step, tx.Wallet, tx.Seq)
		}
		m[tx.Seq] = true
	}
	// цепочка кошелька: seq 2..Seq покрыты ровно один раз (seq 1 у
	// пред-сеяных кошельков введён напрямую, без tx)
	for addr, w := range st.Wallets {
		m := perWallet[addr]
		for seq := uint64(2); seq <= w.Seq; seq++ {
			if !m[seq] {
				t.Fatalf("шаг %d: пропуск seq %d у %x", step, seq, w.Address)
			}
		}
		for seq := range m {
			if seq < 2 || seq > w.Seq {
				t.Fatalf("шаг %d: tx seq %d вне цепочки %x (Seq %d)", step, seq, w.Address, w.Seq)
			}
		}
		// фингерпринт-индекс: один фингерпринт — один адрес (INV-13a)
		if w.Fingerprint != nil {
			if fpAddr, ok := st.FingerprintIndex[string(w.Fingerprint)]; !ok || string(fpAddr) != addr {
				t.Fatalf("шаг %d: фингерпринт-индекс расходится у %x", step, w.Address)
			}
		}
	}
	// сгоревшее не превышает выпущенного (нет underflow-обёртки)
	if st.TotalBurned > st.TotalMinted {
		t.Fatalf("шаг %d: burned %d > minted %d", step, st.TotalBurned, st.TotalMinted)
	}
}

// TestPropertyDemurrageRandomStream — случайный поток операций с включённым
// демереджем: инварианты после каждого шага, сеттлемент на границах периодов,
// FullCheck на границах эпох, консистентность индексов после каждого шага.
func TestPropertyDemurrageRandomStream(t *testing.T) {
	env := newPropEnv(t, 6)
	prof := &env.ledger.State.Net.Profile
	prof.DemurrageRateMicros = 10_000 // 1% за период
	prof.DemurrageFloor = 1000
	ids := []string{"A", "B", "C", "D", "E", "F"}
	rng := rand.New(rand.NewSource(2024))
	period := prof.PeriodSecsOrDefault()

	seqOf := func(id string) uint64 { return env.ledger.State.Wallet(env.addr(id)).Seq + 1 }

	for step := 0; step < 300; step++ {
		from := ids[rng.Intn(len(ids))]
		to := ids[rng.Intn(len(ids))]
		var res Result
		switch rng.Intn(8) {
		case 0, 1, 2: // перевод
			amount := uint64(rng.Intn(5000))
			res = env.ledger.Apply(env.transferTx(from, to, seqOf(from), amount))
		case 3, 4: // выпуск
			p := uint64(13 + rng.Intn(3))
			res = env.ledger.Apply(env.mintTx(from, seqOf(from), p, 1000))
		case 5: // отложенный перевод (выше P) — pending, исполняется на flush
			res = env.ledger.Apply(env.transferTx(from, to, seqOf(from), 9000))
		case 6: // конфликтная ветка: тот же seq
			seq := seqOf(from)
			t1 := env.transferTx(from, to, seq, 100)
			res = env.ledger.Apply(t1)
			if res.Outcome == Accepted {
				t2 := env.transferTx(from, ids[rng.Intn(len(ids))], seq, 200)
				if res2 := env.ledger.Apply(t2); res2.Outcome != Conflict {
					t.Fatalf("шаг %d: конфликтная ветка: %v %s", step, res2.Outcome, res2.Code)
				}
			}
		default: // невалидный seq
			tx := env.transferTx(from, to, seqOf(from)+5, 10)
			res = env.ledger.Apply(tx)
			if res.Code != "E_SEQ" {
				t.Fatalf("шаг %d: ожидается E_SEQ, получен %s", step, res.Code)
			}
		}
		_ = res
		if bad := env.ledger.CheckInvariants(); len(bad) > 0 {
			t.Fatalf("шаг %d: инварианты нарушены: %v", step, bad)
		}
		checkIndexConsistency(t, env.ledger, step)
		// каждые 25 шагов — граница периода: сеттлемент с демереджем
		if step%25 == 24 {
			env.now += period
			env.ledger.settleTo(env.now)
			if bad := env.ledger.CheckInvariants(); len(bad) > 0 {
				t.Fatalf("шаг %d (сеттлемент): инварианты нарушены: %v", step, bad)
			}
			checkIndexConsistency(t, env.ledger, step)
		}
		// раз в 50 шагов — граница эпохи: FullCheck
		if step%50 == 49 {
			if bad := env.ledger.FullCheck(); len(bad) > 0 {
				t.Fatalf("шаг %d (граница эпохи): FullCheck: %v", step, bad)
			}
		}
	}
}

// TestPropertyDemurrageRules — детерминированные правила демереджа (10.3,
// №31): списание (base − floor) × rate за период, base = balance − locked,
// floor и ниже не трогаются, повторный сеттлемент в тот же период
// идемпотентен, сгоревшее попадает в TotalBurned (INV-8).
func TestPropertyDemurrageRules(t *testing.T) {
	env := newPropEnv(t, 0)
	prof := &env.ledger.State.Net.Profile
	const rate = uint64(10_000) // 1%
	const floor = uint64(1000)
	prof.DemurrageRateMicros = rate
	prof.DemurrageFloor = floor
	period := prof.PeriodSecsOrDefault()

	env.registerWalletState("A", wire.StatusActive, 50_000) // выше floor
	env.registerWalletState("B", wire.StatusActive, floor)  // ровно floor
	env.registerWalletState("C", wire.StatusActive, 500)    // ниже floor

	// довести сеттлемент до текущего периода (пред-сеяные балансы
	// «исторически выпущены» — демередж прошлых периодов законен)
	env.ledger.settleTo(env.now)
	if bad := env.ledger.CheckInvariants(); len(bad) > 0 {
		t.Fatalf("после начального сеттлемента: %v", bad)
	}
	a := env.ledger.State.Wallet(env.addr("A"))
	b := env.ledger.State.Wallet(env.addr("B"))
	c := env.ledger.State.Wallet(env.addr("C"))
	baseA := a.TotalBalance() - a.Locked
	if baseA <= floor {
		t.Fatalf("баланс A (%d) не выше floor — тест не имеет смысла", baseA)
	}
	if b.TotalBalance() != floor || c.TotalBalance() != 500 {
		t.Fatalf("floor-кошельки изменились до периода: B=%d C=%d", b.TotalBalance(), c.TotalBalance())
	}
	burnedBefore := env.ledger.State.TotalBurned

	// один период вперёд: ровно одно списание по формуле
	env.now += period
	env.ledger.settleTo(env.now)
	wantBurn := (baseA - floor) * rate / 1_000_000
	if got := a.TotalBalance(); got != baseA-wantBurn {
		t.Fatalf("демередж A: %d, ожидается %d (base %d, burn %d)",
			got, baseA-wantBurn, baseA, wantBurn)
	}
	if env.ledger.State.TotalBurned != burnedBefore+wantBurn {
		t.Fatalf("TotalBurned: %d, ожидается %d (+%d)",
			env.ledger.State.TotalBurned, burnedBefore, wantBurn)
	}
	if b.TotalBalance() != floor {
		t.Fatalf("кошелёк на floor списан: %d", b.TotalBalance())
	}
	if c.TotalBalance() != 500 {
		t.Fatalf("кошелёк ниже floor списан: %d", c.TotalBalance())
	}
	if bad := env.ledger.CheckInvariants(); len(bad) > 0 {
		t.Fatalf("после демереджа: %v", bad)
	}

	// идемпотентность: повторный сеттлемент в том же периоде не списывает
	before := a.TotalBalance()
	env.ledger.settleTo(env.now)
	if a.TotalBalance() != before {
		t.Fatalf("повторный сеттлемент списал: %d -> %d", before, a.TotalBalance())
	}
	// время внутри того же периода — тоже не списывает
	env.ledger.settleTo(env.now + period/2)
	if a.TotalBalance() != before {
		t.Fatalf("сеттлемент внутри периода списал: %d -> %d", before, a.TotalBalance())
	}
	// следующий период — снова ровно одна формула
	env.now += period
	env.ledger.settleTo(env.now)
	base2 := before
	want2 := (base2 - floor) * rate / 1_000_000
	if got := a.TotalBalance(); got != base2-want2 {
		t.Fatalf("демередж A (2-й период): %d, ожидается %d", got, base2-want2)
	}
	if bad := env.ledger.CheckInvariants(); len(bad) > 0 {
		t.Fatalf("после 2-го периода: %v", bad)
	}
}
