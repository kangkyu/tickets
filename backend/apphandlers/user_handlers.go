package apphandlers

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/gorilla/mux"
	"golang.org/x/crypto/bcrypt"

	"tickets-by-uma/middleware"
	"tickets-by-uma/models"
	"tickets-by-uma/repositories"
)

type UserHandlers struct {
	userRepo  repositories.UserRepository
	nwcRepo   repositories.NWCConnectionRepository
	logger    *slog.Logger
	jwtSecret string
}

func NewUserHandlers(userRepo repositories.UserRepository, nwcRepo repositories.NWCConnectionRepository, logger *slog.Logger, jwtSecret string) *UserHandlers {
	return &UserHandlers{
		userRepo:  userRepo,
		nwcRepo:   nwcRepo,
		logger:    logger,
		jwtSecret: jwtSecret,
	}
}

// HandleCreateUser creates a new user
func (h *UserHandlers) HandleCreateUser(w http.ResponseWriter, r *http.Request) {
	var req models.CreateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		middleware.WriteError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Validate request
	if err := h.validateCreateUserRequest(&req); err != nil {
		middleware.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}

	h.logger.Info("Creating new user", "email", req.Email)

	// Check if user already exists
	existingUser, err := h.userRepo.GetByEmail(req.Email)
	if err != nil {
		h.logger.Error("Failed to check existing user", "email", req.Email, "error", err)
		middleware.WriteError(w, http.StatusInternalServerError, "Failed to check existing user")
		return
	}

	if existingUser != nil {
		middleware.WriteError(w, http.StatusConflict, "User with this email already exists")
		return
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		h.logger.Error("Failed to hash password", "error", err)
		middleware.WriteError(w, http.StatusInternalServerError, "Failed to create user")
		return
	}

	// Create new user
	user := &models.User{
		Email:        req.Email,
		Name:         req.Name,
		PasswordHash: string(hashedPassword),
	}

	if err := h.userRepo.Create(user); err != nil {
		h.logger.Error("Failed to create user", "error", err)
		middleware.WriteError(w, http.StatusInternalServerError, "Failed to create user")
		return
	}

	h.logger.Info("User created successfully", "user_id", user.ID)

	middleware.WriteJSON(w, http.StatusCreated, models.SuccessResponse{
		Message: "User created successfully",
		Data:    user,
	})
}

// HandleLogin handles user login and returns JWT token
func (h *UserHandlers) HandleLogin(w http.ResponseWriter, r *http.Request) {
	var req models.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		middleware.WriteError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.Email == "" {
		middleware.WriteError(w, http.StatusBadRequest, "Email is required")
		return
	}

	if req.Password == "" {
		middleware.WriteError(w, http.StatusBadRequest, "Password is required")
		return
	}

	h.logger.Info("User login attempt", "email", req.Email)

	// Get user by email
	user, err := h.userRepo.GetByEmail(req.Email)
	if err != nil {
		h.logger.Error("Failed to fetch user", "email", req.Email, "error", err)
		middleware.WriteError(w, http.StatusInternalServerError, "Failed to fetch user")
		return
	}

	if user == nil {
		h.logger.Warn("Login failed - user not found", "email", req.Email)
		middleware.WriteError(w, http.StatusUnauthorized, "Invalid credentials")
		return
	}

	// Verify password
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		h.logger.Warn("Login failed - invalid password", "email", req.Email)
		middleware.WriteError(w, http.StatusUnauthorized, "Invalid credentials")
		return
	}

	// Generate JWT token
	token, err := middleware.GenerateToken(user, h.jwtSecret)
	if err != nil {
		h.logger.Error("Failed to generate token", "user_id", user.ID, "error", err)
		middleware.WriteError(w, http.StatusInternalServerError, "Failed to generate token")
		return
	}

	h.logger.Info("User logged in successfully", "user_id", user.ID)

	authResponse := models.AuthResponse{
		Token: token,
		User:  user,
	}

	middleware.WriteJSON(w, http.StatusOK, models.SuccessResponse{
		Message: "Login successful",
		Data:    authResponse,
	})
}

