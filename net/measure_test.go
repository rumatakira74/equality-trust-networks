// Замеры этапа 4 (4.12): пропускная способность gossip, время полной
// синхронизации N узлов, память узла, размер служебного трафика.
// Числа фиксируются в reports/REPORT-STAGE-4.md (запуск с -v).
package net

import (
	"context"
	"runtime"
	"testing"
	"time"

	"github.com/rumatakira74/trust-networks-platform/core/wire"
)

// TestMeasureGossipThroughput — распространение M транзакций через gossip
// на N узлах (in-process, localhost).
func TestMeasureGossipThroughput(t *testing.T) {
	if testing.Short() {
		t.Skip("замеры пропущены в -short")
	}
	const nodes = 8
	const txs = 30
	e := newTestEnv(t)
	var all []*Node
	first := e.newNode(nodeOpts{level: LevelFull})
	e.start(first)
	all = append(all, first)
	// звезда: все узлы знают адрес первого (каскад: ручной ввод/PX)
	for i := 1; i < nodes; i++ {
		n := e.newNode(nodeOpts{level: LevelFull, manual: manualAddrs(t, first)})
		e.start(n)
		all = append(all, n)
	}
	deadline := time.Now().Add(30 * time.Second)
	for time.Now().Before(deadline) {
		ok := true
		for _, n := range all {
			if len(n.Host.Network().Peers()) < 1 {
				ok = false
			}
		}
		if ok {
			break
		}
		time.Sleep(200 * time.Millisecond)
	}
	time.Sleep(2 * time.Second) // mesh-формирование

	ctx := context.Background()
	// цепочка транзакций от V1: mint + переводы
	if res, err := first.SubmitTx(ctx, e.mintTx(first, "V1")); err != nil || res.Outcome.String() != "accept" {
		t.Fatalf("mint: %v/%s", res.Outcome, res.Code)
	}
	start := time.Now()
	for i := 0; i < txs; i++ {
		tx := e.transferTx(first, "V1", "V2", 1)
		if res, err := first.SubmitTx(ctx, tx); err != nil || res.Outcome.String() != "accept" {
			t.Fatalf("tx %d: %v/%s", i, res.Outcome, res.Code)
		}
	}
	// ждём доставки всем
	deadline = time.Now().Add(120 * time.Second)
	for time.Now().Before(deadline) {
		ok := true
		for _, n := range all[1:] {
			if n.Stats().JournalLen < txs+1 {
				ok = false
			}
		}
		if ok {
			break
		}
		time.Sleep(200 * time.Millisecond)
	}
	elapsed := time.Since(start)
	minLen := 1 << 30
	for _, n := range all {
		if n.Stats().JournalLen < minLen {
			minLen = n.Stats().JournalLen
		}
	}
	t.Logf("Замер gossip: %d узлов, %d транзакций, минимальный журнал %d, доставка за %v (%.1f tx/с по самому медленному узлу)",
		nodes, txs, minLen, elapsed, float64(minLen)/elapsed.Seconds())
	if minLen < txs+1 {
		t.Fatalf("доставка не завершена: %d", minLen)
	}
	// память узла (полный, журнал %d tx)
	var ms runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&ms)
	t.Logf("Замер память: HeapAlloc процесса с %d узлами = %d МБ (%.1f МБ/узел)", nodes, ms.HeapAlloc>>20, float64(ms.HeapAlloc>>20)/nodes)
}

// TestMeasureSyncTime — время полной синхронизации узла с нуля (genesis +
// журнал, INV-3) от узла с K транзакциями.
func TestMeasureSyncTime(t *testing.T) {
	if testing.Short() {
		t.Skip("замеры пропущены в -short")
	}
	const txCount = 60
	e := newTestEnv(t)
	a := e.newNode(nodeOpts{level: LevelFull})
	e.start(a)
	ctx := context.Background()
	if res, err := a.SubmitTx(ctx, e.mintTx(a, "V1")); err != nil || res.Outcome.String() != "accept" {
		t.Fatalf("mint: %v/%s", res.Outcome, res.Code)
	}
	for i := 0; i < txCount-1; i++ {
		tx := e.transferTx(a, "V1", "V2", 1)
		if res, err := a.SubmitTx(ctx, tx); err != nil || res.Outcome.String() != "accept" {
			t.Fatalf("tx %d: %v/%s", i, res.Outcome, res.Code)
		}
	}
	b := e.newNode(nodeOpts{level: LevelFull, manual: manualAddrs(t, a)})
	e.start(b)
	start := time.Now()
	if !waitJournal(t, b, txCount, 60*time.Second) {
		t.Fatalf("синхронизация не завершилась: %d", b.Stats().JournalLen)
	}
	t.Logf("Замер sync: полная сверка %d транзакций с нуля за %v", txCount, time.Since(start))

	// служебный трафик: журнал против полезной нагрузки
	journal := a.journalSnapshot()
	var payload, service int
	for _, tx := range journal {
		payload += len(tx.Encode())
	}
	service = estimateSyncOverhead(journal)
	t.Logf("Замер трафик: полезная нагрузка %d байт (%d tx), служебные кадры sync ~%d байт (%.0f%% сверху)",
		payload, len(journal), service, float64(service)/float64(payload)*100)
}

// estimateSyncOverhead — оценка служебных байтов протокола синхронизации
// (кадры: тип+длина+счётчики+длины blob) для журнала.
func estimateSyncOverhead(journal []*wire.Transaction) int {
	overhead := 0
	for range journal {
		overhead += 5 + 8 + 8 // кадр + длина tx + (упрощённо) varint-обвязка
	}
	return overhead
}
