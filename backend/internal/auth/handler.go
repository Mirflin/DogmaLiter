package auth

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) Routes() chi.Router {
	r := chi.NewRouter()
	r.Post("/register", h.Register)
	r.Post("/login", h.Login)
	r.Get("/verify", h.VerifyEmail)
	r.Post("/refresh", h.RefreshToken)
	r.Post("/forgot-password", h.ForgotPassword)
	r.Post("/reset-password", h.ResetPassword)
	return r
}

func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	var input RegisterInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		respondJSON(w, 400, map[string]string{"error": "invalid JSON"})
		return
	}

	result, err := h.service.Register(input)
	if err != nil {
		respondJSON(w, 400, map[string]string{"error": err.Error()})
		return
	}

	respondJSON(w, 201, result)
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var input LoginInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		respondJSON(w, 400, map[string]string{"error": "invalid JSON"})
		return
	}

	result, err := h.service.Login(input)
	if err != nil {
		respondJSON(w, 401, map[string]string{"error": err.Error()})
		return
	}

	respondJSON(w, 200, result)
}

func (h *Handler) VerifyEmail(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")
	if token == "" {
		respondJSON(w, 400, map[string]string{"error": "token is required"})
		return
	}

	if err := h.service.VerifyEmail(token); err != nil {
		respondJSON(w, 400, map[string]string{"error": err.Error()})
		return
	}

	respondJSON(w, 200, map[string]string{"message": "Email successfully verified!"})
}

func (h *Handler) RefreshToken(w http.ResponseWriter, r *http.Request) {
	var input struct {
		RefreshToken string `json:"refresh_token"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		respondJSON(w, 400, map[string]string{"error": "invalid JSON"})
		return
	}

	tokens, err := h.service.RefreshToken(input.RefreshToken)
	if err != nil {
		respondJSON(w, 401, map[string]string{"error": err.Error()})
		return
	}

	respondJSON(w, 200, tokens)
}

func (h *Handler) ForgotPassword(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Email string `json:"email"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		respondJSON(w, 400, map[string]string{"error": "invalid JSON"})
		return
	}

	h.service.RequestPasswordReset(input.Email)

	respondJSON(w, 200, map[string]string{
		"message": "If the email is registered, you will receive a password reset link.",
	})
}

func (h *Handler) ResetPassword(w http.ResponseWriter, r *http.Request) {
	var input ResetPasswordInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		respondJSON(w, 400, map[string]string{"error": "invalid JSON"})
		return
	}

	if err := h.service.ResetPassword(input); err != nil {
		respondJSON(w, 400, map[string]string{"error": err.Error()})
		return
	}

	respondJSON(w, 200, map[string]string{"message": "Password successfully reset!"})
}

func respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}
