package main

import (
	"net/http"

	"github.com/ac0d3r/ipa-decrypter/internal/api"
	"github.com/ac0d3r/ipa-decrypter/internal/config"
	"github.com/ac0d3r/ipa-decrypter/internal/db"
	"github.com/ac0d3r/ipa-decrypter/internal/logger"
	"github.com/ac0d3r/ipa-decrypter/internal/services"
)

func main() {
	logger.Init()
	log := logger.WithComponent("server")

	cfg, err := config.Load()
	if err != nil {
		log.WithError(err).Fatal("load config failed")
	}
	database, err := db.Open(cfg.DBPath)
	if err != nil {
		log.WithError(err).Fatal("open database failed")
	}
	defer database.Close()

	svc := services.NewRegistry(cfg, database)
	if err := svc.Auth.EnsureBootstrapAdmin(); err != nil {
		log.WithError(err).Fatal("bootstrap admin failed")
	}

	srv := &api.Server{
		Registry: svc,
		WebDir:   cfg.WebDir,
	}

	log.WithFields(map[string]any{
		"listen": cfg.ListenAddr,
		"data":   cfg.DataDir,
	}).Info("ipa-decrypter starting")

	if err := http.ListenAndServe(cfg.ListenAddr, srv.Handler()); err != nil {
		log.WithError(err).Fatal("http server stopped")
	}
}
