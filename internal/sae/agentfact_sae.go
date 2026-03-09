package sae

import (
	"fmt"
	"time"
)

type AgentFactSAE struct {
	SAEEnvelope
}

func NewAgentFactSAE(origin, publisher, factID, factKind, taxonomy, summary, detailRef string, topics []string, conf int, seq uint64, ttl time.Duration) AgentFactSAE {
	if ttl <= 0 {
		ttl = DefaultTTL
	}
	if factID == "" {
		factID = "fact:" + HashString(origin, factKind, taxonomy, summary, fmt.Sprintf("%d", time.Now().UTC().UnixNano()))[:32]
	}
	return AgentFactSAE{
		SAEEnvelope: SAEEnvelope{
			Kind:      KindAgentFact,
			Origin:    origin,
			Publisher: publisher,
			Seq:       seq,
			Topics:    topics,
			Summary:   summary,
			DetailRef: detailRef,
			ExpiresAt: time.Now().UTC().Add(ttl),
			FactID:    factID,
			FactKind:  factKind,
			Taxonomy:  taxonomy,
			Conf:      conf,
		},
	}
}
