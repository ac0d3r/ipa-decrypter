package services

import (
	"crypto/rand"
	"fmt"
	"strings"
	"time"
	"unicode"

	"github.com/google/uuid"

	"github.com/ac0d3r/ipa-decrypter/internal/db"
)

const (
	voucherAlphabet = "23456789ABCDEFGHJKLMNPQRSTUVWXYZ"
	voucherRawLen   = 8
	voucherMaxBatch = 50
	voucherMaxQuota = 100000
	voucherPageSize = 20
	voucherMaxPage  = 100
)

type VoucherCreateInput struct {
	Count int `json:"count"`
	Quota int `json:"quota"`
}

type VoucherCreated struct {
	Codes []string `json:"codes"`
	Quota int      `json:"quota"`
}

type VoucherView struct {
	ID               string     `json:"id"`
	Code             string     `json:"code"`
	Quota            int        `json:"quota"`
	Redeemed         bool       `json:"redeemed"`
	UserID           string     `json:"user_id,omitempty"`
	RedeemedUsername string     `json:"redeemed_username,omitempty"`
	RedeemedAt       *time.Time `json:"redeemed_at,omitempty"`
	CreatedAt        time.Time  `json:"created_at"`
}

func (s *AuthService) CreateVouchers(in VoucherCreateInput) (VoucherCreated, error) {
	if in.Count < 1 || in.Count > voucherMaxBatch {
		return VoucherCreated{}, fmt.Errorf("count must be 1–%d", voucherMaxBatch)
	}
	if in.Quota < 1 || in.Quota > voucherMaxQuota {
		return VoucherCreated{}, fmt.Errorf("quota must be 1–%d", voucherMaxQuota)
	}
	const maxAttempts = 8
	for attempt := 0; attempt < maxAttempts; attempt++ {
		items, codes, err := generateVoucherBatch(in.Count, in.Quota)
		if err != nil {
			return VoucherCreated{}, err
		}
		if err := s.db.CreateVouchers(items); err != nil {
			if isUniqueViolation(err) {
				continue
			}
			return VoucherCreated{}, err
		}
		return VoucherCreated{Codes: codes, Quota: in.Quota}, nil
	}
	return VoucherCreated{}, fmt.Errorf("failed to allocate unique voucher codes")
}

type VoucherListQuery struct {
	Status string
	UserID string
	Offset int
	Limit  int
}

type VoucherRedeemer struct {
	ID       string `json:"id"`
	Username string `json:"username"`
}

type VoucherListPage struct {
	Items  []VoucherView     `json:"items"`
	Users  []VoucherRedeemer `json:"users,omitempty"`
	Total  int               `json:"total"`
	Offset int               `json:"offset"`
	Limit  int               `json:"limit"`
}

func (s *AuthService) ListVouchers(q VoucherListQuery) (VoucherListPage, error) {
	if q.Offset < 0 {
		q.Offset = 0
	}
	if q.Limit <= 0 {
		q.Limit = voucherPageSize
	}
	if q.Limit > voucherMaxPage {
		q.Limit = voucherMaxPage
	}

	f := db.VoucherListFilter{Offset: q.Offset, Limit: q.Limit}
	page := VoucherListPage{Items: []VoucherView{}, Offset: q.Offset, Limit: q.Limit}

	switch strings.ToLower(strings.TrimSpace(q.Status)) {
	case "", "all":
	case "unused":
		f.Unused = true
	case "used":
		f.Used = true
	case "user":
		f.Used = true
		users, err := s.voucherRedeemers()
		if err != nil {
			return VoucherListPage{}, err
		}
		page.Users = users
		userID := strings.TrimSpace(q.UserID)
		if userID == "" {
			return page, nil
		}
		f.UserIDs = []string{userID}
	default:
		return VoucherListPage{}, fmt.Errorf("invalid status")
	}

	items, total, err := s.db.ListVouchersPage(f)
	if err != nil {
		return VoucherListPage{}, err
	}
	views, err := s.toVoucherViews(items)
	if err != nil {
		return VoucherListPage{}, err
	}
	page.Items = views
	page.Total = int(total)
	return page, nil
}