// HandleGetUser gets a specific user by ID
func (h *UserHandlers) HandleGetUser(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	userID, err := strconv.Atoi(vars["id"])
	if err != nil {
		middleware.WriteError(w, http.StatusBadRequest, "Invalid user ID")
		return
	}

	h.logger.Info("Fetching user", "user_id", userID)

	user, err := h.userRepo.GetByID(userID)
	if err != nil {
		h.logger.Error("Failed to fetch user", "user_id", userID, "error", err)
		middleware.WriteError(w, http.StatusInternalServerError, "Failed to fetch user")
		return
	}

	if user == nil {
		middleware.WriteError(w, http.StatusNotFound, "User not found")
		return
	}

	middleware.WriteJSON(w, http.StatusOK, models.SuccessResponse{
		Message: "User retrieved successfully",
		Data:    user,
	})
}

// HandleUpdateUser updates an existing user
func (h *UserHandlers) HandleUpdateUser(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	userID, err := strconv.Atoi(vars["id"])
	if err != nil {
		middleware.WriteError(w, http.StatusBadRequest, "Invalid user ID")
		return
	}

	var req models.CreateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		middleware.WriteError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Validate request
	if err := h.validateCreateUserRequest(&req); err != nil {
		middleware.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}

	h.logger.Info("Updating user", "user_id", userID)

	// Get existing user
	user, err := h.userRepo.GetByID(userID)
	if err != nil {
		h.logger.Error("Failed to fetch user for update", "user_id", userID, "error", err)
		middleware.WriteError(w, http.StatusInternalServerError, "Failed to fetch user")
		return
	}

	if user == nil {
		middleware.WriteError(w, http.StatusNotFound, "User not found")
		return
	}

	// Check if email is being changed and if it conflicts with existing user
	if req.Email != user.Email {
		existingUser, err := h.userRepo.GetByEmail(req.Email)
		if err != nil {
			h.logger.Error("Failed to check existing user", "email", req.Email, "error", err)
			middleware.WriteError(w, http.StatusInternalServerError, "Failed to check existing user")
			return
		}

		if existingUser != nil && existingUser.ID != userID {
			middleware.WriteError(w, http.StatusConflict, "User with this email already exists")
			return
		}
	}

	// Update user fields
	user.Email = req.Email
	user.Name = req.Name

	if err := h.userRepo.Update(user); err != nil {
		h.logger.Error("Failed to update user", "user_id", userID, "error", err)
		middleware.WriteError(w, http.StatusInternalServerError, "Failed to update user")
		return
	}

	h.logger.Info("User updated successfully", "user_id", userID)

	middleware.WriteJSON(w, http.StatusOK, models.SuccessResponse{
		Message: "User updated successfully",
		Data:    user,
	})
}

// HandleDeleteUser deletes a user
func (h *UserHandlers) HandleDeleteUser(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	userID, err := strconv.Atoi(vars["id"])
	if err != nil {
		middleware.WriteError(w, http.StatusBadRequest, "Invalid user ID")
		return
	}

	h.logger.Info("Deleting user", "user_id", userID)

	// Check if user exists
	user, err := h.userRepo.GetByID(userID)
	if err != nil {
		h.logger.Error("Failed to fetch user for deletion", "user_id", userID, "error", err)
		middleware.WriteError(w, http.StatusInternalServerError, "Failed to fetch user")
		return
	}

	if user == nil {
		middleware.WriteError(w, http.StatusNotFound, "User not found")
		return
	}

	if err := h.userRepo.Delete(userID); err != nil {
		h.logger.Error("Failed to delete user", "user_id", userID, "error", err)
		middleware.WriteError(w, http.StatusInternalServerError, "Failed to delete user")
		return
	}

	h.logger.Info("User deleted successfully", "user_id", userID)

	middleware.WriteJSON(w, http.StatusOK, models.SuccessResponse{
		Message: "User deleted successfully",
	})
}

