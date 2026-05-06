-- migrations/000003_add_user_column.up.sql

ALTER TABLE urls
    ADD COLUMN user_uuid UUID;

CREATE INDEX idx_user_uuid ON urls (user_uuid);