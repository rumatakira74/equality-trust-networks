// Уровни узлов (12.2, 4.8): полный / лёгкий, переключение по условиям
// (зарядка + Wi-Fi). Android-триггеры — этап 6; здесь — интерфейс и
// headless-конфиг (ручное переключение / предикат).
package net

import "time"

// LevelSwitcher — решает, должен ли узел быть полным (12.2: зарядка + Wi-Fi,
// foreground service). Реализация триггеров — этап 6 (Android).
type LevelSwitcher interface {
	// ShouldBeFull — true, если условия позволяют быть полным узлом.
	ShouldBeFull() bool
}

// HeadlessSwitcher — ручное/предикатное переключение для headless-узлов.
type HeadlessSwitcher struct {
	// Pred — предикат; nil — всегда полный.
	Pred func() bool
}

func (s *HeadlessSwitcher) ShouldBeFull() bool {
	if s == nil || s.Pred == nil {
		return true
	}
	return s.Pred()
}

// FixedSwitcher — фиксированный уровень (конфиг).
type FixedSwitcher bool

func (f FixedSwitcher) ShouldBeFull() bool { return bool(f) }

// Snapshot — снапшот состояния из чекпоинта (12.5: снапшоты = state
// commitments — корень Merkle-дерева в чекпоинте). Реальные чекпоинты —
// этап 5; здесь — интерфейс.
type Snapshot struct {
	Epoch     uint64
	StateRoot []byte
	// JournalPos — позиция журнала (id в store), с которой продолжать
	// догон после снапшота.
	JournalPos uint64
}

// SnapshotSource — источник снапшотов из чекпоинтов (интерфейс; этап 5).
type SnapshotSource interface {
	// SnapshotAt возвращает снапшот на чекпоинте эпохи (false — нет).
	SnapshotAt(epoch uint64) (Snapshot, bool)
	// LatestSnapshot — последний доступный снапшот.
	LatestSnapshot() (Snapshot, bool)
}

// NoSnapshots — заглушка до этапа 5: снапшотов нет, синхронизация —
// всегда полная сверка из genesis + журнала (INV-3).
type NoSnapshots struct{}

func (NoSnapshots) SnapshotAt(uint64) (Snapshot, bool) { return Snapshot{}, false }
func (NoSnapshots) LatestSnapshot() (Snapshot, bool)   { return Snapshot{}, false }

// pollTicker — интервал опроса (для лёгкого узла и переключателя).
func pollTicker(d time.Duration) *time.Ticker {
	if d <= 0 {
		d = 30 * time.Second
	}
	return time.NewTicker(d)
}
