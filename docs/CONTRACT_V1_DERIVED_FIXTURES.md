# Companion v1 derived-fixture drift strategy

> **Status:** Phase 1 Validate notes — not a second contract

Yeetcraft owns the canonical ingest contract. This repository records how
later tests may use **derived** copies without forking the protocol.

| Item | Location |
| ---- | -------- |
| Canonical files | `../yeetcraft/contracts/companion/v1/schema/`, `examples/` |
| Canonical checksum set | [`../../yeetcraft/contracts/companion/v1/CHECKSUMS.sha256`](../../yeetcraft/contracts/companion/v1/CHECKSUMS.sha256) |
| Recorded companion copy of that set | [`../testdata/contract/v1/CANONICAL_CHECKSUMS.sha256`](../testdata/contract/v1/CANONICAL_CHECKSUMS.sha256) |
| Verify script | [`../scripts/verify-canonical-checksums.ps1`](../scripts/verify-canonical-checksums.ps1) |
| Producer review | [`CONTRACT_V1_WP1_REVIEW.md`](./CONTRACT_V1_WP1_REVIEW.md) |

Do not copy `CONTRACT.md` or JSON Schema into this repo as an edited source.

## Deferred to Phase 2 / Phase 3 CI

- GitHub Actions that run Yeetcraft `validate-contract.ps1`.
- `go test` that materializes derived JSON under `testdata/contract/v1/derived/`
  and asserts SHA-256 equality with the recorded checksums.
- Any checkout of Yeetcraft from companion CI.

Phase 1 only records the checksum file and the verify script for local sibling
checkouts.
