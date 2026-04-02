package news

import (
	"encoding/json"
	"net/http"
	"strconv"

	"backend/internal/auth"
	"backend/internal/models"

	"github.com/go-chi/chi/v5"
)

type Handler struct {
	service    *Service
	jwtManager *auth.JWTManager
}

func NewHandler(service *Service, jwtManager *auth.JWTManager) *Handler {
	return &Handler{service: service, jwtManager: jwtManager}
}

func (h *Handler) Routes() chi.Router {
	r := chi.NewRouter()
	r.Get("/", h.List)
	r.Get("/{id}", h.GetByID)
	r.Group(func(r chi.Router) {
		r.Use(auth.JWTMiddleware(h.jwtManager))
		r.Use(auth.RequireAdmin)
		r.Post("/", h.Create)
	})
	return r
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))

	posts, total, err := h.service.ListPublished(limit, offset)
	if err != nil {
		respondJSON(w, 500, map[string]string{"error": err.Error()})
		return
	}

	result := make([]map[string]interface{}, 0, len(posts))
	for _, p := range posts {
		result = append(result, postToMap(p))
	}

	respondJSON(w, 200, map[string]interface{}{
		"posts": result,
		"total": total,
	})
}

func (h *Handler) GetByID(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	post, err := h.service.GetByID(id)
	if err != nil {
		respondJSON(w, 404, map[string]string{"error": err.Error()})
		return
	}

	respondJSON(w, 200, postToMap(*post))
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	userID := auth.GetUserID(r)

	if err := r.ParseMultipartForm(10 << 20); err != nil {
		respondJSON(w, 400, map[string]string{"error": "invalid form data"})
		return
	}

	title := r.FormValue("title")
	content := r.FormValue("content")

	file, header, err := r.FormFile("image")
	if err != nil && err != http.ErrMissingFile {
		respondJSON(w, 400, map[string]string{"error": "invalid image"})
		return
	}
	if file != nil {
		defer file.Close()
	}

	post, err := h.service.Create(userID, title, content, file, header)
	if err != nil {
		respondJSON(w, 400, map[string]string{"error": err.Error()})
		return
	}

	respondJSON(w, 201, postToMap(*post))
}

func postToMap(p models.NewsPost) map[string]interface{} {
	m := map[string]interface{}{
		"id":           p.ID,
		"title":        p.Title,
		"content":      p.Content,
		"image_id":     p.ImageID,
		"is_published": p.IsPublished,
		"published_at": p.PublishedAt,
		"created_at":   p.CreatedAt,
		"author": map[string]interface{}{
			"id":        p.Author.ID,
			"username":  p.Author.Username,
			"avatar_id": p.Author.AvatarID,
		},
	}
	return m
}

func respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}
