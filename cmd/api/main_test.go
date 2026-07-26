package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Hardonian/missionledger/internal/mission"
)

func TestHandleHealthz(t *testing.T) {
	srv := &apiServer{store: mission.NewStore()}

	// Test GET method
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rr := httptest.NewRecorder()
	srv.handleHealthz(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	var got map[string]interface{}
	if err := json.NewDecoder(rr.Body).Decode(&got); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if got["status"] != "ok" {
		t.Errorf("handler returned unexpected status: got %v want %v", got["status"], "ok")
	}

	// Test non-GET method
	req = httptest.NewRequest(http.MethodPost, "/healthz", nil)
	rr = httptest.NewRecorder()
	srv.handleHealthz(rr, req)

	if status := rr.Code; status != http.StatusMethodNotAllowed {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusMethodNotAllowed)
	}
}

func TestHandleMissions(t *testing.T) {
	srv := &apiServer{store: mission.NewStore()}

	// Test valid POST
	payload := createMissionRequest{
		TenantID:       "tenant-1",
		Objective:      "Test mission",
		RequestedTools: []string{"read_file"},
		BudgetUSD:      10.0,
		CreatedBy:      "test-user",
	}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/v1/missions", bytes.NewBuffer(body))
	rr := httptest.NewRecorder()
	srv.handleMissions(rr, req)

	if status := rr.Code; status != http.StatusCreated {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusCreated)
	}

	var resp mission.Mission
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.TenantID != payload.TenantID {
		t.Errorf("expected tenant ID %s, got %s", payload.TenantID, resp.TenantID)
	}

	// Test non-POST method — GET should be allowed and return OK
	req = httptest.NewRequest(http.MethodGet, "/v1/missions", nil)
	rr = httptest.NewRecorder()
	srv.handleMissions(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	// Test invalid JSON
	req = httptest.NewRequest(http.MethodPost, "/v1/missions", bytes.NewBufferString("invalid json"))
	rr = httptest.NewRecorder()
	srv.handleMissions(rr, req)

	if status := rr.Code; status != http.StatusBadRequest {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusBadRequest)
	}
}

func TestHandleMissionRoutes(t *testing.T) {
	srv := &apiServer{store: mission.NewStore()}

	// Create a mission for testing
	m, _ := srv.store.CreateMission(mission.CreateRequest{
		TenantID:       "tenant-1",
		Objective:      "Test mission",
		RequestedTools: []string{"read_file"},
		BudgetUSD:      10.0,
		CreatedBy:      "test-user",
	})

	t.Run("GET /v1/missions/{id}", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/v1/missions/"+m.ID, nil)
		rr := httptest.NewRecorder()
		srv.handleMissionRoutes(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
		}
	})

	t.Run("GET /v1/missions/{id} Not Found", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/v1/missions/non-existent-id", nil)
		rr := httptest.NewRecorder()
		srv.handleMissionRoutes(rr, req)

		if status := rr.Code; status != http.StatusNotFound {
			t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusNotFound)
		}
	})

	t.Run("POST /v1/missions/{id}/approve", func(t *testing.T) {
		payload := approveMissionRequest{ApprovedBy: "approver-1"}
		body, _ := json.Marshal(payload)
		req := httptest.NewRequest(http.MethodPost, "/v1/missions/"+m.ID+"/approve", bytes.NewBuffer(body))
		rr := httptest.NewRecorder()
		srv.handleMissionRoutes(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
		}
	})

	t.Run("POST /v1/missions/{id}/tool-calls", func(t *testing.T) {
		payload := toolCallRequest{
			ToolName: "read_file",
			ActorID:  "actor-1",
			CostUSD:  1.0,
		}
		body, _ := json.Marshal(payload)
		req := httptest.NewRequest(http.MethodPost, "/v1/missions/"+m.ID+"/tool-calls", bytes.NewBuffer(body))
		rr := httptest.NewRecorder()
		srv.handleMissionRoutes(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
		}
	})

	t.Run("GET /v1/missions/{id}/proofpack", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/v1/missions/"+m.ID+"/proofpack", nil)
		rr := httptest.NewRecorder()
		srv.handleMissionRoutes(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
		}
	})

	t.Run("GET /v1/missions/{id}/audit", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/v1/missions/"+m.ID+"/audit", nil)
		rr := httptest.NewRecorder()
		srv.handleMissionRoutes(rr, req)
		if status := rr.Code; status != http.StatusOK {
			t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
		}
		if got := rr.Header().Get("Content-Disposition"); got == "" {
			t.Error("expected audit export content disposition")
		}
	})

	t.Run("Invalid Route", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/v1/missions/"+m.ID+"/invalid", nil)
		rr := httptest.NewRecorder()
		srv.handleMissionRoutes(rr, req)

		if status := rr.Code; status != http.StatusNotFound {
			t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusNotFound)
		}
	})
}

