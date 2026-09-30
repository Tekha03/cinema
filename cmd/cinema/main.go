package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Tekha03/cinema/internal/config"
	"github.com/Tekha03/cinema/internal/httpapi"
	"github.com/Tekha03/cinema/migrations"

	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	slog.SetDefault(slog.New(
		slog.NewJSONHandler(os.Stdout, nil),
	))

	if err := run(); err != nil {
		slog.Error("application stopped", "error", err)
		os.Exit(1)
	}
}

func run() error {
	command := "serve"
	if len(os.Args) == 2 {
		command = os.Args[1]
	}
	if len(os.Args) > 2 || (command != "serve" && command != "migrate") {
		return errors.New("usage: cinema [serve|migrate]")
	}

	cfg, err := config.Load()
	if err != nil {
		return err
	}

	slog.SetDefault(slog.New(
		slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			Level: cfg.LogLevel,
		}),
	))

	ctx, stop := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)
	defer stop()

	poolConfig, err := pgxpool.ParseConfig(cfg.DatabaseURL)
	if err != nil {
		// Не выводим строку подключения, которая содержит пароль.
		return errors.New("invalid DATABASE_URL")
	}

	poolConfig.MaxConns = cfg.DBMaxConns
	poolConfig.ConnConfig.ConnectTimeout = 5 * time.Second

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return errors.New("cannot create PostgreSQL connection pool")
	}
	defer pool.Close()

	pingCtx, cancelPing := context.WithTimeout(ctx, 10*time.Second)
	err = pool.Ping(pingCtx)
	cancelPing()
	if err != nil {
		return fmt.Errorf("PostgreSQL connection failed: %w", err)
	}

	if command == "migrate" {
		migrationCtx, cancel := context.WithTimeout(ctx, 60*time.Second)
		defer cancel()

		return migrations.Up(migrationCtx, pool)
	}

	server := &http.Server{
		Addr:              net.JoinHostPort("", cfg.Port),
		Handler:           httpapi.New(pool),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
		ErrorLog: slog.NewLogLogger(
			slog.Default().Handler(),
			slog.LevelError,
		),
	}

	serverErrors := make(chan error, 1)

	go func() {
		slog.Info("HTTP server starting", "address", server.Addr)
		serverErrors <- server.ListenAndServe()
	}()

	select {
	case err := <-serverErrors:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err

	case <-ctx.Done():
		slog.Info("shutdown requested")
	}

	shutdownCtx, cancelShutdown := context.WithTimeout(
		context.Background(),
		10*time.Second,
	)
	defer cancelShutdown()

	if err := server.Shutdown(shutdownCtx); err != nil {
		_ = server.Close()
		return fmt.Errorf("graceful shutdown failed: %w", err)
	}

	slog.Info("HTTP server stopped")
	return nil
}
