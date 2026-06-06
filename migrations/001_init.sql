CREATE SEQUENCE IF NOT EXISTS mission_numbers START WITH 1;

CREATE TABLE IF NOT EXISTS missions (
    id TEXT PRIMARY KEY,
    tenant_id TEXT NOT NULL,
    objective TEXT NOT NULL,
    requested_tools JSONB NOT NULL,
    approved_tools JSONB NOT NULL,
    authority_level TEXT NOT NULL,
    budget_usd DOUBLE PRECISION NOT NULL,
    budget_used_usd DOUBLE PRECISION NOT NULL DEFAULT 0,
    timeout_seconds INTEGER NOT NULL,
    state TEXT NOT NULL,
    created_by TEXT NOT NULL,
    approved_by TEXT,
    created_at TIMESTAMPTZ NOT NULL,
    approved_at TIMESTAMPTZ,
    closed_at TIMESTAMPTZ
);

CREATE TABLE IF NOT EXISTS proof_events (
    mission_id TEXT NOT NULL REFERENCES missions(id) ON DELETE CASCADE,
    sequence INTEGER NOT NULL,
    event_type TEXT NOT NULL,
    actor_type TEXT NOT NULL,
    actor_id TEXT NOT NULL,
    payload_hash TEXT NOT NULL,
    policy_decision TEXT NOT NULL,
    tool_name TEXT,
    spend_delta DOUBLE PRECISION NOT NULL,
    verification_state TEXT NOT NULL,
    reason TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL,
    PRIMARY KEY (mission_id, sequence)
);
