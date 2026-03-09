package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"openagent/internal/p2p"
)

type stringListFlag []string

func (s *stringListFlag) String() string {
	return strings.Join(*s, ",")
}

func (s *stringListFlag) Set(v string) error {
	for _, item := range strings.Split(v, ",") {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		*s = append(*s, item)
	}
	return nil
}

func run(args []string) error {
	fs := flag.NewFlagSet("openagent", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)

	var (
		profile      = fs.String("profile", "", "OpenClaw profile name")
		dataDir      = fs.String("data-dir", "", "data directory for the node")
		profileDir   = fs.String("profile-dir", "", "optional OpenClaw profile directory")
		rendezvous   = fs.String("rendezvous", "openagent/v1/default", "rendezvous namespace")
		p2pPort      = fs.Int("port", 4001, "libp2p listen port")
		apiPort      = fs.Int("api-port", 7401, "local HTTP API port")
		confMin      = fs.Int("conf-min", 700, "minimum confidence for accepted events")
		topicLimit   = fs.Int("topic-limit", 8, "maximum topics per SAE")
		msgSizeLimit = fs.Int("msg-size-limit", 1024, "maximum SAE size in bytes")
		rateLimit    = fs.Int("rate-limit", 120, "maximum accepted events per peer per minute")
	)
	var bootstrapPeers stringListFlag
	var topics stringListFlag
	fs.Var(&bootstrapPeers, "bootstrap", "bootstrap multiaddr(s), comma-separated or repeated")
	fs.Var(&topics, "topic", "topic(s) to subscribe to, comma-separated or repeated")

	if err := fs.Parse(args); err != nil {
		return err
	}
	if *profile == "" {
		return errors.New("missing required --profile")
	}

	rootDir := *dataDir
	if rootDir == "" {
		rootDir = filepath.Join(".local", "profiles", *profile)
	}
	rootDir, err := filepath.Abs(rootDir)
	if err != nil {
		return fmt.Errorf("resolve data dir: %w", err)
	}

	profilePath := *profileDir
	if profilePath != "" {
		profilePath, err = filepath.Abs(profilePath)
		if err != nil {
			return fmt.Errorf("resolve profile dir: %w", err)
		}
	}

	if len(topics) == 0 {
		topics = append(topics, "openagent/v1/general")
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	cfg := p2p.Config{
		Profile:        *profile,
		ProfileDir:     profilePath,
		DataDir:        rootDir,
		P2PPort:        *p2pPort,
		APIPort:        *apiPort,
		Rendezvous:     *rendezvous,
		BootstrapPeers: bootstrapPeers,
		Topics:         topics,
		Policy: p2p.Policy{
			ConfMin:          *confMin,
			TopicLimit:       *topicLimit,
			MsgSizeLimit:     *msgSizeLimit,
			RequireSignature: true,
			RateLimitPerPeer: *rateLimit,
		},
	}

	node, err := p2p.NewNode(ctx, cfg)
	if err != nil {
		return err
	}
	defer node.Close()

	apiServer, err := startAgentAPI(ctx, node)
	if err != nil {
		return err
	}
	defer shutdownServer(apiServer)

	<-ctx.Done()
	return nil
}

func shutdownServer(server *http.Server) {
	if server == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = server.Shutdown(ctx)
}
