package service

import (
	"context"
	"errors"
	"fmt"
	"io"
	"time"

	"scenetrack/internal/domain"
	"scenetrack/internal/recognition"
)

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

type IdentifyResult struct {
	Track  domain.Track
	Usages []domain.Usage
}

func (s *Service) Identify(ctx context.Context, audio io.Reader, contentType string) (IdentifyResult, error) {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	track, err := s.rec.Identify(ctx, audio, contentType)
	if err != nil {
		return IdentifyResult{}, fmt.Errorf("recognize: %w", err)
	}

	usages, err := s.repo.UsagesByTrack(ctx, track)
	if err != nil && !errors.Is(err, domain.ErrNotFound) {
		return IdentifyResult{}, fmt.Errorf("find usages: %w", err)
	}
	// Si no hay usos, no es un error: devolvemos la canción con lista vacía.
	return IdentifyResult{Track: track, Usages: usages}, nil
}
