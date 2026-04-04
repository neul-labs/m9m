package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/neul-labs/m9m/internal/scheduler"
)

func (s *APIServer) ListSchedules(w http.ResponseWriter, r *http.Request) {
	schedules := s.scheduler.ListSchedules()
	s.sendJSON(w, http.StatusOK, map[string]interface{}{
		"data":  schedules,
		"total": len(schedules),
	})
}

func (s *APIServer) CreateSchedule(w http.ResponseWriter, r *http.Request) {
	var config scheduler.ScheduleConfig
	if err := json.NewDecoder(r.Body).Decode(&config); err != nil {
		s.sendError(w, http.StatusBadRequest, "Invalid JSON", err)
		return
	}

	if config.WorkflowID == "" {
		s.sendError(w, http.StatusBadRequest, "workflowId is required", nil)
		return
	}
	if config.CronExpr == "" {
		s.sendError(w, http.StatusBadRequest, "cronExpression is required", nil)
		return
	}

	if config.ID == "" {
		config.ID = fmt.Sprintf("schedule_%d", time.Now().UnixNano())
	}
	if config.Timezone == "" {
		config.Timezone = "UTC"
	}
	config.CreatedAt = time.Now()
	config.UpdatedAt = time.Now()

	if err := s.scheduler.AddSchedule(&config); err != nil {
		s.sendError(w, http.StatusBadRequest, "Failed to create schedule", err)
		return
	}

	s.sendJSON(w, http.StatusCreated, config)
}

func (s *APIServer) GetSchedule(w http.ResponseWriter, r *http.Request) {
	scheduleID := mux.Vars(r)["id"]

	schedule, err := s.scheduler.GetSchedule(scheduleID)
	if err != nil {
		s.sendError(w, http.StatusNotFound, "Schedule not found", err)
		return
	}

	s.sendJSON(w, http.StatusOK, schedule)
}

func (s *APIServer) UpdateSchedule(w http.ResponseWriter, r *http.Request) {
	scheduleID := mux.Vars(r)["id"]

	var updates scheduler.ScheduleConfig
	if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
		s.sendError(w, http.StatusBadRequest, "Invalid JSON", err)
		return
	}

	updates.UpdatedAt = time.Now()

	if err := s.scheduler.UpdateSchedule(scheduleID, &updates); err != nil {
		s.sendError(w, http.StatusBadRequest, "Failed to update schedule", err)
		return
	}

	schedule, _ := s.scheduler.GetSchedule(scheduleID)
	s.sendJSON(w, http.StatusOK, schedule)
}

func (s *APIServer) DeleteSchedule(w http.ResponseWriter, r *http.Request) {
	scheduleID := mux.Vars(r)["id"]

	if err := s.scheduler.RemoveSchedule(scheduleID); err != nil {
		s.sendError(w, http.StatusNotFound, "Schedule not found", err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (s *APIServer) EnableSchedule(w http.ResponseWriter, r *http.Request) {
	scheduleID := mux.Vars(r)["id"]

	updates := &scheduler.ScheduleConfig{Enabled: true, UpdatedAt: time.Now()}
	if err := s.scheduler.UpdateSchedule(scheduleID, updates); err != nil {
		s.sendError(w, http.StatusNotFound, "Schedule not found", err)
		return
	}

	schedule, _ := s.scheduler.GetSchedule(scheduleID)
	s.sendJSON(w, http.StatusOK, schedule)
}

func (s *APIServer) DisableSchedule(w http.ResponseWriter, r *http.Request) {
	scheduleID := mux.Vars(r)["id"]

	updates := &scheduler.ScheduleConfig{Enabled: false, UpdatedAt: time.Now()}
	if err := s.scheduler.UpdateSchedule(scheduleID, updates); err != nil {
		s.sendError(w, http.StatusNotFound, "Schedule not found", err)
		return
	}

	schedule, _ := s.scheduler.GetSchedule(scheduleID)
	s.sendJSON(w, http.StatusOK, schedule)
}

func (s *APIServer) GetScheduleHistory(w http.ResponseWriter, r *http.Request) {
	scheduleID := mux.Vars(r)["id"]
	limit := parseIntParam(r.URL.Query().Get("limit"), 50, 100)

	history, err := s.scheduler.GetExecutionHistory(scheduleID, limit)
	if err != nil {
		s.sendError(w, http.StatusNotFound, "Schedule history not found", err)
		return
	}

	s.sendJSON(w, http.StatusOK, history)
}
