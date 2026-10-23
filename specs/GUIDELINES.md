# zod2go — Agent Guidelines

> Specs in this directory define the module deliverables.

## Workflow

1. Read this file and [`MASTER_IMPLEMENTATION.md`](./MASTER_IMPLEMENTATION.md).
2. Inspect the current checkout, branches, and worktrees before editing.
3. Read the relevant numbered spec and the user's current task.
4. Keep the implementation tracker current.
5. Run `npm test` and `npm run build` before finishing.

## Repository rules

- **Checkout:** use the current authorized checkout. Preserve unrelated changes and consolidate branches only after verifying their contents.
- **IR + runtime together:** TypeScript exporter (`ts/src/export.ts`) and Go validator (`validate.go`, `parse.go`) must stay in sync for every supported type.
- **Real Zod schemas:** examples and tests use Zod v4 from `vendor/zod` / npm `zod@4.4.3`.
- **Fixtures:** exported IR JSON lives in `testdata/`; regenerate with `npm run export:fixtures`.

## Testing philosophy

| Layer | What to test | Tool |
| ----- | ------------ | ---- |
| IR export | Schema node shapes, checks, strict flags | Vitest `export.test.ts` |
| Cross-runtime parity | Zod `safeParse` vs Go `Validator.Parse` agree | Vitest `parity.test.ts` + Go tests |
| Go runtime | Parse/coerce/error codes on fixtures | Go `zod2go_test.go`, `extended_test.go` |

Skip exhaustive combinatorics; focus on representative cases per IR type.

## Code conventions

- **Go:** package `zod2go` at repo root; CLI in `cmd/zod2go`.
- **TypeScript:** workspace `@zod2go/export` under `ts/`.
- **Errors:** mirror Zod v4 issue codes (`invalid_type`, `too_small`, `invalid_format`, …).
- **Dev server:** Vite on `0.0.0.0:4020`, `allowedHosts` includes `a4100b3d.naori.cloud`.

## Go toolchain

The host Go 1.24 install may be broken. Tests use `GOTOOLCHAIN=go1.23.6` (see `package.json` scripts).

## Implemented (formerly out of scope)

- **Transforms:** `.transform()` is a pipe; exporter uses input schema (pre-transform). ✅
- **Refinements:** `.refine()` attaches a `custom` check; exporter skips it and validates the base schema. ✅
- **Lazy/recursive schemas:** exported as `$ref` nodes with a `definitions` map in the document; Go runtime resolves refs at validation time. ✅
- **Date type:** exported as `{"type":"date"}`; Go validates ISO 8601 strings. ✅
- **Set/Map types:** exported as `set`/`map` IR types; Go validates element/value types. ✅

## Remaining out of scope

- **Full upstream Zod test fixture port:** exhaustive combinatoric coverage against every upstream Zod test case; representative parity is in place.
