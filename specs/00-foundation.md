# 00 — Foundation

Initial zod2go stack: TypeScript IR exporter, Go validator, CLI, Zod submodule, basic tests.

## Deliverables

| Item | Description |
| ---- | ----------- |
| IR envelope | `$zod2go.version`, `schema` root in JSON |
| Core types | string, number, int, boolean, null, array, object, enum, literal, union, optional, nullable, default, any/unknown |
| String formats | email, uuid, url, ipv4, cuid2, ulid, nanoid (RE2-safe subset) |
| CLI | `zod2go-export` (TS), `zod2go` (Go) |
| Submodule | `vendor/zod` pinned to Zod v4 |
| Basic tests | `export.test.ts`, `zod2go_test.go` against `testdata/user.schema.json` |

## Acceptance

- `npm run build` succeeds.
- `go test ./...` passes with `GOTOOLCHAIN=go1.23.6`.
- README quick-start commands work.

## Status

Completed in initial commit `c4a5bb2`.
