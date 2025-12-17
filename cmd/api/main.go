package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"telecom-platform/internal/auth"
	"telecom-platform/internal/config"
	"telecom-platform/pkg/logger"
	"telecom-platform/pkg/utils"

	"github.com/gin-gonic/gin"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/redis/go-redis/v9"
)

func main() {
	ctx := context.Background()

	cfg, err := config.Load()
	if err != nil {
		panic(err)
	}

	log := logger.New(cfg.App.Env)
	slog.SetDefault(log)

	authManager, err := auth.NewManager(cfg.Auth)
	if err != nil {
		log.Error("auth init failed", "err", err)
		panic(err)
	}

	db, err := utils.OpenPostgres(ctx, "pgx", cfg.PostgresDSN(), utils.PostgresPoolConfig{})
	if err != nil {
		log.Error("postgres init failed", "err", err)
		panic(err)
	}
	defer func() { _ = db.Close() }()

	rdb, err := utils.OpenRedis(ctx, utils.RedisConfig{Addr: cfg.RedisAddr()})
	if err != nil {
		log.Error("redis init failed", "err", err)
		panic(err)
	}
	defer func() { _ = rdb.Close() }()

	_ = db   // reserved for dependency injection wiring
	_ = rdb // reserved for dependency injection wiring

	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(logger.Middleware(log))

	registerRoutes(r, auth.RequireAccessToken(authManager))

	srv := &http.Server{
		Addr:              cfg.HTTPAddr(),
		Handler:           r,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		log.Info("api listening", "addr", srv.Addr, "env", cfg.App.Env)
		errCh <- srv.ListenAndServe()
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	select {
	case sig := <-stop:
		log.Info("shutdown signal received", "signal", sig.String())
	case err := <-errCh:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Error("server stopped unexpectedly", "err", err)
			panic(err)
		}
		log.Info("server stopped")
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Error("http shutdown failed", "err", err)
	}
	_ = logger.ShutdownFlush(shutdownCtx, 2*time.Second)
}
