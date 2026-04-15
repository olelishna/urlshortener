-- migrations/000002_add_original_url_idx.up.sql

CREATE UNIQUE INDEX idx_original_url ON urls (original_url);