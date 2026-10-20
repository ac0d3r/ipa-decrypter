package main

import (
	"net/http"
	"time"

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
	defer func() { _ = database.Close() }()

	svc := services.NewRegistry(cfg, database)
	if err := svc.Auth.EnsureBootstrapAdmin(); err != nil {
		log.WithError(err).Fatal("bootstrap admin failed")
	}
	svc.Tasks.RecoverInterrupted()

	srv := &api.Server{
		Registry: svc,
		WebDir:   cfg.WebDir,
	}

	log.WithFields(map[string]any{
		"listen": cfg.ListenAddr,
		"data":   cfg.DataDir,
	}).Info("ipa-decrypter starting")

	httpServer := &http.Server{
		Addr:              cfg.ListenAddr,
		Handler:           srv.Engine(),
		ReadHeaderTimeout: 30 * time.Second,
		IdleTimeout:       120 * time.Second,
	}
	if err := httpServer.ListenAndServe(); err != nil {
		log.WithError(err).Fatal("http server stopped")
	}
}
