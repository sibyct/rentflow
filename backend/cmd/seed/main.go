// Command seed creates one admin user for local development, since the
// public /auth/register endpoint always assigns UserRoleManager and
// there is otherwise no way to get an admin account into a fresh
// database. Safe to run repeatedly: an existing email is left untouched
// rather than erroring or duplicating.
//
//	go run ./cmd/seed
//	SEED_ADMIN_EMAIL=me@example.com SEED_ADMIN_PASSWORD=... go run ./cmd/seed
package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/joho/godotenv"
	"golang.org/x/crypto/bcrypt"

	"propertymanagement/internal/config"
	"propertymanagement/internal/domain"
	"propertymanagement/internal/repository/postgres"
)

const (
	defaultEmail    = "admin@propertymanagement.local"
	defaultPassword = "ChangeMe123!"
	connectTimeout  = 5 * time.Second
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	_ = godotenv.Load()

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	email := getEnv("SEED_ADMIN_EMAIL", defaultEmail)
	password := getEnv("SEED_ADMIN_PASSWORD", defaultPassword)

	ctx, cancel := context.WithTimeout(context.Background(), connectTimeout)
	defer cancel()

	pool, err := postgres.NewPool(ctx, cfg.DatabaseURL, cfg.DatabaseMaxConn)
	if err != nil {
		return fmt.Errorf("connecting to database: %w", err)
	}
	defer pool.Close()

	userRepo := postgres.NewUserRepository(pool)

	if _, err := userRepo.GetByEmail(ctx, email); err == nil {
		fmt.Printf("user %s already exists, skipping\n", email)
		return nil
	} else if !errors.Is(err, domain.ErrNotFound) {
		return fmt.Errorf("checking for existing user: %w", err)
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("hashing password: %w", err)
	}

	now := time.Now().UTC()
	user := &domain.User{
		ID:           uuid.New(),
		Email:        email,
		PasswordHash: string(hash),
		Role:         domain.UserRoleAdmin,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	if err := userRepo.Create(ctx, user); err != nil {
		return fmt.Errorf("creating admin user: %w", err)
	}

	fmt.Printf("created admin user\n  email:    %s\n  password: %s\n  role:     %s\n", email, password, user.Role)
	fmt.Println("change this password after logging in — it's only meant for local development.")
	return nil
}

func getEnv(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return fallback
}
