package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Hardonian/missionledger/internal/mission"
)

func setupTestServer() (*apiServer, *http.ServeMux) {
	srv := &apiServer{store: mission.NewStore()}
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", srv.handleHealthz)
	mux.HandleFunc("/v1/missions", srv.handleMissions)
	mux.HandleFunc("/v1/missions/", srv.handleMissionRoutes)
	return srv, mux
}

func TestHandleHealthz(t *testing.T) {
	_, mux := setupTestServer()

	t.Run("GET returns 200 OK", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
		rr := httptest.NewRecorder()
		mux.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d", http.StatusOK, rr.Code)
		}

		var resp map[string]string
		if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}
		if status, ok := resp["status"]; !ok || status != "ok" {
			t.Errorf("expected status 'ok', got %v", status)
		}
	})

	t.Run("POST returns 405 Method Not Allowed", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/healthz", nil)
		rr := httptest.NewRecorder()
		mux.ServeHTTP(rr, req)

		if rr.Code != http.StatusMethodNotAllowed {
			t.Errorf("expected status %d, got %d", http.StatusMethodNotAllowed, rr.Code)
		}
	})
}

func TestHandleMissions(t *testing.T) {
	_, mux := setupTestServer()

	t.Run("POST returns 201 Created", func(t *testing.T) {
		payload := createMissionRequest{
			TenantID:       "tenant-1",
			Objective:      "Test objective",
			RequestedTools: []string{"read_file"},
			BudgetUSD:      10.0,
			CreatedBy:      "user-1",
		}
		body, _ := json.Marshal(payload)

		req := httptest.NewRequest(http.MethodPost, "/v1/missions", bytes.NewReader(body))
		rr := httptest.NewRecorder()
		mux.ServeHTTP(rr, req)

		if rr.Code != http.StatusCreated {
			t.Errorf("expected status %d, got %d. Body: %s", http.StatusCreated, rr.Code, rr.Body.String())
		}

		var resp mission.Mission
		if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
			t.Fatalf("failed to decode response: %v", err)
		}
		if resp.ID == "" {
			t.Errorf("expected mission ID to be set")
		}
		if resp.Objective != payload.Objective {
			t.Errorf("expected objective %q, got %q", payload.Objective, resp.Objective)
		}
	})

	t.Run("GET returns 405 Method Not Allowed", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/v1/missions", nil)
		rr := httptest.NewRecorder()
		mux.ServeHTTP(rr, req)

		if rr.Code != http.StatusMethodNotAllowed {
			t.Errorf("expected status %d, got %d", http.StatusMethodNotAllowed, rr.Code)
		}
	})

	t.Run("POST with invalid JSON returns 400 Bad Request", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/v1/missions", bytes.NewReader([]byte("{invalid-json}")))
		rr := httptest.NewRecorder()
		mux.ServeHTTP(rr, req)

		if rr.Code != http.StatusBadRequest {
			t.Errorf("expected status %d, got %d", http.StatusBadRequest, rr.Code)
		}
	})
}

func TestHandleMissionRoutes(t *testing.T) {
	srv, mux := setupTestServer()

	// Create a mission for use in these tests
	created, _ := srv.store.CreateMission(mission.CreateRequest{
		TenantID:       "tenant-1",
		Objective:      "Test",
		RequestedTools: []string{"read_file"},
		BudgetUSD:      10.0,
		CreatedBy:      "user-1",
	})

	t.Run("GET existing mission returns 200 OK", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/v1/missions/"+created.ID, nil)
		rr := httptest.NewRecorder()
		mux.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d", http.StatusOK, rr.Code)
		}
	})

	t.Run("GET non-existent mission returns 404 Not Found", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/v1/missions/does-not-exist", nil)
		rr := httptest.NewRecorder()
		mux.ServeHTTP(rr, req)

		if rr.Code != http.StatusNotFound {
			t.Errorf("expected status %d, got %d", http.StatusNotFound, rr.Code)
		}
	})

	t.Run("POST approve returns 200 OK", func(t *testing.T) {
		payload := approveMissionRequest{ApprovedBy: "approver-1"}
		body, _ := json.Marshal(payload)

		req := httptest.NewRequest(http.MethodPost, "/v1/missions/"+created.ID+"/approve", bytes.NewReader(body))
		rr := httptest.NewRecorder()
		mux.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d. Body: %s", http.StatusOK, rr.Code, rr.Body.String())
		}
	})

	t.Run("POST tool-calls returns 200 OK", func(t *testing.T) {
		payload := toolCallRequest{
			ToolName: "read_file",
			ActorID:  "agent-1",
			CostUSD:  0.01,
			Metadata: map[string]interface{}{"path": "test.txt"},
		}
		body, _ := json.Marshal(payload)

		req := httptest.NewRequest(http.MethodPost, "/v1/missions/"+created.ID+"/tool-calls", bytes.NewReader(body))
		rr := httptest.NewRecorder()
		mux.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d. Body: %s", http.StatusOK, rr.Code, rr.Body.String())
		}
	})

	t.Run("GET proofpack returns 200 OK", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/v1/missions/"+created.ID+"/proofpack", nil)
		rr := httptest.NewRecorder()
		mux.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("expected status %d, got %d", http.StatusOK, rr.Code)
		}
	})
}
