package services

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ac0d3r/ipa-decrypter/internal/config"
	"github.com/ac0d3r/ipa-decrypter/internal/db"
	"github.com/ac0d3r/ipa-decrypter/internal/ipatool"
	"github.com/ac0d3r/ipa-decrypter/internal/logger"
	"github.com/ac0d3r/ipa-decrypter/internal/util"
	"github.com/google/uuid"
)

var accountLog = logger.WithComponent("account")

type AccountService struct {
	cfg    config.Config
	db     *db.DB
	runner ipatool.Runner
}

func NewAccountService(cfg config.Config, database *db.DB) *AccountService {
	return &AccountService{
		cfg: cfg,
		db:  database,
		runner: ipatool.Runner{
			Bin:        cfg.IPATOOLBin,
			Passphrase: cfg.KeychainPassphrase,
		},
	}
}

type AccountCreateInput struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Label    string `json:"label"`
	Region   string `json:"region"`
	LoginNow bool   `json:"login_now"`
}

// AccountLoginError is returned when login fails. Needs2FA means Apple/ipatool expects a verification code.
type AccountLoginError struct {
	Account  db.Account
	Needs2FA bool
	Message  string
}

func (e *AccountLoginError) Error() string {
	if e.Message != "" {
		return e.Message
	}
	if e.Needs2FA {
		return "2fa required"
	}
	return "login failed"
}

func (s *AccountService) List() ([]db.Account, error) {
	rows, err := s.db.ListAccounts()
	if err != nil {
		return nil, err
	}
	out := make([]db.Account, 0, len(rows))
	for i := range rows {
		out = append(out, rows[i].Account)
	}
	return out, nil
}

func (s *AccountService) Get(id string) (db.Account, error) {
	row, err := s.db.GetAccount(id)
	return row.Account, err
}

func (s *AccountService) Create(in AccountCreateInput) (db.Account, error) {
	in.Email = strings.TrimSpace(in.Email)
	in.Region = strings.ToLower(strings.TrimSpace(in.Region))
	in.Label = strings.TrimSpace(in.Label)
	if in.Email == "" || in.Password == "" {
		return db.Account{}, fmt.Errorf("email and password required")
	}
	if in.Label == "" {
		if in.Region != "" {
			in.Label = in.Region
		} else {
			in.Label = in.Email
		}
	}

	id := uuid.NewString()
	home := filepath.Join(s.cfg.AccountsDir, id)
	if abs, err := filepath.Abs(home); err == nil {
		home = abs
	}
	if err := os.MkdirAll(home, 0o700); err != nil {
		return db.Account{}, err
	}
	enc, err := util.Encrypt(s.cfg.KeychainPassphrase, in.Password)
	if err != nil {
		return db.Account{}, err
	}
	now := time.Now().UTC()
	row := db.AccountRow{
		Account: db.Account{
			ID:        id,
			Email:     in.Email,
			Label:     in.Label,
			Region:    in.Region,
			HomeDir:   home,
			Status:    "logged_out",
			CreatedAt: now,
			UpdatedAt: now,
		},
		PasswordEnc: enc,
	}
	if err := s.db.CreateAccount(row); err != nil {
		return db.Account{}, err
	}
	accountLog.WithFields(map[string]any{
		"account_id": id,
		"email":      in.Email,
		"region":     in.Region,
		"login_now":  in.LoginNow,
	}).Info("account created")
	if in.LoginNow {
		return s.Login(id, "")
	}
	return row.Account, nil
}

func (s *AccountService) Login(id, authCode string) (db.Account, error) {
	row, err := s.db.GetAccount(id)
	if err != nil {
		return db.Account{}, err
	}
	password, err := util.Decrypt(s.cfg.KeychainPassphrase, row.PasswordEnc)
	if err != nil {
		return db.Account{}, err
	}

	// First login may download SAP assets; allow several minutes.
	ctx, cancel := ipatool.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	accountLog.WithFields(map[string]any{"account_id": id, "email": row.Email}).Info("login started")

	res, err := s.runner.AuthLogin(ctx, row.HomeDir, row.Email, password, strings.TrimSpace(authCode))
	row.UpdatedAt = time.Now().UTC()
	msg := firstNonEmpty(ipatool.FailureMessage(res.Stdout), errString(err), ipatool.CombinedOutput(res))
	_, storedErr := s.runner.ReadStoredAccount(row.HomeDir)
	persisted := storedErr == nil

	// ipatool can exit 0 on 2FA-required without writing the keyring account file.
	needs2FA := strings.TrimSpace(authCode) == "" && looksLikeNeeds2FA(msg)
	ok := err == nil && res.ExitCode == 0 && persisted && !needs2FA && !looksLikeAccountDisabled(msg)
	if ok {
		row.Status = "logged_in"
		row.LastError = ""
		s.applyStoreFront(&row)
		accountLog.WithFields(map[string]any{
			"account_id": id,
			"email":      row.Email,
			"region":     row.Region,
			"storefront": row.StoreFront,
		}).Info("login succeeded")
		if err := s.db.UpdateAccount(row); err != nil {
			return row.Account, err
		}
		return row.Account, nil
	}

	if needs2FA {
		row.Status = "needs_2fa"
		row.LastError = "等待输入 2FA 验证码"
		msg = firstNonEmpty(msg, "2FA code required")
		accountLog.WithFields(map[string]any{"account_id": id, "email": row.Email}).Warn("login needs 2FA")
	} else {
		row.Status = "error"
		if looksLikeAccountDisabled(msg) {
			msg = "Apple ID is disabled"
		} else if msg == "" && !persisted {
			msg = "login did not persist account credentials"
		}
		row.LastError = msg
		accountLog.WithFields(map[string]any{
			"account_id": id,
			"email":      row.Email,
			"error":      msg,
			"exit_code":  res.ExitCode,
		}).Error("login failed")
	}
	_ = s.db.UpdateAccount(row)
	return row.Account, &AccountLoginError{Account: row.Account, Needs2FA: needs2FA, Message: msg}
}

