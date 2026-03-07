package app

import (
	"context"
	"os"

	"github.com/chatchawan/agent-orchestrator/internal/models"
	"github.com/chatchawan/agent-orchestrator/internal/repository"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/rs/zerolog/log"
	"golang.org/x/crypto/bcrypt"
)

// SeedUser creates the initial admin user if it does not already exist.
func SeedUser(db *sqlx.DB) error {
	username := os.Getenv("SEED_USERNAME")
	password := os.Getenv("SEED_PASSWORD")
	if username == "" || password == "" {
		log.Warn().Msg("SEED_USERNAME or SEED_PASSWORD not set; skipping user seed")
		return nil
	}

	userRepo := repository.NewUserRepo(db)
	ctx := context.Background()

	if _, err := userRepo.GetByUsername(ctx, username); err == nil {
		log.Info().Str("username", username).Msg("seed user already exists")
		return nil
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	user := &models.User{
		UserID:       uuid.New(),
		Username:     username,
		PasswordHash: string(hash),
	}
	if err := userRepo.Create(ctx, user); err != nil {
		return err
	}
	log.Info().Str("username", username).Msg("seed user created")
	return nil
}
