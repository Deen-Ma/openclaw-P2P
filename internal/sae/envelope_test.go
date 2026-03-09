package sae

import (
	"crypto/rand"
	"testing"
	"time"

	crypto "github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/peer"
)

func TestTaskSAESignVerify(t *testing.T) {
	priv, pub, err := crypto.GenerateEd25519Key(rand.Reader)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	pid, err := peer.IDFromPublicKey(pub)
	if err != nil {
		t.Fatalf("peer id: %v", err)
	}

	task := NewTaskSAE(DIDFromPubKey(pub), pid.String(), "task:123", "crowd.data_labeling", "Need image labels", "openagent://peer/task/task:123/1", []string{"openagent/v1/crowd"}, 900, OpUpsert, 1, time.Hour)
	if err := task.Sign(priv); err != nil {
		t.Fatalf("sign: %v", err)
	}
	if err := task.VerifyPublisher(); err != nil {
		t.Fatalf("verify publisher: %v", err)
	}
	if _, err := DecodeEnvelope(mustEncode(t, task.SAEEnvelope)); err != nil {
		t.Fatalf("decode envelope: %v", err)
	}
}

func TestEnvelopeRejectsOversizedPayload(t *testing.T) {
	priv, pub, err := crypto.GenerateEd25519Key(rand.Reader)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	pid, err := peer.IDFromPublicKey(pub)
	if err != nil {
		t.Fatalf("peer id: %v", err)
	}

	task := NewTaskSAE(DIDFromPubKey(pub), pid.String(), "task:oversized", "crowd.data_labeling", string(make([]byte, MaxEnvelopeSize)), "openagent://peer/task/task:oversized/1", []string{"openagent/v1/crowd"}, 900, OpUpsert, 1, time.Hour)
	if err := task.Sign(priv); err == nil {
		t.Fatalf("expected oversized SAE to fail signing")
	}
}

func mustEncode(t *testing.T, env SAEEnvelope) []byte {
	t.Helper()
	data, err := env.Encode()
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	return data
}
