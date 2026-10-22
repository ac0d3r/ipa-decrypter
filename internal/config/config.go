package config

import (
	"os"
	"path/filepath"
)

type Config struct {
	ListenAddr         string
	WebDir             string
	DataDir            string
	IPATOOLBin         string
	KeychainPassphrase string
	AdminUsername      string
	AdminPassword      string
	DBPath             string
	AccountsDir        string
	IPAsDir            string
	DecryptedDir       string
	// FairPlay transform helper (C) and its constant-data table. When both
	// exist on disk, decryption prefers local FairPlay decryption and only
	// falls back to unfaird devices if the account has no credential.
	FairplayTransformBin   string
	FairplayTransformConst string
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

		FairplayTransformBin:   env("FAIRPLAY_TRANSFORM_BIN", filepath.Join("bin", "transform")),
		FairplayTransformConst: env("FAIRPLAY_TRANSFORM_CONST", filepath.Join(".cache", "fairplay-decrypt", "transform_const_data.bin")),

		KeychainPassphrase: env("KEYCHAIN_PASSPHRASE", "ipatool-ipa-decrypter"),
		AdminUsername:      env("ADMIN_USERNAME", "admin"),
		AdminPassword:      env("ADMIN_PASSWORD", "admin"),
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
