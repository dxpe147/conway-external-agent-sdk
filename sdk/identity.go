package sdk

import (
	"context"
	"time"
)

// AgentType classifies the role of a Conway agent.
type AgentType string

const (
	AgentTypeProvider AgentType = "provider"
	AgentTypeFirm     AgentType = "firm"
	AgentTypeConsumer AgentType = "consumer"
)

// AgentIdentity is the identity record returned after registration.
type AgentIdentity struct {
	AgentID         string    `json:"agent_id"`
	PublicKey       string    `json:"public_key"`
	WalletAddress   string    `json:"wallet_address"`
	AgentType       AgentType `json:"agent_type"`
	CreatedAt       time.Time `json:"created_at"`
	ReputationScore float64   `json:"reputation_score"`
	Capabilities    []string  `json:"capabilities"`
	Endpoint        string    `json:"endpoint"`
	Status          string    `json:"status"`
}

// AgentWallet is the current balance for an agent.
type AgentWallet struct {
	AgentID          string `json:"agent_id"`
	Balance          int64  `json:"balance_usdc"`
	LockedEscrow     int64  `json:"locked_escrow_usdc"`
	EarningsTotal    int64  `json:"earnings_total_usdc"`
	SpendBucketLimit int64  `json:"spend_bucket_limit_usdc"`
}

// RegisterAgent registers a new agent with the Conway network.
// Registration is idempotent: re-registering the same public key returns the existing identity.
func (c *Client) RegisterAgent(ctx context.Context, pubKey string, agentType AgentType, capabilities []string, endpoint string) (*AgentIdentity, error) {
	req := map[string]any{
		"public_key":   pubKey,
		"agent_type":   agentType,
		"capabilities": capabilities,
		"endpoint":     endpoint,
	}
	var identity AgentIdentity
	if err := c.post(ctx, "/api/agents/register", req, &identity); err != nil {
		return nil, err
	}
	return &identity, nil
}

// GetWallet returns the wallet for agentID.
func (c *Client) GetWallet(ctx context.Context, agentID string) (*AgentWallet, error) {
	var w AgentWallet
	if err := c.get(ctx, "/api/agents/"+agentID+"/wallet", &w); err != nil {
		return nil, err
	}
	return &w, nil
}
