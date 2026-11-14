package service

import (
	"context"
	"errors"
	"strings"
	"testing"

	"scenetrack/internal/domain"
)

func TestSubmit(t *testing.T) {
	valid := domain.Submission{Artist: "Queen", Title: "Brighton Rock", Kind: "movie", Work: "Baby Driver", Year: 2017}
	with := func(edit func(*domain.Submission)) domain.Submission {
		s := valid
		edit(&s)
		return s
	}

	tests := []struct {
		name    string
		sub     domain.Submission
		wantErr bool
	}{
		{"válido", valid, false},
		{"serie con temporada", with(func(s *domain.Submission) { s.Kind = "series"; s.Season = 1; s.Episode = 4 }), false},
		{"sin título", with(func(s *domain.Submission) { s.Title = "   " }), true},
		{"tipo desconocido", with(func(s *domain.Submission) { s.Kind = "libro" }), true},
		{"año imposible", with(func(s *domain.Submission) { s.Year = 1500 }), true},
		{"temporada negativa", with(func(s *domain.Submission) { s.Season = -1 }), true},
		{"nota muy larga", with(func(s *domain.Submission) { s.Note = strings.Repeat("a", 501) }), true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			repo := &fakeRepo{}
			err := New(fakeRec{}, repo).Submit(context.Background(), tc.sub)

			if tc.wantErr {
				if !errors.Is(err, domain.ErrInvalidSubmission) {
					t.Fatalf("error = %v, quería ErrInvalidSubmission", err)
				}
				if len(repo.saved) != 0 {
					t.Error("no debía guardar nada")
				}
				return
			}
			if err != nil || len(repo.saved) != 1 {
				t.Errorf("error = %v, guardados = %d; quería nil y 1", err, len(repo.saved))
			}
		})
	}
}
