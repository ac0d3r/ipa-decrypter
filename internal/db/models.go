package db

import (
	"fmt"
	"time"
)

type Account struct {
	ID         string    `gorm:"primaryKey" json:"id"`
	Email      string    `json:"email"`
	Label      string    `json:"label"`
	Region     string    `json:"region"` // storefront country code, e.g. us/jp/cn
	StoreFront string    `json:"store_front,omitempty"`
	HomeDir    string    `json:"home_dir"`
	Status     string    `json:"status"` // unknown|logged_in|logged_out|needs_2fa|error
	LastError  string    `json:"last_error,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

type AccountRow struct {
	Account
	PasswordEnc string `gorm:"column:password_enc" json:"-"`
}

func (AccountRow) TableName() string { return "accounts" }

type Device struct {
	ID        string    `gorm:"primaryKey" json:"id"`
	Name      string    `json:"name"`
	Host      string    `json:"host"`   // IP or hostname
	Port      int       `json:"port"`   // unfaird port, default 6347
	Status    string    `json:"status"` // unknown|online|offline
	LastError string    `json:"last_error,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (d Device) BaseURL() string {
	return fmt.Sprintf("http://%s:%d", d.Host, d.Port)
}

type Task struct {
	ID                string    `gorm:"primaryKey" json:"id"`
	BundleID          string    `json:"bundle_id"`
	AppName           string    `json:"app_name"`
	Region            string    `json:"region"`
	ExternalVersionID string    `json:"external_version_id,omitempty"`
	DisplayVersion    string    `json:"display_version,omitempty"`
	IconURL           string    `json:"icon_url,omitempty"`
	SizeBytes         int64     `json:"size_bytes,omitempty"`
	AppID             int64     `json:"app_id,omitempty"`
	Status            string    `json:"status"` // queued|downloading|decrypting|completed|failed
	Error             string    `json:"error,omitempty"`
	AccountID         string    `json:"account_id,omitempty"`
	DeviceID          string    `json:"device_id,omitempty"`
	OutputPath        string    `json:"output_path,omitempty"`    // decrypted IPA
	EncryptedPath     string    `json:"encrypted_path,omitempty"` // downloaded encrypted IPA
	Logs              []string  `gorm:"serializer:json" json:"logs"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

type Secret struct {
	Key   string `gorm:"primaryKey"`
	Value string
}

func (Secret) TableName() string { return "secrets" }

// AdminUser is the web admin panel operator (not an Apple ID).
type AdminUser struct {
	ID               string     `gorm:"primaryKey" json:"id"`
	Username         string     `gorm:"uniqueIndex;size:64" json:"username"`
	PasswordHash     string     `json:"-"`
	SessionToken     string     `gorm:"index;size:64" json:"-"`
	SessionExpiresAt *time.Time `json:"expires_at,omitempty"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
}

func (AdminUser) TableName() string { return "admin_users" }

type RegionInfo struct {
	Code  string `json:"code"`
	Label string `json:"label"`
}

func normalizeTaskLogs(t *Task) {
	if t.Logs == nil {
		t.Logs = []string{}
	}
}
