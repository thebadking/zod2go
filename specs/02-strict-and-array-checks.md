# 02 — Strict objects and array checks

Object strictness and collection size checks aligned with Zod v4.

## Deliverables

| Item | Description |
| ---- | ----------- |
| Strict objects | `.strict()` → `strict: true` in IR; reject unknown keys with `unrecognized_keys` |
| Array `min_length` | Zod `.min(n)` on arrays → `checks: [{ check: "min_length", value: n }]` |
| Optional nested arrays | min_length on optional inner array (e.g. `tags`) |

## Tests

- `extended_test.go` — `TestExtendedSchemaStrictKeys`, `TestExtendedSchemaArrayMinLength`
- `extended_test.go` — `TestUserSchemaTagsMinLength`
- `export.test.ts` — array min_length export

## Acceptance

Empty `tags: []` fails when schema requires `min(1)`.
Extra object keys fail when schema is strict.
