-- +migrate Up

-- Sandbox reference tables.
--
-- FEAT-001 does NOT own these. Its schema (database.md) declares
-- favorite_items.user_id REFERENCES users(id) and
-- favorite_items.item_id REFERENCES catalog_items(id), and states that both
-- parents already exist and belong to the auth/session and catalog systems.
-- This repository is greenfield, so they are created here as the minimum that
-- makes those foreign keys real. Columns beyond what the FK and REQ-004 need
-- are deliberately absent — growing these into the real auth/catalog tables is
-- those systems' work, not this feature's, and a richer stand-in here would
-- just be schema this feature does not own and cannot maintain.
--
-- When the real services land, replace these two tables with their definitions;
-- favorite_items does not change.

CREATE TABLE IF NOT EXISTS users (
    id          BIGSERIAL    PRIMARY KEY,
    username    TEXT         NOT NULL UNIQUE,
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS catalog_items (
    id            BIGSERIAL    PRIMARY KEY,
    name          TEXT         NOT NULL,
    -- REQ-004: catalog_items rows are never hard-deleted. A discontinued item is
    -- soft-marked here, and favorite_service.List reads this flag live at
    -- list-time so a discontinued favorite stays visible, marked unavailable,
    -- instead of vanishing from the join.
    is_available  BOOLEAN      NOT NULL DEFAULT TRUE,
    created_at    TIMESTAMPTZ  NOT NULL DEFAULT now()
);

-- +migrate Down

DROP TABLE IF EXISTS catalog_items;
DROP TABLE IF EXISTS users;
