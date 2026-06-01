package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/Hardonian/missionledger/internal/mission"
)

type apiServer struct {
	store   mission.Repository
	storage string
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

type missionsResponse struct {
	Storage  string            `json:"storage"`
	Count    int               `json:"count"`
	Missions []mission.Mission `json:"missions"`
}

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	store, cleanup, storage, err := mission.OpenStoreFromEnv()
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		if err := cleanup(); err != nil {
			log.Printf("missionledger cleanup error: %v", err)
		}
	}()

	srv := &apiServer{store: store, storage: storage}
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", srv.handleHealthz)
	mux.HandleFunc("/v1/missions", srv.handleMissions)
	mux.HandleFunc("/v1/missions/", srv.handleMissionRoutes)

	httpServer := &http.Server{
		Addr:              ":" + port,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	log.Printf("missionledger api listening on :%s (storage=%s)", port, storage)
	if err := httpServer.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}

func (s *apiServer) handleHealthz(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeMethodNotAllowed(w)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{
		"status":  "ok",
		"storage": s.storage,
	})
}

func (s *apiServer) handleMissions(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		limit := 20
		if raw := r.URL.Query().Get("limit"); raw != "" {
			parsed, err := strconv.Atoi(raw)
			if err != nil || parsed <= 0 {
				writeJSON(w, http.StatusBadRequest, errorResponse{Error: "limit must be a positive integer"})
				return
			}
			limit = parsed
		}

		missions, err := s.store.ListMissions(limit)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, errorResponse{Error: err.Error()})
			return
		}

		writeJSON(w, http.StatusOK, missionsResponse{Storage: s.storage, Count: len(missions), Missions: missions})
	case http.MethodPost:
		var req createMissionRequest
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&req); err != nil {
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
	default:
		writeMethodNotAllowed(w)
	}
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
		m, ok, err := s.store.GetMission(missionID)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, errorResponse{Error: err.Error()})
			return
		}
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
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&req); err != nil {
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
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&req); err != nil {
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
		m, ok, err := s.store.GetMission(missionID)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, errorResponse{Error: err.Error()})
			return
		}
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
