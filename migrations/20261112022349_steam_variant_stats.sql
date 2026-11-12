-- +goose Up
-- Статистика Steam по варианту предмета. Одна строка на market_hash_name.
-- Деньги — BIGINT в тысячных долях доллара (domain.Price), доли — DOUBLE PRECISION долей единицы.
-- NULL = «нет данных» (нет продаж в окне, история короче окна). Счётчики — NOT NULL, 0 = продаж не было.
-- Описание всех колонок: документ проекта steam-variant-stats.md.
CREATE TABLE steam_variant_stats (
    -- служебное
    hash_name               TEXT PRIMARY KEY,
    group_id                TEXT NOT NULL REFERENCES steam_groups (group_id),
    computed_at             TIMESTAMPTZ NOT NULL,
    last_point_at           TIMESTAMPTZ,
    source_currency         INT NOT NULL,
    usd_rate                DOUBLE PRECISION NOT NULL,
    points_count            INT NOT NULL,

    -- price_: взвешенные перцентили цены продаж за окно
    price_median_24h        BIGINT,
    price_p25_24h           BIGINT,
    price_p75_24h           BIGINT,
    price_median_3d         BIGINT,
    price_p25_3d            BIGINT,
    price_p75_3d            BIGINT,
    price_median_7d         BIGINT,
    price_p25_7d            BIGINT,
    price_p75_7d            BIGINT,
    price_median_30d        BIGINT,
    price_p25_30d           BIGINT,
    price_p75_30d           BIGINT,
    price_median_60d        BIGINT,
    price_p25_60d           BIGINT,
    price_p75_60d           BIGINT,
    price_median_90d        BIGINT,
    price_p25_90d           BIGINT,
    price_p75_90d           BIGINT,

    -- price_: коридор дневных медиан
    price_min_daily_7d      BIGINT,
    price_min_daily_30d     BIGINT,
    price_min_daily_60d     BIGINT,
    price_min_daily_90d     BIGINT,
    price_max_daily_7d      BIGINT,
    price_max_daily_30d     BIGINT,
    price_max_daily_60d     BIGINT,
    price_max_daily_90d     BIGINT,

    -- price_: отношения медиан окон
    price_ratio_24h_7d      DOUBLE PRECISION,
    price_ratio_3d_7d       DOUBLE PRECISION,
    price_ratio_7d_30d      DOUBLE PRECISION,
    price_ratio_30d_90d     DOUBLE PRECISION,

    -- price_: тренд (изменение в день) и его R²
    price_trend_7d          DOUBLE PRECISION,
    price_trend_30d         DOUBLE PRECISION,
    price_trend_r2_7d       DOUBLE PRECISION,
    price_trend_r2_30d      DOUBLE PRECISION,

    -- price_: исторические максимум и минимум дневной медианы
    price_ath               BIGINT,
    price_ath_at            TIMESTAMPTZ,
    price_atl               BIGINT,
    price_atl_at            TIMESTAMPTZ,

    -- price_: самый дешёвый лот на момент обхода (справочно)
    price_min_listing       BIGINT,

    -- liq_: продажи за окно
    liq_sales_24h           INT NOT NULL,
    liq_sales_3d            INT NOT NULL,
    liq_sales_7d            INT NOT NULL,
    liq_sales_30d           INT NOT NULL,
    liq_sales_60d           INT NOT NULL,
    liq_sales_90d           INT NOT NULL,

    -- liq_: регулярность спроса
    liq_sales_per_day_7d    DOUBLE PRECISION NOT NULL,
    liq_sales_per_day_30d   DOUBLE PRECISION NOT NULL,
    liq_days_with_sales_7d  INT NOT NULL,
    liq_days_with_sales_30d INT NOT NULL,
    liq_top_day_share_7d    DOUBLE PRECISION,
    liq_volume_trend_7d     DOUBLE PRECISION,

    -- risk_: изменение цены за H дней по истории
    risk_fwd_p10            DOUBLE PRECISION,
    risk_fwd_p50            DOUBLE PRECISION,
    risk_fwd_days           INT NOT NULL,
    risk_fwd_samples        INT NOT NULL,

    -- risk_: волатильность и просадка
    risk_volatility_7d      DOUBLE PRECISION,
    risk_volatility_30d     DOUBLE PRECISION,
    risk_max_drawdown_90d   DOUBLE PRECISION,

    -- q_: качество данных
    q_first_sale_at         TIMESTAMPTZ,
    q_age_days              INT,
    q_last_sale_at          TIMESTAMPTZ,
    q_last_sale_price       BIGINT,
    q_mixed_phases          BOOLEAN NOT NULL
);

CREATE INDEX steam_variant_stats_group_id_idx ON steam_variant_stats (group_id);

-- +goose Down
DROP TABLE steam_variant_stats;