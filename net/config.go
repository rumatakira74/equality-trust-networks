// Пакет net: сетевой слой поверх ядра этапа 3 (раздел 12 KNOWLEDGE.md).
//
// Сеть только доставляет и запрашивает: валидация и применение транзакций —
// в core/ledger. Этап 5 (голосование, чекпоинты, пороговая подпись) здесь не
// реализуется — только интерфейсы.
package net

import (
	"crypto/ecdsa"
	"crypto/x509"
	"time"

	"github.com/rumatakira74/equality-trust-networks/core/wire"
)

// NodeLevel — уровень узла (12.2).
type NodeLevel int

const (
	// LevelFull — полный узел: хранит и проверяет реестр, ретранслирует,
	// DHT server, добровольный relay (12.2).
	LevelFull NodeLevel = iota
	// LevelLight — лёгкий узел: своя цепочка, входящие, заголовки чекпоинтов,
	// Merkle-доказательства; DHT client (12.5).
	LevelLight
)

func (l NodeLevel) String() string {
	if l == LevelFull {
		return "full"
	}
	return "light"
}

// NodeConfig — конфигурация узла (headless; Android-триггеры — этап 6).
type NodeConfig struct {
	// Genesis — genesis-пакет сети (одинаков у всех узлов сети).
	Genesis *wire.GenesisPackage
	// NodeKey — ключ узла (Peer ID); не ключ кошелька.
	NodeKey *ecdsa.PrivateKey
	// Level — полный/лёгкий узел (12.2).
	Level NodeLevel
	// DBPath — путь к SQLite (core/store); "" — только память (тесты).
	DBPath string
	// ListenAddrs — адреса прослушивания (localhost/эмерджентные в тестах).
	ListenAddrs []string
	// NoAnnounce — не анонсировать адреса (эмуляция CGNAT, 12.2): узел за NAT,
	// входящие только через relay/DCUtR.
	NoAnnounce bool
	// ForcePublic — форсировать «публичную достижимость» (relay-сервис
	// стартует только на достижимых узлах; в тестах/эмуляции CGNAT —
	// явно, в продакшене определяет AutoNAT).
	ForcePublic bool
	// RelayAddrs — адреса добровольных relay-узлов для резерва слота
	// (circuit relay v2; CGNAT-узлы).
	RelayAddrs []string
	// ManualAddrs — QR «добавить узел» и ручной ввод (каскад 12.3, шаг 6):
	// адреса первых узлов; в invite_only — адреса из приглашения.
	ManualAddrs []string
	// EmbeddedAddrs — зашитый список (каскад 12.3, шаг 3; устаревший
	// fallback). Не используется в invite_only (12.3).
	EmbeddedAddrs []string
	// SeedList — подписанный seed-список (каскад 12.3, шаг 2), полезная
	// нагрузка net.SeedList. Не используется в invite_only.
	SeedList []byte
	// SeedPubKeys — ключи подписантов seed-списка (N-из-M, core/threshold).
	// Синтетические для тестов; реальные — трек «Ключи проекта» (13).
	SeedPubKeys map[string]*ecdsa.PublicKey
	// SeedThreshold — N из M для seed-списка.
	SeedThreshold uint64
	// InviteOnly — сеть по приглашению (12.3): каналы и зашитый список не
	// используются; адреса первых узлов — из приглашения (ManualAddrs).
	// Берётся из genesis (NetworkAccessInviteOnly), флаг — для тестов.
	InviteOnly bool
	// MinPeers — минимум подключений (смешивание источников, 12.3).
	MinPeers int
	// CorroborationN — подтверждение данных от N несвязанных пиров (12.3,
	// eclipse). Решение №35: N = 3, параметр genesis-профиля eclipse_n.
	CorroborationN int
	// LinkGroups — оценка несвязанности по группам связи (№35, этап 6):
	// подтверждение — N различных IP-подсетей (/24 IPv4, /64 IPv6), а не
	// N пиров. false — уникальные Peer ID (тесты на localhost; дефолт
	// для продакшена — true, предложение в отчёте этапа 6).
	LinkGroups bool
	// MaxTxSize — лимит размера транзакции для gossip (спам-фильтр).
	// Ориентир ~250 байт на транзакцию (12.5); лимит с запасом на attest-цепочку.
	MaxTxSize int
	// SyncBatch — максимум транзакций в ответе синхронизации.
	SyncBatch int
	// PollInterval — период опроса входящих для лёгкого узла.
	PollInterval time.Duration
	// Switcher — переключатель уровня узла (зарядка + Wi-Fi — этап 6).
	Switcher LevelSwitcher
	// WalletKey/WalletAddr — ключ и адрес кошелька узла: подпись
	// чекпоинтов (участник) и вес голоса (этап 5). Нил — узел не
	// подписывает (наблюдатель).
	WalletKey  *ecdsa.PrivateKey
	WalletAddr []byte
	// VoteInterval — период раундов голосования по конфликтам.
	VoteInterval time.Duration
	// ConsensusEpochSeconds — переопределение epoch_seconds для тестов
	// (0 — из genesis-профиля; №16: предложение 24 ч).
	ConsensusEpochSeconds uint64
	// OwnWallet — адрес собственного кошелька (лёгкий узел, 12.5).
	OwnWallet []byte
	// TrustedRoots — корни аттестации (core/attest; одинаковые у всех
	// узлов сети; распространение — этап 6).
	TrustedRoots []*x509.Certificate
	// Log — журнал узла (nil — тихий режим).
	Log func(format string, args ...any)
}

func (c *NodeConfig) networkID() []byte { return c.Genesis.Genesis.NetworkID }

func (c *NodeConfig) inviteOnly() bool {
	if c.InviteOnly {
		return true
	}
	return c.Genesis.Manifest.Profile != nil &&
		c.Genesis.Manifest.Profile.NetworkAccess == wire.NetworkAccessInviteOnly
}

// defaults — значения по умолчанию для незаданных полей.
func (c *NodeConfig) defaults() {
	if c.MinPeers <= 0 {
		c.MinPeers = 3
	}
	if c.CorroborationN <= 0 {
		// решение №35 (этап 5): N = 3, параметр genesis-профиля eclipse_n
		c.CorroborationN = 3
		if c.Genesis != nil && c.Genesis.Manifest != nil && c.Genesis.Manifest.Profile != nil &&
			c.Genesis.Manifest.Profile.EclipseN > 0 {
			c.CorroborationN = int(c.Genesis.Manifest.Profile.EclipseN)
		}
	}
	if c.MaxTxSize <= 0 {
		c.MaxTxSize = 64 * 1024
	}
	if c.SyncBatch <= 0 {
		c.SyncBatch = 512
	}
	if c.PollInterval <= 0 {
		c.PollInterval = 30 * time.Second
	}
	if c.VoteInterval <= 0 {
		c.VoteInterval = 2 * time.Second
	}
}
