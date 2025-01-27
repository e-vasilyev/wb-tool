-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS wb_content_cards (
    nm_id int NOT NULL,
    imt_id int,
    vendor_code varchar(64) NOT NULL,
    subject_id int NOT NULL,
    subject_name varchar(128) NOT NULL,
    brand varchar(64),
    title varchar(128),
    trashed_at timestamp,
    trashed boolean DEFAULT false,
    deleted boolean DEFAULT false,
    updated_timestamp timestamp NOT NULL,
    PRIMARY KEY (nm_id)
);

CREATE TABLE IF NOT EXISTS wb_content_skus (
    sku varchar(16) NOT NULL,
    nm_id int NOT NULL,
    PRIMARY KEY (sku),
    FOREIGN KEY (nm_id) REFERENCES wb_content_cards (nm_id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS wb_marketplace_stocks (
    sku varchar(16) NOT NULL,
    nm_id int NOT NULL,
    amount int NOT NULL,
    updated_timestamp timestamp NOT NULL,
    PRIMARY KEY (sku),
    FOREIGN KEY (nm_id) REFERENCES wb_content_cards (nm_id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS wb_stocks (
    sku varchar(16) NOT NULL,
    nm_id int NOT NULL,
    quantity int NOT NULL,
    quantity_full int NOT NULL,
    in_way_to_client int NOT NULL,
    in_way_from_client int NOT NULL,
    updated_timestamp timestamp NOT NULL,
    PRIMARY KEY (sku),
    FOREIGN KEY (nm_id) REFERENCES wb_content_cards (nm_id) ON DELETE CASCADE
);
-- +goose StatementEnd