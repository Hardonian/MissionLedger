package mission

import (
	"database/sql"
	"fmt"
	"os"

	_ "github.com/jackc/pgx/v5/stdlib"
)

type Repository interface {
	CreateMission(CreateRequest) (Mission, error)
	GetMission(id string) (Mission, bool, error)
	ListMissions(limit int) ([]Mission, error)
	ApproveMission(id, approvedBy string) (Mission, error)
	RecordToolCall(id string, req ToolCallRequest) (ToolCallResult, Mission, error)
}

func OpenStoreFromEnv() (Repository, func() error, string, error) {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		return NewStore(), func() error { return nil }, "memory", nil
	}

	db, err := sql.Open("pgx", databaseURL)
	if err != nil {
		return nil, nil, "", fmt.Errorf("open postgres: %w", err)
	}

	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)

	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, nil, "", fmt.Errorf("ping postgres: %w", err)
	}

	store, err := NewPostgresStore(db)
	if err != nil {
		_ = db.Close()
		return nil, nil, "", err
	}

	return store, db.Close, "postgres", nil
}
