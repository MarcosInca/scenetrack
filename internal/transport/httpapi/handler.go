package httpapi

import (
	"encoding/json"
	"errors"
	"log/slog"
	"mime"
	"net/http"

	"scenetrack/internal/domain"
	"scenetrack/internal/service"
)

const maxAudioBytes = 5 << 20 // 5 MB

var allowedAudio = map[string]bool{
	"audio/webm": true,
	"audio/mpeg": true,
	"audio/wav":  true,
	"audio/mp4":  true,
	"audio/ogg":  true,
}

type Handler struct {
	svc *service.Service
	log *slog.Logger
}

func New(svc *service.Service, log *slog.Logger) *Handler {
	return &Handler{svc: svc, log: log}
}

func (h *Handler) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /v1/identify", h.identify)
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	return mux
}

// DTOs: lo que ve el cliente. Separados del dominio a propósito.
type usageDTO struct {
	WorkTitle string `json:"work_title"`
	Kind      string `json:"kind"`
	Year      int    `json:"year,omitempty"`
	Season    int    `json:"season,omitempty"`
	Episode   int    `json:"episode,omitempty"`
	Note      string `json:"note,omitempty"`
}

type identifyResponse struct {
	Title  string     `json:"title"`
	Artist string     `json:"artist"`
	Usages []usageDTO `json:"usages"`
}

func (h *Handler) identify(w http.ResponseWriter, r *http.Request) {
	mt, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || !allowedAudio[mt] {
		h.writeError(w, http.StatusUnsupportedMediaType, "unsupported content type")
		return
	}

	// Corta la lectura si el cliente manda más de maxAudioBytes.
	r.Body = http.MaxBytesReader(w, r.Body, maxAudioBytes)

	res, err := h.svc.Identify(r.Context(), r.Body, mt)
	if err != nil {
		h.handleError(w, err)
		return
	}

	out := identifyResponse{
		Title:  res.Track.Title,
		Artist: res.Track.Artist,
		Usages: make([]usageDTO, 0, len(res.Usages)),
	}
	for _, u := range res.Usages {
		out.Usages = append(out.Usages, usageDTO{
			WorkTitle: u.Work.Title,
			Kind:      string(u.Work.Kind),
			Year:      u.Work.Year,
			Season:    u.Season,
			Episode:   u.Episode,
			Note:      u.Note,
		})
	}
	h.writeJSON(w, http.StatusOK, out)
}

func (h *Handler) handleError(w http.ResponseWriter, err error) {
	var maxErr *http.MaxBytesError
	switch {
	case errors.As(err, &maxErr):
		h.writeError(w, http.StatusRequestEntityTooLarge, "audio too large")
	case errors.Is(err, domain.ErrInvalidInput):
		h.writeError(w, http.StatusBadRequest, "invalid audio")
	case errors.Is(err, domain.ErrNoMatch):
		h.writeError(w, http.StatusNotFound, "song not recognized")
	default:
		h.log.Error("identify failed", "err", err) // el detalle va al log, no al cliente
		h.writeError(w, http.StatusInternalServerError, "internal error")
	}
}

func (h *Handler) writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		h.log.Error("encode response", "err", err)
	}
}

func (h *Handler) writeError(w http.ResponseWriter, status int, msg string) {
	h.writeJSON(w, status, map[string]string{"error": msg})
}
