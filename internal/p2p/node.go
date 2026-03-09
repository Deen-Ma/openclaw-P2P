package p2p

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p/core/peer"
	routingdiscovery "github.com/libp2p/go-libp2p/p2p/discovery/routing"

	"openagent/internal/sae"
)

type Config struct {
	Profile        string
	ProfileDir     string
	DataDir        string
	P2PPort        int
	APIPort        int
	Rendezvous     string
	BootstrapPeers []string
	Topics         []string
	Policy         Policy
}

type PublishTaskInput struct {
	TaskID   string
	Op       sae.TaskOp
	Taxonomy string
	Topics   []string
	Summary  string
	Detail   any
	TTL      time.Duration
	Conf     int
}

type PublishFactInput struct {
	FactID   string
	FactKind string
	Taxonomy string
	Topics   []string
	Summary  string
	Detail   any
	TTL      time.Duration
	Conf     int
}

type Node struct {
	Config    Config
	Identity  Identity
	Store     *Store
	Gossip    *Gossip
	DHT       *dht.IpfsDHT
	Routing   *routingdiscovery.RoutingDiscovery
	Discovery *DiscoveryServices

	hostBundle *HostBundle
	cancel     context.CancelFunc
	closeOnce  sync.Once
}

func NewNode(ctx context.Context, cfg Config) (*Node, error) {
	cfg.Policy = cfg.Policy.Normalize()
	if cfg.Profile == "" {
		return nil, errors.New("missing profile")
	}
	if cfg.P2PPort == 0 {
		return nil, errors.New("missing p2p port")
	}
	if cfg.DataDir == "" {
		return nil, errors.New("missing data dir")
	}
	if cfg.Rendezvous == "" {
		cfg.Rendezvous = "openagent/v1/default"
	}
	if len(cfg.Topics) == 0 {
		cfg.Topics = []string{"openagent/v1/general"}
	}

	if err := os.MkdirAll(cfg.DataDir, 0o700); err != nil {
		return nil, fmt.Errorf("create data dir: %w", err)
	}

	store, err := OpenStore(filepath.Join(cfg.DataDir, "state"))
	if err != nil {
		return nil, err
	}
	nodeCtx, cancel := context.WithCancel(ctx)

	hostBundle, err := CreateHost(nodeCtx, cfg.P2PPort, filepath.Join(cfg.DataDir, "identity.key"))
	if err != nil {
		cancel()
		return nil, err
	}

	node := &Node{
		Config:     cfg,
		Identity:   hostBundle.Identity,
		Store:      store,
		hostBundle: hostBundle,
		cancel:     cancel,
	}

	RegisterFetchHandler(hostBundle.Host, node.handleFetch)
	RegisterNegotiateHandler(hostBundle.Host, node.handleNegotiate)

	mdnsService, err := SetupMDNS(hostBundle.Host, cfg.Rendezvous, node.onPeerDiscovered)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("setup mdns: %w", err)
	}

	dhtNode, routing, stopDiscovery, err := SetupRendezvous(nodeCtx, hostBundle.Host, cfg.Rendezvous, cfg.BootstrapPeers, node.onPeerDiscovered)
	if err != nil {
		_ = mdnsService.Close()
		cancel()
		return nil, err
	}

	gossip, err := NewGossip(nodeCtx, hostBundle.Host, node.handleEnvelope)
	if err != nil {
		stopDiscovery()
		cancel()
		return nil, err
	}
	if err := gossip.SubscribeTopics(nodeCtx, cfg.Topics); err != nil {
		_ = mdnsService.Close()
		stopDiscovery()
		cancel()
		return nil, err
	}

	node.Gossip = gossip
	node.DHT = dhtNode
	node.Routing = routing
	node.Discovery = &DiscoveryServices{
		DHT:             dhtNode,
		Routing:         routing,
		MDNSService:     mdnsService,
		cancelDiscovery: stopDiscovery,
	}

	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-nodeCtx.Done():
				return
			case <-ticker.C:
				_ = node.Store.GCExpired(time.Now().UTC())
			}
		}
	}()

	if err := node.Store.AppendEvent(EventLogEntry{
		Type:    "node_started",
		Applied: true,
		Reason:  cfg.Profile,
		PeerID:  node.Identity.PeerID.String(),
	}); err != nil {
		return nil, err
	}
	return node, nil
}

