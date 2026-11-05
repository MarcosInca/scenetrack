package recognition

import (
	"context"
	"fmt"
	"io"

	"scenetrack/internal/domain"
)

// Recognizer es lo que el servicio necesita. No le importa quién lo implemente.
type Recognizer interface {
	Identify(ctx context.Context, audio io.Reader, contentType string) (domain.Track, error)
}

// Mock sirve para desarrollar sin depender de ninguna API externa.
type Mock struct{}

func NewMock() *Mock { return &Mock{} }

func (m *Mock) Identify(ctx context.Context, audio io.Reader, _ string) (domain.Track, error) {
	if err := ctx.Err(); err != nil {
		return domain.Track{}, err
	}
	// Consumimos el cuerpo: así también se aplica el límite de tamaño del handler.
	n, err := io.Copy(io.Discard, audio)
	if err != nil {
		return domain.Track{}, fmt.Errorf("reading audio: %w", err)
	}
	if n == 0 {
		return domain.Track{}, fmt.Errorf("%w: empty audio", domain.ErrInvalidInput)
	}
	return domain.Track{Title: "Demo Song", Artist: "Demo Artist"}, nil
}
