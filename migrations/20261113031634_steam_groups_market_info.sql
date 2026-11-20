-- +goose Up
-- Данные поиска Steam по группе: самая низкая цена среди всех износов и сумма лотов.
-- NULL — группа ещё не перезагружена после этой миграции.
ALTER TABLE steam_groups
    ADD COLUMN min_price     BIGINT,       -- цена покупателя в тысячных долях доллара
    ADD COLUMN sell_listings INT,
    ADD COLUMN updated_at    TIMESTAMPTZ;  -- когда последний раз обновили из поиска

-- +goose Down
ALTER TABLE steam_groups
    DROP COLUMN min_price,
    DROP COLUMN sell_listings,
    DROP COLUMN updated_at;