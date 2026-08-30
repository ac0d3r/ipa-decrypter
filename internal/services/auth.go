package services

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/ac0d3r/ipa-decrypter/internal/config"
	"github.com/ac0d3r/ipa-decrypter/internal/db"
	"github.com/ac0d3r/ipa-decrypter/internal/logger"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

const (
	sessionTTL        = 7 * 24 * time.Hour
	sessionTokenBytes = 32
)

var (
	ErrInvalidCredentials = errors.New("invalid username or password")
	ErrUnauthorized       = errors.New("unauthorized")
)

type AuthService struct {
	cfg config.Config
	db  *db.DB
}

func NewAuthService(cfg config.Config, database *db.DB) *AuthService {
	return &AuthService{cfg: cfg, db: database}
}

// EnsureBootstrapAdmin creates the first admin from env if none exist.
func (s *AuthService) EnsureBootstrapAdmin() error {
	n, err := s.db.CountAdminUsers()
	if err != nil {
		return err
	}
	if n > 0 {
		return nil
	}
	username := s.cfg.AdminUsername
	password := s.cfg.AdminPassword
	if username == "" || password == "" {
		return fmt.Errorf("ADMIN_USERNAME and ADMIN_PASSWORD required to bootstrap admin")
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	u := db.AdminUser{
		ID:           uuid.NewString(),
		Username:     username,
		PasswordHash: string(hash),
	}
	if err := s.db.CreateAdminUser(u); err != nil {
		return err
	}
	logger.WithComponent("auth").WithField("username", username).Info("bootstrapped admin user")
	return nil
}

type LoginResult struct {
	Token     string    `json:"-"`
	Username  string    `json:"username"`
	ExpiresAt time.Time `json:"expires_at"`
}

func (s *AuthService) Login(username, password string) (LoginResult, error) {
	u, err := s.db.GetAdminUserByUsername(username)
	if err != nil {
		return LoginResult{}, ErrInvalidCredentials
	}
	if bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password)) != nil {
		return LoginResult{}, ErrInvalidCredentials
	}
	token, err := randomToken()
	if err != nil {
		return LoginResult{}, err
	}
	expires := time.Now().Add(sessionTTL)
	u.SessionToken = token
	u.SessionExpiresAt = &expires
	if err := s.db.UpdateAdminUser(u); err != nil {
		return LoginResult{}, err
	}
	return LoginResult{Token: token, Username: u.Username, ExpiresAt: expires}, nil
}

func (s *AuthService) Logout(token string) error {
	if token == "" {
		return nil
	}
	u, err := s.db.GetAdminUserBySessionToken(token)
	if err != nil {
		return nil
	}
	u.SessionToken = ""
	u.SessionExpiresAt = nil
	return s.db.UpdateAdminUser(u)
}

func (s *AuthService) Validate(token string) (db.AdminUser, error) {
	if token == "" {
		return db.AdminUser{}, ErrUnauthorized
	}
	u, err := s.db.GetAdminUserBySessionToken(token)
	if err != nil {
		return db.AdminUser{}, ErrUnauthorized
	}
	if u.SessionExpiresAt == nil || time.Now().After(*u.SessionExpiresAt) {
		u.SessionToken = ""
		u.SessionExpiresAt = nil
		_ = s.db.UpdateAdminUser(u)
		return db.AdminUser{}, ErrUnauthorized
	}
	return u, nil
}

func randomToken() (string, error) {
	b := make([]byte, sessionTokenBytes)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
