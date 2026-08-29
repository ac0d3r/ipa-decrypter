package config

import (
	"os"
	"path/filepath"
	"strconv"
)

type Config struct {
	ListenAddr         string
	WebDir             string
	DataDir            string
	IPATOOLBin         string
	KeychainPassphrase string
	DBPath             string
	AccountsDir        string
	IPAsDir            string
	DecryptedDir       string
}

func Load() (Config, error) {
	dataDir := env("DATA_DIR", "./data")
	cfg := Config{
		ListenAddr: env("LISTEN_ADDR", ":8080"),
		DataDir:    dataDir,
		WebDir:     env("WEB_DIR", "./web/dist"),
		IPATOOLBin: env("IPATOOL_BIN", "ipatool"),

		DBPath:       filepath.Join(dataDir, "ipa-decrypter.db"),
		AccountsDir:  filepath.Join(dataDir, "accounts"),
		IPAsDir:      filepath.Join(dataDir, "ipas"),
		DecryptedDir: filepath.Join(dataDir, "decrypted"),

		KeychainPassphrase: env("KEYCHAIN_PASSPHRASE", "ipatool-ipa-decrypter"),
	}
	for _, dir := range []string{cfg.DataDir, cfg.AccountsDir, cfg.IPAsDir, cfg.DecryptedDir} {
		if err := os.MkdirAll(dir, 0o700); err != nil {
			return Config{}, err
		}
	}
	return cfg, nil
}

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func EnvInt(key string, fallback int) int {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return fallback
	}
	return n
}