func (n *Node) Close() error {
	n.closeOnce.Do(func() {
		if n.Gossip != nil {
			n.Gossip.Close()
		}
		if n.Discovery != nil {
			_ = n.Discovery.Close()
		}
		if n.cancel != nil {
			n.cancel()
		}
		if n.hostBundle != nil && n.hostBundle.Host != nil {
			_ = n.hostBundle.Host.Close()
		}
	})
	return nil
}

func (n *Node) PublishTask(ctx context.Context, input PublishTaskInput) (sae.TaskSAE, error) {
	if input.Taxonomy == "" {
		return sae.TaskSAE{}, errors.New("missing taxonomy")
	}
	if input.Summary == "" {
		return sae.TaskSAE{}, errors.New("missing summary")
	}
	if input.Op == "" {
		input.Op = sae.OpUpsert
	}
	topics := normalizeTopics(input.Topics, input.Taxonomy, n.Config.Policy.TopicLimit)
	taskID := input.TaskID
	if taskID == "" {
		taskID = "task:" + sae.HashString(n.Identity.DID, input.Taxonomy, input.Summary, fmt.Sprintf("%d", time.Now().UTC().UnixNano()))[:32]
	}
	seq := n.Store.NextTaskSeq(n.Identity.DID, taskID)

	payload, err := marshalDetail(input.Detail)
	if err != nil {
		return sae.TaskSAE{}, err
	}
	detailRef := BuildDetailRef(n.Identity.PeerID, "task", taskID, seq)
	task := sae.NewTaskSAE(n.Identity.DID, n.Identity.PeerID.String(), taskID, input.Taxonomy, input.Summary, detailRef, topics, input.Conf, input.Op, seq, input.TTL)
	if err := task.Sign(n.Identity.PrivKey); err != nil {
		return sae.TaskSAE{}, err
	}
	if err := n.enforceEnvelopePolicy(task.SAEEnvelope, false); err != nil {
		return sae.TaskSAE{}, err
	}

	if err := n.Store.UpsertPayload(StoredPayload{
		Ref:         detailRef,
		OwnerPeerID: n.Identity.PeerID.String(),
		Kind:        "task",
		ObjectID:    taskID,
		Version:     fmt.Sprintf("%d", seq),
		ContentType: "application/json",
		Data:        payload,
		SHA256:      payloadDigest(payload),
		UpdatedAt:   time.Now().UTC(),
		ExpiresAt:   task.ExpiresAt,
	}); err != nil {
		return sae.TaskSAE{}, err
	}
	if _, applied, reason, err := n.Store.ApplyTask(task.SAEEnvelope); err != nil {
		return sae.TaskSAE{}, err
	} else {
		_ = n.Store.AppendEvent(EventLogEntry{Type: "task_publish_local", Applied: applied, Reason: reason, PeerID: n.Identity.PeerID.String(), Envelope: &task.SAEEnvelope})
	}
	if err := n.Gossip.SubscribeTopics(ctx, topics); err != nil {
		return sae.TaskSAE{}, err
	}
	if err := n.Gossip.PublishSAE(ctx, task.SAEEnvelope); err != nil {
		return sae.TaskSAE{}, err
	}
	return task, nil
}

