// Адаптивная комиссия (10.2, этап 10): детерминированный пересчёт ставки
// на границе эпохи ИЗ ДАННЫХ ЧЕКПОИНТОВ.
//
// Источник — total_supply финализированных чекпоинтов (пороговая подпись,
// одинаковые значения на всех узлах): рост предложения за последние K
// эпох выше целевого — ставка растёт на шаг, иначе снижается; границы
// [rate_min, rate_max] заданы профилем. Пересчёт — чистая функция от
// цепочки чекпоинтов, поэтому все узлы сходятся к одному значению без
// координации; рестарт из БД восстанавливает ряд предложения из store и
// сворачивает ту же цепочку пересчётов (эквивалентность рестарта — тест).
package ledger

import "math/bits"

// OnCheckpointSupply — данные финализированного чекпоинта эпохи epoch
// (total_supply). Вызывается сетевым слоем при адаптации чекпоинта
// (net.adoptCheckpoint) и при восстановлении ряда из БД; порядок —
// строго возрастающие эпохи (цепочка чекпоинтов).
func (l *Ledger) OnCheckpointSupply(epoch, supply uint64) {
	prof := l.State.Net.Profile
	K := prof.FeeAdjustEpochs
	if K == 0 {
		return // адаптивный пересчёт выключен (векторы этапа 0)
	}
	// цепочка строго возрастает; дубль или регрессия игнорируются
	if n := len(l.State.SupplySnapshots); n > 0 {
		if epoch <= l.State.SupplySnapshots[n-1].Epoch {
			return
		}
	}
	l.State.SupplySnapshots = append(l.State.SupplySnapshots, SupplySnapshot{
		Epoch:  epoch,
		Supply: supply,
	})
	// держим только окно K+1 снимков (память не растёт с числом эпох)
	if uint64(len(l.State.SupplySnapshots)) > K+1 {
		l.State.SupplySnapshots = l.State.SupplySnapshots[1:]
	}
	// нужно K+1 снимков: текущий и снимок K эпох назад
	if uint64(len(l.State.SupplySnapshots)) < K+1 {
		return
	}
	base := l.State.SupplySnapshots[len(l.State.SupplySnapshots)-int(K)-1]
	if base.Supply == 0 {
		return // рост не определён — ставка не меняется
	}
	cur := l.State.SupplySnapshots[len(l.State.SupplySnapshots)-1]
	if cur.Supply < base.Supply {
		return // предложение монотонно; защита от аномалии источника
	}
	// growth в micros = (cur − base) × 10^6 / base; 128-битное умножение —
	// без переполнения uint64 при больших объёмах эмиссии
	growth := growthMicros(cur.Supply, base.Supply)
	rate := l.State.FeeRate
	if growth > prof.FeeTargetGrowthMicros {
		rate += prof.FeeStepMicros
	} else {
		if rate < prof.RateMin+prof.FeeStepMicros {
			rate = prof.RateMin
		} else {
			rate -= prof.FeeStepMicros
		}
	}
	if rate < prof.RateMin {
		rate = prof.RateMin
	}
	if rate > prof.RateMax {
		rate = prof.RateMax
	}
	l.State.FeeRate = rate
}

// FeeRateNow — текущая ставка комиссии в micros (для UI и диагностики).
func (l *Ledger) FeeRateNow() uint64 { return l.State.FeeRate }

// growthMicros — (cur − base) × 10^6 / base без переполнения: произведение
// в 128 битах; при сверхбольших значениях насыщается до ^uint64(0).
func growthMicros(cur, base uint64) uint64 {
	if base == 0 || cur <= base {
		return 0
	}
	hi, lo := bits.Mul64(cur-base, 1_000_000)
	if hi >= base {
		return ^uint64(0) // рост больше 2^64 micros — заведомо выше цели
	}
	q, _ := bits.Div64(hi, lo, base)
	return q
}
