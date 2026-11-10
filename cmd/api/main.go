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

	"github.com/jackc/pgx/v5/pgxpool"

	"scenetrack/internal/recognition"
	"scenetrack/internal/repository/postgres"
	"scenetrack/internal/service"
	"scenetrack/internal/transport/httpapi"
)

func main() {
	if err := run(); err != nil {
		slog.Error("fatal", "err", err)
		os.Exit(1)
	}
}

func run() error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	pool, err := connect(ctx, os.Getenv("DATABASE_URL"))
	if err != nil {
		return err
	}
	defer pool.Close()

	svc := service.New(newRecognizer(), postgres.NewUsageRepo(pool))
	srv := &http.Server{
		Addr:              ":8080",
		Handler:           httpapi.New(svc).Routes("web"),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second, // mayor que el timeout del servicio
	}

	errCh := make(chan error, 1)
	go func() {
		slog.Info("escuchando", "addr", srv.Addr)
		errCh <- srv.ListenAndServe()
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done(): // Ctrl+C
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		return srv.Shutdown(shutdownCtx)
	}
}

// connect abre el pool de Postgres y verifica que responda.
func connect(ctx context.Context, dsn string) (*pgxpool.Pool, error) {
	if dsn == "" {
		return nil, errors.New("falta DATABASE_URL")
	}
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return nil, err
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, err
	}
	slog.Info("datos: postgres")
	return pool, nil
}

// newRecognizer usa AudD si hay token; si no, el mock.
func newRecognizer() recognition.Recognizer {
	if token := os.Getenv("AUDD_API_TOKEN"); token != "" {
		slog.Info("reconocimiento: AudD")
		return recognition.NewAudD(token)
	}
	slog.Info("reconocimiento: mock (falta AUDD_API_TOKEN)")
	return recognition.NewMock()
}
