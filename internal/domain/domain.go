package domain

import "errors"

var (
	ErrNoMatch      = errors.New("canción no reconocida")
	ErrInvalidInput = errors.New("entrada inválida")
)

// Track es una canción reconocida.
type Track struct {
	Title  string
	Artist string
}

// Usage es un lugar donde suena una canción (va tal cual al cliente).
type Usage struct {
	Kind    string `json:"kind"` // movie, series o trailer
	Work    string `json:"work_title"`
	Year    int    `json:"year,omitempty"`
	Season  int    `json:"season,omitempty"`
	Episode int    `json:"episode,omitempty"`
	Note    string `json:"note,omitempty"`
}
