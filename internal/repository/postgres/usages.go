package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"scenetrack/internal/domain"
)

type UsageRepo struct {
	pool *pgxpool.Pool
}

func NewUsageRepo(pool *pgxpool.Pool) *UsageRepo {
	return &UsageRepo{pool: pool}
}

func (r *UsageRepo) UsagesByTrack(ctx context.Context, t domain.Track) ([]domain.Usage, error) {
	// $1 es un parámetro: nunca se concatena texto del usuario en el SQL.
	rows, err := r.pool.Query(ctx, `
		SELECT u.id, w.id, w.kind, w.title,
		       COALESCE(w.year, 0), COALESCE(w.poster_url, ''),
		       COALESCE(u.season, 0), COALESCE(u.episode, 0),
		       COALESCE(u.timestamp_sec, 0), COALESCE(u.note, '')
		FROM tracks t
		JOIN usages u ON u.track_id = t.id AND u.status = 'approved'
		JOIN works w ON w.id = u.work_id
		WHERE t.key = $1
		ORDER BY w.year, w.title`, t.Key())
	if err != nil {
		return nil, fmt.Errorf("query usages: %w", err)
	}
	defer rows.Close()

	var out []domain.Usage
	for rows.Next() {
		var u domain.Usage
		var kind string
		err := rows.Scan(&u.ID, &u.Work.ID, &kind, &u.Work.Title,
			&u.Work.Year, &u.Work.PosterURL,
			&u.Season, &u.Episode, &u.TimestampSec, &u.Note)
		if err != nil {
			return nil, fmt.Errorf("scan usage: %w", err)
		}
		u.Work.Kind = domain.WorkKind(kind)
		out = append(out, u)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate usages: %w", err)
	}
	return out, nil
}
