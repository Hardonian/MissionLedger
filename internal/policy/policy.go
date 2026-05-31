package policy

type Decision string

const (
	DecisionAllow    Decision = "allow"
	DecisionEscalate Decision = "escalate"
	DecisionDeny     Decision = "deny"
)

type Result struct {
	Decision         Decision `json:"decision"`
	Reason           string   `json:"reason"`
	RequiresApproval bool     `json:"requires_approval"`
}

var allowTools = map[string]struct{}{
	"read_file":    {},
	"search_files": {},
	"web_search":   {},
}

var escalateTools = map[string]struct{}{
	"terminal":      {},
	"write_file":    {},
	"patch":         {},
	"browser":       {},
	"browser_click": {},
}

var denyTools = map[string]struct{}{
	"secret.read":  {},
	"send_message": {},
}

func Decide(toolName string) Result {
	if _, ok := denyTools[toolName]; ok {
		return Result{
			Decision:         DecisionDeny,
			Reason:           "tool is categorically denied by policy",
			RequiresApproval: false,
		}
	}

	if _, ok := escalateTools[toolName]; ok {
		return Result{
			Decision:         DecisionEscalate,
			Reason:           "tool requires explicit approval",
			RequiresApproval: true,
		}
	}

	if _, ok := allowTools[toolName]; ok {
		return Result{
			Decision:         DecisionAllow,
			Reason:           "tool is permitted for low-risk mission work",
			RequiresApproval: false,
		}
	}

	return Result{
		Decision:         DecisionEscalate,
		Reason:           "tool is unknown and must be escalated",
		RequiresApproval: true,
	}
}
