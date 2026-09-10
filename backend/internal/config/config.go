// Package config loads and validates process configuration from
// environment variables. Loading fails fast: any missing required
// variable or invalid value returns an error from Load so main() can
// refuse to start rather than run half-configured.
package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Env      string // "development" | "staging" | "production"
	HTTPPort string

	DatabaseURL     string
	DatabaseMaxConn int32

	RedisAddr     string
	RedisPassword string
	RedisDB       int

	JWTAccessSecret  string
	JWTRefreshSecret string
	JWTAccessTTL     time.Duration
	JWTRefreshTTL    time.Duration

	// CookieDomain/CookieSecure control the refresh-token cookie. Secure
	// should be true everywhere except local plain-HTTP development.
	CookieDomain string
	CookieSecure bool

	AllowedOrigins []string
}

func (c Config) IsProduction() bool { return c.Env == "production" }

// Load reads configuration from the environment and validates it.
// Required variables have no default and cause Load to return an error
// when unset; everything else falls back to a development-friendly
// default.
func Load() (Config, error) {
	var errs []error

	cfg := Config{
		Env:             getEnv("APP_ENV", "development"),
		HTTPPort:        getEnv("HTTP_PORT", "8080"),
		DatabaseURL:     requireEnv("DATABASE_URL", &errs),
		DatabaseMaxConn: getEnvInt32("DATABASE_MAX_CONNS", 10),
		RedisAddr:       getEnv("REDIS_ADDR", "localhost:6379"),
		RedisPassword:   getEnv("REDIS_PASSWORD", ""),
		RedisDB:         int(getEnvInt32("REDIS_DB", 0)),

		JWTAccessSecret:  requireEnv("JWT_ACCESS_SECRET", &errs),
		JWTRefreshSecret: requireEnv("JWT_REFRESH_SECRET", &errs),
		JWTAccessTTL:     getEnvDuration("JWT_ACCESS_TTL", 15*time.Minute),
		JWTRefreshTTL:    getEnvDuration("JWT_REFRESH_TTL", 7*24*time.Hour),

		CookieDomain: getEnv("COOKIE_DOMAIN", ""),
		CookieSecure: getEnvBool("COOKIE_SECURE", false),

		AllowedOrigins: strings.Split(getEnv("ALLOWED_ORIGINS", "http://localhost:5173"), ","),
	}

	if cfg.Env != "development" && cfg.Env != "staging" && cfg.Env != "production" {
		errs = append(errs, fmt.Errorf("APP_ENV must be one of development|staging|production, got %q", cfg.Env))
	}
	if len(cfg.JWTAccessSecret) < 32 && cfg.JWTAccessSecret != "" {
		errs = append(errs, errors.New("JWT_ACCESS_SECRET must be at least 32 characters"))
	}
	if len(cfg.JWTRefreshSecret) < 32 && cfg.JWTRefreshSecret != "" {
		errs = append(errs, errors.New("JWT_REFRESH_SECRET must be at least 32 characters"))
	}
	if cfg.IsProduction() && !cfg.CookieSecure {
		errs = append(errs, errors.New("COOKIE_SECURE must be true when APP_ENV=production"))
	}

	if len(errs) > 0 {
		return Config{}, fmt.Errorf("invalid configuration: %w", errors.Join(errs...))
	}

	return cfg, nil
}

func getEnv(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return fallback
}

func requireEnv(key string, errs *[]error) string {
	v, ok := os.LookupEnv(key)
	if !ok || v == "" {
		*errs = append(*errs, fmt.Errorf("%s is required", key))
		return ""
	}
	return v
}

func getEnvInt32(key string, fallback int32) int32 {
	v, ok := os.LookupEnv(key)
	if !ok || v == "" {
		return fallback
	}
	n, err := strconv.ParseInt(v, 10, 32)
	if err != nil {
		return fallback
	}
	return int32(n)
}

func getEnvBool(key string, fallback bool) bool {
	v, ok := os.LookupEnv(key)
	if !ok || v == "" {
		return fallback
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return fallback
	}
	return b
}

func getEnvDuration(key string, fallback time.Duration) time.Duration {
	v, ok := os.LookupEnv(key)
	if !ok || v == "" {
		return fallback
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		return fallback
	}
	return d
}
