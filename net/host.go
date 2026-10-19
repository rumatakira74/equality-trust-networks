// Хост libp2p (4.1): QUIC + TCP, Noise, Peer ID из ключа узла.
// Идентификация пиров включает network_id: пир чужой сети отсеивается сразу
// по UserAgent после identify (12.3) — соединение закрывается.
package net

import (
	"context"
	"crypto/ecdsa"
	"crypto/x509"
	"encoding/hex"
	"fmt"

	"github.com/libp2p/go-libp2p"
	libp2pcrypto "github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/event"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
	"github.com/libp2p/go-libp2p/p2p/security/noise"
	quic "github.com/libp2p/go-libp2p/p2p/transport/quic"
	"github.com/libp2p/go-libp2p/p2p/transport/tcp"
	"github.com/multiformats/go-multiaddr"
)

// UserAgentPrefix — префикс UserAgent узлов платформы; далее hex(network_id).
const userAgentPrefix = "tnp/"

// libp2pPrivKey — алиас типа ключа libp2p.
type libp2pPrivKey = libp2pcrypto.PrivKey

// protocolPrefix — префикс всех протоколов узла: /tnp/<hex(network_id)>/...
func protocolPrefix(networkID []byte) protocol.ID {
	return protocol.ID(fmt.Sprintf("/tnp/%s", hex.EncodeToString(networkID)))
}

// протоколы узла (идентификация включает network_id).
func (n *Node) syncProtocol() protocol.ID { return protocolPrefix(n.networkID) + "/sync/1.0.0" }
func (n *Node) pxProtocol() protocol.ID   { return protocolPrefix(n.networkID) + "/px/1.0.0" }

// voteProtocol — протокол голосования по конфликтам (этап 5, 9.1).
func (n *Node) voteProtocol() protocol.ID { return protocolPrefix(n.networkID) + "/vote/1.0.0" }
func (n *Node) txTopic() string           { return "tx/" + hex.EncodeToString(n.networkID) }
func (n *Node) cpTopic() string           { return "checkpoint/" + hex.EncodeToString(n.networkID) }

// nodeKeyToLibp2p — ключ узла P-256 (core/crypto) -> ключ libp2p: Peer ID
// выводится из ключа узла (12.1).
func nodeKeyToLibp2p(priv *ecdsa.PrivateKey) (libp2pcrypto.PrivKey, error) {
	der, err := x509.MarshalECPrivateKey(priv)
	if err != nil {
		return nil, err
	}
	return libp2pcrypto.UnmarshalECDSAPrivateKey(der)
}

// newNodeHost создаёт хост: QUIC + TCP, Noise, relay/DCUtR (4.7).
func (n *Node) newNodeHost() (host.Host, error) {
	cfg := n.Cfg
	opts := []libp2p.Option{
		libp2p.Transport(quic.NewTransport),
		libp2p.Transport(tcp.NewTCPTransport),
		libp2p.Security(noise.ID, noise.New),
		libp2p.Identity(n.privKey),
		libp2p.UserAgent(userAgentPrefix + hex.EncodeToString(n.networkID)),
		libp2p.ProtocolVersion("tnp/1.0.0"),
		libp2p.EnableRelay(),        // circuit relay v2: клиент
		libp2p.EnableHolePunching(), // DCUtR (CGNAT, 12.2)
	}
	if len(cfg.ListenAddrs) > 0 {
		opts = append(opts, libp2p.ListenAddrStrings(cfg.ListenAddrs...))
	}
	if cfg.NoAnnounce {
		// Эмуляция CGNAT: узел не анонсирует адреса — входящие только
		// через relay/DCUtR (12.2).
		opts = append(opts, libp2p.AddrsFactory(func([]multiaddr.Multiaddr) []multiaddr.Multiaddr {
			return nil
		}))
	}
	if cfg.Level == LevelFull {
		// Полный узел: добровольный relay-транслятор и NAT-сервис (12.2).
		// Relay-сервис активен только на достижимых узлах: в продакшене
		// достижимость определяет AutoNAT, в тестах/эмуляции — ForcePublic.
		opts = append(opts, libp2p.EnableRelayService(), libp2p.EnableNATService())
		if cfg.ForcePublic {
			opts = append(opts, libp2p.ForceReachabilityPublic())
		}
	}
	return libp2p.New(opts...)
}

// watchNetworkFilter — фильтрация пиров по network_id (4.6): после identify
// пир с чужим UserAgent (другая сеть) отключается. Смешивание: событие
// EvtPeerIdentificationFailed тоже закрывает соединение.
func (n *Node) watchNetworkFilter(ctx context.Context) error {
	sub, err := n.Host.EventBus().Subscribe(
		[]interface{}{new(event.EvtPeerIdentificationCompleted), new(event.EvtPeerIdentificationFailed)})
	if err != nil {
		return err
	}
	go func() {
		defer sub.Close()
		want := userAgentPrefix + hex.EncodeToString(n.networkID)
		for {
			select {
			case <-ctx.Done():
				return
			case ev, ok := <-sub.Out():
				if !ok {
					return
				}
				switch e := ev.(type) {
				case event.EvtPeerIdentificationCompleted:
					if e.AgentVersion != want {
						// пир чужой сети (или не платформы) — отключаем
						n.closePeerConns(e.Peer)
						n.logf("изоляция: пир %s отключён (agent %q)", e.Peer.ShortString(), e.AgentVersion)
					}
				case event.EvtPeerIdentificationFailed:
					n.closePeerConns(e.Peer)
				}
			}
		}
	}()
	return nil
}

func (n *Node) closePeerConns(p peer.ID) {
	conns := n.Host.Network().ConnsToPeer(p)
	for _, c := range conns {
		_ = c.Close()
	}
}

// connectedPeers — подключённые пиры нашей сети (по протоколу sync).
func (n *Node) connectedPeers() []peer.ID {
	if n.Host == nil {
		return nil
	}
	return n.Host.Network().Peers()
}

// ConnectedPeers — подключённые пиры (экспорт для моста Android, этап 6).
func (n *Node) ConnectedPeers() []peer.ID { return n.connectedPeers() }

// hasProtocolConn — true, если пир говорит хотя бы один наш протокол
// (после identify чужие уже отфильтрованы).
func hasProtocolConn(h host.Host, p peer.ID) bool {
	protos, err := h.Peerstore().GetProtocols(p)
	if err != nil {
		return false
	}
	return len(protos) > 0
}

// addrInfosOf — собственные адреса узла (для peer exchange).
func addrInfosOf(h host.Host) peer.AddrInfo {
	return peer.AddrInfo{ID: h.ID(), Addrs: h.Addrs()}
}
