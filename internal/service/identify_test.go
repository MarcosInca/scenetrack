package service

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"

	"scenetrack/internal/domain"
)

type fakeRec struct {
	track domain.Track
	err   error
}

func (f fakeRec) Identify(context.Context, io.Reader, string) (domain.Track, error) {
	return f.track, f.err
}

type fakeRepo struct {
	usages []domain.Usage
	err    error
}

func (f fakeRepo) UsagesByTrack(context.Context, domain.Track) ([]domain.Usage, error) {
	return f.usages, f.err
}

func TestIdentify(t *testing.T) {
	song := domain.Track{Title: "Brighton Rock", Artist: "Queen"}
	use := domain.Usage{Kind: "movie", Work: "Baby Driver", Year: 2017}
	boom := errors.New("boom")

	tests := []struct {
		name       string
		rec        fakeRec
		repo       fakeRepo
		wantUsages int
		wantErr    error
	}{
		{"con usos", fakeRec{track: song}, fakeRepo{usages: []domain.Usage{use}}, 1, nil},
		{"sin usos devuelve lista vacía", fakeRec{track: song}, fakeRepo{}, 0, nil},
		{"canción no reconocida", fakeRec{err: domain.ErrNoMatch}, fakeRepo{}, 0, domain.ErrNoMatch},
		{"falla la base", fakeRec{track: song}, fakeRepo{err: boom}, 0, boom},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			res, err := New(tc.rec, tc.repo).Identify(context.Background(), strings.NewReader("audio"), "audio/webm")
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("error = %v, quería %v", err, tc.wantErr)
			}
			if err != nil {
				return
			}
			if res.Usages == nil || len(res.Usages) != tc.wantUsages {
				t.Errorf("usos = %v, quería %d (y nunca nil)", res.Usages, tc.wantUsages)
			}
		})
	}
}
