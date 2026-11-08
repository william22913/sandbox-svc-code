-- +migrate Up

-- FEAT-001 Batch 1, exactly as database.md specifies.
--
-- Deliberate deviations from the house table convention (no id / uuid_key /
-- updated_at / soft-delete columns), each argued in database.md's Phase 2
-- normalization review:
--   * Composite primary key (user_id, item_id), no surrogate key — nothing in
--     the spec or the test cases ever addresses a favorite by any identifier
--     other than that pair, so uuid_key would be an unread column. The integer
--     id is never exposed anyway.
--   * No is_deleted/deleted_at/deleted_by — un-favoriting is a literal row
--     DELETE (REQ-003), not a lifecycle state with history value.
--   * No updated_at — rows are inserted or deleted, never mutated in place.
--   * created_at is kept: REQ-002's list is a natural place to sort by
--     recency later, and the deferred sorting pass can then read it instead of
--     needing a migration to recover it.
-- If a later feature (e.g. FEAT-002's notes) needs per-row identity or
-- lineage, that is a change to this table made by that feature, not a reason to
-- pre-add unused columns now.

CREATE TABLE IF NOT EXISTS favorite_items (
    user_id     BIGINT       NOT NULL REFERENCES users(id),
    item_id     BIGINT       NOT NULL REFERENCES catalog_items(id),
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT now(),
    PRIMARY KEY (user_id, item_id)
);

-- No separate index. The composite PK's leading column (user_id) already
-- serves both read paths with no extra structure:
--   * REQ-002 list:      WHERE user_id = $1
--   * REQ-007 cap check: COUNT(*) WHERE user_id = $1

-- +migrate Down

DROP TABLE IF EXISTS favorite_items;
