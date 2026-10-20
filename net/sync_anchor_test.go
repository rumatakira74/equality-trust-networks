// Тесты позиционного якоря сверки (№41 — блокер 8.1, этап 9):
// инкрементальное продолжение SyncJournalFrom по позиции в нумерации
// журнала ПИРА с подтверждением границы txid; выравнивание при расхождении
// истории (перестройка/другой порядок вставки); ранние пропуски не
// теряются (первая сверка с пиром — полный проход).
package net

import (
	"context"
	"github.com/rumatakira74/equality-trust-networks/core/wire"
	"testing"
	"time"
)

// waitFor — поллинг условия с дедлайном (фон-сверка после identify).
func waitFor(t *testing.T, d time.Duration, what string, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(d)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatalf("не дождались: %s", what)
}

// anchorOf — сохранённый позиционный якорь пира (для проверок).
func (n *Node) anchorOf(p string) *syncAnchor {
	n.mu.RLock()
	defer n.mu.RUnlock()
	for k, a := range n.syncAnchors {
		if k.String() == p {
			return a
		}
	}
	return nil
}

// TestSyncAnchorContinuation — позиционное продолжение: первая сверка
// проходит журнал целиком (corroboration всем журналом этим пиром), якорь
// фиксируется на границе; дальнейшие сверки переносят только новые
// транзакции — трафик не растёт с журналом (№41).
func TestSyncAnchorContinuation(t *testing.T) {
	e := newTestEnv(t)
	n1 := e.newNode(nodeOpts{level: LevelFull})
	n1.applyLive(e.mintTx(n1, "V1"))
	n1.applyLive(e.register(n1, "CAND"))
	n1.applyLive(e.transferTx(n1, "V1", "V2", 500))
	e.start(n1)
	n2 := e.newNode(nodeOpts{level: LevelFull, manual: manualAddrs(t, n1)})
	e.start(n2)

	// первая сверка (фон после identify или явная) — полный проход
	waitFor(t, 10*time.Second, "журнал после первой сверки", func() bool {
		return len(n2.journalSnapshot()) == 3
	})
	a := n2.anchorOf(n1.Host.ID().String())
	if a == nil || a.pos != 3 || a.txid == nil {
		t.Fatalf("якорь после первой сверки: %+v, хочу pos=3 с txid", a)
	}

	// в журнале пира две новые транзакции
	n1.applyLive(e.mintTx(n1, "V2"))
	n1.applyLive(e.transferTx(n1, "V1", "V2", 100))

	// сверка-продолжение: только новые транзакции
	k, err := n2.SyncFromPeer(context.Background(), n1.Host.ID())
	if err != nil {
		t.Fatalf("сверка-продолжение: %v", err)
	}
	if k != 2 {
		t.Fatalf("сверка-продолжение обработала %d, хочу 2 (только новые)", k)
	}
	a = n2.anchorOf(n1.Host.ID().String())
	if a == nil || a.pos != 5 {
		t.Fatalf("якорь после продолжения: %+v, хочу pos=5", a)
	}
	// повторная сверка без изменений — пустая
	if k, _ := n2.SyncFromPeer(context.Background(), n1.Host.ID()); k != 0 {
		t.Fatalf("сверка без изменений обработала %d, хочу 0", k)
	}
	// состояния сходятся
	r1, r2 := mustRoot(t, n1), mustRoot(t, n2)
	if string(r1) != string(r2) {
		t.Fatalf("state_root разошёлся: %x != %x", r1, r2)
	}
}

