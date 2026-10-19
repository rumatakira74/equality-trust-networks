// Eclipse-защита (12.3, 4.9): данные считаются подтверждёнными только после
// получения от N несвязанных пиров. До финализации (этап 5) подтверждение
// данными N пиров — единственный статус: единственная версия реестра от
// одного источника не принимается как окончательная.
//
// Вопрос №35 (этап 6): несвязанность оценивается по группам связи —
// IP-подсети (/24 для IPv4, /64 для IPv6; NodeConfig.LinkGroups).
// Пиры одной подсети считаются связанными: подтверждением служат N
// РАЗЛИЧНЫХ групп. ASN-классификация требует внешних данных (BGP-таблиц),
// в протоколе их нет — PENDING, предложение в отчёте этапа 6.
package net

import (
	"net/netip"
	"sync"

	"github.com/libp2p/go-libp2p/core/peer"
	ma "github.com/multiformats/go-multiaddr"
)

// LinkClassifier — оценка несвязанности: группа связи пира (подсеть).
// Пиры с одинаковой группой считаются связанными (12.3).
type LinkClassifier func(peer.ID) string

// PeerSubnetLink — группа связи по IP-адресу соединения: IPv4 → /24,
// IPv6 → /64; без IP-адреса (relay/circuit) — уникальная группа на пира
// (консервативно: не схлопывать неизвестных в одну группу).
func PeerSubnetLink(addrs []ma.Multiaddr) string {
	for _, a := range addrs {
		ip, ok := ipOfMultiaddr(a)
		if !ok {
			continue
		}
		bits := 24
		if ip.Is6() {
			bits = 64
		}
		return netip.PrefixFrom(ip, bits).Masked().String()
	}
	return ""
}

// ipOfMultiaddr — IP-компонент multiaddr (/ip4/... или /ip6/...).
func ipOfMultiaddr(a ma.Multiaddr) (netip.Addr, bool) {
	for _, c := range a {
		if c.Protocol().Code == ma.P_IP4 || c.Protocol().Code == ma.P_IP6 {
			ip, err := netip.ParseAddr(c.Value())
			if err == nil {
				return ip, true
			}
		}
	}
	return netip.Addr{}, false
}

// peerLink — группа связи пира узла (классификатор для Corroboration):
// подсеть удалённого адреса соединения; без соединения — уникальная группа.
func (n *Node) peerLink(p peer.ID) string {
	if n.Host == nil {
		return ""
	}
	addrs := make([]ma.Multiaddr, 0, 2)
	for _, c := range n.Host.Network().ConnsToPeer(p) {
		addrs = append(addrs, c.RemoteMultiaddr())
	}
	return PeerSubnetLink(addrs)
}

// Corroboration — учёт подтверждений txid от пиров.
type Corroboration struct {
	mu     sync.Mutex
	n      int
	linkOf LinkClassifier // nil — несвязанность = уникальные Peer ID
	seen   map[string]map[peer.ID]struct{}
	links  map[string]map[string]struct{} // txid -> группа -> кто доставил
}

// NewCorroboration — трекер с порогом N; несвязанность — уникальные Peer ID.
func NewCorroboration(n int) *Corroboration {
	return newCorroboration(n, nil)
}

// NewCorroborationLinked — трекер с оценкой несвязанности: подтверждение —
// N различных групп связи (подсети), а не N пиров (№35, этап 6).
func NewCorroborationLinked(n int, linkOf LinkClassifier) *Corroboration {
	return newCorroboration(n, linkOf)
}

func newCorroboration(n int, linkOf LinkClassifier) *Corroboration {
	if n <= 0 {
		n = 3
	}
	return &Corroboration{n: n, linkOf: linkOf,
		seen:  make(map[string]map[peer.ID]struct{}),
		links: make(map[string]map[string]struct{})}
}

// linkGroup — группа связи пира (без классификатора — уникальная на пира).
func (c *Corroboration) linkGroup(p peer.ID) string {
	if c.linkOf == nil {
		return "peer:" + string(p)
	}
	if g := c.linkOf(p); g != "" {
		return g
	}
	return "peer:" + string(p)
}

// Add фиксирует доставку txid от пира; возвращает текущее число пиров.
func (c *Corroboration) Add(txid []byte, p peer.ID) int {
	c.mu.Lock()
	defer c.mu.Unlock()
	k := string(txid)
	set, ok := c.seen[k]
	if !ok {
		set = make(map[peer.ID]struct{})
		c.seen[k] = set
	}
	set[p] = struct{}{}
	g := c.linkGroup(p)
	lset, ok := c.links[k]
	if !ok {
		lset = make(map[string]struct{})
		c.links[k] = lset
	}
	lset[g] = struct{}{}
	return len(set)
}

// Count — сколько пиров доставили txid.
func (c *Corroboration) Count(txid []byte) int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return len(c.seen[string(txid)])
}

// LinkSpread — метрика несвязанности (№35): число пиров и число различных
// групп связи, доставивших txid. groups < peers — есть связанные пиры
// (одна подсеть); подтверждение считается по группам.
func (c *Corroboration) LinkSpread(txid []byte) (peers, groups int) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return len(c.seen[string(txid)]), len(c.links[string(txid)])
}

// Confirmed — данные подтверждены N несвязанными пирами (12.3): без
// классификатора — N различных пиров, с классификатором — N различных
// групп связи (подсетей).
func (c *Corroboration) Confirmed(txid []byte) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.linkOf == nil {
		return len(c.seen[string(txid)]) >= c.n
	}
	return len(c.links[string(txid)]) >= c.n
}

// N — порог подтверждения.
func (c *Corroboration) N() int { return c.n }
