// Сетевое время и эпохи (9.2): источник правды — эпохи чекпоинтов, не
// медиана timestamps. Сетевое время узла ограничено последним
// финализированным чекпоинтом: [cp.time, cp.time + epoch + сдвиг].
// Календарь UTC: «1-е число месяца» = первая эпоха, начавшаяся после
// 00:00 UTC 1-го числа (9.2) — период выпуска меняется только на
// границе эпохи.
package cons

import (
	"sync"
	"time"

	"github.com/rumatakira74/equality-trust-networks/core/wire"
)

// Clock — консенсусные часы узла (закрытие эпох, v9.13: валидацию
// реестра не двигают — ядро применяет tx от txNow). Потокобезопасен
// сам по себе (race-аудит этапа 9): OnCheckpoint пишет из горутины
// чекпоинтов, EpochDue/NetTime читают из cpLoop/voteLoop параллельно.
type Clock struct {
	GenesisTime  uint64
	EpochSeconds uint64
	MaxShift     uint64
	mu           sync.RWMutex
	// lastCP — последний финализированный чекпоинт (эпоха/время).
	lastCP *wire.Checkpoint
	// local — локальное системное время (калибровка NTP — вне кода).
	local func() uint64
}

// NewClock — часы по последнему чекпоинту; до первого чекпоинта —
// genesis-время (эпоха 0).
func NewClock(genesisTime, epochSeconds, maxShift uint64) *Clock {
	if epochSeconds == 0 {
		epochSeconds = 24 * 3600
	}
	return &Clock{
		GenesisTime:  genesisTime,
		EpochSeconds: epochSeconds,
		MaxShift:     maxShift,
		lastCP:       nil,
		local:        func() uint64 { return uint64(time.Now().Unix()) },
	}
}

// SetLocal — инъекция локального времени (симуляция/тесты).
func (c *Clock) SetLocal(f func() uint64) { c.local = f }

// OnCheckpoint — принять финализированный чекпоинт (ход времени).
func (c *Clock) OnCheckpoint(cp *wire.Checkpoint) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.lastCP == nil || cp.Epoch > c.lastCP.Epoch {
		c.lastCP = cp
	}
}

// LastCheckpoint — последний принятый чекпоинт (nil — только genesis).
func (c *Clock) LastCheckpoint() *wire.Checkpoint {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.lastCP
}

// NetTime — сетевое время: локальное, ограниченное пределом сдвига от
// последнего чекпоинта (9.2: не может сдвигаться более чем на заданный
// предел между чекпоинтами).
func (c *Clock) NetTime() uint64 {
	c.mu.RLock()
	defer c.mu.RUnlock()
	now := c.local()
	if c.lastCP == nil {
		if now < c.GenesisTime {
			return c.GenesisTime
		}
		return now
	}
	lo := c.lastCP.NetworkTime
	hi := lo + c.EpochSeconds + c.MaxShift
	if now < lo {
		return lo
	}
	if now > hi {
		return hi
	}
	return now
}

// Epoch — номер текущей эпохи (по последнему чекпоинту).
func (c *Clock) Epoch() uint64 {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.lastCP == nil {
		return 0
	}
	return c.lastCP.Epoch
}

// Period — удалён (аудит z-ai AUDIT-Z-07): мёртвый метод с календарной
// конвенцией месяцев UTC противоречил конвенции ядра (арифметические
// периоды от genesis — periodAt, RealClock.Period) и был ловушкой для
// первого же вызова. Единственный вызов .Period() в проде — RealClock
// (mobile/query.go), который и остаётся единственной конвенцией.

// EpochDue — пора закрывать эпоху: от последнего чекпоинта прошло
// epoch_seconds (по сетевому времени узла).
func (c *Clock) EpochDue() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	now := c.local()
	if c.lastCP == nil {
		return now >= c.GenesisTime+c.EpochSeconds
	}
	lo := c.lastCP.NetworkTime
	hi := lo + c.EpochSeconds + c.MaxShift
	if now < lo {
		now = lo
	}
	if now > hi {
		now = hi
	}
	return now >= lo+c.EpochSeconds
}