func (s *AuthService) voucherRedeemers() ([]VoucherRedeemer, error) {
	users, err := s.db.ListVoucherRedeemers()
	if err != nil {
		return nil, err
	}
	out := make([]VoucherRedeemer, 0, len(users))
	for _, u := range users {
		out = append(out, VoucherRedeemer{ID: u.ID, Username: u.Username})
	}
	return out, nil
}

func (s *AuthService) toVoucherViews(items []db.Voucher) ([]VoucherView, error) {
	userIDs := make([]string, 0, len(items))
	for _, v := range items {
		if v.UserID != "" {
			userIDs = append(userIDs, v.UserID)
		}
	}
	users, err := s.db.GetUsersByIDs(userIDs)
	if err != nil {
		return nil, err
	}
	names := make(map[string]string, len(users))
	for _, u := range users {
		names[u.ID] = u.Username
	}
	out := make([]VoucherView, 0, len(items))
	for _, v := range items {
		row := VoucherView{
			ID:        v.ID,
			Code:      v.Code,
			Quota:     v.Quota,
			Redeemed:  v.Used(),
			CreatedAt: v.CreatedAt,
		}
		if v.Used() {
			row.UserID = v.UserID
			row.RedeemedUsername = names[v.UserID]
			row.RedeemedAt = v.RedeemedAt
		}
		out = append(out, row)
	}
	return out, nil
}

func (s *AuthService) DeleteVoucher(id string) error {
	return s.db.DeleteUnusedVoucher(id)
}

type RegisterInput struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Code     string `json:"code"`
}

func (s *AuthService) Register(in RegisterInput) (LoginResult, error) {
	code, err := parseVoucherCode(in.Code)
	if err != nil {
		return LoginResult{}, err
	}
	u, err := s.newUser(in.Username, in.Password, false)
	if err != nil {
		return LoginResult{}, err
	}
	if err := s.db.RedeemVoucherForNewUser(code, &u); err != nil {
		if isUniqueViolation(err) {
			return LoginResult{}, fmt.Errorf("username already exists")
		}
		return LoginResult{}, err
	}
	return s.Login(u.Username, in.Password)
}

func (s *AuthService) RedeemVoucher(userID, code string) (db.User, int, error) {
	parsed, err := parseVoucherCode(code)
	if err != nil {
		return db.User{}, 0, err
	}
	added, err := s.db.RedeemVoucherForUser(parsed, userID)
	if err != nil {
		return db.User{}, 0, err
	}
	u, err := s.getUser(userID)
	return u, added, err
}

func generateVoucherBatch(count, quota int) ([]db.Voucher, []string, error) {
	items := make([]db.Voucher, 0, count)
	codes := make([]string, 0, count)
	seen := make(map[string]struct{}, count)
	for len(items) < count {
		code, err := generateVoucherCode()
		if err != nil {
			return nil, nil, err
		}
		if _, ok := seen[code]; ok {
			continue
		}
		seen[code] = struct{}{}
		items = append(items, db.Voucher{
			ID:    uuid.NewString(),
			Code:  code,
			Quota: quota,
		})
		codes = append(codes, code)
	}
	return items, codes, nil
}

func generateVoucherCode() (string, error) {
	raw := make([]byte, voucherRawLen)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	out := make([]byte, voucherRawLen)
	for i, b := range raw {
		out[i] = voucherAlphabet[int(b)%len(voucherAlphabet)]
	}
	return string(out[:4]) + "-" + string(out[4:]), nil
}

func parseVoucherCode(code string) (string, error) {
	var b strings.Builder
	for _, r := range strings.ToUpper(strings.TrimSpace(code)) {
		if r == '-' || unicode.IsSpace(r) {
			continue
		}
		b.WriteRune(r)
	}
	n := b.String()
	if len(n) != voucherRawLen {
		return "", db.ErrVoucherInvalid
	}
	for _, r := range n {
		if !strings.ContainsRune(voucherAlphabet, r) {
			return "", db.ErrVoucherInvalid
		}
	}
	return n[:4] + "-" + n[4:], nil
}