func looksLikeNeeds2FA(msg string) bool {
	lower := strings.ToLower(msg)
	for _, n := range []string{
		"auth code is required", "2fa code is required", "2fa code",
		"two-factor", "two factor", "verification code", "--auth-code",
	} {
		if strings.Contains(lower, n) {
			return true
		}
	}
	return false
}

func looksLikeAccountDisabled(msg string) bool {
	lower := strings.ToLower(msg)
	return strings.Contains(lower, "account is disabled") ||
		strings.Contains(lower, "your account is disabled")
}

func (s *AccountService) Refresh(id string) (db.Account, error) {
	row, err := s.db.GetAccount(id)
	if err != nil {
		return db.Account{}, err
	}
	ctx, cancel := ipatool.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	res, runErr := s.runner.AuthInfo(ctx, row.HomeDir)
	row.UpdatedAt = time.Now().UTC()
	if runErr != nil || res.ExitCode != 0 {
		row.Status = "logged_out"
		row.LastError = firstNonEmpty(errString(runErr), ipatool.CombinedOutput(res))
		accountLog.WithFields(map[string]any{"account_id": id, "error": row.LastError}).Warn("session refresh failed")
	} else {
		row.Status = "logged_in"
		row.LastError = ""
		s.applyStoreFront(&row)
		accountLog.WithFields(map[string]any{"account_id": id, "region": row.Region}).Info("session refreshed")
	}
	_ = s.db.UpdateAccount(row)
	return row.Account, nil
}

func (s *AccountService) applyStoreFront(row *db.AccountRow) {
	acc, err := s.runner.ReadStoredAccount(row.HomeDir)
	if err != nil || acc.StoreFront == "" {
		return
	}
	row.StoreFront = acc.StoreFront
	if country, err := util.CountryFromStoreFront(acc.StoreFront); err == nil {
		prevRegion := row.Region
		row.Region = country
		if row.Label == "" || row.Label == row.Email || strings.EqualFold(row.Label, prevRegion) {
			row.Label = strings.ToUpper(country)
		}
	}
	if acc.Email != "" {
		row.Email = acc.Email
	}
}

func (s *AccountService) Revoke(id string) (db.Account, error) {
	row, err := s.db.GetAccount(id)
	if err != nil {
		return db.Account{}, err
	}
	ctx, cancel := ipatool.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	res, runErr := s.runner.AuthRevoke(ctx, row.HomeDir)
	row.UpdatedAt = time.Now().UTC()
	row.Status = "logged_out"
	if runErr != nil || res.ExitCode != 0 {
		row.LastError = firstNonEmpty(errString(runErr), ipatool.CombinedOutput(res))
	} else {
		row.LastError = ""
	}
	accountLog.WithField("account_id", id).Info("session revoked")
	_ = s.db.UpdateAccount(row)
	return row.Account, nil
}

func (s *AccountService) Delete(id string) error {
	row, err := s.db.GetAccount(id)
	if err != nil {
		return err
	}
	_, _ = s.Revoke(id)
	if err := s.db.DeleteAccount(id); err != nil {
		return err
	}
	_ = os.RemoveAll(row.HomeDir)
	accountLog.WithField("account_id", id).Info("account deleted")
	return nil
}

func (s *AccountService) HomeDir(id string) (string, error) {
	row, err := s.db.GetAccount(id)
	if err != nil {
		return "", err
	}
	return row.HomeDir, nil
}

func (s *AccountService) Runner() ipatool.Runner { return s.runner }
