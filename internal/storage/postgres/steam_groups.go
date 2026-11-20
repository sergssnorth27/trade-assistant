package postgres

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sergssnorth27/trade-assistant/internal/domain"
	"github.com/sergssnorth27/trade-assistant/internal/steam"
)

const (
	upsertGroups = `
	INSERT INTO steam_groups(group_id,name,min_price,sell_listings,updated_at)
	SELECT id, name, price, listings, now()
	FROM unnest($1::varchar[], $2::varchar[], $3::bigint[], $4::int[]) AS t(id, name, price, listings)
	ON CONFLICT (group_id) DO UPDATE SET name=EXCLUDED.name, min_price=EXCLUDED.min_price, sell_listings=EXCLUDED.sell_listings, updated_at=now()`

	getGroupIDs = `
	SELECT g.group_id FROM steam_groups g
	WHERE g.min_price <= $1
	  AND g.sell_listings >= $3
	  AND (
			g.min_price >= $2
			OR NOT (g.name LIKE 'Sticker |%' OR g.name LIKE 'Patch |%'
					OR g.name LIKE 'Sealed Graffiti |%' OR g.name LIKE 'Charm |%'
					OR g.name LIKE 'Autograph Capsule |%')
		)
	  AND NOT EXISTS (
	      SELECT 1 FROM steam_variant_stats s
	      WHERE s.group_id = g.group_id
	        AND s.computed_at > now() - interval '24 hours'
	  )
	ORDER BY g.group_id`
)

type SteamGroupsRepo struct {
	pool *pgxpool.Pool
}

func NewSteamGroupsRepo(pool *pgxpool.Pool) *SteamGroupsRepo {
	return &SteamGroupsRepo{pool: pool}
}

func (r *SteamGroupsRepo) UpsertGroups(ctx context.Context, groups []steam.Group) error {
	if len(groups) == 0 {
		return nil
	}

	ids := make([]string, len(groups))
	names := make([]string, len(groups))
	prices := make([]*int64, len(groups))
	listings := make([]int, len(groups))

	for i, g := range groups {
		ids[i] = g.ID
		names[i] = g.Name
		prices[i] = priceArg(g.MinPrice) // nil → NULL, если цену не удалось разобрать
		listings[i] = g.SellListings
	}

	_, err := r.pool.Exec(ctx, upsertGroups, ids, names, prices, listings)
	return err
}

// возвращает ID групп с минимальной ценой не выше maxPrice и не меньше minListings лотов
func (r *SteamGroupsRepo) GroupIDs(ctx context.Context, maxPrice domain.Price, minPrice domain.Price, sellListings int) ([]string, error) {
	rows, err := r.pool.Query(ctx, getGroupIDs, int64(maxPrice), int64(minPrice), sellListings)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, pgx.RowTo[string])
}
