package main

import (
	"log"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"backend/internal/auth"
	"backend/internal/config"
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
	authService := auth.NewService(authRepo, jwtManager, mailer, cfg.FrontendURL)
	authHandler := auth.NewHandler(authService)

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

	r.Group(func(r chi.Router) {
		r.Use(auth.JWTMiddleware(jwtManager))

		r.Get("/api/me", func(w http.ResponseWriter, r *http.Request) {
			userID := auth.GetUserID(r)
			user, err := authRepo.GetUserByID(userID)
			if err != nil {
				http.Error(w, `{"error":"User not found"}`, 404)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			respondJSON(w, 200, map[string]interface{}{
				"id":          user.ID,
				"username":    user.Username,
				"email":       user.Email,
				"role":        user.Role,
				"plan_id":     user.PlanID,
				"is_verified": user.IsVerified,
			})
		})

		r.Group(func(r chi.Router) {
			r.Use(auth.RequireAdmin)
			// r.Mount("/api/admin/news", newsHandler.Routes())
		})

		// r.Mount("/api/games", gameHandler.Routes())
	})

	log.Printf("DogmaLiter: http://localhost:%s", cfg.Port)
	log.Fatal(http.ListenAndServe(":"+cfg.Port, r))
}

func respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}
