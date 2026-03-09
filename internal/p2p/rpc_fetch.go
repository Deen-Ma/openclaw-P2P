package p2p

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"path"
	"strings"

	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
)

const FetchProtocolID = protocol.ID("/openagent/fetch/1.0.0")

type FetchRequest struct {
	DetailRef string `json:"detail_ref"`
}

type FetchResponse struct {
	Found       bool            `json:"found"`
	ContentType string          `json:"content_type,omitempty"`
	Payload     json.RawMessage `json:"payload,omitempty"`
	SHA256      string          `json:"sha256,omitempty"`
	Error       string          `json:"error,omitempty"`
}

type DetailRef struct {
	PeerID  peer.ID
	Kind    string
	ID      string
	Version string
}

func BuildDetailRef(peerID peer.ID, kind, objectID string, version uint64) string {
	return fmt.Sprintf("openagent://%s/%s/%s/%d", peerID.String(), kind, url.PathEscape(objectID), version)
}

func ParseDetailRef(raw string) (DetailRef, error) {
	u, err := url.Parse(raw)
	if err != nil {
		return DetailRef{}, err
	}
	if u.Scheme != "openagent" {
		return DetailRef{}, fmt.Errorf("unsupported detail_ref scheme %q", u.Scheme)
	}
	pid, err := peer.Decode(u.Host)
	if err != nil {
		return DetailRef{}, fmt.Errorf("decode peer ID: %w", err)
	}
	parts := strings.Split(strings.Trim(path.Clean(u.Path), "/"), "/")
	if len(parts) < 3 {
		return DetailRef{}, errors.New("detail_ref must include /kind/id/version")
	}
	objectID, err := url.PathUnescape(parts[1])
	if err != nil {
		return DetailRef{}, fmt.Errorf("unescape detail object id: %w", err)
	}
	return DetailRef{
		PeerID:  pid,
		Kind:    parts[0],
		ID:      objectID,
		Version: parts[2],
	}, nil
}

func RegisterFetchHandler(h host.Host, handler func(context.Context, FetchRequest) (FetchResponse, error)) {
	h.SetStreamHandler(FetchProtocolID, func(stream network.Stream) {
		defer stream.Close()
		var req FetchRequest
		if err := json.NewDecoder(stream).Decode(&req); err != nil {
			_ = json.NewEncoder(stream).Encode(FetchResponse{Found: false, Error: err.Error()})
			return
		}
		resp, err := handler(context.Background(), req)
		if err != nil {
			resp = FetchResponse{Found: false, Error: err.Error()}
		}
		_ = json.NewEncoder(stream).Encode(resp)
	})
}

func FetchDetail(ctx context.Context, h host.Host, target peer.ID, detailRef string) (FetchResponse, error) {
	stream, err := h.NewStream(ctx, target, FetchProtocolID)
	if err != nil {
		return FetchResponse{}, err
	}
	defer stream.Close()

	if err := json.NewEncoder(stream).Encode(FetchRequest{DetailRef: detailRef}); err != nil {
		return FetchResponse{}, err
	}

	var resp FetchResponse
	if err := json.NewDecoder(stream).Decode(&resp); err != nil {
		return FetchResponse{}, err
	}
	if !resp.Found && resp.Error != "" {
		return resp, errors.New(resp.Error)
	}
	return resp, nil
}

func DigestPayload(payload json.RawMessage) string {
	sum := sha256.Sum256(payload)
	return hex.EncodeToString(sum[:])
}
