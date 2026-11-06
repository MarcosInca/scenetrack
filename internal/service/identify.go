package service

import (
	"context"
	"fmt"
	"io"
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
	return Result{Track: track, Usages: usages}, nil
}
