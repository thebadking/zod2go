# Master Implementation Tracker — zod2go

> Update this file whenever you complete or start work on a spec section.
> Last updated: **iteration 4 (full upstream compatibility via native IR and original-module runtime)**.

## Legend

| Status        | Meaning                                  |
| ------------- | ---------------------------------------- |
| `done`        | All deliverables complete, tests in place |
| `in_progress` | Partially implemented                    |
| `not_started` | Spec exists, no implementation yet       |
| `blocked`     | Waiting on external dependency           |

## Overall progress

| Spec | Theme                      | Status | File |
| ---- | -------------------------- | ------ | ---- |
| 00   | Foundation                 | `done` | [`00-foundation.md`](./00-foundation.md) |
| 01   | Extended IR types          | `done` | [`01-extended-ir-types.md`](./01-extended-ir-types.md) |
| 02   | Strict & array checks      | `done` | [`02-strict-and-array-checks.md`](./02-strict-and-array-checks.md) |
| 03   | Demo playground            | `done` | [`03-demo-playground.md`](./03-demo-playground.md) |
| 04   | Cross-runtime parity       | `done` | [`04-cross-runtime-parity.md`](./04-cross-runtime-parity.md) |
| 06   | Full upstream compatibility | `done` | [`06-full-compatibility.md`](./06-full-compatibility.md) |
| 05   | Export pipeline            | `done` | [`05-export-pipeline.md`](./05-export-pipeline.md) |

Specs 00–05 established native IR. Spec 06 adds the original-module runtime for complete JavaScript behavior, typed transport, all inventoried feature gates, and full upstream conformance through Go. See spec 06 for the execution boundary and reproducible evidence. Earlier iteration notes below describe historical behavior.

---

## Section detail

### 00 — Foundation

| Deliverable              | Status | Tests |
| ------------------------ | ------ | ----- |
| IR exporter + Go runtime | done   | initial commit |
| CLI tools                | done   | manual |
| User fixture             | done   | `zod2go_test.go` |

---

### 01 — Extended IR types

| Deliverable        | Status | Tests |
| ------------------ | ------ | ----- |
| Tuple export/parse | done   | `export.test.ts`, `extended_test.go`, `parity.test.ts` |
| Record export/parse| done   | same |
| Intersection       | done   | same |
| Extended fixture   | done   | `testdata/extended.schema.json` |

---

### 02 — Strict & array checks

| Deliverable              | Status | Tests |
| ------------------------ | ------ | ----- |
| Strict object validation | done   | `extended_test.go`, `parity.test.ts` |
| Array min_length         | done   | `extended_test.go`, `export.test.ts` |
| Optional tags min_length | done   | `extended_test.go` |

---

### 03 — Demo playground

| Deliverable     | Status | Tests |
| --------------- | ------ | ----- |
| index.html      | done   | manual |
| demo/main.ts    | done   | manual |
| Vite dev config | done   | `npm run dev` |

---

### 04 — Cross-runtime parity

| Deliverable           | Status | Tests |
| --------------------- | ------ | ----- |
| Parity case table     | done   | `parity.test.ts` (shared native cases and legacy JSON wire cases); full runtime coverage in spec 06 |
| Go subprocess harness | done   | `parity.test.ts` |

---

### 05 — Export pipeline

| Deliverable        | Status | Tests |
| ------------------ | ------ | ----- |
| export:extended    | done   | `npm run export:extended` |
| export:fixtures    | done   | `npm run export:fixtures` |
| Regenerate user IR | done   | includes optional `tags` with min(1) |

---

## Changelog

| Date       | Agent       | Change |
| ---------- | ----------- | ------ |
| 2026-08-23 | agent-ipqj  | Created specs/; implemented extended IR, strict/checks, demo, parity, export pipeline; all tests pass |
| 2026-10-23 | sonnet-4-6  | Iteration 2: lazy/recursive schemas ($ref + definitions), date type, set/map types, skip custom checks (refinement pass-through), literal fix for Zod v4 values array, expanded parity suite (97 TS tests, 19 Go tests) |


### Iteration 3 — Verification and consolidation (2026-10-23)

- Inspected Git and managed worktrees: no outstanding worktrees, unreachable commits, or unmerged branch commits. Three legacy agent branches were already ancestors of main.
- Verified the original baseline: 97 TypeScript tests and 19 Go tests passed. The shared parity table contained 40 cases, not the reported 44.
- Fixed reference chains/cycles, recursive object-root export, repeated transform export, lazy getter caching, prototype-named properties, and numeric enum reverse mappings.
- Fixed literal type equality, missing versus null handling, optional defaults, prefault/readonly/nonoptional wrappers, nested error paths, tuple excess items, enum record exhaustiveness, and recursive intersection output merging.
- Fixed JavaScript JSON coercions, UTF-16 string lengths, safe integer checks, decimal multiples, regex source/flags, and date bound issue metadata.
- Parity now compares successful parsed output as well as acceptance. Infrastructure failures fail the test; temporary binaries and fixtures are cleaned up.
- Validation: 220 TypeScript tests and 25 Go tests; TypeScript build, Go vet, fixture regeneration, and demo build/smoke checks.
- Added a runtime TypeScript loader dependency for CLI exports, verified with native Node type stripping disabled.
- Verified the pinned submodule matches npm Zod 4.4.3.
- Compatibility limitations remain explicit in README; exhaustive upstream fixture coverage is outside the defined specs.
