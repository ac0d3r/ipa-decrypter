package ipatool

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	jose "github.com/dvsekhvalnov/jose2go"
)

// StoredAccount is the JSON blob ipatool keeps in its keychain under key "account".
type StoredAccount struct {
	Email               string `json:"email"`
	Name                string `json:"name"`
	StoreFront          string `json:"storeFront"`
	DirectoryServicesID string `json:"directoryServicesIdentifier"`
}

type keyringItem struct {
	Key  string `json:"Key"`
	Data []byte `json:"Data"`
}

func (r Runner) ReadStoredAccount(homeDir string) (StoredAccount, error) {
	var out StoredAccount
	if homeDir == "" {
		return out, fmt.Errorf("home dir required")
	}
	path := filepath.Join(homeDir, ".ipatool", "account")
	raw, err := os.ReadFile(path)
	if err != nil {
		return out, fmt.Errorf("read keyring file: %w", err)
	}
	payload, _, err := jose.Decode(string(raw), r.Passphrase)
	if err != nil {
		return out, fmt.Errorf("decrypt keyring: %w", err)
	}
	var item keyringItem
	if err := json.Unmarshal([]byte(payload), &item); err != nil {
		return out, fmt.Errorf("decode keyring item: %w", err)
	}
	if err := json.Unmarshal(item.Data, &out); err != nil {
		return out, fmt.Errorf("decode account: %w", err)
	}
	return out, nil
}
