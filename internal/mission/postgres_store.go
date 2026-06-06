package mission

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/Hardonian/missionledger/internal/degraded"
)

type PostgresStore struct {
	db *sql.DB
}

func NewPostgresStore(db *sql.DB) (*PostgresStore, error) {
	store := &PostgresStore{db: db}
	if err := store.ensureSchema(); err != nil {
		return nil, fmt.Errorf("ensure postgres schema: %w", err)
	}
	return store, nil
}

func (s *PostgresStore) ensureSchema() error {
	_, err := s.db.Exec(postgresSchema)
	return err
}

func (s *PostgresStore) CreateMission(req CreateRequest) (Mission, error) {
	if err := normalizeCreateRequest(&req); err != nil {
		return Mission{}, err
	}

	tx, err := s.db.Begin()
	if err != nil {
		return Mission{}, err
	}
	defer tx.Rollback()

	var seq int
	if err := tx.QueryRow(`SELECT nextval('mission_numbers')`).Scan(&seq); err != nil {
		return Mission{}, err
	}

	id := fmt.Sprintf("mission-%04d", seq)
	m := initializeMission(id, req, time.Now().UTC())
	if err := insertMissionTx(tx, m); err != nil {
		return Mission{}, err
	}
	if err := insertEventsTx(tx, m.ID, m.Events); err != nil {
		return Mission{}, err
	}

	if err := tx.Commit(); err != nil {
		return Mission{}, err
	}

	return cloneMission(*m), nil
}

func (s *PostgresStore) GetMission(id string) (Mission, bool, error) {
	m, err := getMissionWithEvents(s.db, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Mission{}, false, nil
		}
		return Mission{}, false, err
	}
	return m, true, nil
}

