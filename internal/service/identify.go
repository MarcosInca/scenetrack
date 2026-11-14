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
	repo Repository
}

func New(rec recognition.Recognizer, repo Repository) *Service {
	return &Service{rec: rec, repo: repo}
}

// Result es lo que se devuelve al cliente (JSON).
type Result struct {
	Title  string         `json:"title"`
	Artist string         `json:"artist"`
	Usages []domain.Usage `json:"usages"`
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
	if usages == nil {
		usages = []domain.Usage{} // JSON: [] en vez de null
	}

	// Resumen del pedido: lo que devolvió AudD y cuántos usos hay.
	slog.Info("identify",
		"artista", track.Artist,
		"titulo", track.Title,
		"usos", len(usages),
		"ms", time.Since(start).Milliseconds(),
	)
	return Result{Title: track.Title, Artist: track.Artist, Usages: usages}, nil
}
