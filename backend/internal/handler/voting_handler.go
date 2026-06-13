package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"gamer-club/backend/internal/middleware"
	"gamer-club/backend/internal/service"
)

// VotingHandler handles HTTP requests for the Voting/Ranking choice system
type VotingHandler struct {
	votingService service.VotingService
}

// NewVotingHandler creates a new VotingHandler
func NewVotingHandler(votingService service.VotingService) *VotingHandler {
	return &VotingHandler{votingService: votingService}
}

// GetActiveSession returns the current active session
func (h *VotingHandler) GetActiveSession(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		RespondError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	session, err := h.votingService.GetActiveSession()
	if err != nil {
		RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	if session == nil {
		RespondJSON(w, http.StatusOK, nil)
		return
	}

	RespondJSON(w, http.StatusOK, session)
}

// ListSessions returns all voting sessions in history
func (h *VotingHandler) ListSessions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		RespondError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	sessions, err := h.votingService.ListSessions()
	if err != nil {
		RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	RespondJSON(w, http.StatusOK, sessions)
}

// ListNominations returns nominated games, supporting custom session_id
func (h *VotingHandler) ListNominations(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		RespondError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	sessionIDStr := r.URL.Query().Get("session_id")
	var sessionID int
	if sessionIDStr != "" {
		sessionID, _ = strconv.Atoi(sessionIDStr)
	}

	nominations, err := h.votingService.GetNominations(sessionID)
	if err != nil {
		RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	RespondJSON(w, http.StatusOK, nominations)
}

// ListMyNominations returns nominations created by the logged-in user, supporting custom session_id
func (h *VotingHandler) ListMyNominations(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		RespondError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	user := middleware.GetUserFromContext(r.Context())
	if user == nil {
		RespondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	sessionIDStr := r.URL.Query().Get("session_id")
	var sessionID int
	if sessionIDStr != "" {
		sessionID, _ = strconv.Atoi(sessionIDStr)
	}

	nominations, err := h.votingService.GetUserNominations(sessionID, user.ID)
	if err != nil {
		RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	RespondJSON(w, http.StatusOK, nominations)
}

// NominateGame handles game nominations from a user
func (h *VotingHandler) NominateGame(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		RespondError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	user := middleware.GetUserFromContext(r.Context())
	if user == nil {
		RespondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req struct {
		GameID int `json:"game_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.GameID <= 0 {
		RespondError(w, http.StatusBadRequest, "invalid game ID")
		return
	}

	if err := h.votingService.NominateGame(user.ID, req.GameID); err != nil {
		RespondError(w, http.StatusBadRequest, err.Error())
		return
	}

	RespondJSON(w, http.StatusOK, map[string]string{
		"message": "game nominated successfully",
	})
}

// SubmitVote handles ranked-choice preferences submission
func (h *VotingHandler) SubmitVote(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		RespondError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	user := middleware.GetUserFromContext(r.Context())
	if user == nil {
		RespondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req struct {
		Preference []int `json:"preference"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		RespondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := h.votingService.SubmitVote(user.ID, req.Preference); err != nil {
		RespondError(w, http.StatusBadRequest, err.Error())
		return
	}

	RespondJSON(w, http.StatusOK, map[string]string{
		"message": "preferences recorded successfully",
	})
}

// GetMyVote returns the currently logged-in user's submitted vote, supporting custom session_id
func (h *VotingHandler) GetMyVote(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		RespondError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	user := middleware.GetUserFromContext(r.Context())
	if user == nil {
		RespondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	sessionIDStr := r.URL.Query().Get("session_id")
	var sessionID int
	if sessionIDStr != "" {
		sessionID, _ = strconv.Atoi(sessionIDStr)
	}

	vote, err := h.votingService.GetUserVote(sessionID, user.ID)
	if err != nil {
		RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	RespondJSON(w, http.StatusOK, vote)
}

// GetResults returns Borda Count results, supporting custom session_id
func (h *VotingHandler) GetResults(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		RespondError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	sessionIDStr := r.URL.Query().Get("session_id")
	var sessionID int
	if sessionIDStr != "" {
		sessionID, _ = strconv.Atoi(sessionIDStr)
	}

	results, err := h.votingService.CalculateResults(sessionID)
	if err != nil {
		RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	RespondJSON(w, http.StatusOK, results)
}

// CreateSession initiates a new voting cycle (Admin only)
func (h *VotingHandler) CreateSession(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		RespondError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req struct {
		Name           string `json:"name"`
		MaxNominations int    `json:"max_nominations"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		RespondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	session, err := h.votingService.StartSession(req.Name, req.MaxNominations)
	if err != nil {
		RespondError(w, http.StatusBadRequest, err.Error())
		return
	}

	RespondJSON(w, http.StatusCreated, session)
}

// UpdatePhase advances the voting event phase (Admin only)
func (h *VotingHandler) UpdatePhase(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		RespondError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req struct {
		SessionID int    `json:"session_id"`
		Phase     string `json:"phase"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		RespondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := h.votingService.TransitionPhase(req.SessionID, req.Phase); err != nil {
		RespondError(w, http.StatusBadRequest, err.Error())
		return
	}

	RespondJSON(w, http.StatusOK, map[string]string{
		"message": "session phase updated successfully",
	})
}

// CancelSession cancels the active voting event, clearing nominations and votes (Admin only)
func (h *VotingHandler) CancelSession(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		RespondError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	sessionIDStr := r.URL.Query().Get("session_id")
	var sessionID int
	if sessionIDStr != "" {
		sessionID, _ = strconv.Atoi(sessionIDStr)
	}

	if err := h.votingService.CancelSession(sessionID); err != nil {
		RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	RespondJSON(w, http.StatusOK, map[string]string{
		"message": "voting session canceled successfully",
	})
}
