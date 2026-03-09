package p2p

import (
	"testing"
	"time"

	"openagent/internal/sae"
)

func TestApplyTaskTerminalAbsorbing(t *testing.T) {
	store := mustStore(t)
	env1 := sae.SAEEnvelope{
		Kind:      sae.KindTask,
		Origin:    "did:key:test",
		Publisher: "12D3KooWTestPublisher",
		Seq:       1,
		Topics:    []string{"openagent/v1/crowd"},
		Summary:   "task",
		DetailRef: "openagent://12D3KooWTestPublisher/task/task-1/1",
		ExpiresAt: time.Now().UTC().Add(time.Hour),
		TaskID:    "task-1",
		Op:        sae.OpComplete,
		Taxonomy:  "crowd.data_labeling",
		Conf:      900,
		Sig:       "stub",
	}
	if _, applied, _, err := store.ApplyTask(env1); err != nil || !applied {
		t.Fatalf("apply complete: applied=%v err=%v", applied, err)
	}

	env2 := env1
	env2.Seq = 2
	env2.Op = sae.OpUpsert
	state, applied, reason, err := store.ApplyTask(env2)
	if err != nil {
		t.Fatalf("apply after terminal: %v", err)
	}
	if applied {
		t.Fatalf("expected terminal absorption")
	}
	if reason != "terminal" {
		t.Fatalf("unexpected reason %q", reason)
	}
	if state.Status != TaskStatusComplete {
		t.Fatalf("unexpected status %s", state.Status)
	}
}

func TestApplyFactSoftOverwrite(t *testing.T) {
	store := mustStore(t)
	env1 := sae.SAEEnvelope{
		Kind:      sae.KindAgentFact,
		Origin:    "did:key:test",
		Publisher: "12D3KooWTestPublisher",
		Seq:       1,
		Topics:    []string{"openagent/v1/crowd"},
		Summary:   "fact-1",
		DetailRef: "openagent://12D3KooWTestPublisher/fact/fact-1/1",
		ExpiresAt: time.Now().UTC().Add(time.Hour),
		FactID:    "fact-1",
		FactKind:  "capability",
		Taxonomy:  "crowd.data_labeling",
		Conf:      900,
		Sig:       "stub",
	}
	if _, applied, _, err := store.ApplyFact(env1); err != nil || !applied {
		t.Fatalf("apply fact: applied=%v err=%v", applied, err)
	}

	env2 := env1
	env2.Seq = 2
	env2.Summary = "fact-2"
	state, applied, reason, err := store.ApplyFact(env2)
	if err != nil {
		t.Fatalf("apply fact update: %v", err)
	}
	if !applied || reason != "applied" {
		t.Fatalf("expected fact overwrite, applied=%v reason=%q", applied, reason)
	}
	if state.Envelope.Summary != "fact-2" {
		t.Fatalf("fact summary did not update")
	}
}

func mustStore(t *testing.T) *Store {
	t.Helper()
	dir := t.TempDir()
	store, err := OpenStore(dir)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	return store
}
