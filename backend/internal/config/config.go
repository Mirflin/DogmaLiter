package config

import "os"

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
}

func Load() *Config {
	return &Config{
		DatabaseURL: getEnv("DATABASE_URL", "admin:FD371D79102981608CD4@tcp(89.254.131.120:3307)/DogmaLiter?charset=utf8mb4&parseTime=True&loc=Local"),
		Port:        getEnv("PORT", "8080"),
		JWTSecret:   getEnv("JWT_SECRET", "verysecretkey"),
		FrontendURL: getEnv("FRONTEND_URL", "http://localhost:5173"),

		SMTPHost:     getEnv("SMTP_HOST", "smtp.gmail.com"),
		SMTPPort:     getEnv("SMTP_PORT", "587"),
		SMTPUser:     getEnv("SMTP_USER", ""),
		SMTPPassword: getEnv("SMTP_PASSWORD", ""),
		SMTPFrom:     getEnv("SMTP_FROM", "noreply@dogmaliter.com"),
	}
}

func getEnv(key, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}
