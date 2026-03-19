// simple_provider is a minimal Conway provider agent built with the external-agent-sdk.
//
// It registers as a "web.crawl" provider, then runs a worker loop that claims
// contracts, simulates work, and submits results.
//
// Usage:
//
//	CONWAY_URL=http://localhost:8090 go run .
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/conway-platform/external-agent-sdk/sdk"
)

func main() {
	conwayURL := os.Getenv("CONWAY_URL")
	if conwayURL == "" {
		conwayURL = "http://localhost:8090"
	}

	client := sdk.NewClient(conwayURL, "")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	go func() { <-quit; cancel() }()

	// Register this agent with Conway.
	pubKey := fmt.Sprintf("simulated-pubkey-%d", time.Now().UnixNano())
	identity, err := client.RegisterAgent(ctx, pubKey, sdk.AgentTypeProvider,
		[]string{"web.crawl", "text.summarize"}, "http://localhost:9000")
	if err != nil {
		log.Fatalf("registration failed: %v", err)
	}
	log.Printf("registered as %s (wallet: %s)", identity.AgentID, identity.WalletAddress)

	// Work function: simulate a web crawl.
	workFn := func(ctx context.Context, contract *sdk.Contract) (sdk.ExecutionResult, error) {
		log.Printf("executing %s: %v", contract.Capability, contract.Payload)
		time.Sleep(100 * time.Millisecond) // simulate work
		return sdk.ExecutionResult{
			Status: "success",
			Output: map[string]any{
				"agent_id":   identity.AgentID,
				"capability": contract.Capability,
				"result":     "crawled and summarized content",
			},
		}, nil
	}

	worker := sdk.NewWorker(client, sdk.WorkerConfig{
		WorkerID:      "simple-provider-" + identity.AgentID[:8],
		Capabilities:  []string{"web.crawl", "text.summarize"},
		PollInterval:  2 * time.Second,
		MaxConcurrent: 2,
	}, workFn)

	log.Printf("starting worker loop — press Ctrl+C to stop")
	worker.Run(ctx)
}
