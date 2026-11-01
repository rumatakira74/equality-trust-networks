// Тесты блокера живого прогона мини-тестнета этапа 12: period_seconds
// (genesis-поле 61 — период заданной длины, границы на реальном времени)
// и -transfer-moratorium-days (свежий кошелёк приложения должен дойти
// до E_BURNED, а не застрять в моратории 14 сут).
package main

import "testing"

// TestConfigureShortPeriod — -period-seconds попадает в genesis и
// декодируется; -transfer-moratorium-days 0 убирает мораторий;
// -mint-challenge-window-hours задаёт окно слота челленджа.
func TestConfigureShortPeriod(t *testing.T) {
	gp, _, _ := configureRun(t, "-slug", "cirrus-t",
		"-period-seconds", "120", "-transfer-moratorium-days", "0",
		"-mint-challenge-window-hours", "2400")
	prof := gp.Manifest.Profile
	if prof.PeriodSeconds != 120 {
		t.Fatalf("period_seconds: %d != 120", prof.PeriodSeconds)
	}
	if prof.PeriodSecsOrDefault() != 120 {
		t.Fatalf("эффективная длина периода: %d != 120", prof.PeriodSecsOrDefault())
	}
	if prof.TransferMoratoriumDays != 0 {
		t.Fatalf("мораторий: %d != 0", prof.TransferMoratoriumDays)
	}
	if prof.MintChallengeWindowHours != 2400 {
		t.Fatalf("окно челленджа: %d != 2400", prof.MintChallengeWindowHours)
	}
}

// TestConfigurePeriodSecondsValidation — длина периода короче минуты
// отвергается при сборке (0 — конвенция 30 сут, не ошибка).
func TestConfigurePeriodSecondsValidation(t *testing.T) {
	if err := configureCmd([]string{"-template", "8", "-slug", "cirrus-t",
		"-out", t.TempDir(), "-period-seconds", "30"}); err == nil {
		t.Fatal("period_seconds 30 должно отвергаться (< 60)")
	}
	if err := configureCmd([]string{"-template", "8", "-slug", "cirrus-t",
		"-out", t.TempDir(), "-period-seconds", "0"}); err != nil {
		t.Fatalf("period_seconds 0 (конвенция) не должно отвергаться: %v", err)
	}
}

// TestConfigureMoratoriumDefault — без флага мораторий из шаблона (14 сут).
func TestConfigureMoratoriumDefault(t *testing.T) {
	gp, _, _ := configureRun(t, "-slug", "cirrus-t")
	if gp.Manifest.Profile.TransferMoratoriumDays != 14 {
		t.Fatalf("мораторий по умолчанию: %d != 14", gp.Manifest.Profile.TransferMoratoriumDays)
	}
}