func (s *PostgresStore) ListMissions(limit int) ([]Mission, error) {
	if limit <= 0 {
		limit = 20
	}

	rows, err := s.db.Query(`
		SELECT id, tenant_id, objective, requested_tools, approved_tools, authority_level,
		       budget_usd, budget_used_usd, timeout_seconds, state, created_by, approved_by,
		       created_at, approved_at, closed_at
		FROM missions
		ORDER BY created_at DESC
		LIMIT $1
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	missions := []Mission{}
	for rows.Next() {
		m, err := scanMission(rows)
		if err != nil {
			return nil, err
		}
		events, err := loadEvents(s.db, m.ID)
		if err != nil {
			return nil, err
		}
		m.Events = events
		missions = append(missions, m)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return missions, nil
}

func (s *PostgresStore) ApproveMission(id, approvedBy string) (Mission, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return Mission{}, err
	}
	defer tx.Rollback()

	m, err := getMissionWithEvents(tx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Mission{}, fmt.Errorf("mission not found: %s", id)
		}
		return Mission{}, err
	}

	before := len(m.Events)
	applyApproval(&m, approvedBy)
	if err := updateMissionTx(tx, &m); err != nil {
		return Mission{}, err
	}
	if err := insertEventsTx(tx, m.ID, m.Events[before:]); err != nil {
		return Mission{}, err
	}

	if err := tx.Commit(); err != nil {
		return Mission{}, err
	}

	return m, nil
}

func (s *PostgresStore) RecordToolCall(id string, req ToolCallRequest) (ToolCallResult, Mission, error) {
	if err := validateToolCallRequest(req); err != nil {
		return ToolCallResult{}, Mission{}, err
	}

	tx, err := s.db.Begin()
	if err != nil {
		return ToolCallResult{}, Mission{}, err
	}
	defer tx.Rollback()

	m, err := getMissionWithEvents(tx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ToolCallResult{}, Mission{}, fmt.Errorf("mission not found: %s", id)
		}
		return ToolCallResult{}, Mission{}, err
	}

	before := len(m.Events)
	result := applyToolCall(&m, req)
	if err := updateMissionTx(tx, &m); err != nil {
		return ToolCallResult{}, Mission{}, err
	}
	if err := insertEventsTx(tx, m.ID, m.Events[before:]); err != nil {
		return ToolCallResult{}, Mission{}, err
	}

	if err := tx.Commit(); err != nil {
		return ToolCallResult{}, Mission{}, err
	}

	return result, m, nil
}

type queryer interface {
	QueryRow(query string, args ...interface{}) *sql.Row
	Query(query string, args ...interface{}) (*sql.Rows, error)
}

type scanner interface {
	Scan(dest ...interface{}) error
}

func getMissionWithEvents(q queryer, id string) (Mission, error) {
	row := q.QueryRow(`
		SELECT id, tenant_id, objective, requested_tools, approved_tools, authority_level,
		       budget_usd, budget_used_usd, timeout_seconds, state, created_by, approved_by,
		       created_at, approved_at, closed_at
		FROM missions
		WHERE id = $1
	`, id)

	m, err := scanMission(row)
	if err != nil {
		return Mission{}, err
	}

	events, err := loadEvents(q, id)
	if err != nil {
		return Mission{}, err
	}
	m.Events = events
	return m, nil
}

func loadEvents(q queryer, missionID string) ([]ProofEvent, error) {
	rows, err := q.Query(`
		SELECT sequence, event_type, actor_type, actor_id, payload_hash, policy_decision,
		       tool_name, spend_delta, verification_state, reason, created_at
		FROM proof_events
		WHERE mission_id = $1
		ORDER BY sequence ASC
	`, missionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	events := []ProofEvent{}
	for rows.Next() {
		var event ProofEvent
		var toolName sql.NullString
		var verificationState string
		if err := rows.Scan(
			&event.Sequence,
			&event.EventType,
			&event.ActorType,
			&event.ActorID,
			&event.PayloadHash,
			&event.PolicyDecision,
			&toolName,
			&event.SpendDelta,
			&verificationState,
			&event.Reason,
			&event.CreatedAt,
		); err != nil {
			return nil, err
		}
		if toolName.Valid {
			event.ToolName = toolName.String
		}
		event.VerificationState = degraded.VerificationState(verificationState)
		events = append(events, event)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return events, nil
}

func scanMission(row scanner) (Mission, error) {
	var mission Mission
	var requestedRaw []byte
	var approvedRaw []byte
	var state string
	var approvedBy sql.NullString
	var approvedAt sql.NullTime
	var closedAt sql.NullTime

	err := row.Scan(
		&mission.ID,
		&mission.TenantID,
		&mission.Objective,
		&requestedRaw,
		&approvedRaw,
		&mission.AuthorityLevel,
		&mission.BudgetUSD,
		&mission.BudgetUsedUSD,
		&mission.TimeoutSeconds,
		&state,
		&mission.CreatedBy,
		&approvedBy,
		&mission.CreatedAt,
		&approvedAt,
		&closedAt,
	)
	if err != nil {
		return Mission{}, err
	}

	if err := json.Unmarshal(requestedRaw, &mission.RequestedTools); err != nil {
		return Mission{}, err
	}
	if err := json.Unmarshal(approvedRaw, &mission.ApprovedTools); err != nil {
		return Mission{}, err
	}
	mission.State = MissionState(state)
	if approvedBy.Valid {
		mission.ApprovedBy = approvedBy.String
	}
	if approvedAt.Valid {
		value := approvedAt.Time
		mission.ApprovedAt = &value
	}
	if closedAt.Valid {
		value := closedAt.Time
		mission.ClosedAt = &value
	}
	mission.Events = []ProofEvent{}
	return mission, nil
}

func insertMissionTx(tx *sql.Tx, m *Mission) error {
	requestedRaw, err := json.Marshal(m.RequestedTools)
	if err != nil {
		return err
	}
	approvedRaw, err := json.Marshal(m.ApprovedTools)
	if err != nil {
		return err
	}

	_, err = tx.Exec(`
		INSERT INTO missions (
			id, tenant_id, objective, requested_tools, approved_tools, authority_level,
			budget_usd, budget_used_usd, timeout_seconds, state, created_by, approved_by,
			created_at, approved_at, closed_at
		) VALUES (
			$1, $2, $3, $4::jsonb, $5::jsonb, $6,
			$7, $8, $9, $10, $11, $12,
			$13, $14, $15
		)
	`, m.ID, m.TenantID, m.Objective, string(requestedRaw), string(approvedRaw), m.AuthorityLevel,
		m.BudgetUSD, m.BudgetUsedUSD, m.TimeoutSeconds, string(m.State), m.CreatedBy, nullableString(m.ApprovedBy),
		m.CreatedAt, nullableTime(m.ApprovedAt), nullableTime(m.ClosedAt))
	return err
}

func updateMissionTx(tx *sql.Tx, m *Mission) error {
	requestedRaw, err := json.Marshal(m.RequestedTools)
	if err != nil {
		return err
	}
	approvedRaw, err := json.Marshal(m.ApprovedTools)
	if err != nil {
		return err
	}

	_, err = tx.Exec(`
		UPDATE missions
		SET tenant_id = $2,
		    objective = $3,
		    requested_tools = $4::jsonb,
		    approved_tools = $5::jsonb,
		    authority_level = $6,
		    budget_usd = $7,
		    budget_used_usd = $8,
		    timeout_seconds = $9,
		    state = $10,
		    created_by = $11,
		    approved_by = $12,
		    created_at = $13,
		    approved_at = $14,
		    closed_at = $15
		WHERE id = $1
	`, m.ID, m.TenantID, m.Objective, string(requestedRaw), string(approvedRaw), m.AuthorityLevel,
		m.BudgetUSD, m.BudgetUsedUSD, m.TimeoutSeconds, string(m.State), m.CreatedBy, nullableString(m.ApprovedBy),
		m.CreatedAt, nullableTime(m.ApprovedAt), nullableTime(m.ClosedAt))
	return err
}

func insertEventsTx(tx *sql.Tx, missionID string, events []ProofEvent) error {
	for _, event := range events {
		_, err := tx.Exec(`
			INSERT INTO proof_events (
				mission_id, sequence, event_type, actor_type, actor_id, payload_hash,
				policy_decision, tool_name, spend_delta, verification_state, reason, created_at
			) VALUES (
				$1, $2, $3, $4, $5, $6,
				$7, $8, $9, $10, $11, $12
			)
		`, missionID, event.Sequence, event.EventType, event.ActorType, event.ActorID, event.PayloadHash,
			event.PolicyDecision, nullableString(event.ToolName), event.SpendDelta, string(event.VerificationState), event.Reason, event.CreatedAt)
		if err != nil {
			return err
		}
	}
	return nil
}

func nullableString(value string) interface{} {
	if value == "" {
		return nil
	}
	return value
}

func nullableTime(value *time.Time) interface{} {
	if value == nil {
		return nil
	}
	return *value
}

const postgresSchema = `
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
`
