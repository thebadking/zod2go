# 04 — Cross-runtime parity tests

Ensure Zod TypeScript and Go validator agree on success/failure for shared cases.

## Deliverables

| Item | Description |
| ---- | ----------- |
| Parity cases | Representative schemas + inputs (user, tuple, record, intersection, array min) |
| Zod check | Each case runs `schema.safeParse(input)` |
| Go check | Export schema to temp file, run a prebuilt `./cmd/zod2go` binary with `-schema …` with JSON stdin |
| Agreement | `zodResult.success === goResult.success` for every shared case; compare parsed outputs on success. Date/Set/Map values use documented JSON wire forms. Refinement/transform pass-through differences have separate Go assertions. |

## Tests

- `ts/src/parity.test.ts` — export shape checks + Go subprocess parity
- Existing Go fixture tests remain as regression guard

## Acceptance

All parity cases pass; mismatches fail the test with case name.
