package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"gamer-club/backend/internal/middleware"
	"gamer-club/backend/internal/models"
	"gamer-club/backend/internal/service"
)

// AuthHandler handles HTTP requests related to users and authentication
type AuthHandler struct {
	userService service.UserService
}

// NewAuthHandler creates a new AuthHandler
func NewAuthHandler(userService service.UserService) *AuthHandler {
	return &AuthHandler{userService: userService}
}

// Login handles user login requests
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		RespondError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		RespondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	token, user, err := h.userService.Authenticate(req.Username, req.Password)
	if err != nil {
		RespondError(w, http.StatusUnauthorized, err.Error())
		return
	}

	RespondJSON(w, http.StatusOK, map[string]interface{}{
		"token": token,
		"user":  user,
	})
}

// Me returns the currently authenticated user details
func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		RespondError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	user := middleware.GetUserFromContext(r.Context())
	if user == nil {
		RespondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	RespondJSON(w, http.StatusOK, user)
}

// Refresh generates a new JWT token for an already authenticated user
func (h *AuthHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		RespondError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	user := middleware.GetUserFromContext(r.Context())
	if user == nil {
		RespondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	// Generate a fresh token for the current user
	token, err := h.userService.GenerateTokenForUser(user)
	if err != nil {
		RespondError(w, http.StatusInternalServerError, "failed to generate token: "+err.Error())
		return
	}

	RespondJSON(w, http.StatusOK, map[string]string{
		"token": token,
	})
}

// UpdateAccount allows the logged-in user to update their credentials
func (h *AuthHandler) UpdateAccount(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		RespondError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	user := middleware.GetUserFromContext(r.Context())
	if user == nil {
		RespondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req models.UserUpdateDTO
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		RespondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := h.userService.UpdateUser(user.ID, &req); err != nil {
		RespondError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Load updated user details
	updatedUser, err := h.userService.GetUserByID(user.ID)
	if err != nil {
		RespondError(w, http.StatusInternalServerError, "failed to fetch updated profile")
		return
	}

	// Generate a fresh token because the username might have changed
	token, err := h.userService.GenerateTokenForUser(updatedUser)
	if err != nil {
		RespondError(w, http.StatusInternalServerError, "failed to generate fresh token")
		return
	}

	RespondJSON(w, http.StatusOK, map[string]interface{}{
		"message": "account updated successfully",
		"token":   token,
		"user":    updatedUser,
	})
}

// ListUsers returns all users (Admin only)
func (h *AuthHandler) ListUsers(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		RespondError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	users, err := h.userService.GetAllUsers()
	if err != nil {
		RespondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	RespondJSON(w, http.StatusOK, users)
}

// CreateUser handles user creation by administrators
func (h *AuthHandler) CreateUser(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		RespondError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req models.CreateUserDTO
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		RespondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	newUser, err := h.userService.CreateUser(&req)
	if err != nil {
		RespondError(w, http.StatusBadRequest, err.Error())
		return
	}

	RespondJSON(w, http.StatusCreated, newUser)
}

// DeleteUser handles deleting a user account (Admin only)
func (h *AuthHandler) DeleteUser(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		RespondError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	idStr := r.PathValue("id")
	if idStr == "" {
		// Fallback for custom routing if path value is not set
		idStr = r.URL.Query().Get("id")
	}

	id, err := strconv.Atoi(idStr)
	if err != nil || id <= 0 {
		RespondError(w, http.StatusBadRequest, "invalid user ID")
		return
	}

	if err := h.userService.DeleteUser(id); err != nil {
		RespondError(w, http.StatusBadRequest, err.Error())
		return
	}

	RespondJSON(w, http.StatusOK, map[string]string{
		"message": "user deleted successfully",
	})
}
