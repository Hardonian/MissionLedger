package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/Hardonian/missionledger/internal/mission"
)

type apiServer struct {
	store *mission.Store
}

type errorResponse struct {
	Error string `json:"error"`
}

type createMissionRequest struct {
	TenantID       string   `json:"tenant_id"`
	Objective      string   `json:"objective"`
	RequestedTools []string `json:"requested_tools"`
	BudgetUSD      float64  `json:"budget_usd"`
	CreatedBy      string   `json:"created_by"`
}

type approveMissionRequest struct {
	ApprovedBy string `json:"approved_by"`
}

type toolCallRequest struct {
	ToolName string                 `json:"tool_name"`
	ActorID  string                 `json:"actor_id"`
	CostUSD  float64                `json:"cost_usd"`
	Metadata map[string]interface{} `json:"metadata"`
}

type proofpackResponse struct {
	Mission mission.Mission `json:"mission"`
	Summary struct {
		EventCount      int     `json:"event_count"`
		BudgetUSD       float64 `json:"budget_usd"`
		BudgetUsedUSD   float64 `json:"budget_used_usd"`
		BudgetRemainUSD float64 `json:"budget_remaining_usd"`
	} `json:"summary"`
}

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	srv := &apiServer{store: mission.NewStore()}
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", srv.handleHealthz)
	mux.HandleFunc("/v1/missions", srv.handleMissions)
	mux.HandleFunc("/v1/missions/", srv.handleMissionRoutes)

	log.Printf("missionledger api listening on :%s", port)
	server := &http.Server{
		Addr:         ":" + port,
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  15 * time.Second,
	}
	if err := server.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}

func (s *apiServer) handleHealthz(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeMethodNotAllowed(w)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *apiServer) handleMissions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeMethodNotAllowed(w)
		return
	}

	var req createMissionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid json body"})
		return
	}

	created, err := s.store.CreateMission(mission.CreateRequest{
		TenantID:       req.TenantID,
		Objective:      req.Objective,
		RequestedTools: req.RequestedTools,
		BudgetUSD:      req.BudgetUSD,
		CreatedBy:      req.CreatedBy,
	})
	if err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: err.Error()})
		return
	}

	writeJSON(w, http.StatusCreated, created)
}

func (s *apiServer) handleMissionRoutes(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/v1/missions/")
	path = strings.Trim(path, "/")
	if path == "" {
		writeJSON(w, http.StatusNotFound, errorResponse{Error: "not found"})
		return
	}

	parts := strings.Split(path, "/")
	missionID := parts[0]

	if len(parts) == 1 {
		if r.Method != http.MethodGet {
			writeMethodNotAllowed(w)
			return
		}
		m, ok := s.store.GetMission(missionID)
		if !ok {
			writeJSON(w, http.StatusNotFound, errorResponse{Error: "mission not found"})
			return
		}
		writeJSON(w, http.StatusOK, m)
		return
	}

	if len(parts) != 2 {
		writeJSON(w, http.StatusNotFound, errorResponse{Error: "not found"})
		return
	}

	switch parts[1] {
	case "approve":
		if r.Method != http.MethodPost {
			writeMethodNotAllowed(w)
			return
		}
		var req approveMissionRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid json body"})
			return
		}
		updated, err := s.store.ApproveMission(missionID, req.ApprovedBy)
		if err != nil {
			status := http.StatusBadRequest
			if strings.Contains(err.Error(), "not found") {
				status = http.StatusNotFound
			}
			writeJSON(w, status, errorResponse{Error: err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, updated)
	case "tool-calls":
		if r.Method != http.MethodPost {
			writeMethodNotAllowed(w)
			return
		}
		var req toolCallRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid json body"})
			return
		}
		result, updated, err := s.store.RecordToolCall(missionID, mission.ToolCallRequest{
			ToolName: req.ToolName,
			ActorID:  req.ActorID,
			CostUSD:  req.CostUSD,
			Metadata: req.Metadata,
		})
		if err != nil {
			status := http.StatusBadRequest
			if strings.Contains(err.Error(), "not found") {
				status = http.StatusNotFound
			}
			writeJSON(w, status, errorResponse{Error: err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"result":  result,
			"mission": updated,
		})
	case "proofpack":
		if r.Method != http.MethodGet {
			writeMethodNotAllowed(w)
			return
		}
		m, ok := s.store.GetMission(missionID)
		if !ok {
			writeJSON(w, http.StatusNotFound, errorResponse{Error: "mission not found"})
			return
		}
		resp := proofpackResponse{Mission: m}
		resp.Summary.EventCount = len(m.Events)
		resp.Summary.BudgetUSD = m.BudgetUSD
		resp.Summary.BudgetUsedUSD = m.BudgetUsedUSD
		resp.Summary.BudgetRemainUSD = m.BudgetUSD - m.BudgetUsedUSD
		writeJSON(w, http.StatusOK, resp)
	default:
		writeJSON(w, http.StatusNotFound, errorResponse{Error: "not found"})
	}
}

func writeMethodNotAllowed(w http.ResponseWriter) {
	writeJSON(w, http.StatusMethodNotAllowed, errorResponse{Error: "method not allowed"})
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
