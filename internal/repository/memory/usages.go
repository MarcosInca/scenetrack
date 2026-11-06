package memory

import (
	"context"

	"scenetrack/internal/domain"
)

// UsageRepo guarda usos en memoria (solo demo).
type UsageRepo struct {
	data map[string][]domain.Usage // clave: Track.Key()
}

// NewDemo crea un repo con un dato de ejemplo (después solo se lee).
func NewDemo() *UsageRepo {
	demo := domain.Track{Artist: "Demo Artist", Title: "Demo Song"}
	return &UsageRepo{data: map[string][]domain.Usage{
		demo.Key(): {{
			Work: domain.Work{Kind: domain.KindMovie, Title: "Pelicula Demo", Year: 2001},
			Note: "Escena de la persecución final",
		}},
	}}
}

func (r *UsageRepo) UsagesByTrack(_ context.Context, t domain.Track) ([]domain.Usage, error) {
	return r.data[t.Key()], nil
}
