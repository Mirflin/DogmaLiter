package main

import (
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/joho/godotenv"

	"backend/pkg/database"
)

func main() {
	loadEnv()

	dsn := os.Getenv("DATABASE_URL")

	allowedOrigin := os.Getenv("CORS_ALLOW_ORIGIN")

	db := database.Connect(dsn)
	database.AutoMigrate(db)

	_ = db

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(30 * time.Second))

	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", allowedOrigin)
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
			if r.Method == "OPTIONS" {
				w.WriteHeader(204)
				return
			}
			next.ServeHTTP(w, r)
		})
	})

	r.Get("/api/test", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok"}`))
	})

	port := os.Getenv("PORT")

	log.Printf("Tabletop App: http://localhost:%s", port)
	log.Fatal(http.ListenAndServe(":"+port, r))
}

func loadEnv() {
	candidates := []string{
		".env",
		filepath.Join("..", ".env"),
	}

	for _, envPath := range candidates {
		if err := godotenv.Load(envPath); err == nil {
			return
		}
	}
}
