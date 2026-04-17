-- migrations/000001_create_url_table.up.sql

CREATE TABLE urls
(
    uuid         UUID PRIMARY KEY,
    short_url    VARCHAR(255)  NOT NULL UNIQUE,
    original_url VARCHAR(2048) NOT NULL
);

CREATE INDEX idx_short_url ON urls (short_url);