func (n *Node) PublishFact(ctx context.Context, input PublishFactInput) (sae.AgentFactSAE, error) {
	if input.Taxonomy == "" {
		return sae.AgentFactSAE{}, errors.New("missing taxonomy")
	}
	if input.Summary == "" {
		return sae.AgentFactSAE{}, errors.New("missing summary")
	}
	if input.FactKind == "" {
		return sae.AgentFactSAE{}, errors.New("missing fact_kind")
	}
	topics := normalizeTopics(input.Topics, input.Taxonomy, n.Config.Policy.TopicLimit)
	factID := input.FactID
	if factID == "" {
		factID = "fact:" + sae.HashString(n.Identity.DID, input.FactKind, input.Taxonomy, input.Summary, fmt.Sprintf("%d", time.Now().UTC().UnixNano()))[:32]
	}
	seq := n.Store.NextFactSeq(n.Identity.DID, factID)

	payload, err := marshalDetail(input.Detail)
	if err != nil {
		return sae.AgentFactSAE{}, err
	}
	detailRef := BuildDetailRef(n.Identity.PeerID, "fact", factID, seq)
	fact := sae.NewAgentFactSAE(n.Identity.DID, n.Identity.PeerID.String(), factID, input.FactKind, input.Taxonomy, input.Summary, detailRef, topics, input.Conf, seq, input.TTL)
	if err := fact.Sign(n.Identity.PrivKey); err != nil {
		return sae.AgentFactSAE{}, err
	}
	if err := n.enforceEnvelopePolicy(fact.SAEEnvelope, false); err != nil {
		return sae.AgentFactSAE{}, err
	}

	if err := n.Store.UpsertPayload(StoredPayload{
		Ref:         detailRef,
		OwnerPeerID: n.Identity.PeerID.String(),
		Kind:        "fact",
		ObjectID:    factID,
		Version:     fmt.Sprintf("%d", seq),
		ContentType: "application/json",
		Data:        payload,
		SHA256:      payloadDigest(payload),
		UpdatedAt:   time.Now().UTC(),
		ExpiresAt:   fact.ExpiresAt,
	}); err != nil {
		return sae.AgentFactSAE{}, err
	}
	if _, applied, reason, err := n.Store.ApplyFact(fact.SAEEnvelope); err != nil {
		return sae.AgentFactSAE{}, err
	} else {
		_ = n.Store.AppendEvent(EventLogEntry{Type: "fact_publish_local", Applied: applied, Reason: reason, PeerID: n.Identity.PeerID.String(), Envelope: &fact.SAEEnvelope})
	}
	if err := n.Gossip.SubscribeTopics(ctx, topics); err != nil {
		return sae.AgentFactSAE{}, err
	}
	if err := n.Gossip.PublishSAE(ctx, fact.SAEEnvelope); err != nil {
		return sae.AgentFactSAE{}, err
	}
	return fact, nil
}

func (n *Node) FetchRemoteDetail(ctx context.Context, detailRef string) (FetchResponse, error) {
	ref, err := ParseDetailRef(detailRef)
	if err != nil {
		return FetchResponse{}, err
	}
	if ref.PeerID == n.Identity.PeerID {
		payload, ok := n.Store.Payload(detailRef)
		if !ok {
			return FetchResponse{Found: false, Error: "not found"}, errors.New("not found")
		}
		return FetchResponse{
			Found:       true,
			ContentType: payload.ContentType,
			Payload:     payload.Data,
			SHA256:      payload.SHA256,
		}, nil
	}
	return FetchDetail(ctx, n.hostBundle.Host, ref.PeerID, detailRef)
}

func (n *Node) SendNegotiation(ctx context.Context, peerID string, frame NegotiationFrame) (NegotiationFrame, error) {
	target, err := peer.Decode(peerID)
	if err != nil {
		return NegotiationFrame{}, err
	}
	frame.FromPeer = n.Identity.PeerID.String()
	frame.ToPeer = peerID
	if frame.Timestamp.IsZero() {
		frame.Timestamp = time.Now().UTC()
	}
	if err := n.Store.RecordNegotiation(frame); err != nil {
		return NegotiationFrame{}, err
	}
	resp, err := SendNegotiationFrame(ctx, n.hostBundle.Host, target, frame)
	if err != nil {
		return NegotiationFrame{}, err
	}
	_ = n.Store.RecordNegotiation(resp)
	return resp, nil
}

func (n *Node) TaskStates() []TaskState {
	return n.Store.TaskStates()
}

func (n *Node) FactStates() []FactState {
	return n.Store.FactStates()
}

func (n *Node) Peers() []KnownPeer {
	return n.Store.Peers()
}

func (n *Node) NegotiationSessions() []NegotiationSession {
	return n.Store.Negotiations()
}

