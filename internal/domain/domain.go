package domain

import "errors"

var (
	ErrNoMatch           = errors.New("canción no reconocida")
	ErrInvalidInput      = errors.New("entrada inválida")
	ErrInvalidSubmission = errors.New("datos inválidos")           // lo escrito en el formulario
	ErrUnavailable       = errors.New("reconocedor no disponible") // sin cuota o saturado
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

// Submission es una escena enviada por un usuario, pendiente de revisión.
type Submission struct {
	Artist  string `json:"artist"`
	Title   string `json:"title"`
	Kind    string `json:"kind"`
	Work    string `json:"work"`
	Year    int    `json:"year"`
	Season  int    `json:"season"`
	Episode int    `json:"episode"`
	Note    string `json:"note"`
}
