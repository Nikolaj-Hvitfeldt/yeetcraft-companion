# Yeetcraft integration

Integration architecture and ownership between the [Yeetcraft](https://github.com/Nikolaj-Hvitfeldt/Yeetcraft) web application and the standalone [yeetcraft-companion](https://github.com/Nikolaj-Hvitfeldt/yeetcraft-companion) desktop companion.

| | |
| --- | --- |
| **Status** | Planning — no companion upload client or ingest API implemented in either repository |
| **Last updated** | 2026-07-30 |

This document describes **how the two products relate**. It does not define an approved API contract, payload schema, or retry policy. For proposed designs, see [IMPLEMENTATION_PLAN.md](./IMPLEMENTATION_PLAN.md).

---

## Repository responsibilities

### `./yeetcraft` (Yeetcraft)

| Owns | Does not own |
| ---- | ------------ |
| React/Vite PWA frontend (Vercel) | Combat-log capture or parsing |
| Go backend (`backend/cmd/server`, chi + pgx) | Local SQLite on user machines |
| PostgreSQL / Supabase (canonical server data) | Desktop packaging or Wails UI |
| Public read routes and current write auth | Companion install lifecycle |
| **`PATCH /api/stats/batch`** (today's only write route) | |

**Planned (not implemented yet):**

- Versioned companion ingestion HTTP endpoint
- Canonical companion API contract at `./yeetcraft/contracts/companion/v1/`
- Event-oriented schema, idempotent ingest, and aggregate reconciliation (see implementation plan)

### `./yeetcraft-companion` (this repository)

| Owns | Does not own |
| ---- | ------------ |
| Locating and monitoring the WoW combat log | Website, frontend, or PostgreSQL |
| Parsing and normalizing relevant log events | Server-side authentication policy |
| Run, dungeon, boss, player, and death inference | Canonical API contract source of truth |
| Local SQLite state, retry, and upload coordination | Yeetcraft aggregate authority rules |
| Local review experience | Hosted API deployment |
| Desktop packaging and future Wails UI | |

---

## Integration boundaries

```text
WoW client → combat-log files → Companion (local SQLite)
                                      ↓
                               HTTPS JSON (versioned API — planned)
                                      ↓
                               Yeetcraft Go API → PostgreSQL
                                      ↓
                               Browser PWA (public GET)
```

Hard rules:

- **No cross-repository Go imports.** Module: `github.com/Nikolaj-Hvitfeldt/yeetcraft-companion`.
- **No direct PostgreSQL access** from the companion (no Supabase client, no connection strings to Yeetcraft's database).
- **No runtime dependency** on sibling source paths, build artifacts, or shared local files.
- **No sharing of database files** between products (companion SQLite ≠ Yeetcraft PostgreSQL).
- **Communication only through a versioned HTTP API** once the companion ingest route is approved and implemented.
- **Independent products:** each repository must build, test, and deploy on its own.

Cross-repository features require an **explicit task scope** with separate changes, validation, and commits per repository. See [AGENTS.md](../AGENTS.md).

---

## Tracked and untracked party members

A Mythic+ group contains five players. Yeetcraft initially recognizes four
configured tracked characters; the fifth player is normally an untracked party
member.

The companion's low-level CSV parser must remain roster-neutral. Identity
filtering belongs after normalization, where the companion can apply configured
tracking rules without corrupting the underlying event shape.

| Identity class | Local behavior | Future upload behavior |
| -------------- | -------------- | ---------------------- |
| Configured tracked character | Retain normalized death evidence needed for review | May upload only after identity mapping and review rules are approved |
| Untracked party member | Keep only anonymized context when required to understand a run | Never upload name, realm, GUID, or death statistic |

An untracked party member must not be created automatically as a Yeetcraft
player and must not contribute to Yeetcraft death totals. Final identity mapping
between combat-log GUIDs and configured tracked characters remains an open
design decision.

---

## Contract ownership

| Item | Owner | Location |
| ---- | ----- | -------- |
| Canonical companion API contract | **Yeetcraft** | `./yeetcraft/contracts/companion/v1/` (**planned — directory does not exist yet**) |
| Derived client fixtures / generated types | Companion (optional) | This repo only as copies; never an alternate source of truth |

Contract changes may require **separate pull requests** in both repositories (schema in Yeetcraft, client and tests in companion). Coordinate version bumps explicitly.

**Do not treat this file or companion docs as the contract.** For the proposed ingest endpoint shape, payload fields, and server-side data model, see:

- [IMPLEMENTATION_PLAN.md §5 — Versioned ingest API](./IMPLEMENTATION_PLAN.md#5-versioned-ingest-api) (**proposed, not approved**)
- [IMPLEMENTATION_PLAN.md §4 — Core data model](./IMPLEMENTATION_PLAN.md#4-core-data-model) (**target design, not implemented**)

During development, agents may read `../yeetcraft/contracts/companion/v1/` from a sibling checkout once it exists. The compiled companion must remain testable using local fixtures when the sibling repo is absent.

---

## Current Yeetcraft behavior (verified)

Facts observed in the Yeetcraft repository today:

| Area | Current behavior |
| ---- | ---------------- |
| Stack | Vite React PWA → Go (chi) API → PostgreSQL (typically Supabase) |
| Hosting | Frontend on Vercel; backend on Render (free tier may sleep) |
| Reads | Public `GET /api/*` routes; no API key required |
| Writes | Single route: `PATCH /api/stats/batch` |
| Write auth | `X-API-Key` or `Authorization: Bearer <token>`; server `API_KEY` env var |
| Fail-closed | Empty/missing server `API_KEY` → **503** on mutations |
| Browser unlock | `?token=` on page URLs → `localStorage` → `X-API-Key` on PATCH (not supported as query param on API routes) |
| Data model | Aggregate tables: `players`, `seasons`, `dungeons`, `season_dungeons`, `player_dungeon_stats` |
| Companion ingest | **Not present** — no `/api/ingest/*` routes; no `contracts/companion/` tree |

Source references (read-only): `../yeetcraft/docs/API.md`, `../yeetcraft/docs/ARCHITECTURE.md`, `../yeetcraft/backend/db/schema.sql`.

---

## Planned integration (not implemented)

The implementation plan proposes a **separate versioned ingest endpoint** (e.g. batch death events) rather than reusing `PATCH /api/stats/batch` semantics. That endpoint, its auth model, and PostgreSQL event tables **do not exist yet** and are subject to contract review in Phase 1.

Until ingest is live:

- The companion performs **no server writes**.
- Yeetcraft continues to operate on manual aggregate edits via the existing PATCH route and PWA outbox.

---

## Local development

Typical sibling layout (paths are examples, not requirements):

```text
Repositories/
├── Yeetcraft/              # website + backend + PostgreSQL schema
└── yeetcraft-companion/    # this repository
```

A **multi-root Cursor workspace** may open both folders together. That is a developer convenience only — not a monorepo.

| Activity | Allowed in companion tasks | Requires explicit Yeetcraft scope |
| -------- | -------------------------- | --------------------------------- |
| Read `../yeetcraft` for API, schema, auth patterns | Yes | — |
| Modify Yeetcraft source, migrations, or contracts | No | Yes |
| Run Yeetcraft backend locally for manual testing | Yes (operator choice) | — |

The installed companion on a user's PC talks to a **configured API URL only**. It does not need the Yeetcraft repository at runtime.

---

## Runtime environments

| Environment | Yeetcraft API (example) | Companion config |
| ----------- | ----------------------- | ---------------- |
| Local dev | `http://localhost:8080` | `YEETCRAFT_API_BASE_URL` in `.env` (see `.env.example`) |
| Production | `https://<your-render-service>.onrender.com` (placeholder) | Same variable; user- or deployer-supplied |

Placeholders only — do not commit real URLs with embedded secrets.

| Concern | Notes |
| ------- | ----- |
| Production backend | Hosted on Render; **free tier may sleep**, causing cold-start latency |
| Companion retry | Must tolerate timeouts and 5xx; exact backoff values **not approved yet** — see [IMPLEMENTATION_PLAN.md §7.2](./IMPLEMENTATION_PLAN.md#72-render-cold-start-strategy) |
| Credentials | `YEETCRAFT_API_KEY` placeholder in `.env.example`; not used until upload is implemented |
| Local SQLite | `COMPANION_DB_PATH` (planned); companion-only; never shared with Yeetcraft |

---

## Authentication

### Current (Yeetcraft website editing)

The PWA uses a shared secret unlocked via browser `?token=` and sent as `X-API-Key` on `PATCH /api/stats/batch`. This flow is **browser UX**, not a model for desktop credential storage.

### Expected direction (companion — not implemented)

Architectural expectation: companion writes use **dedicated write credentials**, separate from the browser editing token, so they can be revoked independently.

**Open design decisions (unresolved):**

- Credential provisioning (per-installation vs shared group key)
- Rotation and revocation workflow
- Whether credentials differ from today's single `API_KEY` or use per-client records
- Storage mechanism (environment variable during headless dev; Windows Credential Manager with Wails later)

The companion must **not** assume the browser `?token=` flow transfers unchanged to a desktop app. Header-based auth (`X-API-Key` or `Authorization: Bearer`) aligns with existing server middleware patterns, but the **companion-specific credential model awaits Phase 1 contract review**.

---

## Manual edits and aggregate consistency

**Current:** Yeetcraft stores **manually maintained aggregate** deaths and yeets in `player_dungeon_stats`. The PWA and `PATCH /api/stats/batch` can update those totals directly. There is no server-side death-event audit trail today.

**Requirement:** Event-derived companion uploads must **not silently double-count** data that was entered manually or migrated from legacy aggregates.

The recommended reconciliation approach (events + adjustment ledger, legacy baseline import) is documented in the implementation plan — **not implemented**:

- [§11 — Manual edits, historic data, and aggregate consistency](./IMPLEMENTATION_PLAN.md#11-manual-edits-historic-data-and-aggregate-consistency)
- [§11.1 — Compatibility strategy](./IMPLEMENTATION_PLAN.md#111-compatibility-strategy)

Implementing that migration is **Yeetcraft-side work** in a later phase. The companion must not assume aggregates are purely event-derived until the server enforces that model.

---

## Open questions

Unresolved until contract review and Phase 0 evidence:

| Topic | Question |
| ----- | -------- |
| API contract format | JSON schema, OpenAPI, or hand-maintained examples in `contracts/companion/v1/`? |
| Authentication mechanism | Dedicated companion key vs extended `API_KEY`; per-client credentials |
| Player and character identity | GUID-first mapping from combat log to Yeetcraft `players`; character table introduction |
| Duplicate detection | Client event IDs, batch idempotency keys, server `ON CONFLICT` behavior |
| Manual-data migration | Baseline `stat_adjustments` vs recomputation from events |
| Retention | Local normalized events vs raw log retention; server evidence storage policy |
| Aggregate recomputation | Transactional delta vs async recompute; interaction with manual PATCH |
| API version compatibility | Supported `schemaVersion` range; client behavior on 400/422 |

Track combat-log unknowns separately in [COMBAT_LOG_CAPABILITIES.md](./COMBAT_LOG_CAPABILITIES.md).

---

## Security and privacy

- Never commit credentials, tokens, raw combat logs, or SQLite databases.
- Upload **structured, minimal events** (once implemented) — not full combat-log files unless a future approved contract explicitly requires it.
- Redact secrets from diagnostics, examples, and error messages.
- Companion local data stays on the user's machine; Yeetcraft PostgreSQL holds canonical accepted server state only.

---

## Out of scope for this repository

- Implementing Yeetcraft API routes, migrations, or contract files
- Changing Yeetcraft authentication middleware
- Defining the canonical schema (Yeetcraft owns `contracts/companion/v1/`)

If a companion task requires Yeetcraft changes, stop and describe the cross-repository work instead of editing `./yeetcraft` without scope.

---

## Related documentation

- [IMPLEMENTATION_PLAN.md](./IMPLEMENTATION_PLAN.md) — phased roadmap, proposed ingest API, data model
- [COMBAT_LOG_CAPABILITIES.md](./COMBAT_LOG_CAPABILITIES.md) — combat-log research (Phase 0)
- [AGENTS.md](../AGENTS.md) — agent boundaries and verification checklist
- [README.md](../README.md) — companion purpose, build, and current status
