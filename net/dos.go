// net-DoS (A3, тема net-dos-a3): защита от асимметричного DoS — бюджет
// дорогих операций (Argon2id) на пира в скользящем окне и счёт пира
// (peer score). Порядок валидации «подпись до дорогой работы» — в
// gossip.go (cheapCheck/relayAdmit); libp2p WithPeerScore — там же.
//
// Всё здесь — поведение узла, не протокола: wire-формат и правила
// приёма валидных tx не меняются, protocolVersion не поднимается,
// genesis-полей нет. Значения констант и обоснование — отчёт темы
// (reports/REPORT-NET-DOS-A3.md).
package net

import (
	"math"
	"sync"
	"time"

	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/peer"
)

// Бюджет дорогих операций: N проверок Argon2id на пира за скользящее
// окно T. Обоснование: честный пир не шлёт десятков register/mint в
// секунду (mint — раз в период на кошелёк, register — раз на
// устройство), а 32 проверки Argon2id (32 МиБ, ~30 мс хост / больше
// на телефоне) за окно 10 с ограничивают CPU узла на одного пира
// ~1 с/окно и память ~64 МиБ (DosMaxConcurrentHard × 32 МиБ).
const (
	// DosExpensiveBudget — N дорогих проверок на пира за окно.
	DosExpensiveBudget = 32
	// DosExpensiveWindow — T — скользящее окно бюджета.
	DosExpensiveWindow = 10 * time.Second
	// DosMaxConcurrentHard — максимум параллельных Argon2id на узле
	// (память: каждая проверка — Argon2MemoryKiB, в прод-профиле 32 МиБ;
	// аудит A3: «ограничить число параллельных Argon2»).
	DosMaxConcurrentHard = 2
)

// Счёт пира: события приёма штрафа/награды, экспоненциальный распад
// к нулю (половина каждые dosScoreHalfLife), низкий счёт — разрыв
// соединения (крайняя мера) и пониженный приоритет в mesh pubsub
// (WithPeerScore, AppSpecificScore). Обоснование весов: невалидная
// подпись — прямой признак мусора (штраф больше всех); неверный PoW
// при валидной подписи — мусор от владельца ключа; структурный мусор
// и исчерпание бюджета — агрессивное поведение; честная доставка
// восстанавливает счёт медленнее, чем штрафы (штрафы не «отработать»
// полезной нагрузкой).
const (
	dosPenaltyInvalidSig    = -8.0
	dosPenaltyInvalidPow    = -4.0
	dosPenaltyMalformed     = -4.0
	dosPenaltyBudget        = -4.0
	dosPenaltyUnknownWallet = -2.0
	dosRewardValid          = 1.0
	dosScoreCap             = 16.0
	dosScoreHalfLife        = 60 * time.Second
	dosDisconnectScore      = -64.0
)

// dosPeerState — состояние пира в трекере.
type dosPeerState struct {
	// expensive — моменты дорогих проверок в окне (len <= бюджета;
	// усечённое кольцо: старые вытесняются при очистке по окну).
	expensive []time.Time
	score     float64
	updated   time.Time
}

// dosTracker — бюджет дорогих операций + счёт пира (A3).
type dosTracker struct {
	mu     sync.Mutex
	budget int
	window time.Duration
	peers  map[peer.ID]*dosPeerState
}

func newDosTracker(budget int, window time.Duration) *dosTracker {
	if budget <= 0 {
		budget = DosExpensiveBudget
	}
	if window <= 0 {
		window = DosExpensiveWindow
	}
	return &dosTracker{budget: budget, window: window, peers: make(map[peer.ID]*dosPeerState)}
}

func (t *dosTracker) state(p peer.ID, now time.Time) *dosPeerState {
	st := t.peers[p]
	if st == nil {
		st = &dosPeerState{}
		t.peers[p] = st
	}
	// ленивый распад счёта к нулю (половина за dosScoreHalfLife)
	if !st.updated.IsZero() && st.score != 0 {
		dt := now.Sub(st.updated).Seconds()
		if dt > 0 {
			halfLives := dt / dosScoreHalfLife.Seconds()
			st.score *= powHalf(halfLives)
			if st.score > -0.01 && st.score < 0.01 {
				st.score = 0
			}
		}
	}
	st.updated = now
	return st
}

