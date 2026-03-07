package services

import (
	"context"
	"errors"
	"os"
	"strconv"
	"time"

	"github.com/chatchawan/agent-orchestrator/internal/models"
	"github.com/chatchawan/agent-orchestrator/internal/repository"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrSessionNotFound    = errors.New("session not found")
)

type AuthService interface {
	Login(ctx context.Context, username, password string) (sessionID string, user *models.User, err error)
	Logout(ctx context.Context, sessionID string) error
	ValidateSession(ctx context.Context, sessionID string) (*models.User, error)
}

type authService struct {
	userRepo    repository.UserRepository
	sessionRepo repository.SessionRepository
}

func NewAuthService(userRepo repository.UserRepository, sessionRepo repository.SessionRepository) AuthService {
	return &authService{userRepo: userRepo, sessionRepo: sessionRepo}
}

func (s *authService) Login(ctx context.Context, username, password string) (string, *models.User, error) {
	user, err := s.userRepo.GetByUsername(ctx, username)
	if err != nil {
		return "", nil, ErrInvalidCredentials
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return "", nil, ErrInvalidCredentials
	}

	hours := 168 // 7 days default
	if h := os.Getenv("SESSION_MAX_AGE_HOURS"); h != "" {
		if v, err := strconv.Atoi(h); err == nil {
			hours = v
		}
	}

	session := &models.Session{
		SessionID: uuid.New().String(),
		UserID:    user.UserID,
		ExpiresAt: time.Now().Add(time.Duration(hours) * time.Hour),
	}
	if err := s.sessionRepo.Create(ctx, session); err != nil {
		return "", nil, err
	}

	return session.SessionID, user, nil
}

func (s *authService) Logout(ctx context.Context, sessionID string) error {
	return s.sessionRepo.Delete(ctx, sessionID)
}

func (s *authService) ValidateSession(ctx context.Context, sessionID string) (*models.User, error) {
	session, err := s.sessionRepo.GetByID(ctx, sessionID)
	if err != nil {
		return nil, ErrSessionNotFound
	}
	user, err := s.userRepo.GetByID(ctx, session.UserID)
	if err != nil {
		return nil, ErrSessionNotFound
	}
	return user, nil
}
