# Mission Model

Mission fields currently implemented:
- id
- tenant_id
- objective
- requested_tools
- approved_tools
- authority_level
- budget_usd
- budget_used_usd
- timeout_seconds
- state
- created_by
- approved_by
- created_at
- approved_at
- closed_at
- events

Mission states in code:
- created
- planning
- waiting_approval
- approved
- denied
- running
- paused
- degraded
- completed
- failed
- cancelled

Current truth:
- `planning`, `paused`, `completed`, `failed`, and `cancelled` are declared states for the future model.
- `created`, `waiting_approval`, `approved`, `running`, and `degraded` are the states exercised by the current smoke path.
