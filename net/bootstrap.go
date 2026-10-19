// Bootstrap-каскад (12.3, 4.5) — порядок фиксирован:
//
//  1. кэш прошлых подключений (SQLite, core/store);
//  2. подписанный seed-список (net/seedlist; N-из-M через core/threshold);
//  3. зашитый список адресов (embedded, устаревший fallback);
//  4. NSD/mDNS — libp2p mDNS-дискавери в локальной Wi-Fi сети;
//  5. Bluetooth-скан — заглушка-интерфейс (реальный BT — этап 6);
//  6. QR «добавить узел» и ручной ввод — конфигурация узла;
//  7. peer exchange — обмен списками пиров с подключёнными.
//
// «Смешивание источников» (12.3): узел подключается к пирам из нескольких
// источников и не доверяет единственному (данные — только после
// подтверждения от N пиров, net/eclipse).
//
// invite_only (12.3): каналы (seed-списки) и зашитый список не используются;
// адреса первых узлов — из приглашения (ManualAddrs); несовместимые пиры
// отключаются фильтром network_id (net/host).
package net

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	mdns "github.com/libp2p/go-libp2p/p2p/discovery/mdns"
	ma "github.com/multiformats/go-multiaddr"
)

// peerSourceKind — источник пира каскада (для смешивания, 12.3).
type peerSourceKind int

const (
	srcCache peerSourceKind = iota
	srcSeedList
	srcEmbedded
	srcMDNS
	srcBluetooth
	srcManual
	srcPeerExchange
)

func (k peerSourceKind) name() string {
	switch k {
	case srcCache:
		return "cache"
	case srcSeedList:
		return "seedlist"
	case srcEmbedded:
		return "embedded"
	case srcMDNS:
		return "mdns"
	case srcBluetooth:
		return "bluetooth"
	case srcManual:
		return "manual"
	default:
		return "px"
	}
}

// discoveredPeer — пир с источником.
type discoveredPeer struct {
	info   peer.AddrInfo
	source peerSourceKind
}

// mdnsNotifee — приём mDNS-событий (шаг 4 каскада).
type mdnsNotifee struct {
	mu    sync.Mutex
	found []peer.AddrInfo
}

func (m *mdnsNotifee) HandlePeerFound(p peer.AddrInfo) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.found = append(m.found, p)
}

func (m *mdnsNotifee) take() []peer.AddrInfo {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := m.found
	m.found = nil
	return out
}

// startMDNS — mDNS-дискавери в локальной сети (NSD, 12.3).
func (n *Node) startMDNS() error {
	n.mdnsNotifee = &mdnsNotifee{}
	svc := mdns.NewMdnsService(n.Host, "tnp-"+fmt.Sprintf("%x", n.networkID)[:8], n.mdnsNotifee)
	if err := svc.Start(); err != nil {
		// mDNS недоступен (нет multicast-интерфейса) — источник просто
		// пропускается, каскад продолжается (12.3: best-effort).
		n.mdnsNotifee = nil
		return nil
	}
	n.mdnsSvc = svc
	return nil
}

// discoverCascade — сбор пиров по каскаду в фиксированном порядке (12.3).
// Возвращает кандидатов с источниками; подключение — отдельно (смешивание).
func (n *Node) discoverCascade(ctx context.Context) []discoveredPeer {
	var out []discoveredPeer
	seen := make(map[peer.ID]bool)
	add := func(ps []discoveredPeer) {
		for _, p := range ps {
			if p.info.ID == n.Host.ID() || seen[p.info.ID] {
				continue
			}
			seen[p.info.ID] = true
			out = append(out, p)
		}
	}

	// invite_only (12.3): каналы (2) и зашитый список (3) не используются.
	inviteOnly := n.Cfg.inviteOnly()

	// 1. Кэш прошлых подключений (SQLite).
	add(n.cachePeers())

	// 2. Подписанный seed-список.
	if !inviteOnly && len(n.Cfg.SeedList) > 0 {
		if ps, err := n.seedListPeers(); err != nil {
			n.logf("bootstrap: seed-список отклонён: %v", err)
		} else {
			add(ps)
		}
	}

	// 3. Зашитый список (устаревший fallback).
	if !inviteOnly {
		add(parseAddrInfos(n.Cfg.EmbeddedAddrs, srcEmbedded))
	}

	// 4. NSD/mDNS в локальной Wi-Fi сети.
	if n.mdnsNotifee != nil {
		for _, ai := range n.mdnsNotifee.take() {
			add([]discoveredPeer{{info: ai, source: srcMDNS}})
		}
	}

	// 5. Bluetooth-скан — заглушка (этап 6).
	if ps, err := (BluetoothScanSource{}).Peers(); err == nil {
		add(parseAddrInfos(addrsOf(ps), srcBluetooth))
	}

	// 6. QR «добавить узел» и ручной ввод — конфигурация узла.
	add(parseAddrInfos(n.Cfg.ManualAddrs, srcManual))

	// 7. Peer exchange: обмен списками пиров с подключёнными.
	for _, p := range n.connectedPeers() {
		if ps, err := n.pxRequest(ctx, p); err == nil {
			add(ps)
		}
	}
	return out
}

