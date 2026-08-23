# Master Implementation Tracker — zod2go

> Update this file whenever you complete or start work on a spec section.
> Last updated: **iteration 1 (complete)**.

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
| 05   | Export pipeline            | `done` | [`05-export-pipeline.md`](./05-export-pipeline.md) |

All defined specs fulfilled. Future work (transforms, refinements, lazy schemas) is documented as out of scope in [`GUIDELINES.md`](./GUIDELINES.md).

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
| Parity case table     | done   | `parity.test.ts` (7 cases × 2 tests) |
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
