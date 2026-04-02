package game

import (
	"encoding/json"
	"fmt"
	"net/http"

	"backend/internal/auth"

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
	r.Get("/", h.GetMyGames)
	r.Post("/", h.CreateGame)
	r.Post("/join", h.JoinByCode)
	r.Post("/{gameID}/regenerate-code", h.RegenerateInviteCode)
	r.Get("/{gameID}/invite-code", h.GetInviteCode)
	r.Get("/{gameID}", h.GetGame)
	r.Put("/{gameID}", h.UpdateGame)
	r.Post("/{gameID}/leave", h.LeaveGame)
	r.Delete("/{gameID}", h.DeleteGame)
	r.Post("/{gameID}/cover", h.UploadCoverImage)
	return r
}

func (h *Handler) GetMyGames(w http.ResponseWriter, r *http.Request) {
	userID := auth.GetUserID(r)

	games, err := h.service.GetUserGames(userID)
	if err != nil {
		respondJSON(w, 500, map[string]string{"error": "failed to load games"})
		return
	}

	result := make([]map[string]interface{}, 0, len(games))
	for _, g := range games {
		members := make([]map[string]interface{}, 0, len(g.Members))
		for _, m := range g.Members {
			member := map[string]interface{}{
				"user_id":   m.UserID,
				"role":      m.Role,
				"username":  m.User.Username,
				"avatar_id": m.User.AvatarID,
			}
			members = append(members, member)
		}

		var coverImageID *string
		if g.CoverImageID != nil {
			coverImageID = g.CoverImageID
		}

		result = append(result, map[string]interface{}{
			"id":             g.ID,
			"title":          g.Title,
			"description":    g.Description,
			"system":         g.System,
			"cover_image_id": coverImageID,
			"owner_id":       g.OwnerID,
			"max_players":    g.MaxPlayers,
			"members":        members,
			"updated_at":     g.UpdatedAt,
		})
	}

	respondJSON(w, 200, result)
}

func (h *Handler) CreateGame(w http.ResponseWriter, r *http.Request) {
	userID := auth.GetUserID(r)

	var req struct {
		Title             string `json:"title"`
		Description       string `json:"description"`
		System            string `json:"system"`
		MaxPlayers        int    `json:"max_players"`
		ShowStandardAttrs *bool  `json:"show_standard_attrs"`
		EnableChat        *bool  `json:"enable_chat"`
		EnableItemTrading *bool  `json:"enable_item_trading"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondJSON(w, 400, map[string]string{"error": "invalid request body"})
		return
	}

	showStd := true
	if req.ShowStandardAttrs != nil {
		showStd = *req.ShowStandardAttrs
	}
	chat := true
	if req.EnableChat != nil {
		chat = *req.EnableChat
	}
	trading := true
	if req.EnableItemTrading != nil {
		trading = *req.EnableItemTrading
	}
	if req.System == "" {
		req.System = "custom"
	}

	game, err := h.service.CreateGame(userID, req.Title, req.Description, req.System, req.MaxPlayers, showStd, chat, trading)
	if err != nil {
		respondJSON(w, 400, map[string]string{"error": err.Error()})
		return
	}

	respondJSON(w, 201, map[string]interface{}{
		"id":                     game.ID,
		"title":                  game.Title,
		"invite_code":            game.InviteCode,
		"invite_code_expires_at": game.InviteCodeExpiresAt,
	})
}

func (h *Handler) JoinByCode(w http.ResponseWriter, r *http.Request) {
	userID := auth.GetUserID(r)

	var req struct {
		Code string `json:"code"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondJSON(w, 400, map[string]string{"error": "invalid request body"})
		return
	}

	game, err := h.service.JoinByCode(userID, req.Code)
	if err != nil {
		respondJSON(w, 400, map[string]string{"error": err.Error()})
		return
	}

	respondJSON(w, 200, map[string]interface{}{
		"game_id": game.ID,
		"title":   game.Title,
	})
}

func (h *Handler) RegenerateInviteCode(w http.ResponseWriter, r *http.Request) {
	userID := auth.GetUserID(r)
	gameID := chi.URLParam(r, "gameID")

	newCode, expiresAt, err := h.service.RegenerateInviteCode(userID, gameID)
	if err != nil {
		respondJSON(w, 400, map[string]string{"error": err.Error()})
		return
	}

	respondJSON(w, 200, map[string]interface{}{
		"invite_code":            newCode,
		"invite_code_expires_at": expiresAt,
	})
}

func (h *Handler) GetInviteCode(w http.ResponseWriter, r *http.Request) {
	userID := auth.GetUserID(r)
	gameID := chi.URLParam(r, "gameID")

	game, err := h.service.repo.GetGameByID(gameID)
	if err != nil {
		respondJSON(w, 404, map[string]string{"error": "game not found"})
		return
	}
	if game.OwnerID != userID {
		respondJSON(w, 403, map[string]string{"error": "only the game owner can view the invite code"})
		return
	}

	respondJSON(w, 200, map[string]interface{}{
		"invite_code":            game.InviteCode,
		"invite_code_expires_at": game.InviteCodeExpiresAt,
	})
}