// TestSyncAnchorRealignsOnDivergence — корректность якоря: пир перестроил
// журнал (rebuildToFinality меняет порядок вставки) — txid на границе не
// совпадает, сверка выравнивается полным проходом (дубликаты отсеиваются
// дедупом), якорь переустанавливается, состояние не расходится (№41).
func TestSyncAnchorRealignsOnDivergence(t *testing.T) {
	e := newTestEnv(t)
	n1 := e.newNode(nodeOpts{level: LevelFull})
	n1.applyLive(e.mintTx(n1, "V1"))
	n1.applyLive(e.register(n1, "CAND"))
	n1.applyLive(e.mintTx(n1, "V2"))
	e.start(n1)
	n2 := e.newNode(nodeOpts{level: LevelFull, manual: manualAddrs(t, n1)})
	e.start(n2)

	waitFor(t, 10*time.Second, "журнал после первой сверки", func() bool {
		return len(n2.journalSnapshot()) == 3
	})
	rootBefore := mustRoot(t, n2)

	// перестройка у пира: порядок вставки разошёлся (транзакции разных
	// кошельков меняются местами). Новый срез, как при rebuildToFinality
	// (продукт никогда не мутирует элементы журнала на месте — читатели
	// держат заголовок среза под RLock, элементы читают после Unlock)
	n1.mu.Lock()
	j := n1.journal
	n1.journal = []*wire.Transaction{j[0], j[2], j[1]}
	n1.mu.Unlock()

	// якорь не сходится — выравнивание полным проходом
	k, err := n2.SyncFromPeer(context.Background(), n1.Host.ID())
	if err != nil {
		t.Fatalf("сверка после расхождения: %v", err)
	}
	if k != 3 {
		t.Fatalf("сверка после расхождения обработала %d, хочу 3 (выравнивание)", k)
	}
	// якорь переустановлен на новой нумерации пира
	a := n2.anchorOf(n1.Host.ID().String())
	if a == nil || a.pos != 3 || a.txid == nil {
		t.Fatalf("якорь после выравнивания: %+v, хочу pos=3 с txid", a)
	}
	// дальнейшие сверки снова инкрементальны
	if k, _ := n2.SyncFromPeer(context.Background(), n1.Host.ID()); k != 0 {
		t.Fatalf("сверка после выравнивания обработала %d, хочу 0", k)
	}
	// состояние не разошлось
	if r := mustRoot(t, n2); string(r) != string(rootBefore) {
		t.Fatalf("state_root изменился при выравнивании: %x -> %x", rootBefore, r)
	}
	if r := mustRoot(t, n1); string(r) != string(rootBefore) {
		t.Fatalf("state_root узлов разошёлся: %x != %x", r, rootBefore)
	}
}

// TestSyncAnchorDeliversEarlyGap — исходный дефект позиционной сверки:
// узел имеет «среднюю» транзакцию журнала (позиция 1 у пира), но не имеет
// ранней (позиция 0). Прежняя семантика «после своей длины» пропускала
// ранние транзакции навсегда; якорь №41 (первая сверка с пиром — с
// начала) доставляет пробел (№41).
func TestSyncAnchorDeliversEarlyGap(t *testing.T) {
	e := newTestEnv(t)
	n1 := e.newNode(nodeOpts{level: LevelFull})
	a := e.mintTx(n1, "V1")
	b := e.register(n1, "CAND")
	c := e.mintTx(n1, "V2")
	n1.applyLive(a)
	n1.applyLive(b)
	n1.applyLive(c)

	// n2 имеет только среднюю транзакцию (доставлена мимо сверки):
	// журнал длины 1, у пира ранние позиции ему неизвестны
	e.start(n1)
	n2 := e.newNode(nodeOpts{level: LevelFull, manual: manualAddrs(t, n1)})
	n2.applyLive(b)
	e.start(n2)

	// сверка доставляет ранний пропуск: якорь первой сверки — с начала,
	// полный проход по журналу пира (раньше «после своей длины» = 1
	// пропускал позицию 0 навсегда)
	waitFor(t, 10*time.Second, "сверка с ранним пропуском", func() bool {
		return len(n2.journalSnapshot()) == 3
	})
	r1, r2 := mustRoot(t, n1), mustRoot(t, n2)
	if string(r1) != string(r2) {
		t.Fatalf("state_root разошёлся (ранний пропуск не доставлен): %x != %x", r1, r2)
	}
}
