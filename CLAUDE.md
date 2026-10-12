# sandbox-svc-code

Service code repo for `sandbox-svc` in the nexai Agent-Assisted Dev Pipeline.

Greenfield as of the first feature (FEAT-001): no source tree existed before it.

## Stack

- **Go**, using the team's shared `nexcommon` library
  (`github.com/nexsoft-git/nexcommon`) — controller wrapping, DTO/validation,
  DAO/repository pattern, request-context user ID, Prometheus metrics.
- **Postgres** for persistence, via nexcommon's data-access layer.
- Auth/session is a shared middleware that is *consumed*, not built here; it
  places the authenticated user ID into request context.
- **Not behind Kong.** Every route therefore uses nexcommon's plain
  `httpController.UserAccessValidator()`, not the `...WithKong()` variant — the
  caller's identity is resolved from the raw `Authorization` header against
  Redis, and no gateway header is consulted.

## Feature docs

- [FEAT-001](docs/FEAT-001/spec.md) — Personal favorites list for catalog items
  (favorite/un-favorite, My Favorites list, 500-per-user cap, row-ownership 404).
- [ERD](docs/ERD.md) — cumulative entity-relationship diagram for this service.
