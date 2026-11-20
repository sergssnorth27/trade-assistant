-- +goose Up
CREATE TABLE market_variant_stats (
    -- служебное
    hash_name            TEXT        PRIMARY KEY,
    market_id            BIGINT      NOT NULL,  -- id из словаря full-history/all.json
    computed_at          TIMESTAMPTZ NOT NULL,

    -- price_: цены в тысячных доллара; NULL — продаж нет или окно не покрыто историей
    price_median_24h     BIGINT,
    price_median_7d      BIGINT,
    price_median_30d     BIGINT,
    price_median_90d     BIGINT,
    price_p25_24h        BIGINT,
    price_p25_7d         BIGINT,
    price_p25_30d        BIGINT,
    price_p25_90d        BIGINT,

    -- liq_: продано штук за окно
    liq_sales_24h        INT         NOT NULL DEFAULT 0,
    liq_sales_7d         INT         NOT NULL DEFAULT 0,
    liq_sales_30d        INT         NOT NULL DEFAULT 0,
    liq_sales_90d        INT         NOT NULL DEFAULT 0,

    -- q_: можно ли верить данным
    q_oldest_sale_at     TIMESTAMPTZ,           -- до какого момента хватило истории (6000 продаж)
    q_last_sale_at       TIMESTAMPTZ
);

CREATE INDEX market_variant_stats_computed_at_idx ON market_variant_stats (computed_at);

-- +goose Down
DROP TABLE market_variant_stats;