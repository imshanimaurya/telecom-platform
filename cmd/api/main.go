package main

import (
	"context"
	"errors"
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
)

func main() {
	// Root context that cancels on shutdown
	rootCtx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	cfg, err := config.Load()
	if err != nil {
		slog.Error("config load failed", "err", err)
		os.Exit(1)
	}

	log := logger.New(cfg.App.Env)
	slog.SetDefault(log)

	if cfg.App.Env == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	authManager, err := auth.NewManager(cfg.Auth)
	if err != nil {
		log.Error("auth init failed", "err", err)
		os.Exit(1)
	}

	db, err := utils.OpenPostgres(rootCtx, "pgx", cfg.PostgresDSN(), utils.PostgresPoolConfig{})
	if err != nil {
		log.Error("postgres init failed", "err", err)
		os.Exit(1)
	}
	defer db.Close()

	rdb, err := utils.OpenRedis(rootCtx, utils.RedisConfig{Addr: cfg.RedisAddr()})
	if err != nil {
		log.Error("redis init failed", "err", err)
		os.Exit(1)
	}
	defer rdb.Close()

	// Gin router
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(logger.Middleware(log))

	// Attach shared deps to context (no globals)
	r.Use(func(c *gin.Context) {
		c.Set("db", db)
		c.Set("redis", rdb)
		c.Next()
	})

	// Route groups
	registerPublicRoutes(r) // webhooks, health
	registerAuthRoutes(r, authManager)
	registerProtectedRoutes(r, auth.RequireAccessToken(authManager))

	srv := &http.Server{
		Addr:              cfg.HTTPAddr(),
		Handler:           r,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	go func() {
		log.Info("api listening", "addr", srv.Addr, "env", cfg.App.Env)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Error("http server failed", "err", err)
			stop()
		}
	}()

	<-rootCtx.Done()
	log.Info("shutdown initiated")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Error("http shutdown failed", "err", err)
	}

	_ = logger.ShutdownFlush(shutdownCtx, 2*time.Second)
}
