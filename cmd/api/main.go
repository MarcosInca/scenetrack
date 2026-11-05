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

	"github.com/jackc/pgx/v5/pgxpool"

	"scenetrack/internal/domain"
	"scenetrack/internal/recognition"
	"scenetrack/internal/repository/memory"
	"scenetrack/internal/repository/postgres"
	"scenetrack/internal/service"
	"scenetrack/internal/transport/httpapi"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	if err := run(logger); err != nil {
		logger.Error("fatal", "err", err)
		os.Exit(1)
	}
}

func run(logger *slog.Logger) error {
	var repo service.UsageRepository

	if dsn := os.Getenv("DATABASE_URL"); dsn != "" {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		pool, err := pgxpool.New(ctx, dsn)
		if err != nil {
			return fmt.Errorf("create pool: %w", err)
		}
		defer pool.Close()
		if err := pool.Ping(ctx); err != nil {
			return fmt.Errorf("connect database: %w", err)
		}
		repo = postgres.NewUsageRepo(pool)
		logger.Info("using postgres")
	} else {
		mem := memory.NewUsageRepo()
		mem.Add(
			domain.Track{Title: "Demo Song", Artist: "Demo Artist"},
			domain.Usage{
				Work: domain.Work{Kind: domain.KindMovie, Title: "Pelicula Demo", Year: 2001},
				Note: "Escena de la persecución final",
			},
		)
		repo = mem
		logger.Info("using in-memory data")
	}

	svc := service.New(recognition.NewMock(), repo)
	h := httpapi.New(svc, logger)

	root := http.NewServeMux()
	root.Handle("/v1/", h.Routes())
	root.Handle("/healthz", h.Routes())
	root.Handle("/", http.FileServer(http.Dir("web")))

	srv := &http.Server{
		Addr:              ":8080",
		Handler:           root,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go func() {
		logger.Info("listening", "addr", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("server failed", "err", err)
			stop()
		}
	}()

	<-ctx.Done()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return srv.Shutdown(shutdownCtx)
}
