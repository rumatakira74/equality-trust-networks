// Часы узла с периодом заданной длины (genesis-поле 61, блокер живого
// прогона мини-тестнета этапа 12): 0 — конвенция 30 сут.
package net

import (
	"testing"
	"time"
)

func TestRealClockPeriodLength(t *testing.T) {
	// genesis в прошлом (как в прогонах: возраст посевного круга)
	genesis := uint64(time.Now().Unix()) - 95*86400
	// конвенция: 30 сут
	c := NewClock(genesis, 75, 0)
	if c.PeriodSecs != 30*86400 {
		t.Fatalf("дефолт: %d != 30 сут", c.PeriodSecs)
	}
	if got, want := c.Period(), (uint64(time.Now().Unix())-genesis)/(30*86400); got != want {
		t.Fatalf("период (конвенция): %d != %d", got, want)
	}
	// тестовая сеть: период 120 с — границы на реальном времени
	c2 := NewClock(genesis, 75, 120)
	if c2.PeriodSecs != 120 {
		t.Fatalf("period_secs: %d != 120", c2.PeriodSecs)
	}
	if got, want := c2.Period(), (uint64(time.Now().Unix())-genesis)/120; got != want {
		t.Fatalf("период (120 с): %d != %d", got, want)
	}
	if c2.Period() < 68_400 { // 95 сут / 120 с
		t.Fatalf("период 120 с при genesis 95 сут назад: %d (< 68400 — не короткие периоды?)", c2.Period())
	}
	// replay-часы — та же длина периода
	rc := NewReplayClock(genesis, 75, 120)
	if rc.PeriodSecs != 120 {
		t.Fatalf("replay period_secs: %d != 120", rc.PeriodSecs)
	}
}
