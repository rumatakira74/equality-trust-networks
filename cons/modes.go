// Режимы чекпоинтов (spec/07 7.3, 9.1): decentralized (выборка
// валидаторов эпохи, N ≥ 2/3 M — вопрос №17), members_quorum (≥ 2/3
// участников, малые сети), переходный режим холодного старта (посевной
// круг N-из-M до seed_sunset — вопрос №15).
package cons

import (
	"crypto/sha256"
	"fmt"
	"sort"

	"github.com/rumatakira74/trust-networks-platform/core/ledger"
	"github.com/rumatakira74/trust-networks-platform/core/wire"
)

// Participants — состав подписантов M и порог N для эпохи:
//
//   - переходный (SeedRightsActive и sunset не достигнут): посевной круг
//     из genesis, N = max(2, ceil(2/3 M));
//   - decentralized: детерминированная выборка ValidatorsM active-кошельков
//     (сортировка по SHA256(prev_hash ‖ адрес) — предсказуема только после
//     финализации предыдущего чекпоинта), N = ceil(2/3 M);
//   - members_quorum: все active-кошельки, N = ceil(2/3 M).
func Participants(mode wire.CheckpointMode, l *ledger.Ledger, gp *wire.GenesisPackage,
	p Params, epoch uint64, prevHash []byte) ([][]byte, uint64, error) {
	// переходный режим холодного старта: посевной круг до seed_sunset
	if l.State.SeedRightsActive && !SeedSunsetMet(l, gp, p, epoch) {
		return seedCircle(gp)
	}
	switch mode {
	case wire.CheckpointModeDecentralized:
		acts := activeWallets(l)
		if len(acts) == 0 {
			return nil, 0, fmt.Errorf("cons: нет active-кошельков для выборки")
		}
		m := p.ValidatorsM
		if m > len(acts) {
			m = len(acts)
		}
		sample := weightedSample(acts, prevHash, m)
		return sample, twoThirds(len(sample)), nil
	case wire.CheckpointModeMembersQuorum:
		acts := activeWallets(l)
		if len(acts) == 0 {
			return nil, 0, fmt.Errorf("cons: нет active-кошельков для кворума")
		}
		return acts, twoThirds(len(acts)), nil
	default:
		return nil, 0, fmt.Errorf("cons: неизвестный режим чекпоинтов %d", mode)
	}
}

// seedCircle — посевной круг из genesis: M = состав круга, N = max(2, 2/3 M).
func seedCircle(gp *wire.GenesisPackage) ([][]byte, uint64, error) {
	if len(gp.Manifest.SeedCircle) == 0 {
		return nil, 0, fmt.Errorf("cons: посевной круг пуст")
	}
	out := make([][]byte, 0, len(gp.Manifest.SeedCircle))
	for _, m := range gp.Manifest.SeedCircle {
		out = append(out, m.Address)
	}
	n := twoThirds(len(out))
	if n < 2 {
		n = 2
	}
	if uint64(n) > uint64(len(out)) {
		n = uint64(len(out))
	}
	return out, n, nil
}

// SeedSunsetMet — условие самоликвидации посевного круга (решение №15,
// утверждено 18.10.2026): active ≥ SeedSunsetActive или эпоха ≥
// SeedSunsetEpochs — срабатывает любое; genesis задаёт значения через
// SeedSunset (оба условия кодируются).
func SeedSunsetMet(l *ledger.Ledger, gp *wire.GenesisPackage, p Params, epoch uint64) bool {
	if gp == nil || gp.Manifest == nil {
		return false
	}
	ss := gp.Manifest.SeedSunset
	if ss != nil {
		if ss.UseActiveWallets && countActiveWallets(l) >= ss.ActiveWallets {
			return true
		}
		if ss.DeadlineEpoch > 0 && epoch >= ss.DeadlineEpoch {
			return true
		}
	}
	// дефолты (решение №15: 100 active или 24 эпохи)
	return countActiveWallets(l) >= p.SeedSunsetActive || epoch >= p.SeedSunsetEpochs
}

func countActiveWallets(l *ledger.Ledger) uint64 {
	n := uint64(0)
	for _, w := range l.State.Wallets {
		if w.Status == wire.StatusActive {
			n++
		}
	}
	return n
}

func activeWallets(l *ledger.Ledger) [][]byte {
	var out [][]byte
	for _, w := range l.State.Wallets {
		if w.Status == wire.StatusActive {
			out = append(out, w.Address)
		}
	}
	sort.Slice(out, func(i, j int) bool { return string(out[i]) < string(out[j]) })
	return out
}

// weightedSample — детерминированная выборка m из active: сортировка по
// SHA256(prevHash ‖ адрес). Равномерна по адресам; взвешивание по весу
// кошельков — этап 6 (нужны данные о распределении возрастов).
func weightedSample(acts [][]byte, prevHash []byte, m int) [][]byte {
	type keyed struct {
		addr []byte
		h    [32]byte
	}
	ks := make([]keyed, 0, len(acts))
	for _, a := range acts {
		h := sha256.Sum256(append(append([]byte{}, prevHash...), a...))
		ks = append(ks, keyed{addr: a, h: h})
	}
	sort.Slice(ks, func(i, j int) bool {
		if string(ks[i].h[:]) != string(ks[j].h[:]) {
			return string(ks[i].h[:]) < string(ks[j].h[:])
		}
		return string(ks[i].addr) < string(ks[j].addr)
	})
	out := make([][]byte, 0, m)
	for i := 0; i < m && i < len(ks); i++ {
		out = append(out, ks[i].addr)
	}
	return out
}

// twoThirds — порог N ≥ 2/3 M (округление вверх; spec/07 7.3).
func twoThirds(m int) uint64 {
	if m == 0 {
		return 0
	}
	return uint64((2*m + 2) / 3)
}
