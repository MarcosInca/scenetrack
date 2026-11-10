package httpapi

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"scenetrack/internal/domain"
	"scenetrack/internal/service"
)

type fakeRec struct{ err error }

func (f fakeRec) Identify(context.Context, io.Reader, string) (domain.Track, error) {
	return domain.Track{Title: "Brighton Rock", Artist: "Queen"}, f.err
}

type fakeRepo struct{}

func (fakeRepo) UsagesByTrack(context.Context, domain.Track) ([]domain.Usage, error) {
	return []domain.Usage{{Kind: "movie", Work: "Baby Driver", Year: 2017}}, nil
}

// post hace un POST a /v1/identify y devuelve la respuesta.
func post(h http.Handler, contentType string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, "/v1/identify", strings.NewReader("audio"))
	req.Header.Set("Content-Type", contentType)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func TestIdentifyStatus(t *testing.T) {
	tests := []struct {
		name        string
		rec         fakeRec
		contentType string
		want        int
	}{
		{"ok", fakeRec{}, "audio/webm", http.StatusOK},
		{"no es audio", fakeRec{}, "text/plain", http.StatusUnsupportedMediaType},
		{"no reconocida", fakeRec{err: domain.ErrNoMatch}, "audio/webm", http.StatusNotFound},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h := New(service.New(tc.rec, fakeRepo{}), false).Routes(".")
			if got := post(h, tc.contentType).Code; got != tc.want {
				t.Errorf("status = %d, quería %d", got, tc.want)
			}
		})
	}
}

func TestRateLimit(t *testing.T) {
	h := New(service.New(fakeRec{}, fakeRepo{}), false).Routes(".")
	var last int
	for i := 0; i <= perMinute; i++ { // un pedido más que el máximo
		last = post(h, "audio/webm").Code
	}
	if last != http.StatusTooManyRequests {
		t.Errorf("status = %d, quería 429", last)
	}
}
