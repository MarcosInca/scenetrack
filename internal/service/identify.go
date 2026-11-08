package service

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"time"

	"scenetrack/internal/domain"
	"scenetrack/internal/recognition"
)

// UsageRepository busca dónde se usó una canción.
type UsageRepository interface {
	UsagesByTrack(ctx context.Context, t domain.Track) ([]domain.Usage, error)
}

type Service struct {
	rec  recognition.Recognizer
	repo UsageRepository
}

func New(rec recognition.Recognizer, repo UsageRepository) *Service {
	return &Service{rec: rec, repo: repo}
}

// Result es la canción reconocida y sus usos.
type Result struct {
	Track  domain.Track
	Usages []domain.Usage
}

func (s *Service) Identify(ctx context.Context, audio io.Reader, contentType string) (Result, error) {
	start := time.Now()
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	track, err := s.rec.Identify(ctx, audio, contentType)
	if err != nil {
		return Result{}, fmt.Errorf("reconocer: %w", err)
	}
	usages, err := s.repo.UsagesByTrack(ctx, track)
	if err != nil {
		return Result{}, fmt.Errorf("buscar usos: %w", err)
	}

	// Resumen del pedido: lo que devolvió AudD y cuántos usos hay.
	slog.Info("identify",
		"artista", track.Artist,
		"titulo", track.Title,
		"usos", len(usages),
		"ms", time.Since(start).Milliseconds(),
	)
	return Result{Track: track, Usages: usages}, nil
}
