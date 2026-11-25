-- +goose Up
CREATE TABLE steam_orderbooks (
    hash_name         TEXT        PRIMARY KEY,
    fetched_at        TIMESTAMPTZ NOT NULL,
    max_buy_order     BIGINT,              -- тысячные доллара; NULL — заявок нет
    min_sell_order    BIGINT,              -- NULL — лотов нет
    buy_orders_total  INT         NOT NULL, -- всего штук в заявках (cBuyOrders)
    sell_orders_total INT         NOT NULL, -- всего лотов (cSellOrders)
    buy_levels        BIGINT[]    NOT NULL, -- верх заявок: [цена, штук, цена, штук, ...], по убыванию
    sell_levels       BIGINT[]    NOT NULL  -- первые уровни лотов, тот же формат, по возрастанию
);

CREATE INDEX steam_orderbooks_fetched_at_idx ON steam_orderbooks (fetched_at);

-- +goose Down
DROP TABLE steam_orderbooks;