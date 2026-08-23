# zod2go — Agent Guidelines

> **Read this file first** after `.cursor/ralph/directive.md`.
> Specs in this directory are the source of truth for deliverables.

## Workflow (mandatory order)

1. Read [`.cursor/ralph/directive.md`](../.cursor/ralph/directive.md) — current task and environment.
2. Read this file — conventions and testing rules.
3. Open [`MASTER_IMPLEMENTATION.md`](./MASTER_IMPLEMENTATION.md) — find the next incomplete deliverable.
4. Open the relevant numbered spec — implement only what it defines.
5. Update `MASTER_IMPLEMENTATION.md` and directive notes each iteration.
6. Run `npm test` before finishing an iteration.
7. Set `complete: true` in directive frontmatter when every spec deliverable is `done`.

## Repository rules

- **Worktree only:** edit files in this agent worktree (`/.cp-worktrees/agent-ipqj`), not other worktrees or the main repo unless merging.
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

## Out of scope (document, do not implement yet)

- **Transforms / refinements / lazy:** runtime-only in Zod; exporter falls back to `unknown`.
- **Full upstream Zod fixture port:** future work once core IR is stable.
