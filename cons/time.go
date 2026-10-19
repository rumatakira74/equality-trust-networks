// Сетевое время и эпохи (9.2): источник правды — эпохи чекпоинтов, не
// медиана timestamps. Сетевое время узла ограничено последним
// финализированным чекпоинтом: [cp.time, cp.time + epoch + сдвиг].
// Календарь UTC: «1-е число месяца» = первая эпоха, начавшаяся после
// 00:00 UTC 1-го числа (9.2) — период выпуска меняется только на
// границе эпохи.
package cons

import (
	"time"

	"github.com/rumatakira74/trust-networks-platform/core/wire"
)

// Clock — консенсусные часы узла (ledger.EpochSource).
type Clock struct {
	GenesisTime  uint64
	EpochSeconds uint64
	MaxShift     uint64
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
	if c.lastCP == nil || cp.Epoch > c.lastCP.Epoch {
		c.lastCP = cp
	}
}

// LastCheckpoint — последний принятый чекпоинт (nil — только genesis).
func (c *Clock) LastCheckpoint() *wire.Checkpoint { return c.lastCP }

// NetTime — сетевое время: локальное, ограниченное пределом сдвига от
// последнего чекпоинта (9.2: не может сдвигаться более чем на заданный
// предел между чекпоинтами).
func (c *Clock) NetTime() uint64 {
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
	if c.lastCP == nil {
		return 0
	}
	return c.lastCP.Epoch
}

// Period — текущий период выпуска: календарные месяцы UTC, начавшиеся
// после genesis к началу текущей эпохи (9.2: «1-е число месяца» =
// первая эпоха, начавшаяся после 00:00 UTC 1-го).
func (c *Clock) Period() uint64 {
	epochStart := c.GenesisTime
	if c.lastCP != nil {
		epochStart = c.lastCP.NetworkTime
	}
	return PeriodOfEpoch(c.GenesisTime, epochStart)
}

// PeriodOfEpoch — число «1-х чисел месяца» 00:00 UTC в интервале
// (genesisTime, epochStartTime]: первая эпоха, начавшаяся после границы,
// открывает новый период.
func PeriodOfEpoch(genesisTime, epochStartTime uint64) uint64 {
	g := time.Unix(int64(genesisTime), 0).UTC()
	e := time.Unix(int64(epochStartTime), 0).UTC()
	if !e.After(g) {
		return 0
	}
	// первая граница «1-е число 00:00 UTC» строго после genesis
	b := time.Date(g.Year(), g.Month(), 1, 0, 0, 0, 0, time.UTC)
	if !b.After(g) {
		b = b.AddDate(0, 1, 0)
	}
	count := uint64(0)
	for !b.After(e) {
		count++
		b = b.AddDate(0, 1, 0)
	}
	return count
}

// EpochDue — пора закрывать эпоху: от последнего чекпоинта прошло
// epoch_seconds (по сетевому времени узла).
func (c *Clock) EpochDue() bool {
	if c.lastCP == nil {
		return c.NetTime() >= c.GenesisTime+c.EpochSeconds
	}
	return c.NetTime() >= c.lastCP.NetworkTime+c.EpochSeconds
}
