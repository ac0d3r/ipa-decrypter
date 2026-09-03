package services

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"github.com/ac0d3r/ipa-decrypter/internal/config"
	"github.com/ac0d3r/ipa-decrypter/internal/db"
	"github.com/ac0d3r/ipa-decrypter/internal/logger"
)

const (
	sessionTTL        = 7 * 24 * time.Hour
	sessionTokenBytes = 32
)

var (
	ErrInvalidCredentials = errors.New("invalid username or password")
	ErrUnauthorized       = errors.New("unauthorized")
	ErrForbidden          = errors.New("forbidden")
	ErrNotFound           = errors.New("not found")
)

type AuthService struct {
	cfg config.Config
	db  *db.DB
}

func NewAuthService(cfg config.Config, database *db.DB) *AuthService {
	return &AuthService{cfg: cfg, db: database}
}

// EnsureBootstrapAdmin creates the admin from env if none exists.
func (s *AuthService) EnsureBootstrapAdmin() error {
	n, err := s.db.CountAdmins()
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
	if _, err := s.createUser(username, password, true); err != nil {
		return err
	}
	logger.WithComponent("auth").WithField("username", username).Info("bootstrapped admin user")
	return nil
}

type LoginResult struct {
	Token     string
	Username  string
	IsAdmin   bool
	ExpiresAt time.Time
	Quota     map[string]map[string]int
}

func (s *AuthService) Login(username, password string) (LoginResult, error) {
	u, err := s.db.GetUserByUsername(strings.TrimSpace(username))
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
	if err := s.db.UpdateUser(u); err != nil {
		return LoginResult{}, err
	}
	out := LoginResult{
		Token:     token,
		Username:  u.Username,
		IsAdmin:   u.IsAdmin,
		ExpiresAt: expires,
	}
	if !u.IsAdmin {
		out.Quota = u.Quota()
	}
	return out, nil
}

func (s *AuthService) Logout(token string) error {
	if token == "" {
		return nil
	}
	u, err := s.db.GetUserBySessionToken(token)
	if err != nil {
		return nil
	}
	u.SessionToken = ""
	u.SessionExpiresAt = nil
	return s.db.UpdateUser(u)
}

func (s *AuthService) Validate(token string) (db.User, error) {
	if token == "" {
		return db.User{}, ErrUnauthorized
	}
	u, err := s.db.GetUserBySessionToken(token)
	if err != nil {
		return db.User{}, ErrUnauthorized
	}
	if u.SessionExpiresAt == nil || time.Now().After(*u.SessionExpiresAt) {
		u.SessionToken = ""
		u.SessionExpiresAt = nil
		_ = s.db.UpdateUser(u)
		return db.User{}, ErrUnauthorized
	}
	return u, nil
}

func (s *AuthService) ValidateAdmin(token string) (db.User, error) {
	u, err := s.Validate(token)
	if err != nil {
		return db.User{}, err
	}
	if !u.IsAdmin {
		return db.User{}, ErrForbidden
	}
	return u, nil
}

type UserCreateInput struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type UserUpdateInput struct {
	Password *string `json:"password"`
	Limit    *int    `json:"limit"` // applies to both decrypt & download totals
}

func (s *AuthService) ListUsers() ([]db.User, error) {
	items, err := s.db.ListUsers()
	if err != nil {
		return nil, err
	}
	if items == nil {
		items = []db.User{}
	}
	return items, nil
}

func (s *AuthService) CreateUser(in UserCreateInput) (db.User, error) {
	return s.createUser(in.Username, in.Password, false)
}

func (s *AuthService) createUser(username, password string, isAdmin bool) (db.User, error) {
	username = strings.TrimSpace(username)
	if username == "" || password == "" {
		return db.User{}, fmt.Errorf("username and password required")
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return db.User{}, err
	}
	u := db.User{
		ID:           uuid.NewString(),
		Username:     username,
		PasswordHash: string(hash),
		IsAdmin:      isAdmin,
	}
	if !isAdmin {
		u.DecryptLimit = db.DefaultQuota
		u.DownloadLimit = db.DefaultQuota
	}
	if err := s.db.CreateUser(&u); err != nil {
		if isUniqueViolation(err) {
			return db.User{}, fmt.Errorf("username already exists")
		}
		return db.User{}, err
	}
	return u, nil
}

func (s *AuthService) UpdateUser(id string, in UserUpdateInput) (db.User, error) {
	u, err := s.getUser(id)
	if err != nil {
		return db.User{}, err
	}
	if in.Password != nil {
		pw := strings.TrimSpace(*in.Password)
		if pw == "" {
			return db.User{}, fmt.Errorf("password required")
		}
		hash, err := bcrypt.GenerateFromPassword([]byte(pw), bcrypt.DefaultCost)
		if err != nil {
			return db.User{}, err
		}
		u.PasswordHash = string(hash)
		u.SessionToken = ""
		u.SessionExpiresAt = nil
	}
	if in.Limit != nil {
		if *in.Limit < -1 {
			return db.User{}, fmt.Errorf("limit must be -1 or >= 0")
		}
		u.DecryptLimit = *in.Limit
		u.DownloadLimit = *in.Limit
	}
	if err := s.db.UpdateUser(u); err != nil {
		return db.User{}, err
	}
	return u, nil
}

func (s *AuthService) DeleteUser(id, actorID string) error {
	u, err := s.getUser(id)
	if err != nil {
		return err
	}
	if u.ID == actorID {
		return fmt.Errorf("cannot delete yourself")
	}
	return s.db.DeleteUser(id)
}

func (s *AuthService) getUser(id string) (db.User, error) {
	u, err := s.db.GetUser(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return db.User{}, ErrNotFound
		}
		return db.User{}, err
	}
	return u, nil
}

func (s *AuthService) ConsumeUserQuota(userID, qtype string) error {
	return s.db.ConsumeUserQuota(userID, qtype)
}

func randomToken() (string, error) {
	b := make([]byte, sessionTokenBytes)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func isUniqueViolation(err error) bool {
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "unique") || strings.Contains(msg, "constraint failed")
}
