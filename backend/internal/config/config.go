package config

import (
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Port               string
	DatabaseDSN        string
	JWTSecret          string
	TokenTTL           time.Duration
	UploadDir          string
	MaxUploadBytes     int64
	CORSAllowedOrigins []string
	AdminUsername      string
	AdminPassword      string
}

func Load() Config {
	port := envOr("APP_PORT", "8080")
	ttlHours := envIntOr("TOKEN_TTL_HOURS", 48)
	maxUploadMB := envIntOr("MAX_UPLOAD_MB", 100)
	origins := strings.Split(envOr("CORS_ALLOWED_ORIGINS", "*"), ",")
	if len(origins) == 1 && strings.TrimSpace(origins[0]) == "" {
		origins = []string{"*"}
	}

	return Config{
		Port:               port,
		DatabaseDSN:        envOr("DATABASE_DSN", "postgres://file_upload:file_upload@localhost:5432/file_upload?sslmode=disable"),
		JWTSecret:          envOr("JWT_SECRET", "dev-secret"),
		TokenTTL:           time.Duration(ttlHours) * time.Hour,
		UploadDir:          envOr("UPLOAD_DIR", "./uploads"),
		MaxUploadBytes:     int64(maxUploadMB) * 1024 * 1024,
		CORSAllowedOrigins: origins,
		AdminUsername:      envOr("ADMIN_USERNAME", ""),
		AdminPassword:      envOr("ADMIN_PASSWORD", ""),
	}
}

func envOr(name, fallback string) string {
	value := strings.TrimSpace(os.Getenv(name))
	if value == "" {
		return fallback
	}
	return value
}

func envIntOr(name string, fallback int) int {
	value := strings.TrimSpace(os.Getenv(name))
	if value == "" {
		return fallback
	}
	n, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return n
}

