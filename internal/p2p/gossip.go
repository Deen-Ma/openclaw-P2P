package p2p

import (
	"context"
	"sync"

	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/host"

	"openagent/internal/sae"
)

type EnvelopeHandler func(context.Context, sae.SAEEnvelope) error

type Gossip struct {
	ctx     context.Context
	host    host.Host
	pubsub  *pubsub.PubSub
	handler EnvelopeHandler

	mu     sync.Mutex
	topics map[string]*pubsub.Topic
	subs   map[string]*pubsub.Subscription
}

func NewGossip(ctx context.Context, h host.Host, handler EnvelopeHandler) (*Gossip, error) {
	ps, err := pubsub.NewGossipSub(ctx, h)
	if err != nil {
		return nil, err
	}
	return &Gossip{
		ctx:     ctx,
		host:    h,
		pubsub:  ps,
		handler: handler,
		topics:  map[string]*pubsub.Topic{},
		subs:    map[string]*pubsub.Subscription{},
	}, nil
}

func (g *Gossip) SubscribeTopics(ctx context.Context, topics []string) error {
	for _, topic := range uniqueStrings(topics) {
		if _, err := g.ensureTopic(ctx, topic, true); err != nil {
			return err
		}
	}
	return nil
}

func (g *Gossip) PublishSAE(ctx context.Context, env sae.SAEEnvelope) error {
	payload, err := env.Encode()
	if err != nil {
		return err
	}
	for _, topicName := range uniqueStrings(env.Topics) {
		topic, err := g.ensureTopic(ctx, topicName, false)
		if err != nil {
			return err
		}
		if err := topic.Publish(ctx, payload); err != nil {
			return err
		}
	}
	return nil
}

func (g *Gossip) ensureTopic(ctx context.Context, topicName string, subscribe bool) (*pubsub.Topic, error) {
	g.mu.Lock()
	defer g.mu.Unlock()

	topic, ok := g.topics[topicName]
	if !ok {
		var err error
		topic, err = g.pubsub.Join(topicName)
		if err != nil {
			return nil, err
		}
		g.topics[topicName] = topic
	}
	if subscribe {
		if _, ok := g.subs[topicName]; !ok {
			sub, err := topic.Subscribe()
			if err != nil {
				return nil, err
			}
			g.subs[topicName] = sub
			go g.readLoop(ctx, sub)
		}
	}
	return topic, nil
}

func (g *Gossip) readLoop(ctx context.Context, sub *pubsub.Subscription) {
	for {
		msg, err := sub.Next(ctx)
		if err != nil {
			return
		}
		if msg.ReceivedFrom == g.host.ID() {
			continue
		}
		env, err := sae.DecodeEnvelope(msg.Data)
		if err != nil {
			continue
		}
		if g.handler != nil {
			_ = g.handler(ctx, env)
		}
	}
}

func (g *Gossip) Close() {
	g.mu.Lock()
	defer g.mu.Unlock()
	for _, sub := range g.subs {
		sub.Cancel()
	}
	for _, topic := range g.topics {
		_ = topic.Close()
	}
}

func uniqueStrings(input []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(input))
	for _, item := range input {
		if item == "" {
			continue
		}
		if _, ok := seen[item]; ok {
			continue
		}
		seen[item] = struct{}{}
		out = append(out, item)
	}
	return out
}
