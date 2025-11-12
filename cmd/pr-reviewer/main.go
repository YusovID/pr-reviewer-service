package main

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/YusovID/pr-reviewer-service/internal/config"
	"github.com/YusovID/pr-reviewer-service/internal/repository/postgres"
	myhttp "github.com/YusovID/pr-reviewer-service/internal/transport/http"

	"github.com/YusovID/pr-reviewer-service/pkg/logger/slogpretty"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	if err := run(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}
}

func run(ctx context.Context) error {
	cfg := config.MustLoad()
	log := slogpretty.SetupLogger(cfg.Env)

	log.Info("starting pr-reviewer-service", slog.String("env", cfg.Env))

	errChan := make(chan error, 1)

	db, err := postgres.NewDB(cfg.Postgres, log)
	if err != nil {
		return fmt.Errorf("failed to init db: %v", err)
	}
	defer func() {
		err = db.DB().Close()
		if err != nil {
			errChan <- fmt.Errorf("db close failed: %v", err)
		}
	}()

	srv := myhttp.NewServer(db.DB(), log)
	httpServer := &http.Server{
		Addr:    net.JoinHostPort(cfg.Server.Host, cfg.Server.Port),
		Handler: srv.Routes(),
	}

	go startServer(log, httpServer, errChan)

	select {
	case err := <-errChan:
		return fmt.Errorf("http server error: %v", err)

	case <-ctx.Done():
		log.Info("stopping server...")
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("error shuting down http server: %v", err)
	}

	return nil
}

func startServer(log *slog.Logger, httpServer *http.Server, errChan chan error) {
	defer close(errChan)

	log.Info("service started", slog.String("addr", httpServer.Addr))

	if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		errChan <- fmt.Errorf("error listening and serving: %v", err)
	}
}
