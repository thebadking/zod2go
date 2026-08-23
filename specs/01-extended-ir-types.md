# 01 — Extended IR types

Add tuple, record, and intersection to the IR exporter and Go runtime.

## Deliverables

| Item | TS exporter | Go runtime |
| ---- | ----------- | ---------- |
| Tuple | `items[]`, optional `rest` | Fixed-length validation, per-index paths |
| Record | `keyType`, `valueType` | Key/value validation |
| Intersection | `left`, `right` | Both branches must pass; merge object outputs |

## Example schema

`examples/schemas.ts` exports `ExtendedSchema` with tuple, record, intersection, and strict object.

## Fixtures

- `testdata/extended.schema.json` — exported from `ExtendedSchema`

## Tests

- `ts/src/export.test.ts` — tuple/record/intersection export shapes
- `extended_test.go` — valid input, tuple type error

## Acceptance

Export and Go parse agree on valid/invalid cases for all three types.
