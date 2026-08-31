package main

import (
	"context"
	"log"
	"net/http"

	"github.com/vpn-project/backend/internal/auth"
	"github.com/vpn-project/backend/internal/config"
	"github.com/vpn-project/backend/internal/nodes"
	"github.com/vpn-project/backend/internal/server"
	"github.com/vpn-project/backend/internal/storage"
	"github.com/vpn-project/backend/internal/users"
)

func main() {
	cfg := config.Load()

	ctx := context.Background()

	if err := storage.MigrateUp(cfg.DatabaseURL); err != nil {
		log.Fatalf("migration failed: %v", err)
	}
	log.Println("database migrations applied")

	st, err := storage.New(ctx, cfg)
	if err != nil {
		log.Fatalf("storage init failed: %v", err)
	}
	defer st.Close()

	userRepo := users.NewRepository(st.Postgres.Pool)
	nodeRepo := nodes.NewRepository(st.Postgres.Pool)
	tokenService := auth.NewTokenService(cfg.JWTSecret)

	srv := server.New(cfg, st, userRepo, nodeRepo, tokenService)

	port := cfg.Port
	if port == "" {
		port = "8080"
	}

	log.Printf("VPN panel starting on :%s", port)
	if err := http.ListenAndServe(":"+port, srv.Handler()); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}
