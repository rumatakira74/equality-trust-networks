// Пакет cons: консенсус этапа 5 — выборочное голосование (Snowball-подобное,
// 9.1), чекпоинты (spec/07) и сетевое время/эпохи (9.2). Поверх ядра
// (core/ledger, core/smt, core/threshold) и сети (net); судей и вердиктов
// нет (7.5): голосование только выбирает ветку при конфликте, метка
// flagged — только через evidence любым узлом.
package cons

import (
	"github.com/rumatakira74/equality-trust-networks/core/wire"
)

// Params — параметры консенсуса. Решения №15/№16/№17 утверждены
// пользователем 18.10.2026 (закреплены этапом 6; обоснование симуляцией —
// reports/REPORT-STAGE-5.md):
//
//   - EpochSeconds (№16): 24 ч — чекпоинт раз в сутки: финальность
//     «на следующий день», трафик подписей пренебрежим; песочница —
//     time_scale из профиля.
//   - MaxTimeShift (9.2): EpochSeconds/4 (6 ч при 24-часовой эпохе) —
//     сетевое время чекпоинта не может сдвинуться сильнее.
//   - SnowballK/Alpha/Beta: k = 10 пиров в выборке, α = 0,7 (кворум
//     веса: живость при доле злонамеренных ≤ 30%, победа атакующего
//     требует ≥ 70% веса — маловероятна), β = 4 последовательных
//     успешных раунда (см. симуляцию).
//   - ValidatorsM (№17): M = 21 валидатор в режиме decentralized
//     (параметр genesis-профиля validators_m); порог N = ceil(2/3 M) = 14.
//   - SeedSunsetActive/Epochs (№15): 100 active-кошельков или 24 эпохи.
//   - BuilderQueue (№42, решение владельца (A), этап 9): K = 3 строителя —
//     очередь первых K участников состава, ротация через epoch/2; после
//     исчерпания очереди застрявшая эпоха пропускается (решение (D)).
//     Значение черновое (как eclipse_n = 3), симуляцией не подбирали.
type Params struct {
	EpochSeconds     uint64
	MaxTimeShift     uint64
	SnowballK        int
	SnowballAlpha    float64
	SnowballBeta     int
	ValidatorsM      int
	BuilderQueue     int
	SeedSunsetActive uint64
	SeedSunsetEpochs uint64
}

// DefaultParams — предложения этапа 5 (обоснование — отчёт).
func DefaultParams() Params {
	return Params{
		EpochSeconds:     24 * 3600,
		MaxTimeShift:     6 * 3600,
		SnowballK:        10,
		SnowballAlpha:    0.6,
		SnowballBeta:     4,
		ValidatorsM:      21,
		BuilderQueue:     3,
		SeedSunsetActive: 100,
		SeedSunsetEpochs: 24,
	}
}

// FromProfile — параметры из genesis-профиля (epoch_seconds, validators_m)
// с дефолтами утверждённых решений.
func FromProfile(p *wire.ProfileParams) Params {
	dp := DefaultParams()
	if p != nil && p.EpochSeconds > 0 {
		dp.EpochSeconds = p.EpochSeconds
		dp.MaxTimeShift = p.EpochSeconds / 4
	}
	if p != nil && p.ValidatorsM > 0 {
		dp.ValidatorsM = int(p.ValidatorsM)
	}
	if p != nil && p.CpBuilderQueue > 0 {
		dp.BuilderQueue = int(p.CpBuilderQueue)
	}
	return dp
}
