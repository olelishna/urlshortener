-- migrations/000003_add_user_column.down.sql

DROP INDEX IF EXISTS idx_user_uuid;

ALTER TABLE urls DROP COLUMN IF EXISTS user_uuid;