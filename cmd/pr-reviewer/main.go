package main

import (
	"context"
	"errors"
	"log"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/YusovID/pr-reviewer-service/internal/config"
	"github.com/YusovID/pr-reviewer-service/internal/repository/postgres"
	"github.com/YusovID/pr-reviewer-service/internal/service"
	myhttp "github.com/YusovID/pr-reviewer-service/internal/transport/http"
	"github.com/YusovID/pr-reviewer-service/pkg/logger/sl"
	"github.com/YusovID/pr-reviewer-service/pkg/logger/slogpretty"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	log := slogpretty.SetupLogger(cfg.Env)
	log.Info("starting pr-reviewer-service", slog.String("env", cfg.Env))

	db, err := postgres.NewDB(cfg.Postgres, log)
	if err != nil {
		log.Error("failed to init db", sl.Err(err))
		os.Exit(1)
	}

	defer func() {
		if err := db.Close(); err != nil {
			log.Error("db close failed", sl.Err(err))
		}
	}()

	teamRepo := postgres.NewTeamRepository(db, log)
	userRepo := postgres.NewUserRepository(db, log)
	prRepo := postgres.NewPullRequestRepository(db, log)

	teamService := service.NewTeamService(teamRepo)
	userService := service.NewUserService(userRepo)
	prService := service.NewPullRequestService(db, log, prRepo, prRepo, prRepo)

	handler := myhttp.NewServer(log, teamService, userService, prService)

	httpServer := &http.Server{
		Addr:         net.JoinHostPort(cfg.Server.Host, cfg.Server.Port),
		Handler:      handler.Routes(),
		ReadTimeout:  cfg.Server.Timeout,
		WriteTimeout: cfg.Server.Timeout,
		IdleTimeout:  cfg.Server.Timeout * 2,
	}

	go func() {
		log.Info("server started", slog.String("addr", httpServer.Addr))

		if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Error("server failed to start", sl.Err(err))
			stop()
		}
	}()

	<-ctx.Done()
	log.Info("stopping server...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		log.Error("server shutdown failed", sl.Err(err))
	}

	log.Info("server stopped")
}
