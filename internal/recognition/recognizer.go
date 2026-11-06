package recognition

import (
	"context"
	"io"

	"scenetrack/internal/domain"
)

// Recognizer identifica una canción a partir de audio.
type Recognizer interface {
	Identify(ctx context.Context, audio io.Reader, contentType string) (domain.Track, error)
}

// Mock devuelve siempre la misma canción.
type Mock struct{}

func NewMock() Mock { return Mock{} }

func (Mock) Identify(_ context.Context, audio io.Reader, _ string) (domain.Track, error) {
	n, err := io.Copy(io.Discard, audio)
	if err != nil {
		return domain.Track{}, err
	}
	if n == 0 {
		return domain.Track{}, domain.ErrInvalidInput
	}
	return domain.Track{Title: "Demo Song", Artist: "Demo Artist"}, nil
}
