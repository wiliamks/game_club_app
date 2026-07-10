package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"gamer-club/backend/internal/middleware"
	"gamer-club/backend/internal/models"
	"gamer-club/backend/internal/service"
)

// GameHandler handles HTTP requests related to games, reviews, and IGDB integration
type GameHandler struct {
	gameService service.GameService
	igdbClient  service.IGDBClient
}

// NewGameHandler creates a new GameHandler
func NewGameHandler(gameService service.GameService, igdbClient service.IGDBClient) *GameHandler {
	return &GameHandler{
		gameService: gameService,
		igdbClient:  igdbClient,
	}
}

// ListGames returns all games stored in the local system
func (h *GameHandler) ListGames(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		RespondError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	games, err := h.gameService.ListGames()
	if err != nil {
		RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	RespondJSON(w, http.StatusOK, games)
}

// GetGameDetails returns full details for a game, importing it if not found locally
func (h *GameHandler) GetGameDetails(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		RespondError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	idStr := r.PathValue("id")
	if idStr == "" {
		idStr = r.URL.Query().Get("id")
	}

	id, err := strconv.Atoi(idStr)
	if err != nil || id <= 0 {
		RespondError(w, http.StatusBadRequest, "invalid game ID")
		return
	}

	user := middleware.GetUserFromContext(r.Context())
	userID := 0
	if user != nil {
		userID = user.ID
	}

	details, err := h.gameService.GetGameDetails(id, userID)
	if err != nil {
		RespondError(w, http.StatusNotFound, err.Error())
		return
	}

	RespondJSON(w, http.StatusOK, details)
}

// SubmitReview handles submitting or updating a user's game review
func (h *GameHandler) SubmitReview(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		RespondError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	idStr := r.PathValue("id")
	if idStr == "" {
		idStr = r.URL.Query().Get("id")
	}

	id, err := strconv.Atoi(idStr)
	if err != nil || id <= 0 {
		RespondError(w, http.StatusBadRequest, "invalid game ID")
		return
	}

	user := middleware.GetUserFromContext(r.Context())
	if user == nil {
		RespondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req models.Review
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		RespondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := h.gameService.SubmitReview(user.ID, user.Username, id, &req); err != nil {
		RespondError(w, http.StatusBadRequest, err.Error())
		return
	}

	RespondJSON(w, http.StatusOK, map[string]string{
		"message": "review submitted successfully",
	})
}

// SetActiveGame sets a specific game as the Active Game (Admin only)
func (h *GameHandler) SetActiveGame(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		RespondError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req struct {
		GameID int `json:"game_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.GameID <= 0 {
		RespondError(w, http.StatusBadRequest, "invalid or missing game ID")
		return
	}

	if err := h.gameService.SetActiveGame(req.GameID); err != nil {
		RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	RespondJSON(w, http.StatusOK, map[string]string{
		"message": "active game set successfully",
	})
}

// DeactivateActiveGame deactivates the current Active Game, if any (Admin only)
func (h *GameHandler) DeactivateActiveGame(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		RespondError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	if err := h.gameService.DeactivateActiveGame(); err != nil {
		RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	RespondJSON(w, http.StatusOK, map[string]string{
		"message": "active game deactivated successfully",
	})
}

// SearchIGDB proxies query requests to the IGDB Client
func (h *GameHandler) SearchIGDB(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		RespondError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	query := r.URL.Query().Get("q")
	if query == "" {
		RespondJSON(w, http.StatusOK, []interface{}{})
		return
	}

	results, err := h.igdbClient.SearchGames(query)
	if err != nil {
		RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	RespondJSON(w, http.StatusOK, results)
}

// DeleteReview handles deleting a user's game review (or any review if admin)
func (h *GameHandler) DeleteReview(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		RespondError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	idStr := r.PathValue("id")
	if idStr == "" {
		idStr = r.URL.Query().Get("id")
	}

	gameID, err := strconv.Atoi(idStr)
	if err != nil || gameID <= 0 {
		RespondError(w, http.StatusBadRequest, "invalid game ID")
		return
	}

	user := middleware.GetUserFromContext(r.Context())
	if user == nil {
		RespondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	targetUserID := user.ID
	if user.Role == "admin" {
		qUserID := r.URL.Query().Get("user_id")
		if qUserID != "" {
			if id, err := strconv.Atoi(qUserID); err == nil && id > 0 {
				targetUserID = id
			}
		}
	}

	if err := h.gameService.DeleteReview(targetUserID, gameID); err != nil {
		RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	RespondJSON(w, http.StatusOK, map[string]string{
		"message": "review deleted successfully",
	})
}

// ToggleReaction handles toggling an emoji reaction on a review
func (h *GameHandler) ToggleReaction(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		RespondError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	idStr := r.PathValue("id")
	reviewID, err := strconv.Atoi(idStr)
	if err != nil || reviewID <= 0 {
		RespondError(w, http.StatusBadRequest, "invalid review ID")
		return
	}

	user := middleware.GetUserFromContext(r.Context())
	if user == nil {
		RespondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req struct {
		Emoji string `json:"emoji"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Emoji == "" {
		RespondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := h.gameService.ToggleReaction(reviewID, user.ID, req.Emoji); err != nil {
		RespondError(w, http.StatusBadRequest, err.Error())
		return
	}

	RespondJSON(w, http.StatusOK, map[string]string{
		"message": "reaction toggled successfully",
	})
}
