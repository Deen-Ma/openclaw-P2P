package p2p

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
)

const NegotiateProtocolID = protocol.ID("/openagent/negotiate/1.0.0")

type NegotiationOp string

const (
	NegotiationOffer    NegotiationOp = "offer"
	NegotiationAccept   NegotiationOp = "accept"
	NegotiationReject   NegotiationOp = "reject"
	NegotiationCancel   NegotiationOp = "cancel"
	NegotiationMessage  NegotiationOp = "message"
	NegotiationComplete NegotiationOp = "complete"
)

type NegotiationFrame struct {
	SessionID  string        `json:"session_id"`
	RefID      string        `json:"ref_id"`
	Op         NegotiationOp `json:"op"`
	FromPeer   string        `json:"from_peer"`
	ToPeer     string        `json:"to_peer,omitempty"`
	Status     string        `json:"status,omitempty"`
	Body       string        `json:"body,omitempty"`
	PayloadRef string        `json:"payload_ref,omitempty"`
	Timestamp  time.Time     `json:"timestamp"`
}

func RegisterNegotiateHandler(h host.Host, handler func(context.Context, NegotiationFrame) (NegotiationFrame, error)) {
	h.SetStreamHandler(NegotiateProtocolID, func(stream network.Stream) {
		defer stream.Close()
		var frame NegotiationFrame
		if err := json.NewDecoder(stream).Decode(&frame); err != nil {
			_ = json.NewEncoder(stream).Encode(NegotiationFrame{
				SessionID: frame.SessionID,
				RefID:     frame.RefID,
				Op:        frame.Op,
				Status:    "decode_error",
				Body:      err.Error(),
				Timestamp: time.Now().UTC(),
			})
			return
		}
		resp, err := handler(context.Background(), frame)
		if err != nil {
			resp = NegotiationFrame{
				SessionID: frame.SessionID,
				RefID:     frame.RefID,
				Op:        frame.Op,
				FromPeer:  frame.ToPeer,
				ToPeer:    frame.FromPeer,
				Status:    "error",
				Body:      err.Error(),
				Timestamp: time.Now().UTC(),
			}
		}
		_ = json.NewEncoder(stream).Encode(resp)
	})
}

func SendNegotiationFrame(ctx context.Context, h host.Host, target peer.ID, frame NegotiationFrame) (NegotiationFrame, error) {
	stream, err := h.NewStream(ctx, target, NegotiateProtocolID)
	if err != nil {
		return NegotiationFrame{}, err
	}
	defer stream.Close()

	if frame.Timestamp.IsZero() {
		frame.Timestamp = time.Now().UTC()
	}
	if err := validateNegotiation(frame); err != nil {
		return NegotiationFrame{}, err
	}
	if err := json.NewEncoder(stream).Encode(frame); err != nil {
		return NegotiationFrame{}, err
	}
	var resp NegotiationFrame
	if err := json.NewDecoder(stream).Decode(&resp); err != nil {
		return NegotiationFrame{}, err
	}
	return resp, nil
}

func validateNegotiation(frame NegotiationFrame) error {
	if frame.SessionID == "" {
		return errors.New("missing session_id")
	}
	if frame.RefID == "" {
		return errors.New("missing ref_id")
	}
	switch frame.Op {
	case NegotiationOffer, NegotiationAccept, NegotiationReject, NegotiationCancel, NegotiationMessage, NegotiationComplete:
		return nil
	default:
		return errors.New("invalid negotiation op")
	}
}