func (h *Handler) GetGame(w http.ResponseWriter, r *http.Request) {
	userID := auth.GetUserID(r)
	gameID := chi.URLParam(r, "gameID")

	game, err := h.service.repo.GetGameByID(gameID)
	if err != nil {
		respondJSON(w, 404, map[string]string{"error": "game not found"})
		return
	}

	isMember, _ := h.service.repo.IsMember(gameID, userID)
	if game.OwnerID != userID && !isMember {
		respondJSON(w, 403, map[string]string{"error": "not a member of this game"})
		return
	}

	members := make([]map[string]interface{}, 0, len(game.Members))
	for _, m := range game.Members {
		members = append(members, map[string]interface{}{
			"user_id":   m.UserID,
			"role":      m.Role,
			"username":  m.User.Username,
			"avatar_id": m.User.AvatarID,
			"joined_at": m.JoinedAt,
		})
	}

	owner := map[string]interface{}{
		"id":         game.Owner.ID,
		"username":   game.Owner.Username,
		"avatar_id":  game.Owner.AvatarID,
		"plan_name":  game.Owner.Plan.Name,
		"created_at": game.Owner.CreatedAt,
	}

	result := map[string]interface{}{
		"id":                  game.ID,
		"title":               game.Title,
		"description":         game.Description,
		"system":              game.System,
		"max_players":         game.MaxPlayers,
		"owner_id":            game.OwnerID,
		"cover_image_id":      game.CoverImageID,
		"show_standard_attrs": game.ShowStandardAttrs,
		"enable_chat":         game.EnableChat,
		"enable_item_trading": game.EnableItemTrading,
		"invite_code":         game.InviteCode,
		"invite_code_expires_at": game.InviteCodeExpiresAt,
		"created_at":          game.CreatedAt,
		"updated_at":          game.UpdatedAt,
		"members":             members,
		"owner":               owner,
	}

	if game.OwnerID != userID {
		delete(result, "invite_code")
		delete(result, "invite_code_expires_at")
	}

	respondJSON(w, 200, result)
}

func (h *Handler) UpdateGame(w http.ResponseWriter, r *http.Request) {
	userID := auth.GetUserID(r)
	gameID := chi.URLParam(r, "gameID")

	game, err := h.service.repo.GetGameByID(gameID)
	if err != nil {
		respondJSON(w, 404, map[string]string{"error": "game not found"})
		return
	}
	if game.OwnerID != userID {
		respondJSON(w, 403, map[string]string{"error": "only the game owner can update settings"})
		return
	}

	var req struct {
		Title             *string `json:"title"`
		Description       *string `json:"description"`
		MaxPlayers        *int    `json:"max_players"`
		ShowStandardAttrs *bool   `json:"show_standard_attrs"`
		EnableChat        *bool   `json:"enable_chat"`
		EnableItemTrading *bool   `json:"enable_item_trading"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondJSON(w, 400, map[string]string{"error": "invalid request body"})
		return
	}

	if req.Title != nil {
		if len(*req.Title) < 1 || len(*req.Title) > 200 {
			respondJSON(w, 400, map[string]string{"error": "title must be 1-200 characters"})
			return
		}
		game.Title = *req.Title
	}
	if req.Description != nil {
		game.Description = *req.Description
	}
	if req.MaxPlayers != nil && *req.MaxPlayers >= 1 {
		plan, err := h.service.repo.GetUserPlan(userID)
		if err == nil && plan.MaxPlayersPerGame != -1 && *req.MaxPlayers > plan.MaxPlayersPerGame {
			respondJSON(w, 400, map[string]string{"error": fmt.Sprintf("your plan allows max %d players", plan.MaxPlayersPerGame)})
			return
		}
		game.MaxPlayers = *req.MaxPlayers
	}
	if req.ShowStandardAttrs != nil {
		game.ShowStandardAttrs = *req.ShowStandardAttrs
	}
	if req.EnableChat != nil {
		game.EnableChat = *req.EnableChat
	}
	if req.EnableItemTrading != nil {
		game.EnableItemTrading = *req.EnableItemTrading
	}

	if err := h.service.repo.UpdateGame(game); err != nil {
		respondJSON(w, 500, map[string]string{"error": "failed to update game"})
		return
	}

	respondJSON(w, 200, map[string]interface{}{
		"id":                  game.ID,
		"title":               game.Title,
		"description":         game.Description,
		"max_players":         game.MaxPlayers,
		"show_standard_attrs": game.ShowStandardAttrs,
		"enable_chat":         game.EnableChat,
		"enable_item_trading": game.EnableItemTrading,
	})
}

func (h *Handler) LeaveGame(w http.ResponseWriter, r *http.Request) {
	userID := auth.GetUserID(r)
	gameID := chi.URLParam(r, "gameID")

	if err := h.service.LeaveGame(userID, gameID); err != nil {
		respondJSON(w, 400, map[string]string{"error": err.Error()})
		return
	}
	respondJSON(w, 200, map[string]string{"message": "left game"})
}

func (h *Handler) DeleteGame(w http.ResponseWriter, r *http.Request) {
	userID := auth.GetUserID(r)
	gameID := chi.URLParam(r, "gameID")

	if err := h.service.DeleteGame(userID, gameID); err != nil {
		respondJSON(w, 400, map[string]string{"error": err.Error()})
		return
	}
	respondJSON(w, 200, map[string]string{"message": "game deleted"})
}

func (h *Handler) UploadCoverImage(w http.ResponseWriter, r *http.Request) {
	userID := auth.GetUserID(r)
	gameID := chi.URLParam(r, "gameID")

	r.Body = http.MaxBytesReader(w, r.Body, 5*1024*1024+512)
	if err := r.ParseMultipartForm(5 << 20); err != nil {
		respondJSON(w, 400, map[string]string{"error": "file too large (max 5MB)"})
		return
	}

	file, header, err := r.FormFile("cover")
	if err != nil {
		respondJSON(w, 400, map[string]string{"error": "cover file is required"})
		return
	}
	defer file.Close()

	uploadID, err := h.service.UploadCoverImage(userID, gameID, file, header)
	if err != nil {
		respondJSON(w, 400, map[string]string{"error": err.Error()})
		return
	}

	respondJSON(w, 200, map[string]interface{}{"cover_image_id": uploadID})
}

func respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}