// powHalf — 0.5^k (распад счёта: половина за dosScoreHalfLife).
func powHalf(k float64) float64 {
	return math.Pow(0.5, k)
}

// chargeExpensive — попытка расхода бюджета дорогих операций: true —
// можно выполнять (момент учтён), false — окно пира исчерпано (дорогая
// операция не выполняется, сообщение дёшево отклоняется).
func (t *dosTracker) chargeExpensive(p peer.ID, now time.Time) bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	st := t.state(p, now)
	// чистка окна
	cut := len(st.expensive)
	for i, ts := range st.expensive {
		if now.Sub(ts) <= t.window {
			cut = i
			break
		}
	}
	st.expensive = st.expensive[cut:]
	if len(st.expensive) >= t.budget {
		return false
	}
	st.expensive = append(st.expensive, now)
	return true
}

// penalty — событие мусора: счёт пира снижается.
func (t *dosTracker) penalty(p peer.ID, delta float64, now time.Time) {
	t.mu.Lock()
	defer t.mu.Unlock()
	st := t.state(p, now)
	st.score += delta
	if st.score < -dosScoreCap*8 {
		st.score = -dosScoreCap * 8 // потолок долга: распад измерим
	}
}

// reward — полезная активность: счёт восстанавливается (медленнее
// штрафов: dosRewardValid < |штрафы|).
func (t *dosTracker) reward(p peer.ID, delta float64, now time.Time) {
	t.mu.Lock()
	defer t.mu.Unlock()
	st := t.state(p, now)
	st.score += delta
	if st.score > dosScoreCap {
		st.score = dosScoreCap
	}
}

// score — текущий счёт пира (с распадом; чтение не мутирует состояние).
func (t *dosTracker) score(p peer.ID) float64 {
	t.mu.Lock()
	defer t.mu.Unlock()
	st, ok := t.peers[p]
	if !ok {
		return 0
	}
	s := st.score
	if !st.updated.IsZero() && s != 0 {
		dt := time.Since(st.updated).Seconds()
		if dt > 0 {
			s *= powHalf(dt / dosScoreHalfLife.Seconds())
		}
	}
	return s
}

// peerScoreParams — параметры libp2p peer score (A3): счёт темы tx
// (P1 время в mesh, P2 первые доставки, P4 невалидные сообщения —
// автоматически по ValidationReject валидатора) + счёт приложения
// (dosTracker). Пороги: ниже GossipThreshold — пир вырезается из
// mesh (пониженный приоритет/бандвит), ниже GraylistThreshold —
// pubsub игнорирует пира; разрыв соединения — dosDisconnectScore
// (крайняя мера, в relayAdmit). IP-colocation отключён: все пиры
// localhost-тестов в одной /24 (включение — рекомендация отчёта).
func (n *Node) peerScoreParams() (*pubsub.PeerScoreParams, *pubsub.PeerScoreThresholds) {
	params := &pubsub.PeerScoreParams{
		SkipAtomicValidation: true,
		Topics: map[string]*pubsub.TopicScoreParams{
			n.txTopic(): {
				SkipAtomicValidation:           true,
				TopicWeight:                    1,
				TimeInMeshWeight:               0.01,
				TimeInMeshQuantum:              time.Second,
				TimeInMeshCap:                  3600,
				FirstMessageDeliveriesWeight:   1,
				FirstMessageDeliveriesDecay:    0.5,
				FirstMessageDeliveriesCap:      64,
				InvalidMessageDeliveriesWeight: -16,
				InvalidMessageDeliveriesDecay:  0.5,
			},
		},
		TopicScoreCap: 32,
		AppSpecificScore: func(p peer.ID) float64 {
			return n.dos.score(p)
		},
		AppSpecificWeight:           1,
		IPColocationFactorWeight:    0,
		IPColocationFactorThreshold: 1,
		BehaviourPenaltyWeight:      -16,
		BehaviourPenaltyThreshold:   8,
		BehaviourPenaltyDecay:       0.9,
		DecayInterval:               time.Second,
		DecayToZero:                 0.01,
		RetainScore:                 10 * time.Minute,
	}
	thresholds := &pubsub.PeerScoreThresholds{
		SkipAtomicValidation:        true,
		GossipThreshold:             -10,
		PublishThreshold:            -40,
		GraylistThreshold:           -80,
		OpportunisticGraftThreshold: 2,
	}
	return params, thresholds
}
