-- FEAT-003 live-verification fixture. Applied by postgres-seed.sh in the
-- docker-compose.verify.yml overlay only — it is not a migration and not part
-- of the shipped stack.
--
-- User 1 owns five favorites chosen so one export row exercises each rendering
-- rule at once:
--
--   103  integer price 0, empty category, UTF-8 note
--   102  name and note both start with '=' → formula neutralisation, price
--        without a fractional part, discontinued → Tidak Tersedia
--   101  name contains a comma → RFC 4180 quoting, fractional price
--   104  name contains an embedded LF → quoting that must preserve the byte,
--        four-decimal price that must not be rounded
--   105  name contains doubled quotes and a comma → quote doubling
--
-- User 3 exists with no favorites: the "header-only file" case.
-- User 2 is what the collection-wide token maps to (FEAT-001's three requests).

INSERT INTO users (id, username) VALUES
    (1, 'live-user-1'),
    (2, 'live-user-2'),
    (3, 'live-user-3')
ON CONFLICT (id) DO NOTHING;

INSERT INTO catalog_items (id, name, category, price, is_available) VALUES
    (2,   'Sandbox Item 2',      '',       0,        TRUE),
    (101, 'Notebook, A5',        'Office', 19.99,    TRUE),
    (102, '=cmd|calc',           'Office', 1500,     FALSE),
    (103, 'Pen',                 '',       0,        TRUE),
    (104, E'Line1\nLine2',       'Misc',   1234.5678, TRUE),
    (105, 'He said "hi", twice', '',       7.5,       TRUE)
ON CONFLICT (id) DO UPDATE SET
    name         = EXCLUDED.name,
    category     = EXCLUDED.category,
    price        = EXCLUDED.price,
    is_available = EXCLUDED.is_available;

INSERT INTO favorite_items (user_id, item_id, created_at, note) VALUES
    (1, 101, '2026-04-09T14:30:00Z', 'Buy more'),
    (1, 102, '2026-04-10T00:00:00Z', '=1+1'),
    (1, 103, '2026-04-11T08:00:00Z', 'Café — 🎁'),
    (1, 104, '2026-04-01T00:00:00Z', NULL),
    (1, 105, '2026-03-05T00:00:00Z', NULL)
ON CONFLICT (user_id, item_id) DO UPDATE SET
    created_at = EXCLUDED.created_at,
    note       = EXCLUDED.note;
