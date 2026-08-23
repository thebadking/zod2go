# 04 — Cross-runtime parity tests

Ensure Zod TypeScript and Go validator agree on success/failure for shared cases.

## Deliverables

| Item | Description |
| ---- | ----------- |
| Parity cases | Representative schemas + inputs (user, tuple, record, intersection, array min) |
| Zod check | Each case runs `schema.safeParse(input)` |
| Go check | Export schema to temp file, run `go run ./cmd/zod2go -schema …` with JSON stdin |
| Agreement | `zodResult.success === goResult.success` for every case |

## Tests

- `ts/src/parity.test.ts` — export shape checks + Go subprocess parity
- Existing Go fixture tests remain as regression guard

## Acceptance

All parity cases pass; mismatches fail the test with case name.
