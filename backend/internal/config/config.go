package config

import (
	"os"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	DatabaseURL string
	Port        string
	JWTSecret   string
	FrontendURL string
	FrontendOrigins []string

	SMTPHost     string
	SMTPPort     string
	SMTPUser     string
	SMTPPassword string
	SMTPFrom     string

	UploadDir string

	StripeSecretKey  string
	StripeWebhookKey string
	StripePlusPriceID string
	StripeProPriceID  string
}

func Load() *Config {
	loadDotEnv()
	frontendURL := getEnv("FRONTEND_URL", "http://89.254.131.120:5175")
	frontendOrigins := getEnvList("FRONTEND_URLS")
	if len(frontendOrigins) == 0 {
		frontendOrigins = []string{frontendURL}
	} else if !contains(frontendOrigins, frontendURL) {
		frontendOrigins = append([]string{frontendURL}, frontendOrigins...)
	}

	return &Config{
		DatabaseURL: getEnv("DATABASE_URL", "admin:FD371D79102981608CD4@tcp(89.254.131.120:3307)/DogmaLiter?charset=utf8mb4&parseTime=True&loc=Local"),
		Port:        getEnv("PORT", "8006"),
		JWTSecret:   getEnv("JWT_SECRET", "verysecretkey"),
		FrontendURL: frontendURL,
		FrontendOrigins: frontendOrigins,

		SMTPHost:     getEnv("SMTP_HOST", "smtp.gmail.com"),
		SMTPPort:     getEnv("SMTP_PORT", "587"),
		SMTPUser:     getEnv("SMTP_USER", ""),
		SMTPPassword: getEnv("SMTP_PASSWORD", ""),
		SMTPFrom:     getEnv("SMTP_FROM", "noreply@dogmaliter.com"),

		UploadDir: getEnv("UPLOAD_DIR", "./uploads"),

		StripeSecretKey:   getEnv("STRIPE_SECRET_KEY", ""),
		StripeWebhookKey:  getEnv("STRIPE_WEBHOOK_KEY", ""),
		StripePlusPriceID: getEnv("STRIPE_PLUS_PRICE_ID", ""),
		StripeProPriceID:  getEnv("STRIPE_PRO_PRICE_ID", ""),
	}
}

func getEnv(key, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}

func getEnvList(key string) []string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return nil
	}

	parts := strings.Split(value, ",")
	values := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed == "" || contains(values, trimmed) {
			continue
		}
		values = append(values, trimmed)
	}
	return values
}

func loadDotEnv() {
	for _, path := range []string{".env", "backend/.env", "../.env", "../../.env"} {
		if _, err := os.Stat(path); err == nil {
			_ = godotenv.Load(path)
		}
	}
}

func contains(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}
