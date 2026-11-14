package recognition

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"time"

	audd "github.com/AudDMusic/audd-go"

	"scenetrack/internal/domain"
)

// AudD reconoce canciones con el SDK oficial de audd.io.
type AudD struct{ client *audd.Client }

func NewAudD(token string) *AudD {
	return &AudD{client: audd.NewClient(token,
		audd.WithStandardTimeout(12*time.Second),
		audd.WithMaxAttempts(2), // los reintentos gastan cuota
	)}
}

func (a *AudD) Identify(ctx context.Context, audio io.Reader, contentType string) (domain.Track, error) {
	start := time.Now()
	data, err := io.ReadAll(audio) // bytes: nunca pasamos textos del usuario al SDK
	if err != nil {
		return domain.Track{}, err
	}
	if len(data) == 0 {
		return domain.Track{}, domain.ErrInvalidInput
	}

	result, err := a.client.RecognizeContext(ctx, data, nil)

	// Resumen de cada llamada a AudD.
	slog.Info("audd",
		"bytes", len(data),
		"tipo", contentType,
		"ms", time.Since(start).Milliseconds(),
		"coincidencia", result != nil,
	)

	switch {
	case errors.Is(err, audd.ErrQuota), errors.Is(err, audd.ErrRateLimit):
		return domain.Track{}, fmt.Errorf("%w: %w", domain.ErrUnavailable, err)
	case errors.Is(err, audd.ErrInvalidAudio):
		return domain.Track{}, fmt.Errorf("%w: %w", domain.ErrInvalidInput, err)
	case err != nil:
		return domain.Track{}, fmt.Errorf("audd: %w", err)
	case result == nil:
		return domain.Track{}, domain.ErrNoMatch // sin coincidencia
	}
	return domain.Track{Title: result.Title, Artist: result.Artist}, nil
}