// HandleGetCurrentUser gets the current authenticated user
func (h *UserHandlers) HandleGetCurrentUser(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUserFromContext(r.Context())
	if user == nil {
		middleware.WriteError(w, http.StatusUnauthorized, "User not authenticated")
		return
	}

	h.logger.Info("Fetching current user", "user_id", user.ID)

	// Get fresh user data from database
	freshUser, err := h.userRepo.GetByID(user.ID)
	if err != nil {
		h.logger.Error("Failed to fetch current user", "user_id", user.ID, "error", err)
		middleware.WriteError(w, http.StatusInternalServerError, "Failed to fetch user")
		return
	}

	if freshUser == nil {
		middleware.WriteError(w, http.StatusNotFound, "User not found")
		return
	}

	middleware.WriteJSON(w, http.StatusOK, models.SuccessResponse{
		Message: "Current user retrieved successfully",
		Data:    freshUser,
	})
}

// HandleStoreNWCConnection stores an NWC connection for the authenticated user
func (h *UserHandlers) HandleStoreNWCConnection(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUserFromContext(r.Context())
	if user == nil {
		middleware.WriteError(w, http.StatusUnauthorized, "User not authenticated")
		return
	}

	var req models.StoreNWCConnectionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		middleware.WriteError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if req.NWCConnectionURI == "" {
		middleware.WriteError(w, http.StatusBadRequest, "NWC connection URI is required")
		return
	}

	// Validate that it looks like an NWC URI
	if !strings.HasPrefix(req.NWCConnectionURI, "nostr+walletconnect://") {
		middleware.WriteError(w, http.StatusBadRequest, "Invalid NWC connection URI format")
		return
	}

	h.logger.Info("Storing NWC connection for user", "user_id", user.ID)

	if err := h.nwcRepo.Upsert(user.ID, req.NWCConnectionURI, req.ExpiresAt); err != nil {
		h.logger.Error("Failed to store NWC connection", "user_id", user.ID, "error", err)
		middleware.WriteError(w, http.StatusInternalServerError, "Failed to store NWC connection")
		return
	}

	h.logger.Info("NWC connection stored successfully", "user_id", user.ID)

	middleware.WriteJSON(w, http.StatusOK, models.SuccessResponse{
		Message: "NWC connection stored successfully",
	})
}

// HandleGetNWCConnection checks if the authenticated user has a stored NWC connection
func (h *UserHandlers) HandleGetNWCConnection(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUserFromContext(r.Context())
	if user == nil {
		middleware.WriteError(w, http.StatusUnauthorized, "User not authenticated")
		return
	}

	conn, err := h.nwcRepo.GetByUserID(user.ID)
	if err != nil {
		middleware.WriteError(w, http.StatusInternalServerError, "Failed to check wallet connection")
		return
	}

	if conn == nil {
		middleware.WriteJSON(w, http.StatusOK, models.SuccessResponse{
			Message: "No wallet connection found",
			Data: map[string]interface{}{
				"connected": false,
			},
		})
		return
	}

	middleware.WriteJSON(w, http.StatusOK, models.SuccessResponse{
		Message: "Wallet connection found",
		Data: map[string]interface{}{
			"connected":  true,
			"expires_at": conn.ExpiresAt,
			"created_at": conn.CreatedAt,
		},
	})
}

// validateCreateUserRequest validates the create user request
func (h *UserHandlers) validateCreateUserRequest(req *models.CreateUserRequest) error {
	if req.Email == "" {
		return fmt.Errorf("email is required")
	}

	if req.Name == "" {
		return fmt.Errorf("name is required")
	}

	if req.Password == "" {
		return fmt.Errorf("password is required")
	}

	// Basic email validation
	if len(req.Email) < 5 || !strings.Contains(req.Email, "@") {
		return fmt.Errorf("invalid email format")
	}

	// Basic name validation
	if len(req.Name) < 2 {
		return fmt.Errorf("name must be at least 2 characters long")
	}

	// Password minimum length
	if len(req.Password) < 8 {
		return fmt.Errorf("password must be at least 8 characters long")
	}

	return nil
}
