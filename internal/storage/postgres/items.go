package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Item struct {
	MarketID       int
	MarketHashName string
}

type ItemRepo struct {
	pool *pgxpool.Pool
}

func NewItemRepo(pool *pgxpool.Pool) *ItemRepo {
	return &ItemRepo{pool: pool}
}

func (r *ItemRepo) Save(ctx context.Context, i Item) (int64, error) {
	query := `
	INSERT INTO items (market_id, market_hash_name)
	VALUES ($1, $2)
	RETURNING ID`

	var id int64

	if err := r.pool.QueryRow(ctx, query, i.MarketID, i.MarketHashName).Scan(&id); err != nil {
		return 0, fmt.Errorf("insert item price: %w", err)
	}
	return id, nil
}
