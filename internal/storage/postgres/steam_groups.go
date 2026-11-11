package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	UpsertGroups = `
	INSERT INTO steam_groups(group_id,name)
	SELECT * FROM unnest($1::varchar[], $2::varchar[])
	ON CONFLICT (group_id) DO UPDATE SET name=EXCLUDED.name`
)

type SteamGroupsRepo struct {
	pool *pgxpool.Pool
}

func NewSteamGroupsRepo(pgx *pgxpool.Pool) *SteamGroupsRepo {
	return &SteamGroupsRepo{pool: pgx}
}

func (r *SteamGroupsRepo) UpsertGroups(ctx context.Context, ids, names []string) error {
	if len(ids) != len(names) {
		return fmt.Errorf("upsert groups: %d ids, %d names", len(ids), len(names))
	}
	if len(ids) == 0 {
		return nil
	}

	if _, err := r.pool.Exec(ctx, UpsertGroups, ids, names); err != nil {
		return fmt.Errorf("upsert groups: %w", err)
	}
	return nil
}
