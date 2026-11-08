# sandbox-svc — Entity Relationship Diagram

Cumulative ERD across every feature built for this service, maintained
by `/code-preparation` as part of each feature's own PR — see my_need.md
§2.1. Not a substitute for a feature's own `database.md` — this is the
combined picture, that document has the rationale.

```mermaid
erDiagram
    users ||--o{ favorite_items : favorites
    catalog_items ||--o{ favorite_items : "is favorited as"

    users {
        bigint id PK
        text username UK
        timestamptz created_at
    }

    catalog_items {
        bigint id PK
        text name
        boolean is_available
        timestamptz created_at
        text category "NEW (FEAT-003): NOT NULL DEFAULT ''"
        numeric price "NEW (FEAT-003): NOT NULL DEFAULT 0"
    }

    favorite_items {
        bigint user_id PK "FK -> users.id"
        bigint item_id PK "FK -> catalog_items.id"
        timestamptz created_at "favorited_at"
        text note "NEW (FEAT-003): NULL; written later by FEAT-002"
    }
```

`users` and `catalog_items` are existing tables owned by other systems
(auth/session and catalog respectively) — stand-in tables that this
service's features have since grown (see FEAT-003).

## Tables by feature

| Feature | Tables added/changed |
|---|---|
| [FEAT-001](feature/FEAT-001/database.md) | favorite_items |
| [FEAT-003](feature/FEAT-003/database.md) | catalog_items (category, price), favorite_items (note) |
