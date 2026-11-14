package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"scenetrack/internal/domain"
)

type UsageRepo struct{ pool *pgxpool.Pool }

func NewUsageRepo(pool *pgxpool.Pool) *UsageRepo { return &UsageRepo{pool: pool} }

// El orden de las columnas debe coincidir con los campos de domain.Usage.
const usagesQuery = `
	SELECT w.kind, w.title, COALESCE(w.year, 0),
	       COALESCE(u.season, 0), COALESCE(u.episode, 0), COALESCE(u.note, '')
	FROM tracks t
	JOIN usages u ON u.track_id = t.id AND u.status = 'approved'
	JOIN works w ON w.id = u.work_id
	WHERE t.key = public.norm_key($1, $2)
	ORDER BY w.year, w.title`

func (r *UsageRepo) UsagesByTrack(ctx context.Context, t domain.Track) ([]domain.Usage, error) {
	rows, err := r.pool.Query(ctx, usagesQuery, t.Artist, t.Title)
	if err != nil {
		return nil, fmt.Errorf("query usages: %w", err)
	}
	return pgx.CollectRows(rows, pgx.RowToStructByPos[domain.Usage])
}

// AddSubmission guarda un envío como pendiente (0 y "" se guardan como NULL).
func (r *UsageRepo) AddSubmission(ctx context.Context, s domain.Submission) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO submissions (artist, title, kind, work, year, season, episode, note)
		VALUES ($1, $2, $3, $4, NULLIF($5, 0), NULLIF($6, 0), NULLIF($7, 0), NULLIF($8, ''))`,
		s.Artist, s.Title, s.Kind, s.Work, s.Year, s.Season, s.Episode, s.Note)
	if err != nil {
		return fmt.Errorf("insert submission: %w", err)
	}
	return nil
}
