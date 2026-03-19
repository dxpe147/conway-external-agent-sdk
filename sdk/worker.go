package sdk

import (
	"context"
	"fmt"
	"log"
	"time"
)

// WorkFn is the function a provider implements to execute contract work.
// It receives the contract payload and must return a result or an error.
type WorkFn func(ctx context.Context, contract *Contract) (ExecutionResult, error)

// WorkerConfig configures a Conway worker loop.
type WorkerConfig struct {
	// WorkerID uniquely identifies this worker instance.
	WorkerID string

	// Capabilities is the list of contract capabilities this worker handles.
	Capabilities []string

	// PollInterval is how long to wait between claim attempts when idle (default: 2s).
	PollInterval time.Duration

	// MaxConcurrent is the maximum number of concurrent contract executions (default: 4).
	MaxConcurrent int

	// Logger is an optional logger (defaults to log.Default output).
	Logger *log.Logger
}

// Worker is a Conway provider worker that continuously polls for and executes contracts.
type Worker struct {
	client *Client
	cfg    WorkerConfig
	workFn WorkFn
	sem    chan struct{}
	log    *log.Logger
}

// NewWorker creates a Worker from the given client, config, and work function.
func NewWorker(client *Client, cfg WorkerConfig, fn WorkFn) *Worker {
	if cfg.PollInterval <= 0 {
		cfg.PollInterval = 2 * time.Second
	}
	if cfg.MaxConcurrent <= 0 {
		cfg.MaxConcurrent = 4
	}
	logger := cfg.Logger
	if logger == nil {
		logger = log.Default()
	}
	return &Worker{
		client: client,
		cfg:    cfg,
		workFn: fn,
		sem:    make(chan struct{}, cfg.MaxConcurrent),
		log:    logger,
	}
}

// Run starts the worker poll loop. It blocks until ctx is cancelled.
//
// Poll cycle:
//  1. POST /api/contracts/claim with WorkerID + Capabilities
//  2. If a contract is returned, execute WorkFn in a goroutine
//  3. POST /api/contracts/submit with the signed ExecutionProof
//  4. Sleep PollInterval, repeat
func (w *Worker) Run(ctx context.Context) {
	w.log.Printf("[conway-worker] %s starting — capabilities: %v", w.cfg.WorkerID, w.cfg.Capabilities)
	for {
		select {
		case <-ctx.Done():
			w.log.Printf("[conway-worker] %s shutting down", w.cfg.WorkerID)
			return
		default:
		}

		contract, err := w.client.ClaimContract(ctx, w.cfg.WorkerID, w.cfg.Capabilities)
		if err != nil {
			w.log.Printf("[conway-worker] claim error: %v", err)
			time.Sleep(w.cfg.PollInterval)
			continue
		}
		if contract == nil {
			// Nothing available — back off.
			time.Sleep(w.cfg.PollInterval)
			continue
		}

		// Acquire concurrency slot.
		w.sem <- struct{}{}
		go func(c *Contract) {
			defer func() { <-w.sem }()
			w.execute(ctx, c)
		}(contract)
	}
}

func (w *Worker) execute(ctx context.Context, contract *Contract) {
	w.log.Printf("[conway-worker] executing %s (%s)", contract.ContractID, contract.Capability)
	result, err := w.workFn(ctx, contract)
	if err != nil {
		w.log.Printf("[conway-worker] work error for %s: %v", contract.ContractID, err)
		result = ExecutionResult{
			Status: "error",
			Output: map[string]any{"error": err.Error()},
		}
	}
	meta := map[string]any{
		"worker_id":        w.cfg.WorkerID,
		"executed_at":      time.Now().UTC(),
		"execution_status": result.Status,
	}
	if submitErr := w.client.SubmitResult(ctx, contract.ContractID, w.cfg.WorkerID, result, meta); submitErr != nil {
		w.log.Printf("[conway-worker] submit error for %s: %v", contract.ContractID, submitErr)
		return
	}
	w.log.Printf("[conway-worker] completed %s → %s", contract.ContractID, result.Status)
}

// RunSingle executes exactly one contract claim+execute cycle. Useful for testing.
func (w *Worker) RunSingle(ctx context.Context) error {
	contract, err := w.client.ClaimContract(ctx, w.cfg.WorkerID, w.cfg.Capabilities)
	if err != nil {
		return fmt.Errorf("claim: %w", err)
	}
	if contract == nil {
		return fmt.Errorf("no contracts available")
	}
	w.execute(ctx, contract)
	return nil
}
