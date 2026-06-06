package mission

import "testing"

func TestToolLifecycle(t *testing.T) {
	store := NewStore()

	created, err := store.CreateMission(CreateRequest{
		TenantID:       "tenant-1",
		Objective:      "Investigate an issue safely",
		RequestedTools: []string{"read_file", "terminal", "secret.read"},
		BudgetUSD:      1.00,
		CreatedBy:      "scott",
	})
	if err != nil {
		t.Fatalf("create mission: %v", err)
	}

	safe, updated, err := store.RecordToolCall(created.ID, ToolCallRequest{ToolName: "read_file", ActorID: "agent-1", CostUSD: 0.10})
	if err != nil {
		t.Fatalf("safe tool call: %v", err)
	}
	if safe.Decision != "allow" {
		t.Fatalf("expected allow, got %s", safe.Decision)
	}
	if updated.BudgetUsedUSD != 0.10 {
		t.Fatalf("expected budget used 0.10, got %.2f", updated.BudgetUsedUSD)
	}

	risky, _, err := store.RecordToolCall(created.ID, ToolCallRequest{ToolName: "terminal", ActorID: "agent-1", CostUSD: 0.20})
	if err != nil {
		t.Fatalf("risky tool call: %v", err)
	}
	if risky.Decision != "escalate" {
		t.Fatalf("expected escalate, got %s", risky.Decision)
	}

	approved, err := store.ApproveMission(created.ID, "human-1")
	if err != nil {
		t.Fatalf("approve mission: %v", err)
	}
	if approved.State != StateApproved {
		t.Fatalf("expected approved state, got %s", approved.State)
	}

	postApproval, updated, err := store.RecordToolCall(created.ID, ToolCallRequest{ToolName: "terminal", ActorID: "agent-1", CostUSD: 0.20})
	if err != nil {
		t.Fatalf("approved tool call: %v", err)
	}
	if postApproval.Decision != "allow" {
		t.Fatalf("expected allow after approval, got %s", postApproval.Decision)
	}
	if updated.State != StateRunning {
		t.Fatalf("expected running state, got %s", updated.State)
	}
}

func TestDeniedAndBudgetExceeded(t *testing.T) {
	store := NewStore()

	created, err := store.CreateMission(CreateRequest{
		TenantID:       "tenant-2",
		Objective:      "Run within budget",
		RequestedTools: []string{"read_file", "secret.read"},
		BudgetUSD:      0.30,
		CreatedBy:      "scott",
	})
	if err != nil {
		t.Fatalf("create mission: %v", err)
	}

	denied, _, err := store.RecordToolCall(created.ID, ToolCallRequest{ToolName: "secret.read", ActorID: "agent-2", CostUSD: 0.05})
	if err != nil {
		t.Fatalf("denied tool call: %v", err)
	}
	if denied.Decision != "deny" {
		t.Fatalf("expected deny, got %s", denied.Decision)
	}

	_, _, err = store.RecordToolCall(created.ID, ToolCallRequest{ToolName: "read_file", ActorID: "agent-2", CostUSD: 0.20})
	if err != nil {
		t.Fatalf("first budgeted call: %v", err)
	}

	budgetExceeded, updated, err := store.RecordToolCall(created.ID, ToolCallRequest{ToolName: "read_file", ActorID: "agent-2", CostUSD: 0.20})
	if err != nil {
		t.Fatalf("budget exceeded call: %v", err)
	}
	if budgetExceeded.Decision != "deny" {
		t.Fatalf("expected deny on budget breach, got %s", budgetExceeded.Decision)
	}
	if updated.State != StateDegraded {
		t.Fatalf("expected degraded state, got %s", updated.State)
	}
}

func TestMissionIDUnpredictability(t *testing.T) {
	store := NewStore()

	created1, err := store.CreateMission(CreateRequest{
		TenantID:  "tenant-test",
		Objective: "Test predictability 1",
	})
	if err != nil {
		t.Fatalf("create mission 1: %v", err)
	}

	created2, err := store.CreateMission(CreateRequest{
		TenantID:  "tenant-test",
		Objective: "Test predictability 2",
	})
	if err != nil {
		t.Fatalf("create mission 2: %v", err)
	}

	if created1.ID == created2.ID {
		t.Fatalf("mission IDs should be unique, got %s for both", created1.ID)
	}

	// Example prefix check: "mission-" followed by 36 characters (UUID length)
	if len(created1.ID) != len("mission-")+36 {
		t.Fatalf("mission ID 1 unexpected length: %s", created1.ID)
	}

	if len(created2.ID) != len("mission-")+36 {
		t.Fatalf("mission ID 2 unexpected length: %s", created2.ID)
	}
}
