package sae

import (
	"fmt"
	"time"
)

type TaskSAE struct {
	SAEEnvelope
}

func NewTaskSAE(origin, publisher, taskID, taxonomy, summary, detailRef string, topics []string, conf int, op TaskOp, seq uint64, ttl time.Duration) TaskSAE {
	if ttl <= 0 {
		ttl = DefaultTTL
	}
	if taskID == "" {
		taskID = "task:" + HashString(origin, taxonomy, summary, fmt.Sprintf("%d", time.Now().UTC().UnixNano()))[:32]
	}
	return TaskSAE{
		SAEEnvelope: SAEEnvelope{
			Kind:      KindTask,
			Origin:    origin,
			Publisher: publisher,
			Seq:       seq,
			Topics:    topics,
			Summary:   summary,
			DetailRef: detailRef,
			ExpiresAt: time.Now().UTC().Add(ttl),
			TaskID:    taskID,
			Op:        op,
			Taxonomy:  taxonomy,
			Conf:      conf,
		},
	}
}
