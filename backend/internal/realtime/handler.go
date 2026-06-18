package realtime

import (
	"context"
	"net/http"

	"github.com/coder/websocket"
	"github.com/go-chi/chi/v5"
)

// Handler exposes the WebSocket upgrade endpoint. Auth and game-access checks
// are injected as plain functions so this package stays free of auth/game
// imports (game depends on realtime, not the other way around).
type Handler struct {
	hub            *Hub
	validateToken  func(token string) (userID string, err error)
	canAccessGame  func(userID, gameID string) bool
	originPatterns []string
}

func NewHandler(
	hub *Hub,
	validateToken func(token string) (string, error),
	canAccessGame func(userID, gameID string) bool,
	originPatterns []string,
) *Handler {
	return &Handler{
		hub:            hub,
		validateToken:  validateToken,
		canAccessGame:  canAccessGame,
		originPatterns: originPatterns,
	}
}

// HandleWS upgrades the connection. The browser cannot set an Authorization
// header on a WebSocket handshake, so the JWT arrives as the `token` query
// parameter and is validated here before the upgrade.
func (h *Handler) HandleWS(w http.ResponseWriter, r *http.Request) {
	gameID := chi.URLParam(r, "gameID")

	userID, err := h.validateToken(r.URL.Query().Get("token"))
	if err != nil || userID == "" {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	if !h.canAccessGame(userID, gameID) {
		http.Error(w, `{"error":"forbidden"}`, http.StatusForbidden)
		return
	}

	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		OriginPatterns: h.originPatterns,
	})
	if err != nil {
		return
	}
	defer conn.CloseNow()

	// Use a standalone context, not r.Context(): the global request timeout
	// middleware would otherwise close the socket after 30s. Disconnects are
	// still detected via CloseRead inside the hub's serve loop.
	h.hub.Serve(context.Background(), gameID, conn)
}
