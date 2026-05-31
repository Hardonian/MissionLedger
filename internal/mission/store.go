package mission

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/Hardonian/missionledger/internal/degraded"
	"github.com/Hardonian/missionledger/internal/policy"
)

type Store struct {
	mu       sync.RWMutex
	seq      int
	missions map[string]*Mission
}

func NewStore() *Store {
	return &Store{missions: make(map[string]*Mission)}
}

func (s *Store) CreateMission(req CreateRequest) (Mission, error) {
	if req.TenantID == "" {
		return Mission{}, errors.New("tenant_id is required")
	}
	if req.Objective == "" {
		return Mission{}, errors.New("objective is required")
	}
	if req.CreatedBy == "" {
		req.CreatedBy = "unknown"
	}
	if len(req.RequestedTools) == 0 {
		req.RequestedTools = []string{"read_file"}
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.seq++
	now := time.Now().UTC()
	id := fmt.Sprintf("mission-%04d", s.seq)
	m := &Mission{
		ID:             id,
		TenantID:       req.TenantID,
		Objective:      req.Objective,
		RequestedTools: copyStrings(req.RequestedTools),
		ApprovedTools:  []string{},
		AuthorityLevel: "L1",
		BudgetUSD:      req.BudgetUSD,
		TimeoutSeconds: 900,
		State:          StateCreated,
		CreatedBy:      req.CreatedBy,
		CreatedAt:      now,
		Events:         []ProofEvent{},
	}

	s.addEvent(m, "mission.created", "user", req.CreatedBy, map[string]interface{}{
		"objective":       req.Objective,
		"requested_tools": req.RequestedTools,
		"budget_usd":      req.BudgetUSD,
	}, "allow", "", 0, degraded.StateVerified, "mission created")

	s.missions[id] = m
	return cloneMission(*m), nil
}

func (s *Store) GetMission(id string) (Mission, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	m, ok := s.missions[id]
	if !ok {
		return Mission{}, false
	}
	return cloneMission(*m), true
}

func (s *Store) ApproveMission(id, approvedBy string) (Mission, error) {
	if approvedBy == "" {
		approvedBy = "unknown-approver"
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	m, ok := s.missions[id]
	if !ok {
		return Mission{}, fmt.Errorf("mission not found: %s", id)
	}

	now := time.Now().UTC()
	m.State = StateApproved
	m.ApprovedBy = approvedBy
	m.ApprovedAt = &now
	m.ApprovedTools = copyStrings(m.RequestedTools)

	s.addEvent(m, "mission.approved", "human", approvedBy, map[string]interface{}{
		"approved_tools": m.ApprovedTools,
	}, "allow", "", 0, degraded.StateVerified, "mission approved for risky tools")

	return cloneMission(*m), nil
}

func (s *Store) RecordToolCall(id string, req ToolCallRequest) (ToolCallResult, Mission, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	m, ok := s.missions[id]
	if !ok {
		return ToolCallResult{}, Mission{}, fmt.Errorf("mission not found: %s", id)
	}

	if req.ActorID == "" {
		req.ActorID = "agent"
	}

	decision := policy.Decide(req.ToolName)

	if !contains(m.RequestedTools, req.ToolName) {
		s.addEvent(m, "tool.denied", "agent", req.ActorID, req.Metadata, "deny", req.ToolName, 0, degraded.StateDenied, "tool is outside mission scope")
		return ToolCallResult{Decision: "deny", Reason: "tool is outside mission scope"}, cloneMission(*m), nil
	}

	if decision.Decision == policy.DecisionDeny {
		s.addEvent(m, "tool.denied", "agent", req.ActorID, req.Metadata, string(decision.Decision), req.ToolName, 0, degraded.StateDenied, decision.Reason)
		return ToolCallResult{Decision: string(decision.Decision), Reason: decision.Reason}, cloneMission(*m), nil
	}

	requiresApproval := decision.Decision == policy.DecisionEscalate && !contains(m.ApprovedTools, req.ToolName)
	if requiresApproval {
		m.State = StateWaitingApproval
		s.addEvent(m, "tool.escalated", "agent", req.ActorID, req.Metadata, string(decision.Decision), req.ToolName, 0, degraded.StatePartial, decision.Reason)
		return ToolCallResult{Decision: string(decision.Decision), Reason: decision.Reason}, cloneMission(*m), nil
	}

	if m.BudgetUSD > 0 && m.BudgetUsedUSD+req.CostUSD > m.BudgetUSD {
		m.State = StateDegraded
		s.addEvent(m, "budget.exceeded", "agent", req.ActorID, req.Metadata, "deny", req.ToolName, 0, degraded.StateUnavailable, "budget cap exceeded")
		return ToolCallResult{Decision: "deny", Reason: "budget cap exceeded"}, cloneMission(*m), nil
	}

	m.BudgetUsedUSD += req.CostUSD
	m.State = StateRunning
	reason := decision.Reason
	if decision.Decision == policy.DecisionEscalate && contains(m.ApprovedTools, req.ToolName) {
		reason = "tool use allowed after explicit approval"
	}
	s.addEvent(m, "tool.allowed", "agent", req.ActorID, req.Metadata, "allow", req.ToolName, req.CostUSD, degraded.StateVerified, reason)
	return ToolCallResult{Decision: "allow", Reason: reason}, cloneMission(*m), nil
}

func (s *Store) addEvent(m *Mission, eventType, actorType, actorID string, payload interface{}, policyDecision, toolName string, spendDelta float64, verificationState degraded.VerificationState, reason string) {
	event := ProofEvent{
		Sequence:          len(m.Events) + 1,
		EventType:         eventType,
		ActorType:         actorType,
		ActorID:           actorID,
		PayloadHash:       hashPayload(payload),
		PolicyDecision:    policyDecision,
		ToolName:          toolName,
		SpendDelta:        spendDelta,
		VerificationState: verificationState,
		Reason:            reason,
		CreatedAt:         time.Now().UTC(),
	}
	m.Events = append(m.Events, event)
}

func hashPayload(payload interface{}) string {
	raw, err := json.Marshal(payload)
	if err != nil {
		return "unavailable"
	}
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}

func contains(items []string, wanted string) bool {
	for _, item := range items {
		if item == wanted {
			return true
		}
	}
	return false
}

func copyStrings(items []string) []string {
	out := make([]string, len(items))
	copy(out, items)
	return out
}

func cloneMission(in Mission) Mission {
	out := in
	out.RequestedTools = copyStrings(in.RequestedTools)
	out.ApprovedTools = copyStrings(in.ApprovedTools)
	out.Events = make([]ProofEvent, len(in.Events))
	copy(out.Events, in.Events)
	return out
}
