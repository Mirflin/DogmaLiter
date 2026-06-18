package main

import (
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"backend/internal/auth"
	"backend/internal/config"
	"backend/internal/game"
	"backend/internal/news"
	"backend/internal/payment"
	"backend/internal/realtime"
	"backend/pkg/database"
	"encoding/json"
)

func main() {
	cfg := config.Load()

	db := database.Connect(cfg.DatabaseURL)
	database.AutoMigrate(db)

	jwtManager := auth.NewJWTManager(cfg.JWTSecret)
	mailer := auth.NewMailer(cfg.SMTPHost, cfg.SMTPPort, cfg.SMTPUser, cfg.SMTPPassword, cfg.SMTPFrom)
	authRepo := auth.NewRepository(db)
	authService := auth.NewService(authRepo, jwtManager, mailer, cfg.FrontendURL, cfg.UploadDir)
	authHandler := auth.NewHandler(authService)

	hub := realtime.NewHub()

	gameRepo := game.NewRepository(db)
	gameService := game.NewService(gameRepo, cfg.UploadDir)
	gameHandler := game.NewHandler(gameService, hub)

	newsRepo := news.NewRepository(db)
	newsService := news.NewService(newsRepo, cfg.UploadDir)
	newsHandler := news.NewHandler(newsService, jwtManager)

	paymentHandler := payment.NewHandler(db, cfg, jwtManager)

	if err := os.MkdirAll(cfg.UploadDir, 0755); err != nil {
		log.Fatalf("Failed to create upload directory: %v", err)
	}

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(30 * time.Second))

	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")
			if origin != "" {
				for _, allowedOrigin := range cfg.FrontendOrigins {
					if allowedOrigin == origin {
						w.Header().Set("Access-Control-Allow-Origin", origin)
						w.Header().Set("Vary", "Origin")
						break
					}
				}
			}
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
			if r.Method == "OPTIONS" {
				w.WriteHeader(204)
				return
			}
			next.ServeHTTP(w, r)
		})
	})

	r.Get("/api/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok"}`))
	})

	r.Mount("/api/auth", authHandler.Routes())

	r.Get("/api/uploads/{id}", authHandler.ServeUpload)
	r.Get("/api/plans", authHandler.GetPlans)
	r.Mount("/api/news", newsHandler.Routes())
	r.Mount("/api/payment", paymentHandler.Routes())

	r.Get("/api/games/invite/{code}", func(w http.ResponseWriter, r *http.Request) {
		code := chi.URLParam(r, "code")
		game, err := gameRepo.GetGameByInviteCode(code)
		if err != nil {
			respondJSON(w, 404, map[string]string{"error": "invalid invite code"})
			return
		}
		if game.InviteCodeExpiresAt != nil && time.Now().After(*game.InviteCodeExpiresAt) {
			respondJSON(w, 410, map[string]string{"error": "invite code has expired"})
			return
		}
		respondJSON(w, 200, map[string]interface{}{
			"game_id":                game.ID,
			"title":                  game.Title,
			"system":                 game.System,
			"invite_code_expires_at": game.InviteCodeExpiresAt,
		})
	})

	// Realtime WebSocket endpoint. The JWT arrives as the `token` query param
	// (browsers can't set Authorization on a WS handshake), so this route lives
	// outside the bearer-auth group and validates the token itself.
	wsHandler := realtime.NewHandler(
		hub,
		func(token string) (string, error) {
			claims, err := jwtManager.ValidateToken(token)
			if err != nil {
				return "", err
			}
			return claims.UserID, nil
		},
		func(userID, gameID string) bool {
			g, err := gameRepo.GetGameByID(gameID)
			if err != nil {
				return false
			}
			if g.OwnerID == userID {
				return true
			}
			isMember, _ := gameRepo.IsMember(gameID, userID)
			return isMember
		},
		allowedOriginHosts(cfg.FrontendOrigins),
	)
	// Path kept separate from the "/api/games" mount to avoid a chi routing
	// conflict with that subtree's wildcard.
	r.Get("/api/ws/games/{gameID}", wsHandler.HandleWS)

	r.Group(func(r chi.Router) {
		r.Use(auth.JWTMiddleware(jwtManager))

		r.Get("/api/me", func(w http.ResponseWriter, r *http.Request) {
			userID := auth.GetUserID(r)
			user, err := authRepo.GetUserWithPlan(userID)
			if err != nil {
				http.Error(w, `{"error":"User not found"}`, 404)
				return
			}

			tokens, err := jwtManager.GenerateTokenPair(user.ID, user.Username, user.Role)
			if err != nil {
				respondJSON(w, 500, map[string]string{"error": "Failed to generate tokens"})
				return
			}

			respondJSON(w, 200, map[string]interface{}{
				"id":                   user.ID,
				"username":             user.Username,
				"email":                user.Email,
				"role":                 user.Role,
				"avatar_id":            user.AvatarID,
				"plan_id":              user.PlanID,
				"plan_name":            user.Plan.Name,
				"max_games_owned":      user.Plan.MaxGamesOwned,
				"max_players_per_game": user.Plan.MaxPlayersPerGame,
				"subscription_ends_at": user.SubscriptionEndsAt,
				"storage_frozen":       user.StorageFrozen,
				"is_verified":          user.IsVerified,
				"created_at":           user.CreatedAt.Format("02.01.2006"),
				"access_token":         tokens.AccessToken,
				"refresh_token":        tokens.RefreshToken,
			})
		})

		r.Put("/api/me/username", authHandler.UpdateUsername)
		r.Put("/api/me/password", authHandler.ChangePassword)
		r.Post("/api/me/avatar", authHandler.UploadAvatar)
		r.Delete("/api/me/avatar", authHandler.DeleteAvatar)
		r.Get("/api/me/storage", authHandler.GetStorageUsage)

		r.Mount("/api/games", gameHandler.Routes())

		r.Route("/api/admin", func(r chi.Router) {
			r.Use(auth.RequireAdmin)

			r.Get("/stats", func(w http.ResponseWriter, r *http.Request) {
				stats, err := authRepo.AdminStats()
				if err != nil {
					respondJSON(w, 500, map[string]string{"error": "Failed to load stats"})
					return
				}
				respondJSON(w, 200, stats)
			})

			r.Get("/users", func(w http.ResponseWriter, r *http.Request) {
				users, err := authRepo.ListRecentUsers(100)
				if err != nil {
					respondJSON(w, 500, map[string]string{"error": "Failed to load users"})
					return
				}
				result := make([]map[string]interface{}, 0, len(users))
				for _, u := range users {
					result = append(result, map[string]interface{}{
						"id":                   u.ID,
						"username":             u.Username,
						"email":                u.Email,
						"role":                 u.Role,
						"is_verified":          u.IsVerified,
						"plan_id":              u.PlanID,
						"plan_name":            u.Plan.Name,
						"subscription_ends_at": u.SubscriptionEndsAt,
						"created_at":           u.CreatedAt.Format("02.01.2006"),
					})
				}
				respondJSON(w, 200, map[string]interface{}{"users": result})
			})

			r.Get("/plans", func(w http.ResponseWriter, r *http.Request) {
				plans, err := authRepo.ListPlans()
				if err != nil {
					respondJSON(w, 500, map[string]string{"error": "Failed to load plans"})
					return
				}
				result := make([]map[string]interface{}, 0, len(plans))
				for _, p := range plans {
					result = append(result, map[string]interface{}{
						"id":            p.ID,
						"name":          p.Name,
						"price_monthly": p.PriceMonthly,
					})
				}
				respondJSON(w, 200, map[string]interface{}{"plans": result})
			})

			r.Patch("/users/{userID}", func(w http.ResponseWriter, r *http.Request) {
				userID := chi.URLParam(r, "userID")

				var req struct {
					Role               *string `json:"role"`
					IsVerified         *bool   `json:"is_verified"`
					PlanID             *string `json:"plan_id"`
					SubscriptionEndsAt *string `json:"subscription_ends_at"`
				}
				if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
					respondJSON(w, 400, map[string]string{"error": "Invalid request body"})
					return
				}

				updates := map[string]interface{}{}
				if req.Role != nil {
					if *req.Role != "user" && *req.Role != "admin" {
						respondJSON(w, 400, map[string]string{"error": "Role must be 'user' or 'admin'"})
						return
					}
					updates["role"] = *req.Role
				}
				if req.IsVerified != nil {
					updates["is_verified"] = *req.IsVerified
				}
				if req.PlanID != nil {
					exists, err := authRepo.PlanExists(*req.PlanID)
					if err != nil {
						respondJSON(w, 500, map[string]string{"error": "Failed to verify plan"})
						return
					}
					if !exists {
						respondJSON(w, 400, map[string]string{"error": "Unknown plan"})
						return
					}
					updates["plan_id"] = *req.PlanID
				}
				if req.SubscriptionEndsAt != nil {
					value := strings.TrimSpace(*req.SubscriptionEndsAt)
					if value == "" {
						updates["subscription_ends_at"] = nil
					} else {
						parsed, err := time.Parse("2006-01-02", value)
						if err != nil {
							parsed, err = time.Parse(time.RFC3339, value)
						}
						if err != nil {
							respondJSON(w, 400, map[string]string{"error": "Invalid subscription end date"})
							return
						}
						updates["subscription_ends_at"] = parsed
					}
				}
				if len(updates) == 0 {
					respondJSON(w, 400, map[string]string{"error": "No changes were provided"})
					return
				}

				if err := authRepo.AdminUpdateUser(userID, updates); err != nil {
					respondJSON(w, 500, map[string]string{"error": "Failed to update user"})
					return
				}
				respondJSON(w, 200, map[string]interface{}{"success": true})
			})

			r.Delete("/users/{userID}", func(w http.ResponseWriter, r *http.Request) {
				userID := chi.URLParam(r, "userID")
				adminID := auth.GetUserID(r)
				if userID == adminID {
					respondJSON(w, 400, map[string]string{"error": "You cannot delete your own account"})
					return
				}

				if err := authRepo.DeleteUser(userID, adminID); err != nil {
					respondJSON(w, 500, map[string]string{"error": "Failed to delete user"})
					return
				}
				respondJSON(w, 200, map[string]interface{}{"success": true})
			})

			r.Get("/games", func(w http.ResponseWriter, r *http.Request) {
				games, err := gameRepo.ListAllGames()
				if err != nil {
					respondJSON(w, 500, map[string]string{"error": "Failed to load games"})
					return
				}
				result := make([]map[string]interface{}, 0, len(games))
				for _, g := range games {
					result = append(result, map[string]interface{}{
						"id":          g.ID,
						"title":       g.Title,
						"system":      g.System,
						"owner":       g.Owner.Username,
						"owner_id":    g.OwnerID,
						"players":     len(g.Members),
						"max_players": g.MaxPlayers,
						"created_at":  g.CreatedAt.Format("02.01.2006"),
					})
				}
				respondJSON(w, 200, map[string]interface{}{"games": result})
			})

			r.Patch("/games/{gameID}", func(w http.ResponseWriter, r *http.Request) {
				gameID := chi.URLParam(r, "gameID")
				var req struct {
					Title      *string `json:"title"`
					MaxPlayers *int    `json:"max_players"`
				}
				if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
					respondJSON(w, 400, map[string]string{"error": "Invalid request body"})
					return
				}
				updates := map[string]interface{}{}
				if req.Title != nil {
					title := strings.TrimSpace(*req.Title)
					if title == "" {
						respondJSON(w, 400, map[string]string{"error": "Title cannot be empty"})
						return
					}
					updates["title"] = title
				}
				if req.MaxPlayers != nil {
					if *req.MaxPlayers < 1 || *req.MaxPlayers > 100 {
						respondJSON(w, 400, map[string]string{"error": "Max players must be between 1 and 100"})
						return
					}
					updates["max_players"] = *req.MaxPlayers
				}
				if len(updates) == 0 {
					respondJSON(w, 400, map[string]string{"error": "No changes were provided"})
					return
				}
				if err := gameRepo.AdminUpdateGame(gameID, updates); err != nil {
					respondJSON(w, 500, map[string]string{"error": "Failed to update game"})
					return
				}
				respondJSON(w, 200, map[string]interface{}{"success": true})
			})

			r.Delete("/games/{gameID}", func(w http.ResponseWriter, r *http.Request) {
				gameID := chi.URLParam(r, "gameID")
				if err := gameRepo.DeleteGame(gameID); err != nil {
					respondJSON(w, 500, map[string]string{"error": "Failed to delete game"})
					return
				}
				respondJSON(w, 200, map[string]interface{}{"success": true})
			})
		})

	})

	log.Printf("DogmaLiter: http://0.0.0.0:%s", cfg.Port)
	log.Fatal(http.ListenAndServe("0.0.0.0:"+cfg.Port, r))
}

func respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

// allowedOriginHosts converts configured frontend origins (full URLs) into the
// host patterns coder/websocket checks against the WS handshake Origin header.
func allowedOriginHosts(origins []string) []string {
	hosts := make([]string, 0, len(origins))
	for _, origin := range origins {
		if parsed, err := url.Parse(strings.TrimSpace(origin)); err == nil && parsed.Host != "" {
			hosts = append(hosts, parsed.Host)
		}
	}
	return hosts
}
