package main

import (
	"log"
	"net/http"
	"os"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"backend/internal/auth"
	"backend/internal/config"
	"backend/internal/game"
	"backend/internal/news"
	"backend/internal/payment"
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

	gameRepo := game.NewRepository(db)
	gameService := game.NewService(gameRepo, cfg.UploadDir)
	gameHandler := game.NewHandler(gameService)

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
			w.Header().Set("Access-Control-Allow-Origin", cfg.FrontendURL)
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
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
				"id":                    user.ID,
				"username":              user.Username,
				"email":                 user.Email,
				"role":                  user.Role,
				"avatar_id":             user.AvatarID,
				"plan_id":               user.PlanID,
				"plan_name":             user.Plan.Name,
				"max_games_owned":       user.Plan.MaxGamesOwned,
				"max_players_per_game":  user.Plan.MaxPlayersPerGame,
				"subscription_ends_at":  user.SubscriptionEndsAt,
				"storage_frozen":        user.StorageFrozen,
				"is_verified":           user.IsVerified,
				"created_at":            user.CreatedAt.Format("02.01.2006"),
				"access_token":          tokens.AccessToken,
				"refresh_token":         tokens.RefreshToken,
			})
		})

		r.Put("/api/me/username", authHandler.UpdateUsername)
		r.Put("/api/me/password", authHandler.ChangePassword)
		r.Post("/api/me/avatar", authHandler.UploadAvatar)
		r.Delete("/api/me/avatar", authHandler.DeleteAvatar)
		r.Get("/api/me/storage", authHandler.GetStorageUsage)

		r.Mount("/api/games", gameHandler.Routes())


	})

	log.Printf("DogmaLiter: http://0.0.0.0:%s", cfg.Port)
	log.Fatal(http.ListenAndServe("0.0.0.0:"+cfg.Port, r))
}

func respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}
