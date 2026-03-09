package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"openagent/internal/p2p"
	"openagent/internal/sae"
)

type publishTaskRequest struct {
	TaskID    string     `json:"task_id"`
	Op        sae.TaskOp `json:"op"`
	Taxonomy  string     `json:"taxonomy"`
	Topics    []string   `json:"topics"`
	Summary   string     `json:"summary"`
	Detail    any        `json:"detail"`
	TTLSecond int        `json:"ttl_sec"`
	Conf      int        `json:"conf"`
}

type publishFactRequest struct {
	FactID    string   `json:"fact_id"`
	FactKind  string   `json:"fact_kind"`
	Taxonomy  string   `json:"taxonomy"`
	Topics    []string `json:"topics"`
	Summary   string   `json:"summary"`
	Detail    any      `json:"detail"`
	TTLSecond int      `json:"ttl_sec"`
	Conf      int      `json:"conf"`
}

type fetchRequest struct {
	DetailRef string `json:"detail_ref"`
}

type negotiateRequest struct {
	PeerID     string            `json:"peer_id"`
	SessionID  string            `json:"session_id"`
	RefID      string            `json:"ref_id"`
	Op         p2p.NegotiationOp `json:"op"`
	Body       string            `json:"body"`
	PayloadRef string            `json:"payload_ref"`
	Status     string            `json:"status"`
}

func startAgentAPI(ctx context.Context, node *p2p.Node) (*http.Server, error) {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]any{
			"profile": node.Config.Profile,
			"peer_id": node.Identity.PeerID.String(),
			"did":     node.Identity.DID,
			"topics":  node.Config.Topics,
		})
	})
	mux.HandleFunc("/v1/tasks", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, node.TaskStates())
	})
	mux.HandleFunc("/v1/facts", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, node.FactStates())
	})
	mux.HandleFunc("/v1/peers", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, node.Peers())
	})
	mux.HandleFunc("/v1/sessions", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, node.NegotiationSessions())
	})
	mux.HandleFunc("/v1/tasks/publish", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		var req publishTaskRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		task, err := node.PublishTask(r.Context(), p2p.PublishTaskInput{
			TaskID:   req.TaskID,
			Op:       req.Op,
			Taxonomy: req.Taxonomy,
			Topics:   req.Topics,
			Summary:  req.Summary,
			Detail:   req.Detail,
			TTL:      ttlDuration(req.TTLSecond),
			Conf:     req.Conf,
		})
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, task)
	})
	mux.HandleFunc("/v1/facts/publish", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		var req publishFactRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		fact, err := node.PublishFact(r.Context(), p2p.PublishFactInput{
			FactID:   req.FactID,
			FactKind: req.FactKind,
			Taxonomy: req.Taxonomy,
			Topics:   req.Topics,
			Summary:  req.Summary,
			Detail:   req.Detail,
			TTL:      ttlDuration(req.TTLSecond),
			Conf:     req.Conf,
		})
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, fact)
	})
	mux.HandleFunc("/v1/fetch", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		var req fetchRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		resp, err := node.FetchRemoteDetail(r.Context(), req.DetailRef)
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, resp)
	})
	mux.HandleFunc("/v1/negotiate", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		var req negotiateRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		frame := p2p.NegotiationFrame{
			SessionID:  req.SessionID,
			RefID:      req.RefID,
			Op:         req.Op,
			Body:       req.Body,
			PayloadRef: req.PayloadRef,
			Status:     req.Status,
			Timestamp:  time.Now().UTC(),
			FromPeer:   node.Identity.PeerID.String(),
		}
		if frame.SessionID == "" {
			frame.SessionID = deriveSessionID(frame.RefID, frame.Body)
		}
		resp, err := node.SendNegotiation(r.Context(), req.PeerID, frame)
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, resp)
	})

	server := &http.Server{
		Addr:              fmt.Sprintf("127.0.0.1:%d", node.Config.APIPort),
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}
	go func() {
		<-ctx.Done()
		shutdownServer(server)
	}()
	go func() {
		_ = server.ListenAndServe()
	}()
	return server, nil
}

func ttlDuration(seconds int) time.Duration {
	if seconds <= 0 {
		return 24 * time.Hour
	}
	return time.Duration(seconds) * time.Second
}

func deriveSessionID(refID, body string) string {
	sum := sha256.Sum256([]byte(refID + ":" + body + ":" + time.Now().UTC().Format(time.RFC3339Nano)))
	return hex.EncodeToString(sum[:16])
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func writeError(w http.ResponseWriter, status int, message string) {
	if message == "" {
		message = errors.New("request failed").Error()
	}
	writeJSON(w, status, map[string]any{"error": message})
}
