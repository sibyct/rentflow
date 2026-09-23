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

	// Object storage (S3-compatible: MinIO locally, S3/Tigris in production)
	// for receipts, deposit documents and generated statement PDFs. All
	// optional — with S3Endpoint unset the app runs without file uploads
	// and the attachment endpoints answer 503. Endpoints are full URLs
	// (http://minio:9000). S3PublicEndpoint is the address a *browser*
	// reaches the same store at (it signs the upload URLs); it defaults
	// to S3Endpoint, and differs in Docker where the API talks to
	// minio:9000 but the browser to localhost:9000.
	S3Endpoint       string
	S3PublicEndpoint string
	S3Bucket         string
	S3AccessKey      string
	S3SecretKey      string
	S3Region         string

	// SMTP for outbound email (owner statements). Optional: with SMTPHost
	// unset, emails stay queued in email_outbox and the UI says so.
	SMTPHost     string
	SMTPPort     int
	SMTPUser     string
	SMTPPassword string
	SMTPFrom     string
	// SMTPTLS is "starttls" (default, port 587), "tls" (implicit, 465) or "none" (local Mailpit).
	SMTPTLS string

	// PublicAppURL is the frontend's origin, used for links inside emails.
	PublicAppURL string
}

func (c Config) IsProduction() bool { return c.Env == "production" }

// StorageEnabled reports whether object storage is configured.
func (c Config) StorageEnabled() bool { return c.S3Endpoint != "" }

// MailEnabled reports whether SMTP is configured.
func (c Config) MailEnabled() bool { return c.SMTPHost != "" }

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

		S3Endpoint:   getEnv("S3_ENDPOINT", ""),
		S3Bucket:     getEnv("S3_BUCKET", "rentflow"),
		S3AccessKey:  getEnv("S3_ACCESS_KEY", ""),
		S3SecretKey:  getEnv("S3_SECRET_KEY", ""),
		S3Region:     getEnv("S3_REGION", "us-east-1"),
		SMTPHost:     getEnv("SMTP_HOST", ""),
		SMTPPort:     int(getEnvInt32("SMTP_PORT", 587)),
		SMTPUser:     getEnv("SMTP_USER", ""),
		SMTPPassword: getEnv("SMTP_PASSWORD", ""),
		SMTPFrom:     getEnv("SMTP_FROM", "RentFlow <no-reply@rentflow.local>"),
		SMTPTLS:      getEnv("SMTP_TLS", "starttls"),
		PublicAppURL: getEnv("PUBLIC_APP_URL", "http://localhost:5173"),
	}
	cfg.S3PublicEndpoint = getEnv("S3_PUBLIC_ENDPOINT", cfg.S3Endpoint)

	if cfg.S3Endpoint != "" && (cfg.S3AccessKey == "" || cfg.S3SecretKey == "") {
		errs = append(errs, errors.New("S3_ACCESS_KEY and S3_SECRET_KEY are required when S3_ENDPOINT is set"))
	}
	if cfg.SMTPTLS != "starttls" && cfg.SMTPTLS != "tls" && cfg.SMTPTLS != "none" {
		errs = append(errs, fmt.Errorf("SMTP_TLS must be one of starttls|tls|none, got %q", cfg.SMTPTLS))
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