// bootstrap — подключение к минимальному числу пиров со смешиванием
// источников (12.3): хотя бы по одному пиру из каждого доступного
// источника, затем добор до MinPeers.
func (n *Node) bootstrap(ctx context.Context) error {
	candidates := n.discoverCascade(ctx)
	if len(candidates) == 0 {
		// холодный старт (12.3): первых узлам не к кому подключаться —
		// узел запускается и ждёт входящих (invite_only: приглашение).
		n.logf("bootstrap: нет источников пиров — ожидаю входящих")
		return nil
	}
	// группировка по источникам (смешивание: минимум по одному из каждого)
	bySource := make(map[peerSourceKind][]discoveredPeer)
	var order []peerSourceKind
	for _, c := range candidates {
		if _, ok := bySource[c.source]; !ok {
			order = append(order, c.source)
		}
		bySource[c.source] = append(bySource[c.source], c)
	}
	// детерминированное смешивание: по одному из каждого источника по кругу
	var mixed []discoveredPeer
	for round := 0; len(mixed) < len(candidates); round++ {
		progress := false
		for _, src := range order {
			list := bySource[src]
			if round < len(list) {
				mixed = append(mixed, list[round])
				progress = true
			}
		}
		if !progress {
			break
		}
	}
	connected := 0
	for _, c := range mixed {
		// Смешивание источников (12.3): подключаем до MinPeers*2 пиров из
		// разных источников — узел не доверяет единственному источнику.
		if connected >= n.Cfg.MinPeers*2 {
			break
		}
		if len(c.info.Addrs) == 0 {
			continue
		}
		if err := n.Host.Connect(ctx, c.info); err != nil {
			n.logf("bootstrap: %s: %v: %v", c.source.name(), c.info.ID.ShortString(), err)
			continue
		}
		n.savePeerCache(c.info)
		connected++
	}
	if connected == 0 {
		return errors.New("net: bootstrap: ни один пир не доступен")
	}
	return nil
}

// ---------------------------------------------------------------------------
// Кэш прошлых подключений (шаг 1; SQLite core/store)
// ---------------------------------------------------------------------------

const peerCacheKey = "peer_cache"

func (n *Node) savePeerCache(ai peer.AddrInfo) {
	if n.Store == nil || len(ai.Addrs) == 0 {
		return
	}
	n.mu.Lock()
	defer n.mu.Unlock()
	cache := n.loadPeerCacheLocked()
	addrs := make(map[string]bool)
	for _, a := range cache {
		addrs[a] = true
	}
	for _, a := range ai.Addrs {
		addrs[a.String()] = true
	}
	out := make([]string, 0, len(addrs))
	for a := range addrs {
		out = append(out, a)
	}
	// не более 64 адресов в кэше
	if len(out) > 64 {
		out = out[:64]
	}
	blob, _ := json.Marshal(out)
	_ = n.Store.SaveMeta(peerCacheKey, blob)
}

func (n *Node) loadPeerCacheLocked() []string {
	if n.Store == nil {
		return nil
	}
	blob, ok, err := n.Store.LoadMeta(peerCacheKey)
	if !ok || err != nil {
		return nil
	}
	var out []string
	if json.Unmarshal(blob, &out) != nil {
		return nil
	}
	return out
}

func (n *Node) cachePeers() []discoveredPeer {
	n.mu.Lock()
	addrs := n.loadPeerCacheLocked()
	n.mu.Unlock()
	return parseAddrInfos(addrs, srcCache)
}

