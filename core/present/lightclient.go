// Лёгкий клиент-синхронизатор (6.1): проверяющая библиотека сверяет
// предъявление с чекпоинтом; чекпоинты клиент получает С ЛЮБОГО узла
// (синхронизация с любым узлом, серверы проекта не нужны) и проверяет
// цепочку (prev_hash, возраст эпох, окно времени) и пороговую подпись
// N-из-M каждого чекпоинта. Источник открытых ключей подписантов
// (PubSource) — манифест сети (посевной круг) и/или доверенный снимок
// состава валидаторов; без полного узла.
package present

import (
	"github.com/rumatakira74/equality-trust-networks/cons"
	"github.com/rumatakira74/equality-trust-networks/core/wire"
)

// LightSync — лёгкий клиент цепочки чекпоинтов: принимает чекпоинты от
// любого узла, проверяет и хранит финализированную цепочку.
type LightSync struct {
	networkID    []byte
	genesisTime  uint64
	epochSeconds uint64
	maxShift     uint64
	pubOf        PubSource
	chain        []*wire.Checkpoint
}

// NewLightSync — клиент для сети с заданными параметрами genesis.
func NewLightSync(networkID []byte, genesisTime, epochSeconds, maxShift uint64,
	pubOf PubSource) *LightSync {
	return &LightSync{
		networkID:    append([]byte(nil), networkID...),
		genesisTime:  genesisTime,
		epochSeconds: epochSeconds,
		maxShift:     maxShift,
		pubOf:        pubOf,
	}
}

// Adopt — принять чекпоинт от узла: проверка цепочки и пороговой подписи
// (cons.Verify), только возрастающие эпохи. Чекпоинты можно передавать
// в любом порядке поставки — принимаются строго последовательно.
func (ls *LightSync) Adopt(cp *wire.Checkpoint) error {
	var prev *wire.Checkpoint
	if n := len(ls.chain); n > 0 {
		prev = ls.chain[n-1]
		// уже известная эпоха — дубль поставки, не ошибка
		if cp.Epoch <= prev.Epoch {
			return nil
		}
	}
	if err := cons.Verify(cp, prev, ls.genesisTime, ls.epochSeconds, ls.maxShift, ls.pubOf); err != nil {
		return err
	}
	ls.chain = append(ls.chain, cp)
	return nil
}

// Latest — последний принятый чекпоинт (nil — пока ничего).
func (ls *LightSync) Latest() *wire.Checkpoint {
	if n := len(ls.chain); n > 0 {
		return ls.chain[n-1]
	}
	return nil
}

// Chain — принятая цепочка (копия).
func (ls *LightSync) Chain() []*wire.Checkpoint {
	out := make([]*wire.Checkpoint, len(ls.chain))
	copy(out, ls.chain)
	return out
}

// AtEpoch — чекпоинт конкретной эпохи (nil — нет).
func (ls *LightSync) AtEpoch(epoch uint64) *wire.Checkpoint {
	for _, cp := range ls.chain {
		if cp.Epoch == epoch {
			return cp
		}
	}
	return nil
}
