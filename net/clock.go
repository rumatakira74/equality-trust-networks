// Сетевое время (9.2): источник правды — эпохи чекпоинтов; до этапа 5 —
// системное время узла. Периоды выпуска — календарные месяцы; в headless-ядре
// месяц = 30 суток (этап 3), здесь та же конвенция (уточнение по UTC — этап 5).
//
// v9.13 (этап фикса времени): часы узла больше не участвуют в валидации
// реестра — ядро применяет каждую tx от txNow = timestamp_claim.
// RealClock остаётся для отображения (mobile) и допуска C9 (приём/
// распространение tx в net): claim вне ±timestamp_tolerance_seconds
// часов узла не принимается и не распространяется; replay C9 не применяет.
package net

import (
	"time"
)

// RealClock — реальное время узла (ledger.EpochSource).
type RealClock struct {
	GenesisTime  uint64 // genesis_time из genesis-записи
	EpochSeconds uint64 // epoch_seconds из профиля
	PeriodSecs   uint64 // 30 суток (конвенция headless-ядра)
}

// NewClock строит RealClock по полям genesis. periodSeconds — длина
// периода выпуска (genesis-поле 61; 0 — конвенция headless-ядра, 30 сут).
func NewClock(genesisTime, epochSeconds, periodSeconds uint64) *RealClock {
	if epochSeconds == 0 {
		epochSeconds = 86400
	}
	if periodSeconds == 0 {
		periodSeconds = 30 * 86400
	}
	return &RealClock{GenesisTime: genesisTime, EpochSeconds: epochSeconds, PeriodSecs: periodSeconds}
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
