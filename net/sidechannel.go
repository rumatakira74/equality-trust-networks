// Офлайн-канал (12.4, 4.10): прямая передача подписанной транзакции
// получателю (Bluetooth; реальный BT-стек — этап 6). Полезная нагрузка —
// wire-Transaction из core/wire. «Получено по Bluetooth» ≠ оплата: пока
// транзакция не опубликована и не подтверждена сетью, она не является
// платой; интерфейс отдаёт статус финализации (заглушка до этапа 5).
package net

import (
	"errors"
	"sync"

	"github.com/rumatakira74/equality-trust-networks/core/wire"
)

// OfflineStatus — статус транзакции офлайн-канала (12.4).
type OfflineStatus int

const (
	// OfflineAwaiting — получено, ожидает публикации.
	OfflineAwaiting OfflineStatus = iota
	// OfflinePublished — опубликовано в сеть (доставлено пирам).
	OfflinePublished
	// OfflineFinalized — финализировано (заглушка до этапа 5: подтверждение
	// данными N пиров; голосование/чекпоинты — этап 5).
	OfflineFinalized
)

func (s OfflineStatus) String() string {
	switch s {
	case OfflineAwaiting:
		return "awaiting_publication"
	case OfflinePublished:
		return "published"
	default:
		return "finalized"
	}
}

// SideChannel — интерфейс офлайн-канала (12.4). Реальный Bluetooth-ткан —
// этап 6; здесь — интерфейс и in-memory реализация для тестов.
type SideChannel interface {
	// Send передаёт подписанную транзакцию получателю.
	Send(tx *wire.Transaction) error
	// Receive — канал принятых транзакций (полезная нагрузка wire-Transaction).
	Receive() <-chan *wire.Transaction
	// Close закрывает канал.
	Close() error
}

// MemSideChannel — in-memory офлайн-канал для тестов (пара: отправитель
// пишет в канал получателя).
type MemSideChannel struct {
	mu   sync.Mutex
	ch   chan *wire.Transaction
	done bool
}

// NewMemSideChannel — канал с буфером.
func NewMemSideChannel(buf int) *MemSideChannel {
	if buf <= 0 {
		buf = 16
	}
	return &MemSideChannel{ch: make(chan *wire.Transaction, buf)}
}

func (m *MemSideChannel) Send(tx *wire.Transaction) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.done {
		return errors.New("sidechannel: канал закрыт")
	}
	select {
	case m.ch <- tx:
		return nil
	default:
		return errors.New("sidechannel: переполнение буфера")
	}
}

func (m *MemSideChannel) Receive() <-chan *wire.Transaction { return m.ch }

func (m *MemSideChannel) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.done {
		return nil
	}
	m.done = true
	close(m.ch)
	return nil
}

// BluetoothScanSource — заглушка Bluetooth-скана (каскад 12.3, шаг 5;
// реальный BT — этап 6).
type BluetoothScanSource struct{}

func (BluetoothScanSource) Name() string { return "bluetooth" }

func (BluetoothScanSource) Peers() ([]peerAddrMsg, error) {
	// Этап 6: реальный BT-стек. Пока — канал-интерфейс без пиров.
	return nil, ErrNotStage6
}

// ErrNotStage6 — механизм отложен до этапа 6 (Bluetooth/Android).
var ErrNotStage6 = errors.New("net: механизм отложен до этапа 6 (Bluetooth/Android)")
