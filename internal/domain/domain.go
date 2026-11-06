package domain

import (
	"errors"
	"strings"
)

var (
	ErrNoMatch      = errors.New("canción no reconocida")
	ErrInvalidInput = errors.New("entrada inválida")
)

type Track struct {
	Title  string
	Artist string
}

// Key normaliza artista y título para buscar en la base.
func (t Track) Key() string {
	return strings.ToLower(strings.TrimSpace(t.Artist)) + "|" +
		strings.ToLower(strings.TrimSpace(t.Title))
}

type WorkKind string

const (
	KindMovie   WorkKind = "movie"
	KindSeries  WorkKind = "series"
	KindTrailer WorkKind = "trailer"
)

type Work struct {
	ID        int64
	Kind      WorkKind
	Title     string
	Year      int
	PosterURL string
}

type Usage struct {
	ID           int64
	Work         Work
	Season       int
	Episode      int
	TimestampSec int
	Note         string
}
