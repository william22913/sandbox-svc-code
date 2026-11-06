-- +migrate Up

-- FEAT-003, the single additive batch from database.md.
--
-- No new table: the export reads favorite_items joined to catalog_items, and
-- the stand-in catalog simply lacks the two attributes the CSV header names.
--
-- * category/price live on catalog_items: they are item attributes, and
--   REQ-015 requires the export to read the item's *current* state rather
--   than a snapshot taken at favorite-time (TC-032).
-- * note lives on favorite_items: it describes one user's relationship to the
--   item, so it cannot be shared between users and must disappear with the
--   favorite. FEAT-003 only adds the nullable column and reads it; FEAT-002
--   owns the write path and its validation, and must not re-add the column.
--
-- Both defaults are constants, so PostgreSQL 11+ stores them as a metadata-only
-- default: existing rows read the constant without a table rewrite. No
-- backfill, no index (the composite primary key already covers both read
-- paths), no constraint changes.

SET LOCAL lock_timeout = '5s';

ALTER TABLE catalog_items
    ADD COLUMN category TEXT    NOT NULL DEFAULT '',
    ADD COLUMN price    NUMERIC NOT NULL DEFAULT 0;

ALTER TABLE favorite_items
    ADD COLUMN note TEXT NULL;

-- +migrate Down

SET LOCAL lock_timeout = '5s';

ALTER TABLE favorite_items DROP COLUMN note;
ALTER TABLE catalog_items DROP COLUMN price;
ALTER TABLE catalog_items DROP COLUMN category;
