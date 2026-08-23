# 05 — Export pipeline

Scripts to export example schemas to committed fixtures.

## Deliverables

| Script | Output |
| ------ | ------ |
| `npm run export:example` | `testdata/user.schema.json` from `UserSchema` |
| `npm run export:extended` | `testdata/extended.schema.json` from `ExtendedSchema` |
| `npm run export:fixtures` | Both of the above |

## Acceptance

Running `npm run export:fixtures` after `npm run build` regenerates fixtures that match `examples/schemas.ts`.

User fixture includes optional `tags` array with `min(1)` check.
