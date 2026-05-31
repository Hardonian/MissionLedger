package policy

import "testing"

func TestDecide(t *testing.T) {
	tests := []struct {
		name     string
		toolName string
		decision Decision
	}{
		{name: "allow known safe tool", toolName: "read_file", decision: DecisionAllow},
		{name: "escalate risky tool", toolName: "terminal", decision: DecisionEscalate},
		{name: "deny secret tool", toolName: "secret.read", decision: DecisionDeny},
		{name: "escalate unknown tool", toolName: "mcp.supertool", decision: DecisionEscalate},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := Decide(tc.toolName)
			if result.Decision != tc.decision {
				t.Fatalf("expected %s, got %s", tc.decision, result.Decision)
			}
		})
	}
}
