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

type fakeRepo struct{ saved []domain.Submission }

func (*fakeRepo) UsagesByTrack(context.Context, domain.Track) ([]domain.Usage, error) {
	return []domain.Usage{{Kind: "movie", Work: "Baby Driver", Year: 2017}}, nil
}

func (f *fakeRepo) AddSubmission(_ context.Context, s domain.Submission) error {
	f.saved = append(f.saved, s)
	return nil
}

// do hace un POST a la ruta indicada y devuelve la respuesta.
func do(h http.Handler, path, contentType, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
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
		{"sin cuota", fakeRec{err: domain.ErrUnavailable}, "audio/webm", http.StatusServiceUnavailable},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h := New(service.New(tc.rec, &fakeRepo{}), false).Routes(".")
			if got := do(h, "/v1/identify", tc.contentType, "audio").Code; got != tc.want {
				t.Errorf("status = %d, quería %d", got, tc.want)
			}
		})
	}
}

func TestSubmit(t *testing.T) {
	tests := []struct {
		name      string
		body      string
		want      int
		wantSaved int
	}{
		{"válido", `{"artist":"Queen","title":"Brighton Rock","kind":"movie","work":"Baby Driver","year":2017}`, 201, 1},
		{"faltan datos", `{"artist":"Queen"}`, 400, 0},
		{"json roto", `{`, 400, 0},
		{"bot (campo trampa)", `{"artist":"Q","title":"T","kind":"movie","work":"W","website":"http://spam"}`, 201, 0},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			repo := &fakeRepo{}
			h := New(service.New(fakeRec{}, repo), false).Routes(".")
			if got := do(h, "/v1/submissions", "application/json", tc.body).Code; got != tc.want {
				t.Errorf("status = %d, quería %d", got, tc.want)
			}
			if len(repo.saved) != tc.wantSaved {
				t.Errorf("guardados = %d, quería %d", len(repo.saved), tc.wantSaved)
			}
		})
	}
}

func TestRateLimit(t *testing.T) {
	routes := []struct {
		name, path, contentType, body string
		max                           int
	}{
		{"reconocer", "/v1/identify", "audio/webm", "audio", identifyPerMin},
		{"enviar", "/v1/submissions", "application/json", `{"website":"bot"}`, submitPerMin},
	}
	for _, rt := range routes {
		t.Run(rt.name, func(t *testing.T) {
			h := New(service.New(fakeRec{}, &fakeRepo{}), false).Routes(".")
			var last int
			for i := 0; i <= rt.max; i++ { // un pedido más que el máximo
				last = do(h, rt.path, rt.contentType, rt.body).Code
			}
			if last != http.StatusTooManyRequests {
				t.Errorf("status = %d, quería 429", last)
			}
		})
	}
}
