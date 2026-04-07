-- +goose Up
-- +goose StatementBegin
DROP TABLE IF EXISTS wb_content_skus;
DROP TABLE IF EXISTS wb_marketplace_stocks;
DROP TABLE IF EXISTS wb_stocks;

CREATE TABLE IF NOT EXISTS wb_content_sizes (
    chrt_id int NOT NULL,
    nm_id int NOT NULL,
    PRIMARY KEY (chrt_id),
    FOREIGN KEY (nm_id) REFERENCES wb_content_cards (nm_id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS wb_content_skus (
    sku varchar(16) NOT NULL,
    chrt_id int NOT NULL,
    PRIMARY KEY (sku),
    FOREIGN KEY (chrt_id) REFERENCES wb_content_sizes (chrt_id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS wb_marketplace_stocks (
    chrt_id int NOT NULL,
    nm_id int NOT NULL,
    amount int NOT NULL,
    updated_timestamp timestamp NOT NULL,
    PRIMARY KEY (chrt_id),
    FOREIGN KEY (nm_id) REFERENCES wb_content_cards (nm_id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS wb_stocks (
    chrt_id int NOT NULL,
    nm_id int NOT NULL,
    quantity int NOT NULL,
    quantity_full int NOT NULL,
    in_way_to_client int NOT NULL,
    in_way_from_client int NOT NULL,
    updated_timestamp timestamp NOT NULL,
    PRIMARY KEY (chrt_id),
    FOREIGN KEY (nm_id) REFERENCES wb_content_cards (nm_id) ON DELETE CASCADE
);
-- +goose StatementEnd