func (n *Node) onPeerDiscovered(info peer.AddrInfo) {
	if info.ID == "" || info.ID == n.Identity.PeerID {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = n.hostBundle.Host.Connect(ctx, info)
	_ = n.Store.RecordPeer(info)
}

func (n *Node) handleEnvelope(ctx context.Context, env sae.SAEEnvelope) error {
	if err := n.enforceEnvelopePolicy(env, true); err != nil {
		_ = n.Store.AppendEvent(EventLogEntry{Type: "sae_rejected", Applied: false, Reason: err.Error(), PeerID: env.Publisher, Envelope: &env})
		return err
	}

	switch env.Kind {
	case sae.KindTask:
		_, applied, reason, err := n.Store.ApplyTask(env)
		_ = n.Store.AppendEvent(EventLogEntry{Type: "task_received", Applied: applied, Reason: reason, PeerID: env.Publisher, Envelope: &env})
		return err
	case sae.KindAgentFact:
		_, applied, reason, err := n.Store.ApplyFact(env)
		_ = n.Store.AppendEvent(EventLogEntry{Type: "fact_received", Applied: applied, Reason: reason, PeerID: env.Publisher, Envelope: &env})
		return err
	default:
		return fmt.Errorf("unsupported SAE kind %q", env.Kind)
	}
}

func (n *Node) handleFetch(_ context.Context, req FetchRequest) (FetchResponse, error) {
	payload, ok := n.Store.Payload(req.DetailRef)
	if !ok {
		return FetchResponse{Found: false, Error: "not found"}, errors.New("not found")
	}
	return FetchResponse{
		Found:       true,
		ContentType: payload.ContentType,
		Payload:     payload.Data,
		SHA256:      payload.SHA256,
	}, nil
}

func (n *Node) handleNegotiate(_ context.Context, frame NegotiationFrame) (NegotiationFrame, error) {
	if frame.Timestamp.IsZero() {
		frame.Timestamp = time.Now().UTC()
	}
	if err := validateNegotiation(frame); err != nil {
		return NegotiationFrame{}, err
	}
	if err := n.Store.RecordNegotiation(frame); err != nil {
		return NegotiationFrame{}, err
	}
	status := "received"
	switch frame.Op {
	case NegotiationAccept:
		status = "accepted"
	case NegotiationReject:
		status = "rejected"
	case NegotiationCancel:
		status = "cancelled"
	case NegotiationComplete:
		status = "completed"
	}
	resp := NegotiationFrame{
		SessionID: frame.SessionID,
		RefID:     frame.RefID,
		Op:        frame.Op,
		FromPeer:  n.Identity.PeerID.String(),
		ToPeer:    frame.FromPeer,
		Status:    status,
		Timestamp: time.Now().UTC(),
	}
	_ = n.Store.RecordNegotiation(resp)
	return resp, nil
}

func (n *Node) enforceEnvelopePolicy(env sae.SAEEnvelope, inbound bool) error {
	if len(env.Topics) > n.Config.Policy.TopicLimit {
		return fmt.Errorf("topic limit exceeded: %d > %d", len(env.Topics), n.Config.Policy.TopicLimit)
	}
	if env.Conf < n.Config.Policy.ConfMin {
		return fmt.Errorf("confidence below threshold: %d < %d", env.Conf, n.Config.Policy.ConfMin)
	}
	encoded, err := env.Encode()
	if err != nil {
		return err
	}
	if len(encoded) > n.Config.Policy.MsgSizeLimit {
		return fmt.Errorf("message size exceeded: %d > %d", len(encoded), n.Config.Policy.MsgSizeLimit)
	}
	if n.Config.Policy.RequireSignature {
		if err := env.VerifyPublisher(); err != nil {
			return err
		}
	}
	if inbound && env.Publisher != n.Identity.PeerID.String() && !n.Store.AllowPeer(env.Publisher, n.Config.Policy.RateLimitPerPeer) {
		return fmt.Errorf("rate limit exceeded for %s", env.Publisher)
	}
	return nil
}

func normalizeTopics(topics []string, taxonomy string, limit int) []string {
	if len(topics) == 0 {
		topics = TopicsFromTaxonomy(taxonomy)
	}
	out := make([]string, 0, len(topics))
	seen := map[string]struct{}{}
	for _, topic := range topics {
		topic = strings.TrimSpace(topic)
		if topic == "" {
			continue
		}
		if _, ok := seen[topic]; ok {
			continue
		}
		seen[topic] = struct{}{}
		out = append(out, topic)
		if limit > 0 && len(out) >= limit {
			break
		}
	}
	if len(out) == 0 {
		return []string{"openagent/v1/general"}
	}
	return out
}

func TopicsFromTaxonomy(taxonomy string) []string {
	taxonomy = strings.TrimSpace(strings.Trim(taxonomy, "."))
	if taxonomy == "" {
		return []string{"openagent/v1/general"}
	}
	parts := strings.Split(taxonomy, ".")
	topics := make([]string, 0, len(parts))
	prefix := "openagent/v1"
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		prefix += "/" + part
		topics = append(topics, prefix)
	}
	if len(topics) == 0 {
		return []string{"openagent/v1/general"}
	}
	return topics
}

func marshalDetail(detail any) (json.RawMessage, error) {
	if detail == nil {
		return json.RawMessage(`{}`), nil
	}
	payload, err := json.Marshal(detail)
	if err != nil {
		return nil, err
	}
	return payload, nil
}
