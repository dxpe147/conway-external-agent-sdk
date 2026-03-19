package sdk

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"time"
)

// Contract is a queued contract available for workers to claim.
type Contract struct {
	ContractID  string         `json:"contract_id"`
	Capability  string         `json:"capability"`
	Payload     map[string]any `json:"payload"`
	MaxPayment  int64          `json:"max_payment_usdc"`
	Deadline    time.Time      `json:"deadline"`
	LeaseExpiry time.Time      `json:"lease_expiry"`
}

// ExecutionResult holds the worker's output after completing a contract.
type ExecutionResult struct {
	Status string         `json:"status"`
	Output map[string]any `json:"output"`
}

// ClaimContract asks Conway for the next available contract matching capabilities.
// Returns nil, nil when no contracts are available.
func (c *Client) ClaimContract(ctx context.Context, workerID string, capabilities []string) (*Contract, error) {
	req := map[string]any{
		"worker_id":    workerID,
		"capabilities": capabilities,
	}
	var contract Contract
	if err := c.post(ctx, "/api/contracts/claim", req, &contract); err != nil {
		return nil, nil // 204 No Content → nothing available
	}
	if contract.ContractID == "" {
		return nil, nil
	}
	return &contract, nil
}

// SubmitResult submits the execution proof for a completed contract.
func (c *Client) SubmitResult(ctx context.Context, contractID, workerID string, result ExecutionResult, metadata map[string]any) error {
	sig := computeSignature(contractID, workerID)
	proof := map[string]any{
		"contract_id":        contractID,
		"worker_id":          workerID,
		"result":             result,
		"execution_metadata": metadata,
		"worker_signature":   sig,
	}
	return c.post(ctx, "/api/contracts/submit", proof, nil)
}

// PostContract posts a new contract to the queue.
func (c *Client) PostContract(ctx context.Context, capability string, payload map[string]any, maxPayment int64, requesterID string) (*Contract, error) {
	req := map[string]any{
		"capability":         capability,
		"payload":            payload,
		"max_payment_usdc":   maxPayment,
		"requester_agent_id": requesterID,
	}
	var contract Contract
	if err := c.post(ctx, "/api/contracts/post", req, &contract); err != nil {
		return nil, err
	}
	return &contract, nil
}

// computeSignature returns the SHA-256 hex digest of {"contract_id":…,"worker_id":…}.
func computeSignature(contractID, workerID string) string {
	body, _ := json.Marshal(map[string]string{
		"contract_id": contractID,
		"worker_id":   workerID,
	})
	h := sha256.Sum256(body)
	return hex.EncodeToString(h[:])
}
