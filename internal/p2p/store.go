package p2p

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"

	"openagent/internal/sae"
)

type TaskStatus string

const (
	TaskStatusActive    TaskStatus = "ACTIVE"
	TaskStatusComplete  TaskStatus = "COMPLETE"
	TaskStatusWithdrawn TaskStatus = "WITHDRAW"
	TaskStatusExpired   TaskStatus = "EXPIRED"
)

type TaskState struct {
	Key       string          `json:"key"`
	Origin    string          `json:"origin"`
	TaskID    string          `json:"task_id"`
	LastSeq   uint64          `json:"last_seq"`
	Status    TaskStatus      `json:"status"`
	Envelope  sae.SAEEnvelope `json:"envelope"`
	UpdatedAt time.Time       `json:"updated_at"`
	ExpiresAt time.Time       `json:"expires_at"`
}

type FactState struct {
	Key       string          `json:"key"`
	Origin    string          `json:"origin"`
	FactID    string          `json:"fact_id"`
	FactKind  string          `json:"fact_kind"`
	LastSeq   uint64          `json:"last_seq"`
	Envelope  sae.SAEEnvelope `json:"envelope"`
	UpdatedAt time.Time       `json:"updated_at"`
	ExpiresAt time.Time       `json:"expires_at"`
}

type KnownPeer struct {
	ID       string    `json:"id"`
	Addrs    []string  `json:"addrs"`
	LastSeen time.Time `json:"last_seen"`
}

type StoredPayload struct {
	Ref         string          `json:"ref"`
	OwnerPeerID string          `json:"owner_peer_id"`
	Kind        string          `json:"kind"`
	ObjectID    string          `json:"object_id"`
	Version     string          `json:"version"`
	ContentType string          `json:"content_type"`
	Data        json.RawMessage `json:"data"`
	SHA256      string          `json:"sha256"`
	UpdatedAt   time.Time       `json:"updated_at"`
	ExpiresAt   time.Time       `json:"expires_at"`
}

type NegotiationSession struct {
	SessionID string             `json:"session_id"`
	RefID     string             `json:"ref_id"`
	PeerID    string             `json:"peer_id"`
	State     string             `json:"state"`
	Frames    []NegotiationFrame `json:"frames"`
	UpdatedAt time.Time          `json:"updated_at"`
}

type EventLogEntry struct {
	Timestamp time.Time         `json:"timestamp"`
	Type      string            `json:"type"`
	Applied   bool              `json:"applied"`
	Reason    string            `json:"reason,omitempty"`
	PeerID    string            `json:"peer_id,omitempty"`
	Envelope  *sae.SAEEnvelope  `json:"envelope,omitempty"`
	Frame     *NegotiationFrame `json:"frame,omitempty"`
}

type peerRate struct {
	WindowStart time.Time
	Count       int
}

type Store struct {
	root         string
	eventLogPath string

	mu           sync.RWMutex
	tasks        map[string]TaskState
	facts        map[string]FactState
	payloads     map[string]StoredPayload
	peers        map[string]KnownPeer
	negotiations map[string]NegotiationSession
	rates        map[string]peerRate
}

func OpenStore(root string) (*Store, error) {
	if err := os.MkdirAll(root, 0o700); err != nil {
		return nil, fmt.Errorf("create store root: %w", err)
	}
	s := &Store{
		root:         root,
		eventLogPath: filepath.Join(root, "events.jsonl"),
		tasks:        map[string]TaskState{},
		facts:        map[string]FactState{},
		payloads:     map[string]StoredPayload{},
		peers:        map[string]KnownPeer{},
		negotiations: map[string]NegotiationSession{},
		rates:        map[string]peerRate{},
	}
	if err := s.load(); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *Store) load() error {
	if err := readJSON(filepath.Join(s.root, "tasks.json"), &s.tasks); err != nil {
		return err
	}
	if err := readJSON(filepath.Join(s.root, "facts.json"), &s.facts); err != nil {
		return err
	}
	if err := readJSON(filepath.Join(s.root, "payloads.json"), &s.payloads); err != nil {
		return err
	}
	if err := readJSON(filepath.Join(s.root, "peers.json"), &s.peers); err != nil {
		return err
	}
	if err := readJSON(filepath.Join(s.root, "negotiations.json"), &s.negotiations); err != nil {
		return err
	}
	return nil
}

func taskKey(origin, taskID string) string {
	return origin + "::" + taskID
}

func factKey(origin, factID string) string {
	return origin + "::" + factID
}

func (s *Store) NextTaskSeq(origin, taskID string) uint64 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if state, ok := s.tasks[taskKey(origin, taskID)]; ok {
		return state.LastSeq + 1
	}
	return 1
}

func (s *Store) NextFactSeq(origin, factID string) uint64 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if state, ok := s.facts[factKey(origin, factID)]; ok {
		return state.LastSeq + 1
	}
	return 1
}

