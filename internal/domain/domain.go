package domain

import (
	"errors"
	"strings"
)

var (
	ErrNotFound     = errors.New("not found")
	ErrNoMatch      = errors.New("no recognition match")
	ErrInvalidInput = errors.New("invalid input")
)

type Track struct {
	ID     int64
	Title  string
	Artist string
	ISRC   string
}

// Key normaliza para comparar "The Killers" con " the killers ".
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
	Season       int // 0 si no aplica
	Episode      int // 0 si no aplica
	TimestampSec int // 0 si no se sabe
	Note         string
}
