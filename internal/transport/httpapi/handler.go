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

const (
	maxAudioBytes = 5 << 20 // 5 MB
	maxFormBytes  = 8 << 10 // 8 KB
)

type Handler struct {
	svc    *service.Service
	lim    *limiter // reconocer
	subLim *limiter // enviar escenas
}

func New(svc *service.Service, trustProxy bool) *Handler {
	return &Handler{
		svc:    svc,
		lim:    newLimiter(identifyPerMin, identifyPerDay, trustProxy),
		subLim: newLimiter(submitPerMin, submitPerDay, trustProxy),
	}
}

// Routes registra la API (con límites de uso) y sirve la carpeta web.
func (h *Handler) Routes(webDir string) http.Handler {
	mux := http.NewServeMux()
	mux.Handle("POST /v1/identify", h.lim.wrap(h.identify))
	mux.Handle("POST /v1/submissions", h.subLim.wrap(h.submit))
	mux.Handle("/", http.FileServer(http.Dir(webDir)))
	return mux
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
	writeJSON(w, http.StatusOK, res)
}

func (h *Handler) submit(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxFormBytes)

	var body struct {
		domain.Submission
		Website string `json:"website"` // campo trampa: las personas no lo ven
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "formulario inválido")
		return
	}
	if body.Website != "" { // lo completó un bot: fingimos éxito y no guardamos nada
		writeJSON(w, http.StatusCreated, map[string]string{"status": "pending"})
		return
	}
	if err := h.svc.Submit(r.Context(), body.Submission); err != nil {
		fail(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]string{"status": "pending"})
}

// fail traduce el error a un código HTTP.
func fail(w http.ResponseWriter, err error) {
	var tooBig *http.MaxBytesError
	switch {
	case errors.As(err, &tooBig):
		writeError(w, http.StatusRequestEntityTooLarge, "audio muy pesado")
	case errors.Is(err, domain.ErrInvalidSubmission):
		writeError(w, http.StatusBadRequest, err.Error()) // el mensaje es fijo, sin datos del usuario
	case errors.Is(err, domain.ErrInvalidInput):
		writeError(w, http.StatusBadRequest, "audio inválido")
	case errors.Is(err, domain.ErrNoMatch):
		writeError(w, http.StatusNotFound, "no reconocí la canción")
	case errors.Is(err, domain.ErrUnavailable):
		slog.Error("reconocedor no disponible", "err", err) // revisá la cuota de AudD
		writeError(w, http.StatusServiceUnavailable, "el reconocedor no está disponible ahora, probá más tarde")
	default:
		slog.Error("pedido fallido", "err", err) // detalle solo en el log
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
