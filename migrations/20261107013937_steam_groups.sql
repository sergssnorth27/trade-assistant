-- +goose Up
CREATE TABLE steam_groups(
    group_id VARCHAR PRIMARY KEY,
    name VARCHAR NOT NULL
);
CREATE INDEX steam_groups_name_idx ON steam_groups (name);

-- +goose Down
DROP TABLE steam_groups;
