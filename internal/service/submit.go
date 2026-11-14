package service

import (
	"context"
	"fmt"
	"strings"
	"unicode/utf8"

	"scenetrack/internal/domain"
)

// SubmissionRepository guarda envíos pendientes de revisión.
type SubmissionRepository interface {
	AddSubmission(ctx context.Context, s domain.Submission) error
}

// Repository junta todo lo que el servicio necesita de la base.
type Repository interface {
	UsageRepository
	SubmissionRepository
}

// Submit valida un envío y lo guarda como pendiente.
func (s *Service) Submit(ctx context.Context, sub domain.Submission) error {
	sub.Artist = strings.TrimSpace(sub.Artist)
	sub.Title = strings.TrimSpace(sub.Title)
	sub.Work = strings.TrimSpace(sub.Work)
	sub.Note = strings.TrimSpace(sub.Note)

	if err := validate(sub); err != nil {
		return err
	}
	if err := s.repo.AddSubmission(ctx, sub); err != nil {
		return fmt.Errorf("guardar envío: %w", err)
	}
	return nil
}

// invalid arma el error con un mensaje fijo (nunca repite lo que escribió el usuario).
func invalid(msg string) error { return fmt.Errorf("%w: %s", domain.ErrInvalidSubmission, msg) }

func validate(s domain.Submission) error {
	long := func(text string, max int) bool { return utf8.RuneCountInString(text) > max }
	switch {
	case s.Artist == "" || s.Title == "" || s.Work == "":
		return invalid("faltan la canción, el artista o la película/serie")
	case long(s.Artist, 200) || long(s.Title, 200) || long(s.Work, 200):
		return invalid("algún nombre es demasiado largo")
	case long(s.Note, 500):
		return invalid("la nota es demasiado larga (máximo 500 caracteres)")
	case s.Kind != "movie" && s.Kind != "series" && s.Kind != "trailer":
		return invalid("tipo desconocido")
	case s.Year != 0 && (s.Year < 1888 || s.Year > 2100):
		return invalid("el año no es válido")
	case s.Season < 0 || s.Season > 999 || s.Episode < 0 || s.Episode > 999:
		return invalid("la temporada o el episodio no son válidos")
	}
	return nil
}
