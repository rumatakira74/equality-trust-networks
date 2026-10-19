// Сетевое время (9.2): источник правды — эпохи чекпоинтов; до этапа 5 —
// системное время узла. Периоды выпуска — календарные месяцы; в headless-ядре
// месяц = 30 суток (этап 3), здесь та же конвенция (уточнение по UTC — этап 5).
package net

import (
	"time"

	"github.com/rumatakira74/equality-trust-networks/core/ledger"
)

// RealClock — реальное время узла (ledger.EpochSource).
type RealClock struct {
	GenesisTime  uint64 // genesis_time из genesis-записи
	EpochSeconds uint64 // epoch_seconds из профиля
	PeriodSecs   uint64 // 30 суток (конвенция headless-ядра)
}

// NewClock строит RealClock по полям genesis.
func NewClock(genesisTime, epochSeconds uint64) *RealClock {
	if epochSeconds == 0 {
		epochSeconds = 86400
	}
	return &RealClock{GenesisTime: genesisTime, EpochSeconds: epochSeconds, PeriodSecs: 30 * 86400}
}

func (c *RealClock) NetTime() uint64 { return uint64(time.Now().Unix()) }

func (c *RealClock) Epoch() uint64 {
	now := c.NetTime()
	if now <= c.GenesisTime {
		return 0
	}
	return (now - c.GenesisTime) / c.EpochSeconds
}

func (c *RealClock) Period() uint64 {
	now := c.NetTime()
	if now <= c.GenesisTime {
		return 0
	}
	return (now - c.GenesisTime) / c.PeriodSecs
}

// ReplayClock — часы, пинируемые на время каждой транзакции при replay
// журнала (INV-3): C9 (допуск timestamp ±30 мин) проверяется на момент
// публикации, а не на момент пересчёта.
type ReplayClock struct {
	RealClock
	pinned uint64
}

// NewReplayClock — часы для replay.
func NewReplayClock(genesisTime, epochSeconds uint64) *ReplayClock {
	return &ReplayClock{RealClock: *NewClock(genesisTime, epochSeconds)}
}

// Pin фиксирует время транзакции (ее timestamp_claim).
func (c *ReplayClock) Pin(t uint64) { c.pinned = t }

func (c *ReplayClock) NetTime() uint64 { return c.pinned }

func (c *ReplayClock) Epoch() uint64 {
	if c.pinned <= c.GenesisTime {
		return 0
	}
	return (c.pinned - c.GenesisTime) / c.EpochSeconds
}

func (c *ReplayClock) Period() uint64 {
	if c.pinned <= c.GenesisTime {
		return 0
	}
	return (c.pinned - c.GenesisTime) / c.PeriodSecs
}

var _ ledger.EpochSource = (*RealClock)(nil)
var _ ledger.EpochSource = (*ReplayClock)(nil)
