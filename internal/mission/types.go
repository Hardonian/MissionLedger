package mission

import (
	"time"

	"github.com/Hardonian/missionledger/internal/degraded"
)

type MissionState string

const (
	StateCreated         MissionState = "created"
	StatePlanning        MissionState = "planning"
	StateWaitingApproval MissionState = "waiting_approval"
	StateApproved        MissionState = "approved"
	StateDenied          MissionState = "denied"
	StateRunning         MissionState = "running"
	StatePaused          MissionState = "paused"
	StateDegraded        MissionState = "degraded"
	StateCompleted       MissionState = "completed"
	StateFailed          MissionState = "failed"
	StateCancelled       MissionState = "cancelled"
)

type Mission struct {
	ID             string       `json:"id"`
	TenantID       string       `json:"tenant_id"`
	Objective      string       `json:"objective"`
	RequestedTools []string     `json:"requested_tools"`
	ApprovedTools  []string     `json:"approved_tools"`
	AuthorityLevel string       `json:"authority_level"`
	BudgetUSD      float64      `json:"budget_usd"`
	BudgetUsedUSD  float64      `json:"budget_used_usd"`
	TimeoutSeconds int          `json:"timeout_seconds"`
	State          MissionState `json:"state"`
	CreatedBy      string       `json:"created_by"`
	ApprovedBy     string       `json:"approved_by,omitempty"`
	CreatedAt      time.Time    `json:"created_at"`
	ApprovedAt     *time.Time   `json:"approved_at,omitempty"`
	ClosedAt       *time.Time   `json:"closed_at,omitempty"`
	Events         []ProofEvent `json:"events"`
}

type ProofEvent struct {
	Sequence          int                        `json:"sequence"`
	EventType         string                     `json:"event_type"`
	ActorType         string                     `json:"actor_type"`
	ActorID           string                     `json:"actor_id"`
	PayloadHash       string                     `json:"payload_hash"`
	PolicyDecision    string                     `json:"policy_decision"`
	ToolName          string                     `json:"tool_name,omitempty"`
	SpendDelta        float64                    `json:"spend_delta"`
	VerificationState degraded.VerificationState `json:"verification_state"`
	Reason            string                     `json:"reason"`
	CreatedAt         time.Time                  `json:"created_at"`
}

type CreateRequest struct {
	TenantID       string
	Objective      string
	RequestedTools []string
	BudgetUSD      float64
	CreatedBy      string
}

type ToolCallRequest struct {
	ToolName string
	ActorID  string
	CostUSD  float64
	Metadata map[string]interface{}
}

type ToolCallResult struct {
	Decision string `json:"decision"`
	Reason   string `json:"reason"`
}
