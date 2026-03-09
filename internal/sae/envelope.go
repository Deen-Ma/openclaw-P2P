package sae

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	crypto "github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/peer"
)

const (
	MaxEnvelopeSize = 1024
	DefaultTTL      = 24 * time.Hour
)

type Kind string

const (
	KindTask      Kind = "TaskSAE"
	KindAgentFact Kind = "AgentFactSAE"
)

type TaskOp string

const (
	OpUpsert   TaskOp = "UPSERT"
	OpComplete TaskOp = "COMPLETE"
	OpWithdraw TaskOp = "WITHDRAW"
)

type SAEEnvelope struct {
	Kind      Kind      `json:"kind"`
	Origin    string    `json:"origin"`
	Publisher string    `json:"publisher"`
	Seq       uint64    `json:"seq"`
	Topics    []string  `json:"topics"`
	Summary   string    `json:"summary"`
	DetailRef string    `json:"detail_ref"`
	ExpiresAt time.Time `json:"expires_at"`
	Sig       string    `json:"sig,omitempty"`

	TaskID   string `json:"task_id,omitempty"`
	Op       TaskOp `json:"op,omitempty"`
	Taxonomy string `json:"taxonomy,omitempty"`
	Conf     int    `json:"conf,omitempty"`

	FactID   string `json:"fact_id,omitempty"`
	FactKind string `json:"fact_kind,omitempty"`
}

type canonicalEnvelope struct {
	Kind      Kind      `json:"kind"`
	Origin    string    `json:"origin"`
	Publisher string    `json:"publisher"`
	Seq       uint64    `json:"seq"`
	Topics    []string  `json:"topics"`
	Summary   string    `json:"summary"`
	DetailRef string    `json:"detail_ref"`
	ExpiresAt time.Time `json:"expires_at"`

	TaskID   string `json:"task_id,omitempty"`
	Op       TaskOp `json:"op,omitempty"`
	Taxonomy string `json:"taxonomy,omitempty"`
	Conf     int    `json:"conf,omitempty"`

	FactID   string `json:"fact_id,omitempty"`
	FactKind string `json:"fact_kind,omitempty"`
}

func (e SAEEnvelope) ValidateUnsigned() error {
	if e.Kind != KindTask && e.Kind != KindAgentFact {
		return fmt.Errorf("unknown SAE kind %q", e.Kind)
	}
	if e.Origin == "" {
		return errors.New("missing origin")
	}
	if e.Publisher == "" {
		return errors.New("missing publisher")
	}
	if e.Seq == 0 {
		return errors.New("missing seq")
	}
	if len(e.Topics) == 0 {
		return errors.New("missing topics")
	}
	if e.Summary == "" {
		return errors.New("missing summary")
	}
	if e.DetailRef == "" {
		return errors.New("missing detail_ref")
	}
	if e.ExpiresAt.IsZero() {
		return errors.New("missing expires_at")
	}
	switch e.Kind {
	case KindTask:
		if e.TaskID == "" {
			return errors.New("missing task_id")
		}
		if e.Op != OpUpsert && e.Op != OpComplete && e.Op != OpWithdraw {
			return fmt.Errorf("invalid task op %q", e.Op)
		}
		if e.Taxonomy == "" {
			return errors.New("missing taxonomy")
		}
	case KindAgentFact:
		if e.FactID == "" {
			return errors.New("missing fact_id")
		}
		if e.FactKind == "" {
			return errors.New("missing fact_kind")
		}
		if e.Taxonomy == "" {
			return errors.New("missing taxonomy")
		}
	}
	return nil
}

func (e SAEEnvelope) ValidateSigned() error {
	if err := e.ValidateUnsigned(); err != nil {
		return err
	}
	if e.Sig == "" {
		return errors.New("missing signature")
	}
	encoded, err := e.Encode()
	if err != nil {
		return err
	}
	if len(encoded) > MaxEnvelopeSize {
		return fmt.Errorf("SAE exceeds %d bytes", MaxEnvelopeSize)
	}
	return nil
}

