package httpapi

import (
	"encoding/json"
	"errors"
	"log/slog"
	"mime"
	"net/http"
	"strings"

	"scenetrack/internal/domain"
	"scenetrack/internal/service"
)

const maxAudioBytes = 5 << 20 // 5 MB

type Handler struct{ svc *service.Service }

func New(svc *service.Service) *Handler { return &Handler{svc: svc} }

// Routes registra la API y sirve la carpeta web.
func (h *Handler) Routes(webDir string) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /v1/identify", h.identify)
	mux.Handle("/", http.FileServer(http.Dir(webDir)))
	return mux
}

// Lo que ve el cliente (separado del dominio).
type usageJSON struct {
	WorkTitle string `json:"work_title"`
	Kind      string `json:"kind"`
	Year      int    `json:"year,omitempty"`
	Season    int    `json:"season,omitempty"`
	Episode   int    `json:"episode,omitempty"`
	Note      string `json:"note,omitempty"`
}

type identifyJSON struct {
	Title  string      `json:"title"`
	Artist string      `json:"artist"`
	Usages []usageJSON `json:"usages"`
}

func (h *Handler) identify(w http.ResponseWriter, r *http.Request) {
	mt, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || !strings.HasPrefix(mt, "audio/") {
		writeError(w, http.StatusUnsupportedMediaType, "se esperaba audio")
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxAudioBytes) // corta si pesa de más

	res, err := h.svc.Identify(r.Context(), r.Body, mt)
	if err != nil {
		fail(w, err)
		return
	}

	out := identifyJSON{Title: res.Track.Title, Artist: res.Track.Artist, Usages: []usageJSON{}}
	for _, u := range res.Usages {
		out.Usages = append(out.Usages, usageJSON{
			WorkTitle: u.Work.Title,
			Kind:      string(u.Work.Kind),
			Year:      u.Work.Year,
			Season:    u.Season,
			Episode:   u.Episode,
			Note:      u.Note,
		})
	}
	writeJSON(w, http.StatusOK, out)
}

// fail traduce el error a un código HTTP.
func fail(w http.ResponseWriter, err error) {
	var tooBig *http.MaxBytesError
	switch {
	case errors.As(err, &tooBig):
		writeError(w, http.StatusRequestEntityTooLarge, "audio muy pesado")
	case errors.Is(err, domain.ErrInvalidInput):
		writeError(w, http.StatusBadRequest, "audio inválido")
	case errors.Is(err, domain.ErrNoMatch):
		writeError(w, http.StatusNotFound, "no reconocí la canción")
	default:
		slog.Error("identify", "err", err) // detalle solo en el log
		writeError(w, http.StatusInternalServerError, "error interno")
	}
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}