func (s *Store) ApplyTask(env sae.SAEEnvelope) (TaskState, bool, string, error) {
	if env.Kind != sae.KindTask {
		return TaskState{}, false, "", errors.New("not a task SAE")
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().UTC()
	if now.After(env.ExpiresAt) {
		return TaskState{}, false, "expired", nil
	}

	key := taskKey(env.Origin, env.TaskID)
	current, exists := s.tasks[key]
	if exists {
		if current.Status == TaskStatusComplete || current.Status == TaskStatusWithdrawn {
			return current, false, "terminal", nil
		}
		if env.Seq <= current.LastSeq {
			return current, false, "replay_or_old", nil
		}
	}

	status := TaskStatusActive
	switch env.Op {
	case sae.OpComplete:
		status = TaskStatusComplete
	case sae.OpWithdraw:
		status = TaskStatusWithdrawn
	}

	next := TaskState{
		Key:       key,
		Origin:    env.Origin,
		TaskID:    env.TaskID,
		LastSeq:   env.Seq,
		Status:    status,
		Envelope:  env,
		UpdatedAt: now,
		ExpiresAt: env.ExpiresAt,
	}
	s.tasks[key] = next
	return next, true, "applied", s.persistLocked("tasks.json", s.tasks)
}

func (s *Store) ApplyFact(env sae.SAEEnvelope) (FactState, bool, string, error) {
	if env.Kind != sae.KindAgentFact {
		return FactState{}, false, "", errors.New("not an agent fact SAE")
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().UTC()
	if now.After(env.ExpiresAt) {
		return FactState{}, false, "expired", nil
	}

	key := factKey(env.Origin, env.FactID)
	current, exists := s.facts[key]
	if exists && env.Seq <= current.LastSeq {
		return current, false, "replay_or_old", nil
	}

	next := FactState{
		Key:       key,
		Origin:    env.Origin,
		FactID:    env.FactID,
		FactKind:  env.FactKind,
		LastSeq:   env.Seq,
		Envelope:  env,
		UpdatedAt: now,
		ExpiresAt: env.ExpiresAt,
	}
	s.facts[key] = next
	return next, true, "applied", s.persistLocked("facts.json", s.facts)
}

func (s *Store) UpsertPayload(payload StoredPayload) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.payloads[payload.Ref] = payload
	return s.persistLocked("payloads.json", s.payloads)
}

func (s *Store) Payload(ref string) (StoredPayload, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	payload, ok := s.payloads[ref]
	return payload, ok
}

func (s *Store) RecordPeer(info peer.AddrInfo) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	addrs := make([]string, 0, len(info.Addrs))
	for _, addr := range info.Addrs {
		addrs = append(addrs, addr.String())
	}
	s.peers[info.ID.String()] = KnownPeer{
		ID:       info.ID.String(),
		Addrs:    addrs,
		LastSeen: time.Now().UTC(),
	}
	return s.persistLocked("peers.json", s.peers)
}

func (s *Store) AllowPeer(peerID string, limit int) bool {
	if limit <= 0 || peerID == "" {
		return true
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now().UTC()
	rate := s.rates[peerID]
	if rate.WindowStart.IsZero() || now.Sub(rate.WindowStart) >= time.Minute {
		rate = peerRate{WindowStart: now, Count: 0}
	}
	rate.Count++
	s.rates[peerID] = rate
	return rate.Count <= limit
}

func (s *Store) RecordNegotiation(frame NegotiationFrame) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	session := s.negotiations[frame.SessionID]
	session.SessionID = frame.SessionID
	session.RefID = frame.RefID
	if frame.FromPeer != "" {
		session.PeerID = frame.FromPeer
	}
	if frame.ToPeer != "" && session.PeerID == "" {
		session.PeerID = frame.ToPeer
	}
	session.State = string(frame.Op)
	session.Frames = append(session.Frames, frame)
	session.UpdatedAt = time.Now().UTC()
	s.negotiations[frame.SessionID] = session
	return s.persistLocked("negotiations.json", s.negotiations)
}

func (s *Store) AppendEvent(entry EventLogEntry) error {
	if entry.Timestamp.IsZero() {
		entry.Timestamp = time.Now().UTC()
	}
	data, err := json.Marshal(entry)
	if err != nil {
		return err
	}
	f, err := os.OpenFile(s.eventLogPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	defer f.Close()
	if _, err := f.Write(append(data, '\n')); err != nil {
		return err
	}
	return nil
}

func (s *Store) GCExpired(now time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	changedTasks := false
	for key, task := range s.tasks {
		if now.After(task.ExpiresAt) {
			task.Status = TaskStatusExpired
			s.tasks[key] = task
			changedTasks = true
		}
	}
	if changedTasks {
		if err := s.persistLocked("tasks.json", s.tasks); err != nil {
			return err
		}
	}

	changedFacts := false
	for key, fact := range s.facts {
		if now.After(fact.ExpiresAt) {
			delete(s.facts, key)
			changedFacts = true
		}
	}
	if changedFacts {
		if err := s.persistLocked("facts.json", s.facts); err != nil {
			return err
		}
	}

	changedPayloads := false
	for key, payload := range s.payloads {
		if !payload.ExpiresAt.IsZero() && now.After(payload.ExpiresAt) {
			delete(s.payloads, key)
			changedPayloads = true
		}
	}
	if changedPayloads {
		if err := s.persistLocked("payloads.json", s.payloads); err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) TaskStates() []TaskState {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]TaskState, 0, len(s.tasks))
	for _, task := range s.tasks {
		out = append(out, task)
	}
	return out
}

func (s *Store) FactStates() []FactState {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]FactState, 0, len(s.facts))
	for _, fact := range s.facts {
		out = append(out, fact)
	}
	return out
}

func (s *Store) Peers() []KnownPeer {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]KnownPeer, 0, len(s.peers))
	for _, peer := range s.peers {
		out = append(out, peer)
	}
	return out
}

func (s *Store) Negotiations() []NegotiationSession {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]NegotiationSession, 0, len(s.negotiations))
	for _, session := range s.negotiations {
		out = append(out, session)
	}
	return out
}

func (s *Store) persistLocked(name string, v any) error {
	return writeJSONAtomic(filepath.Join(s.root, name), v)
}

func readJSON(path string, target any) error {
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	if len(data) == 0 {
		return nil
	}
	return json.Unmarshal(data, target)
}

func writeJSONAtomic(path string, v any) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func payloadDigest(raw json.RawMessage) string {
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}