func TestHandleMissions_CreateError(t *testing.T) {
	srv := &apiServer{store: mission.NewStore()}

	// Test invalid payload (empty objective will trigger an error from the store)
	payload := createMissionRequest{
		TenantID:       "tenant-1",
		Objective:      "", // Missing objective should fail
		RequestedTools: []string{"read_file"},
		BudgetUSD:      10.0,
		CreatedBy:      "test-user",
	}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/v1/missions", bytes.NewBuffer(body))
	rr := httptest.NewRecorder()
	srv.handleMissions(rr, req)

	if status := rr.Code; status != http.StatusBadRequest {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusBadRequest)
	}
}

func TestHandleMissionRoutes_InvalidJSONs(t *testing.T) {
	srv := &apiServer{store: mission.NewStore()}
	m, _ := srv.store.CreateMission(mission.CreateRequest{
		TenantID:       "tenant-1",
		Objective:      "Test mission",
		RequestedTools: []string{"read_file"},
		BudgetUSD:      10.0,
		CreatedBy:      "test-user",
	})

	// Test approve invalid JSON
	req := httptest.NewRequest(http.MethodPost, "/v1/missions/"+m.ID+"/approve", bytes.NewBufferString("invalid json"))
	rr := httptest.NewRecorder()
	srv.handleMissionRoutes(rr, req)
	if status := rr.Code; status != http.StatusBadRequest {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusBadRequest)
	}

	// Test approve wrong method
	req = httptest.NewRequest(http.MethodGet, "/v1/missions/"+m.ID+"/approve", nil)
	rr = httptest.NewRecorder()
	srv.handleMissionRoutes(rr, req)
	if status := rr.Code; status != http.StatusMethodNotAllowed {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusMethodNotAllowed)
	}

	// Test tool-calls invalid JSON
	req = httptest.NewRequest(http.MethodPost, "/v1/missions/"+m.ID+"/tool-calls", bytes.NewBufferString("invalid json"))
	rr = httptest.NewRecorder()
	srv.handleMissionRoutes(rr, req)
	if status := rr.Code; status != http.StatusBadRequest {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusBadRequest)
	}

	// Test tool-calls wrong method
	req = httptest.NewRequest(http.MethodGet, "/v1/missions/"+m.ID+"/tool-calls", nil)
	rr = httptest.NewRecorder()
	srv.handleMissionRoutes(rr, req)
	if status := rr.Code; status != http.StatusMethodNotAllowed {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusMethodNotAllowed)
	}

	// Test proofpack wrong method
	req = httptest.NewRequest(http.MethodPost, "/v1/missions/"+m.ID+"/proofpack", nil)
	rr = httptest.NewRecorder()
	srv.handleMissionRoutes(rr, req)
	if status := rr.Code; status != http.StatusMethodNotAllowed {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusMethodNotAllowed)
	}
}

func TestHandleMissionRoutes_BaseRouteErrors(t *testing.T) {
	srv := &apiServer{store: mission.NewStore()}

	// Test empty path after trimming
	req := httptest.NewRequest(http.MethodGet, "/v1/missions/", nil)
	rr := httptest.NewRecorder()
	srv.handleMissionRoutes(rr, req)
	if status := rr.Code; status != http.StatusNotFound {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusNotFound)
	}

	// Test GET base mission with wrong method
	req = httptest.NewRequest(http.MethodPost, "/v1/missions/mission-0001", bytes.NewBufferString("{}"))
	rr = httptest.NewRecorder()
	srv.handleMissionRoutes(rr, req)
	if status := rr.Code; status != http.StatusMethodNotAllowed {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusMethodNotAllowed)
	}
}

func TestHandleMissions_InvalidLimit(t *testing.T) {
	srv := &apiServer{store: mission.NewStore()}
	for _, raw := range []string{"0", "-1", "not-a-number"} {
		t.Run(raw, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/v1/missions?limit="+raw, nil)
			rr := httptest.NewRecorder()
			srv.handleMissions(rr, req)
			if rr.Code != http.StatusBadRequest {
				t.Fatalf("invalid limit returned %d, want %d", rr.Code, http.StatusBadRequest)
			}
		})
	}
}

func TestHandleMissionRoutes_Errors(t *testing.T) {
	srv := &apiServer{store: mission.NewStore()}

	// Test Approve on non-existent mission
	payload := approveMissionRequest{ApprovedBy: "approver-1"}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/v1/missions/non-existent/approve", bytes.NewBuffer(body))
	rr := httptest.NewRecorder()
	srv.handleMissionRoutes(rr, req)
	if status := rr.Code; status != http.StatusNotFound {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusNotFound)
	}

	// Test ToolCall on non-existent mission
	tcPayload := toolCallRequest{ToolName: "read_file", ActorID: "actor-1", CostUSD: 1.0}
	tcBody, _ := json.Marshal(tcPayload)
	req = httptest.NewRequest(http.MethodPost, "/v1/missions/non-existent/tool-calls", bytes.NewBuffer(tcBody))
	rr = httptest.NewRecorder()
	srv.handleMissionRoutes(rr, req)
	if status := rr.Code; status != http.StatusNotFound {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusNotFound)
	}

	// Test too many path parts
	req = httptest.NewRequest(http.MethodGet, "/v1/missions/mission-1/approve/extra", nil)
	rr = httptest.NewRecorder()
	srv.handleMissionRoutes(rr, req)
	if status := rr.Code; status != http.StatusNotFound {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusNotFound)
	}
}
