// CGNAT (12.2, 4.7): телефон за мобильным NAT не принимает входящих.
// Hole punching (DCUtR) — для прямых соединений; circuit relay v2 —
// ретрансляция через добровольные полные узлы (не серверы проекта).
// Ретранслятор — конфигурация полного узла (EnableRelayService в host.go);
// здесь — клиентская часть: резерв слота на relay-узлах.
package net

import (
	"context"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
	relayv2client "github.com/libp2p/go-libp2p/p2p/protocol/circuitv2/client"
	ma "github.com/multiformats/go-multiaddr"
)

// reserveRelays — подключение к relay-узлам и резерв circuit-v2-слота:
// после резерва узел доступен по адресу relay/p2p-circuit (CGNAT, 12.2).
func (n *Node) reserveRelays() error {
	for _, addr := range n.Cfg.RelayAddrs {
		m, err := ma.NewMultiaddr(addr)
		if err != nil {
			n.logf("relay: некорректный адрес %q: %v", addr, err)
			continue
		}
		ai, err := peer.AddrInfoFromP2pAddr(m)
		if err != nil {
			n.logf("relay: некорректный адрес %q: %v", addr, err)
			continue
		}
		ctx, cancel := context.WithTimeout(n.ctx, 15*time.Second)
		err = n.Host.Connect(ctx, *ai)
		if err == nil {
			if _, err := relayv2client.Reserve(ctx, n.Host, *ai); err == nil {
				n.relayReserved++
				n.logf("relay: резерв на %s", ai.ID.ShortString())
			} else {
				n.logf("relay: резерв на %s: %v", ai.ID.ShortString(), err)
			}
		} else {
			n.logf("relay: подключение к %s: %v", ai.ID.ShortString(), err)
		}
		cancel()
	}
	if n.relayReserved == 0 && len(n.Cfg.RelayAddrs) > 0 {
		return errNoRelayReservation
	}
	return nil
}

// errNoRelayReservation — ни один relay не принял резерв.
var errNoRelayReservation = errString("net: relay-резерв не получен")

type errString string

func (e errString) Error() string { return string(e) }

// CircuitAddr — адрес узла через relay (для тестов эмуляции CGNAT):
// relayAddr/p2p/<relayID>/p2p-circuit/p2p/<node>.
func CircuitAddr(relay peer.AddrInfo, node peer.ID) (ma.Multiaddr, error) {
	base, err := peer.AddrInfoToP2pAddrs(&relay)
	if err != nil || len(base) == 0 {
		return nil, err
	}
	circuit, err := ma.NewMultiaddr("/p2p-circuit/p2p/" + node.String())
	if err != nil {
		return nil, err
	}
	return base[0].Encapsulate(circuit), nil
}
