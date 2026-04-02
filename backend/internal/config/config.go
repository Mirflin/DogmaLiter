package config

import (
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	DatabaseURL string
	Port        string
	JWTSecret   string
	FrontendURL string

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
	_ = godotenv.Load("../.env")
	return &Config{
		DatabaseURL: getEnv("DATABASE_URL", "admin:FD371D79102981608CD4@tcp(89.254.131.120:3307)/DogmaLiter?charset=utf8mb4&parseTime=True&loc=Local"),
		Port:        getEnv("PORT", "8006"),
		JWTSecret:   getEnv("JWT_SECRET", "verysecretkey"),
		FrontendURL: getEnv("FRONTEND_URL", "http://89.254.131.120:5175"),

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
