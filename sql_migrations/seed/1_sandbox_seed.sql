-- Sandbox seed data. Not a migration — never applied by the service's migrator.
--
-- This is the fixture config/config.go's AuthorizationAPI comment points at:
-- the rows the seeded Redis token's `oauth.user_id` must resolve to, plus a
-- couple of catalog items to favorite. Run it after the service has started (it
-- applies migrations/main itself, so these tables exist by then):
--
--   docker compose exec -T postgres \
--     psql -U postgres -d sandbox_svc -f /seed/1_sandbox_seed.sql
--
-- Idempotent, so re-running against a persistent volume is safe.

INSERT INTO users (id, username) VALUES
    (1, 'sandbox-user-1'),
    (2, 'sandbox-user-2')
ON CONFLICT DO NOTHING;

INSERT INTO catalog_items (id, name, is_available) VALUES
    (1, 'Sandbox Item 1', TRUE),
    (2, 'Sandbox Item 2', TRUE),
    -- Discontinued: still favoritable and still listed, but marked unavailable
    -- (REQ-004). Exercises the live-join read path.
    (3, 'Sandbox Item 3 (discontinued)', FALSE)
ON CONFLICT DO NOTHING;
