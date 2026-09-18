# Companion v1 derived fixtures

> **Status:** Phase 1 Validate — checksum record only; not a contract fork

Canonical schemas and examples live only in Yeetcraft:

`../yeetcraft/contracts/companion/v1/`

This repository **must not** independently edit those files. Phase 2 tests may
generate copies under `testdata/contract/v1/derived/` by reading the sibling
checkout (developer machines) or a CI checkout of Yeetcraft. Generated copies
are verified against [`CANONICAL_CHECKSUMS.sha256`](./CANONICAL_CHECKSUMS.sha256),
which is a recorded snapshot of Yeetcraft
[`CHECKSUMS.sha256`](../../../../Yeetcraft/contracts/companion/v1/CHECKSUMS.sha256).

## Drift strategy

| Event | Action |
| ----- | ------ |
| Yeetcraft fixture/schema change | Re-run Yeetcraft `scripts/validate-contract.ps1 -WriteChecksums`, then copy the new `CHECKSUMS.sha256` over this file in a companion PR. Do not hand-edit hashes. |
| Companion test needs JSON bytes | Generate derived copies in the test helper; never commit an independently edited schema tree. |
| Sibling Yeetcraft missing (CI of companion-only) | Trust this recorded checksum file; full byte comparison against Yeetcraft is **deferred to Phase 2/3 CI** that checks out both repos or vendors the checksum. |
| Hash mismatch | Fail the test. Do not “fix” companion copies to match local experiments. |

## What is in this directory now

- [`CANONICAL_CHECKSUMS.sha256`](./CANONICAL_CHECKSUMS.sha256) — recorded SHA-256 of canonical schemas and examples.
- This README.

There is **no** copied `schema/` or `examples/` tree here.

Verify (sibling checkout required for live hash compare):

```powershell
powershell -NoProfile -File scripts/verify-canonical-checksums.ps1
```

Pinned Yeetcraft validator (reported, not a companion Go/npm dependency):
`ajv-cli@5.0.0` / `ajv@8.17.1`.
