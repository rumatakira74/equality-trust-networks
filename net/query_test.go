// Тесты net/query.go (этап 7): конкурентные Apply + read-запросы UI.
// Гоняется с -race: go test -race -run TestQuery ./net/
package net

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/rumatakira74/equality-trust-networks/core/wire"
)

// TestQueryConcurrentApplyRead — data race между фоновым applyLive/SubmitTx
// и read-геттерами UI (причина net/query.go; исключение из правила 3,
// авторизовано 19.10.2026). Писатели публикуют register-транзакции
// (независимые, seq=1), читатели параллельно опрашивают все геттеры.
func TestQueryConcurrentApplyRead(t *testing.T) {
	e := newTestEnv(t)
	n := e.newNode(nodeOpts{level: LevelFull})
	e.start(n)
	ctx := context.Background()

	// начальное состояние: посевной круг уже в состоянии (SeedMembers);
	// выпуск V1 даёт историю и баланс для запросов
	if _, err := n.SubmitTx(ctx, e.mintTx(n, "V1")); err != nil {
		t.Fatalf("mint V1: %v", err)
	}
	addr := e.addr("V1")
	now := uint64(time.Now().Unix())

	var wg sync.WaitGroup
	stop := make(chan struct{})

	// читатели: все геттеры net/query.go по кругу
	for r := 0; r < 4; r++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-stop:
					return
				default:
				}
				n.QueryClock()
				if w := n.QueryWallet(addr); w == nil {
					t.Error("QueryWallet: снапшот V1 пуст")
					return
				}
				if recv, k := n.QueryVouchProgress(addr); k == 0 {
					t.Error("QueryVouchProgress: K=0")
					return
				} else if recv > k {
					t.Error("QueryVouchProgress: recv > K")
					return
				}
				if cap := n.QuerySlotCapacity(addr); cap == 0 {
					t.Error("QuerySlotCapacity: 0 у посевного")
					return
				}
				if h := n.QueryHistory(addr); len(h) == 0 {
					t.Error("QueryHistory: цепочка V1 пуста")
					return
				}
				n.QueryPendingTransfers(addr)
				n.QueryPendingMigrations(addr)
				n.QueryMintWindowOpen(addr, 1, now)
				n.QueryFee(1000)
				if d := n.QueryVouchDepth(addr); d != 0 {
					t.Errorf("QueryVouchDepth: посевной = %d, хочу 0", d)
					return
				}
				if g := n.QueryPublicGuardians(); len(g) != 0 {
					t.Error("QueryPublicGuardians: ожидался пустой список")
					return
				}
				if reg := n.QueryRegistry(); len(reg) < 3 {
					t.Errorf("QueryRegistry: %d кошельков, хочу >= 3", len(reg))
					return
				}
			}
		}()
	}

	// писатели: регистрация новых кошельков (register — seq=1, строится
	// независимо от состояния, параллельные сборки безопасны). Ключи
	// предгенерируются: testEnv.key пишет в общий map (не конкурентен).
	for w := 0; w < 8; w++ {
		id := fmt.Sprintf("race-%d", w)
		_ = e.key(id)
		_ = e.addr(id)
	}
	for w := 0; w < 8; w++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			id := fmt.Sprintf("race-%d", i)
			tx := e.register(n, id)
			if _, err := n.SubmitTx(ctx, tx); err != nil {
				t.Errorf("register %s: %v", id, err)
			}
		}(w)
	}

	// писатели закончили — даём читателям отработать и останавливаем
	time.Sleep(100 * time.Millisecond)
	close(stop)
	wg.Wait()

	// итог: все регистрации приняты
	if _, k := n.QueryVouchProgress(e.addr("race-3")); k == 0 {
		t.Fatal("профиль недоступен")
	} else if n.QueryWallet(e.addr("race-3")) == nil {
		t.Fatalf("кошелёк race-3 не в реестре (K=%d)", k)
	}
}

// TestQueryLightNode — геттеры на лёгком узле (12.5): Ledger нет, история —
// своя цепочка; не паникуют и не блокируются.
func TestQueryLightNode(t *testing.T) {
	e := newTestEnv(t)
	full := e.newNode(nodeOpts{level: LevelFull})
	e.start(full)
	// свежий кошелёк (посевной круг уже в состоянии из genesis — его
	// register отклоняется)
	light := e.newNode(nodeOpts{level: LevelLight, ownWallet: e.addr("L1"),
		manual: manualAddrs(t, full)})
	e.start(light)

	ctx := context.Background()
	if _, err := full.SubmitTx(ctx, e.register(full, "L1")); err != nil {
		t.Fatalf("register L1: %v", err)
	}
	if !waitPeers(t, light, 1, 15*time.Second) {
		t.Fatal("лёгкий узел не подключился")
	}
	if err := light.LightSync(ctx); err != nil {
		t.Fatalf("LightSync: %v", err)
	}
	if got := light.QueryHistory(e.addr("L1")); len(got) == 0 {
		t.Fatal("лёгкий узел не получил свою цепочку")
	} else if got[0].Type != wire.Register {
		t.Fatalf("первая транзакция лёгкой цепочки: %s, хочу register", got[0].Type)
	}
	// Ledger на лёгком узле отсутствует: геттеры возвращают нули, не паникуя
	light.QueryClock()
	if w := light.QueryWallet(e.addr("L1")); w != nil {
		t.Error("QueryWallet на лёгком узле: ожидался nil (нет Ledger)")
	}
	if d := light.QueryVouchDepth(e.addr("L1")); d != -1 {
		t.Errorf("QueryVouchDepth на лёгком узле: %d, хочу -1", d)
	}
	light.QueryPendingTransfers(e.addr("L1"))
	light.QueryPublicGuardians()
	light.QueryRegistry()
}
