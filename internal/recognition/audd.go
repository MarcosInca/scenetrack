package recognition

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"time"

	"scenetrack/internal/domain"
)

const auddURL = "https://api.audd.io/"

// Extensión según formato, para que AudD lea el archivo.
var extensions = map[string]string{
	"audio/webm": ".webm",
	"audio/mp4":  ".m4a",
	"audio/ogg":  ".ogg",
	"audio/mpeg": ".mp3",
	"audio/wav":  ".wav",
}

// AudD reconoce canciones con la API de audd.io.
type AudD struct {
	token  string
	client *http.Client
}

func NewAudD(token string) *AudD {
	return &AudD{token: token, client: &http.Client{Timeout: 12 * time.Second}}
}

// Forma de la respuesta de AudD.
type auddResponse struct {
	Result *struct {
		Artist string `json:"artist"`
		Title  string `json:"title"`
	} `json:"result"`
	Error *struct {
		Code    int    `json:"error_code"`
		Message string `json:"error_message"`
	} `json:"error"`
}

func (a *AudD) Identify(ctx context.Context, audio io.Reader, contentType string) (domain.Track, error) {
	data, err := io.ReadAll(audio)
	if err != nil {
		return domain.Track{}, err
	}
	if len(data) == 0 {
		return domain.Track{}, domain.ErrInvalidInput
	}

	// Formulario multipart: token + archivo (bytes.Buffer no falla).
	var body bytes.Buffer
	form := multipart.NewWriter(&body)
	form.WriteField("api_token", a.token)
	file, _ := form.CreateFormFile("file", "clip"+extensions[contentType])
	file.Write(data)
	form.Close()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, auddURL, &body)
	if err != nil {
		return domain.Track{}, err
	}
	req.Header.Set("Content-Type", form.FormDataContentType())

	resp, err := a.client.Do(req)
	if err != nil {
		return domain.Track{}, fmt.Errorf("audd: %w", err)
	}
	defer resp.Body.Close()

	var out auddResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return domain.Track{}, fmt.Errorf("audd: respuesta inválida: %w", err)
	}
	switch {
	case out.Error != nil:
		return domain.Track{}, fmt.Errorf("audd error %d: %s", out.Error.Code, out.Error.Message)
	case out.Result == nil:
		return domain.Track{}, domain.ErrNoMatch // sin coincidencia
	}
	return domain.Track{Title: out.Result.Title, Artist: out.Result.Artist}, nil
}
