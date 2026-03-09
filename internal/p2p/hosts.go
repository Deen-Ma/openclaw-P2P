package p2p

import (
	"context"
	"crypto/rand"
	"fmt"
	"os"
	"path/filepath"

	libp2p "github.com/libp2p/go-libp2p"
	crypto "github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"

	"openagent/internal/sae"
)

type Identity struct {
	PrivKey crypto.PrivKey
	PubKey  crypto.PubKey
	PeerID  peer.ID
	DID     string
	KeyPath string
}

type HostBundle struct {
	Host     host.Host
	Identity Identity
}

func CreateHost(ctx context.Context, listenPort int, identityPath string) (*HostBundle, error) {
	priv, err := loadOrCreateIdentity(identityPath)
	if err != nil {
		return nil, err
	}
	hostNode, err := libp2p.New(
		libp2p.ListenAddrStrings(fmt.Sprintf("/ip4/0.0.0.0/tcp/%d", listenPort)),
		libp2p.Identity(priv),
		libp2p.NATPortMap(),
	)
	if err != nil {
		return nil, fmt.Errorf("create host: %w", err)
	}
	pub := priv.GetPublic()
	return &HostBundle{
		Host: hostNode,
		Identity: Identity{
			PrivKey: priv,
			PubKey:  pub,
			PeerID:  hostNode.ID(),
			DID:     sae.DIDFromPubKey(pub),
			KeyPath: identityPath,
		},
	}, nil
}

func loadOrCreateIdentity(identityPath string) (crypto.PrivKey, error) {
	if err := os.MkdirAll(filepath.Dir(identityPath), 0o700); err != nil {
		return nil, fmt.Errorf("create identity dir: %w", err)
	}
	if raw, err := os.ReadFile(identityPath); err == nil {
		return crypto.UnmarshalPrivateKey(raw)
	} else if !os.IsNotExist(err) {
		return nil, err
	}

	priv, _, err := crypto.GenerateEd25519Key(rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("generate Ed25519 key: %w", err)
	}
	raw, err := crypto.MarshalPrivateKey(priv)
	if err != nil {
		return nil, fmt.Errorf("marshal private key: %w", err)
	}
	if err := os.WriteFile(identityPath, raw, 0o600); err != nil {
		return nil, fmt.Errorf("persist private key: %w", err)
	}
	return priv, nil
}
