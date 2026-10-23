-- +goose Up
CREATE TABLE items (
    id SERIAL PRIMARY KEY,
    market_id INT NOT NULL,
    market_hash_name TEXT NOT NULL,
    update_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- +goose Down
DROP TABLE items;

