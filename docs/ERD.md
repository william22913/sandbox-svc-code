# sandbox-svc — Entity Relationship Diagram

Cumulative ERD across every feature built for this service, maintained
by `/code-preparation` as part of each feature's own PR — see my_need.md
§2.1. Not a substitute for a feature's own `database.md` — this is the
combined picture, that document has the rationale.

```mermaid
erDiagram
    users ||--o{ favorite_items : favorites
    catalog_items ||--o{ favorite_items : "is favorited as"

    favorite_items {
        bigint user_id PK "FK -> users.id"
        bigint item_id PK "FK -> catalog_items.id"
        timestamptz created_at "favorited_at"
    }
```

`users` and `catalog_items` are existing tables owned by other systems
(auth/session and catalog respectively) — shown only to anchor the
relationship; this service's own features have not yet defined them.

## Tables by feature

| Feature | Tables added/changed |
|---|---|
| [FEAT-001](feature/FEAT-001/database.md) | favorite_items |
