package p2p

import (
	"context"
	"fmt"
	"strings"
	"time"

	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/p2p/discovery/mdns"
	routingdiscovery "github.com/libp2p/go-libp2p/p2p/discovery/routing"
	ma "github.com/multiformats/go-multiaddr"
)

type DiscoveryServices struct {
	DHT             *dht.IpfsDHT
	Routing         *routingdiscovery.RoutingDiscovery
	MDNSService     mdns.Service
	cancelDiscovery context.CancelFunc
}

func (d *DiscoveryServices) Close() error {
	if d == nil {
		return nil
	}
	if d.cancelDiscovery != nil {
		d.cancelDiscovery()
	}
	if d.MDNSService != nil {
		return d.MDNSService.Close()
	}
	return nil
}

type discoveryNotifee struct {
	onPeer func(peer.AddrInfo)
}

func (n *discoveryNotifee) HandlePeerFound(info peer.AddrInfo) {
	if n.onPeer != nil {
		n.onPeer(info)
	}
}

func SetupMDNS(h host.Host, rendezvous string, onPeer func(peer.AddrInfo)) (mdns.Service, error) {
	service := mdns.NewMdnsService(h, rendezvous, &discoveryNotifee{onPeer: onPeer})
	if err := service.Start(); err != nil {
		return nil, err
	}
	return service, nil
}

func SetupRendezvous(ctx context.Context, h host.Host, rendezvous string, bootstrapPeers []string, onPeer func(peer.AddrInfo)) (*dht.IpfsDHT, *routingdiscovery.RoutingDiscovery, context.CancelFunc, error) {
	discoveryCtx, cancel := context.WithCancel(ctx)

	kad, err := dht.New(discoveryCtx, h)
	if err != nil {
		cancel()
		return nil, nil, nil, fmt.Errorf("create DHT: %w", err)
	}
	bootstrapInfos, err := parseBootstrapPeers(bootstrapPeers)
	if err != nil {
		cancel()
		return nil, nil, nil, err
	}
	if err := ensureConnectedPeers(discoveryCtx, h, bootstrapInfos, onPeer); err != nil {
		cancel()
		return nil, nil, nil, err
	}
	if err := kad.Bootstrap(discoveryCtx); err != nil {
		cancel()
		return nil, nil, nil, fmt.Errorf("bootstrap DHT: %w", err)
	}

	routing := routingdiscovery.NewRoutingDiscovery(kad)
	_ = tryAdvertise(discoveryCtx, routing, rendezvous)

	go func() {
		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()
		for {
			_ = ensureConnectedPeers(discoveryCtx, h, bootstrapInfos, onPeer)
			_ = tryAdvertise(discoveryCtx, routing, rendezvous)
			findAndConnect(discoveryCtx, h, routing, rendezvous, onPeer)
			select {
			case <-discoveryCtx.Done():
				return
			case <-ticker.C:
			}
		}
	}()

	return kad, routing, cancel, nil
}

func tryAdvertise(ctx context.Context, routing *routingdiscovery.RoutingDiscovery, rendezvous string) error {
	_, err := routing.Advertise(ctx, rendezvous)
	if err == nil {
		return nil
	}
	// The first bootstrap node may start with an empty routing table.
	// In that case we keep the node alive and retry after peers arrive.
	if strings.Contains(err.Error(), "failed to find any peer in table") {
		return nil
	}
	return fmt.Errorf("advertise rendezvous: %w", err)
}

func parseBootstrapPeers(bootstrapPeers []string) ([]peer.AddrInfo, error) {
	infos := make([]peer.AddrInfo, 0, len(bootstrapPeers))
	for _, bootstrap := range bootstrapPeers {
		addr, err := ma.NewMultiaddr(bootstrap)
		if err != nil {
			return nil, fmt.Errorf("parse bootstrap %q: %w", bootstrap, err)
		}
		info, err := peer.AddrInfoFromP2pAddr(addr)
		if err != nil {
			return nil, fmt.Errorf("bootstrap addr info %q: %w", bootstrap, err)
		}
		infos = append(infos, *info)
	}
	return infos, nil
}

func ensureConnectedPeers(ctx context.Context, h host.Host, peers []peer.AddrInfo, onPeer func(peer.AddrInfo)) error {
	for _, info := range peers {
		if info.ID == h.ID() {
			continue
		}
		if h.Network().Connectedness(info.ID) != network.Connected {
			if err := h.Connect(ctx, info); err != nil {
				return fmt.Errorf("connect peer %s: %w", info.ID, err)
			}
		}
		if onPeer != nil {
			onPeer(info)
		}
	}
	return nil
}

func findAndConnect(ctx context.Context, h host.Host, routing *routingdiscovery.RoutingDiscovery, rendezvous string, onPeer func(peer.AddrInfo)) {
	peers, err := routing.FindPeers(ctx, rendezvous)
	if err != nil {
		return
	}
	for info := range peers {
		if info.ID == "" || info.ID == h.ID() {
			continue
		}
		if h.Network().Connectedness(info.ID) == network.Connected || h.Connect(ctx, info) == nil {
			if onPeer != nil {
				onPeer(info)
			}
		}
	}
}