func (e SAEEnvelope) canonical() canonicalEnvelope {
	topics := make([]string, 0, len(e.Topics))
	for _, topic := range e.Topics {
		if topic = strings.TrimSpace(topic); topic != "" {
			topics = append(topics, topic)
		}
	}
	return canonicalEnvelope{
		Kind:      e.Kind,
		Origin:    e.Origin,
		Publisher: e.Publisher,
		Seq:       e.Seq,
		Topics:    topics,
		Summary:   e.Summary,
		DetailRef: e.DetailRef,
		ExpiresAt: e.ExpiresAt.UTC(),
		TaskID:    e.TaskID,
		Op:        e.Op,
		Taxonomy:  e.Taxonomy,
		Conf:      e.Conf,
		FactID:    e.FactID,
		FactKind:  e.FactKind,
	}
}

func (e SAEEnvelope) CanonicalBytes() ([]byte, error) {
	if err := e.ValidateUnsigned(); err != nil {
		return nil, err
	}
	return json.Marshal(e.canonical())
}

func (e *SAEEnvelope) Sign(priv crypto.PrivKey) error {
	canonical, err := e.CanonicalBytes()
	if err != nil {
		return err
	}
	sig, err := priv.Sign(canonical)
	if err != nil {
		return fmt.Errorf("sign SAE: %w", err)
	}
	e.Sig = base64.RawURLEncoding.EncodeToString(sig)
	return e.ValidateSigned()
}

func (e SAEEnvelope) Verify(pub crypto.PubKey) error {
	if err := e.ValidateSigned(); err != nil {
		return err
	}
	canonical, err := e.CanonicalBytes()
	if err != nil {
		return err
	}
	sig, err := base64.RawURLEncoding.DecodeString(e.Sig)
	if err != nil {
		return fmt.Errorf("decode signature: %w", err)
	}
	ok, err := pub.Verify(canonical, sig)
	if err != nil {
		return fmt.Errorf("verify signature: %w", err)
	}
	if !ok {
		return errors.New("invalid signature")
	}
	return nil
}

func (e SAEEnvelope) VerifyPublisher() error {
	pid, err := peer.Decode(e.Publisher)
	if err != nil {
		return fmt.Errorf("decode publisher peer ID: %w", err)
	}
	pub, err := pid.ExtractPublicKey()
	if err != nil || pub == nil {
		return fmt.Errorf("extract publisher public key: %w", err)
	}
	if err := e.Verify(pub); err != nil {
		return err
	}
	if got := DIDFromPubKey(pub); got != e.Origin {
		return fmt.Errorf("origin DID mismatch: got %s want %s", e.Origin, got)
	}
	return nil
}

func (e SAEEnvelope) Encode() ([]byte, error) {
	body, err := json.Marshal(e)
	if err != nil {
		return nil, err
	}
	if len(body) > MaxEnvelopeSize {
		return nil, fmt.Errorf("SAE exceeds %d bytes", MaxEnvelopeSize)
	}
	return body, nil
}

func DecodeEnvelope(data []byte) (SAEEnvelope, error) {
	if len(data) > MaxEnvelopeSize {
		return SAEEnvelope{}, fmt.Errorf("SAE exceeds %d bytes", MaxEnvelopeSize)
	}
	var env SAEEnvelope
	if err := json.Unmarshal(data, &env); err != nil {
		return SAEEnvelope{}, err
	}
	return env, env.ValidateSigned()
}

func DIDFromPubKey(pub crypto.PubKey) string {
	raw, err := pub.Raw()
	if err != nil {
		marshaled, marshalErr := crypto.MarshalPublicKey(pub)
		if marshalErr != nil {
			marshaled = []byte(fmt.Sprintf("%T", pub))
		}
		sum := sha256.Sum256(marshaled)
		return "did:key:" + hex.EncodeToString(sum[:16])
	}
	return "did:key:" + base64.RawURLEncoding.EncodeToString(raw)
}

func HashString(parts ...string) string {
	sum := sha256.Sum256([]byte(strings.Join(parts, "::")))
	return hex.EncodeToString(sum[:])
}
