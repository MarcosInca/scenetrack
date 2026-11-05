package memory

import (
	"context"
	"sync"

	"scenetrack/internal/domain"
)

type UsageRepo struct {
	mu   sync.RWMutex
	data map[string][]domain.Usage
}

func NewUsageRepo() *UsageRepo {
	return &UsageRepo{data: make(map[string][]domain.Usage)}
}

func (r *UsageRepo) Add(t domain.Track, u domain.Usage) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.data[t.Key()] = append(r.data[t.Key()], u)
}

func (r *UsageRepo) UsagesByTrack(ctx context.Context, t domain.Track) ([]domain.Usage, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	usages, ok := r.data[t.Key()]
	if !ok {
		return nil, domain.ErrNotFound
	}
	// Copia, para que nadie modifique nuestro slice interno desde afuera.
	out := make([]domain.Usage, len(usages))
	copy(out, usages)
	return out, nil
}
