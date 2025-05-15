package httphandler

import (
	"ecomGateway/internal/processor"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi"
)

type HTTPHandler struct {
	processor processor.Processor
	logger    *slog.Logger
}

func NewHTTPHandler(processor processor.Processor, logger *slog.Logger) *HTTPHandler {
	return &HTTPHandler{
		processor: processor,
		logger:    logger,
	}
}

func (h *HTTPHandler) RegisterRoutes(router *chi.Mux) {
	// Публичные роуты
	router.Post("/register", h.register)
	router.Post("/login", h.login)

}

type registerRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Login    string `json:"login"`
}

type registerResponse struct {
	UserID  int64  `json:"user_id"`
	Message string `json:"message"`
}

type loginRequest struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

type loginResponse struct {
	Token   string `json:"token"`
	Message string `json:"message"`
}

type errorResponse struct {
	Error string `json:"error"`
}

func (h *HTTPHandler) register(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		h.logger.Error("Failed to read request body", slog.String("error", err.Error()))
		h.respondWithError(w, http.StatusBadRequest, "Failed to read request body")
		return
	}
	defer r.Body.Close()

	var req registerRequest
	if err := json.Unmarshal(body, &req); err != nil {
		h.logger.Error("Failed to unmarshal request JSON", slog.String("error", err.Error()), slog.String("body", string(body)))
		h.respondWithError(w, http.StatusBadRequest, "Invalid JSON payload")
		return
	}

	if req.Email == "" || req.Password == "" || req.Login == "" {
		h.logger.Warn("Missing required fields for registration",
			slog.String("email", req.Email),
			slog.String("login", req.Login),
		)
		h.respondWithError(w, http.StatusBadRequest, "Email, password, and login are required")
		return
	}

	userID, err := h.processor.RegisterUser(r.Context(), req.Email, req.Password, req.Login)
	if err != nil {
		h.logger.Error("Processor failed to register user", slog.String("error", err.Error()))
		h.respondWithError(w, http.StatusInternalServerError, "Failed to register user")
		return
	}

	h.logger.Info("User registered successfully", slog.Int64("userID", userID))
	h.respondWithJSON(w, http.StatusCreated, registerResponse{
		UserID:  userID,
		Message: "User registered successfully",
	})
}

func (h *HTTPHandler) login(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		h.logger.Error("Failed to read request body for login", slog.String("error", err.Error()))
		h.respondWithError(w, http.StatusBadRequest, "Failed to read request body")
		return
	}
	defer r.Body.Close()

	var req loginRequest
	if err := json.Unmarshal(body, &req); err != nil {
		h.logger.Error("Failed to unmarshal login request JSON", slog.String("error", err.Error()), slog.String("body", string(body)))
		h.respondWithError(w, http.StatusBadRequest, "Invalid JSON payload")
		return
	}

	if req.Login == "" || req.Password == "" {
		h.logger.Warn("Missing required fields for login", slog.String("login", req.Login))
		h.respondWithError(w, http.StatusBadRequest, "Login and password are required")
		return
	}

	token, err := h.processor.LoginUser(r.Context(), req.Login, req.Password)
	if err != nil {
		h.logger.Error("Processor failed to login user", slog.String("login", req.Login), slog.String("error", err.Error()))
		h.respondWithError(w, http.StatusUnauthorized, "Login failed. Check credentials.")
		return
	}

	h.logger.Info("User logged in successfully", slog.String("login", req.Login))
	h.respondWithJSON(w, http.StatusOK, loginResponse{
		Token:   token,
		Message: "Login successful",
	})
}

func (h *HTTPHandler) respondWithJSON(w http.ResponseWriter, code int, payload interface{}) {
	response, err := json.Marshal(payload)
	if err != nil {
		h.logger.Error("Failed to marshal JSON response", slog.String("error", err.Error()))
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error":"failed to marshal response"}`))
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	w.Write(response)
}

func (h *HTTPHandler) respondWithError(w http.ResponseWriter, code int, message string) {
	h.respondWithJSON(w, code, errorResponse{Error: message})
}