// ---------------------------------------------------------------------------
// Seed-список (шаг 2)
// ---------------------------------------------------------------------------

func (n *Node) seedListPeers() ([]discoveredPeer, error) {
	if len(n.Cfg.SeedPubKeys) == 0 {
		return nil, errors.New("нет ключей подписантов seed-списка")
	}
	sl, err := VerifySeedList(n.Cfg.SeedList, n.networkID, n.Cfg.SeedThreshold, n.Cfg.SeedPubKeys, 0)
	if err != nil {
		return nil, err
	}
	if string(sl.NetworkID) != string(n.networkID) {
		return nil, errors.New("seed-список чужой сети")
	}
	var out []discoveredPeer
	for _, p := range sl.Peers {
		ai, err := parsePeerAddr(p)
		if err != nil {
			continue
		}
		out = append(out, discoveredPeer{info: ai, source: srcSeedList})
	}
	return out, nil
}

// ---------------------------------------------------------------------------
// Peer exchange (шаг 7)
// ---------------------------------------------------------------------------

// servePX — обработчик peer exchange.
func (n *Node) servePX(s network.Stream) {
	defer s.Close()
	// отдаём своих пиров (адреса из peerstore)
	protos := n.connectedPeers()
	var peers []peerAddrMsg
	for _, p := range protos {
		addrs := n.Host.Peerstore().Addrs(p)
		if len(addrs) == 0 {
			continue
		}
		pm := peerAddrMsg{ID: p.String()}
		for _, a := range addrs {
			pm.Addrs = append(pm.Addrs, a.String())
		}
		peers = append(peers, pm)
	}
	// добавляем себя
	self := addrInfosOf(n.Host)
	if len(self.Addrs) > 0 {
		pm := peerAddrMsg{ID: self.ID.String()}
		for _, a := range self.Addrs {
			pm.Addrs = append(pm.Addrs, a.String())
		}
		peers = append(peers, pm)
	}
	_, _ = s.Write(encodePeers(peers))
}

// pxRequest — запрос списка пиров у подключённого пира.
func (n *Node) pxRequest(ctx context.Context, p peer.ID) ([]discoveredPeer, error) {
	s, err := n.Host.NewStream(ctx, p, n.pxProtocol())
	if err != nil {
		return nil, err
	}
	defer s.Close()
	s.SetDeadline(time.Now().Add(10 * time.Second))
	if _, err := s.Write(writeFrame(&enc{}, msgGetPeers)); err != nil {
		return nil, err
	}
	raw, err := readFrameStream(s)
	if err != nil {
		return nil, err
	}
	msgType, d, err := readFrame(raw)
	if err != nil || msgType != msgPeers {
		return nil, errors.New("net: некорректный px-ответ")
	}
	peers, err := decodePeers(d)
	if err != nil {
		return nil, err
	}
	var out []discoveredPeer
	for _, pm := range peers {
		ai, err := parsePeerAddr(pm)
		if err != nil {
			continue
		}
		if ai.ID == n.Host.ID() {
			continue
		}
		out = append(out, discoveredPeer{info: ai, source: srcPeerExchange})
	}
	return out, nil
}

// ---------------------------------------------------------------------------
// Разбор адресов
// ---------------------------------------------------------------------------

func parseAddrInfos(addrs []string, src peerSourceKind) []discoveredPeer {
	var out []discoveredPeer
	for _, a := range addrs {
		if a == "" {
			continue
		}
		m, err := ma.NewMultiaddr(a)
		if err != nil {
			continue
		}
		ai, err := peer.AddrInfoFromP2pAddr(m)
		if err != nil || ai == nil {
			continue
		}
		out = append(out, discoveredPeer{info: *ai, source: src})
	}
	return out
}

func parsePeerAddr(p peerAddrMsg) (peer.AddrInfo, error) {
	var out peer.AddrInfo
	id, err := peer.Decode(p.ID)
	if err != nil {
		return out, err
	}
	out.ID = id
	for _, a := range p.Addrs {
		m, err := ma.NewMultiaddr(a)
		if err != nil {
			continue
		}
		out.Addrs = append(out.Addrs, m)
	}
	return out, nil
}

func addrsOf(ps []peerAddrMsg) []string {
	var out []string
	for _, p := range ps {
		out = append(out, p.Addrs...)
	}
	return out
}
