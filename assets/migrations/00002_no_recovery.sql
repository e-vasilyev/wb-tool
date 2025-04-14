-- +goose Up
-- +goose StatementBegin
ALTER TABLE wb_content_cards 
    ADD COLUMN IF NOT EXISTS no_recovery boolean DEFAULT false;
-- +goose StatementEnd