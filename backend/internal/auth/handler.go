package auth

import (
	"encoding/json"
	"fmt"
	"net/http"
	"path/filepath"

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
	r.Post("/resend-verification", h.ResendVerification)
	r.Post("/refresh", h.RefreshToken)
	r.Post("/forgot-password", h.ForgotPassword)
	r.Post("/reset-password", h.ResetPassword)
	return r
}

func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	var input RegisterInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		respondJSON(w, 400, map[string]string{"error": "Invalid JSON"})
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
		respondJSON(w, 400, map[string]string{"error": "Invalid JSON"})
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
	fmt.Printf("Received verification request with token: %s\n", token)
	if token == "" {
		respondJSON(w, 400, map[string]string{"error": "Token is required"})
		return
	}

	if err := h.service.VerifyEmail(token); err != nil {
		respondJSON(w, 400, map[string]string{"error": err.Error()})
		return
	}

	respondJSON(w, 200, map[string]string{"message": "Email successfully verified!"})
}

func (h *Handler) ResendVerification(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Email string `json:"email"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		respondJSON(w, 400, map[string]string{"error": "Invalid JSON"})
		return
	}

	if err := h.service.ResendVerification(input.Email); err != nil {
		respondJSON(w, 400, map[string]string{"error": err.Error()})
		return
	}

	respondJSON(w, 200, map[string]string{"message": "Verification email sent!"})
}

func (h *Handler) RefreshToken(w http.ResponseWriter, r *http.Request) {
	var input struct {
		RefreshToken string `json:"refresh_token"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		respondJSON(w, 400, map[string]string{"error": "Invalid JSON"})
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
		respondJSON(w, 400, map[string]string{"error": "Invalid JSON"})
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
		respondJSON(w, 400, map[string]string{"error": "Invalid JSON"})
		return
	}

	if err := h.service.ResetPassword(input); err != nil {
		respondJSON(w, 400, map[string]string{"error": err.Error()})
		return
	}

	respondJSON(w, 200, map[string]string{"message": "Password successfully reset!"})
}

func (h *Handler) UpdateUsername(w http.ResponseWriter, r *http.Request) {
	userID := GetUserID(r)
	var input UpdateUsernameInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		respondJSON(w, 400, map[string]string{"error": "Invalid JSON"})
		return
	}

	if err := h.service.UpdateUsername(userID, input); err != nil {
		respondJSON(w, 400, map[string]string{"error": err.Error()})
		return
	}

	respondJSON(w, 200, map[string]string{"message": "Username updated"})
}

func (h *Handler) ChangePassword(w http.ResponseWriter, r *http.Request) {
	userID := GetUserID(r)
	var input ChangePasswordInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		respondJSON(w, 400, map[string]string{"error": "Invalid JSON"})
		return
	}

	if err := h.service.ChangePassword(userID, input); err != nil {
		respondJSON(w, 400, map[string]string{"error": err.Error()})
		return
	}

	respondJSON(w, 200, map[string]string{"message": "Password changed"})
}

func (h *Handler) UploadAvatar(w http.ResponseWriter, r *http.Request) {
	userID := GetUserID(r)

	r.Body = http.MaxBytesReader(w, r.Body, 2*1024*1024+1024)

	if err := r.ParseMultipartForm(2 * 1024 * 1024); err != nil {
		respondJSON(w, 400, map[string]string{"error": "File too large, maximum 2MB"})
		return
	}

	file, header, err := r.FormFile("avatar")
	if err != nil {
		respondJSON(w, 400, map[string]string{"error": "Avatar file is required"})
		return
	}
	defer file.Close()

	upload, err := h.service.UploadAvatar(userID, file, header)
	if err != nil {
		respondJSON(w, 400, map[string]string{"error": err.Error()})
		return
	}

	respondJSON(w, 200, map[string]interface{}{
		"message":   "Avatar uploaded",
		"avatar_id": upload.ID,
	})
}

func (h *Handler) DeleteAvatar(w http.ResponseWriter, r *http.Request) {
	userID := GetUserID(r)

	if err := h.service.DeleteAvatar(userID); err != nil {
		respondJSON(w, 400, map[string]string{"error": err.Error()})
		return
	}

	respondJSON(w, 200, map[string]string{"message": "Avatar removed"})
}

func (h *Handler) ServeUpload(w http.ResponseWriter, r *http.Request) {
	uploadID := chi.URLParam(r, "id")
	upload, err := h.service.GetUploadByID(uploadID)
	if err != nil {
		http.Error(w, "not found", 404)
		return
	}

	fullPath := filepath.Join(h.service.GetUploadDir(), upload.StorageKey)
	w.Header().Set("Content-Type", upload.MimeType)
	w.Header().Set("Cache-Control", "public, max-age=86400")
	http.ServeFile(w, r, fullPath)
}

func (h *Handler) GetStorageUsage(w http.ResponseWriter, r *http.Request) {
	userID := GetUserID(r)
	usage, err := h.service.GetStorageUsage(userID)
	if err != nil {
		respondJSON(w, 400, map[string]string{"error": err.Error()})
		return
	}
	respondJSON(w, 200, usage)
}

func (h *Handler) GetPlans(w http.ResponseWriter, r *http.Request) {
	plans, err := h.service.GetAllPlans()
	if err != nil {
		respondJSON(w, 500, map[string]string{"error": "Failed to load plans"})
		return
	}
	respondJSON(w, 200, plans)
}

func (h *Handler) GetProfile(w http.ResponseWriter, r *http.Request) {
	userID := GetUserID(r)
	profile, err := h.service.GetProfile(userID)
	if err != nil {
		respondJSON(w, 400, map[string]string{"error": err.Error()})
		return
	}
	respondJSON(w, 200, profile)
}

func respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}
