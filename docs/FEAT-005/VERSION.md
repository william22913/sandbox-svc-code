---
feature_code: FEAT-005
service: sandbox-svc
version: 1
spec_version: 1
testcase_version: 1
architecture_version: 1
database_version: 1
last_updated: 2026-11-08
---

# FEAT-005 — Version Lock

Version lock for this feature's helicopter-view overviews (`TRD.md`,
`BackendSchema.md`, `ImplementationPlan.md`), maintained by
`/code-preparation`. Not a source of truth for anything else — `spec.md`,
`testcase.md`, `architecture.md`, and `database.md` each carry their own
version/spec_version fields; this file only pins the three summaries
above to the design state they were last refreshed against.

- `spec_version: 1` — spec.md's merged version (PR 23); the design and
  every test case are built against it.
- `testcase_version: 1` — testcase.md's merged version (PR 24), which
  pins `spec_version: 1`, matching the spec.
- `architecture_version: 1` — architecture.md's version, first design
  pass for this feature (this PR).
- `database_version: 1` — database.md's version, first design pass; no
  schema change, no migration batch.
