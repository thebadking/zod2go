# zod2go

Validate data in Go using **real [Zod](https://zod.dev) v4 schemas** defined in TypeScript.

Export existing Zod schemas to portable IR for native Go validation, or execute the original schema module through a persistent Node.js worker for complete JavaScript behavior, including closures and async callbacks.

## Why zod2go?

| Approach | Uses real Zod schemas | Stays in sync with upstream |
|----------|----------------------|----------------------------|
| zod-go (inspiration) | No — Go-native fluent API | Manual parity work |
| **zod2go** | **Yes — export from TypeScript** | **`vendor/zod` git submodule** |

## License

**Use the [MIT License](LICENSE).**

MIT is the recommended choice because:

- **[Zod itself is MIT](vendor/zod/LICENSE)** — matching upstream avoids license friction when vendoring and referencing Zod.
- **Permissive and commercial-friendly** — use, modify, and distribute freely with minimal obligations (preserve copyright notice).
- **Widely understood** — compatible with most open-source and proprietary stacks.

Alternatives: **Apache 2.0** (similar permissiveness + explicit patent grant) or **BSD-2/3-Clause**. Avoid copyleft (GPL) if you want maximum adoption in Go services and libraries.

## Upstream Zod submodule

Zod is vendored as a git submodule at `vendor/zod` (from `https://github.com/colinhacks/zod.git`):

```bash
# Initialize / update submodule
git submodule update --init --recursive vendor/zod

# Pull latest upstream Zod
npm run sync:zod
# or: ./scripts/sync-zod.sh
```

The TypeScript exporter (`@zod2go/export`) uses the pinned npm dependency `zod@4.4.3`. The submodule pins `2d90846a` (19 commits after v4.4.3) for upstream conformance tests. Module-backed validation uses the Zod version imported by your module; the two source revisions are recorded separately.

## Quick start

Use Node.js 22.12+ (Node 24 recommended) and Go 1.23.6+ for development. The export CLI loads TypeScript schemas through its `tsx` runtime dependency.

### 1. Define schemas in TypeScript

```typescript
// examples/schemas.ts
import { z } from "zod";

export const UserSchema = z.object({
  name: z.string().min(2),
  email: z.email(),
  role: z.enum(["admin", "user"]).default("user"),
});
```

### 2. Export to zod2go IR

```bash
npm install
npm run build
npx zod2go-export examples/schemas.ts UserSchema testdata/user.schema.json
```

### 3. Validate in Go

```go
package main

import (
    "fmt"
    "github.com/thebadking/zod2go"
)

func main() {
    doc, _ := zod2go.LoadDocument("testdata/user.schema.json")
    v := zod2go.NewValidator(doc)

    out, err := v.Parse(map[string]any{
        "name":  "Ada",
        "email": "ada@example.com",
    })
    if err != nil {
        panic(err)
    }
    fmt.Printf("%+v\n", out)
}
```

CLI:

```bash
go run ./cmd/zod2go -schema testdata/user.schema.json '{"name":"Ada","email":"ada@example.com"}'
```

## Architecture

```
TypeScript (Zod v4)          zod2go IR (JSON)           Go runtime
───────────────────          ────────────────           ──────────
z.object({...})      ──►    { "$zod2go": {...},   ──►  Validator.Parse()
z.string().email()           "schema": { ... } }        Zod-compatible errors
```

The IR mirrors Zod v4 `$ZodTypeDef` nodes: `string`, `number`, `object`, `array`, `union`, `enum`, `optional`, `default`, and attached `$ZodCheck` entries (`min_length`, `greater_than`, `string_format`, …).

## Development

Development and tests require Node.js 22.12+ and an installed Go toolchain.

See [`specs/`](specs/) for agent guidelines, implementation tracker, and per-feature specs.

```bash
# Dev server (Control Plane / Traefik)
npm run dev -- --host 0.0.0.0 --port 4020

# Regenerate IR fixtures from examples/schemas.ts
npm run export:fixtures

# Tests
npm test          # TS + Go
go test ./...
npm run test:ts
```

## Inspiration

The Go validation patterns and error ergonomics are informed by [aymaneallaoui/zod-go](https://github.com/aymaneallaoui/zod-go) (also MIT), but zod2go’s goal is **direct compatibility with Zod TypeScript schemas**, not a separate Go builder API.

## Original-module runtime

For callbacks, closures, asynchronous schemas, codecs, or native JavaScript values,
export the original module and execute it with a persistent Node.js worker:

```bash
npm run build
node ts/dist/cli.js --runtime examples/runtime-schemas.ts ClosureSchema /tmp/schema.runtime.json
go run ./cmd/zod2go -runtime -schema /tmp/schema.runtime.json '"Ada"'
```

Module exports are bundled with their dependencies. Node.js is needed at runtime.
Use `--runtime-reference` for modules that depend on their original file location,
local assets, or dynamic imports; retain the source files and installed dependencies.
In Go, opt in explicitly and close the worker when finished:

```go
ctx := context.Background()
v, err := zod2go.NewRuntimeValidator(ctx, doc, zod2go.RuntimeOptions{})
if err != nil { panic(err) }
defer v.Close()
out, err := v.Parse("Ada")
```

`ParseAsync`, `Encode`, and `EncodeAsync` preserve the corresponding Zod operations.
The CLI selects these with `-op parseAsync`, `-op encode`, or `-op encodeAsync`.
Its runtime timeout defaults to 30 seconds; set `-timeout 0` to disable it.
Ordinary `NewValidator` and document loading never execute bundled code.

The runtime accepts Go `*big.Int`, `time.Time`, `RuntimeSet`, `RuntimeMap`,
`RuntimeFile`, `RuntimePromise`, `RuntimeRegExp`, and `Undefined`. Arbitrary map keys
and non-finite numbers retain their types. `ExportValue` retrieves original module
values such as symbols; `RuntimeValue` handles support `Call`, `Get`, `Invoke`, and
`Construct`, `Await`, and `Release` while their worker remains open. Reference variants such as `ParseReference` preserve original JavaScript object identity, getters, and property descriptors. Per-parse parameters and error callbacks are supported with `ParseWithParams` and
the corresponding async/encode methods. `RuntimeException` preserves thrown
JavaScript exception names, messages, and stacks. `RuntimeError.Issues` retains nested Zod
issue fields. The `-tagged` CLI flag, TS `encodeValue`/`decodeValue`, and Go
`EncodeTaggedValue`/`DecodeTaggedValue` APIs use tagged transport to preserve values outside JSON's type system.

Native IR supports legacy JSON wire forms and typed transport. Export with
`--tagged` or `{encoding:"tagged"}` and use `NewTypedValidator` for native
Date/Set/Map/bigint values. The CLI honors the document encoding. Use `-stream`
with runtime mode for persistent typed RPC, or the Go API for concurrent requests.

## Compatibility status

- **Full semantics:** module-backed validation supports every inventoried Zod v4 schema type, check, string format, callbacks, async operations, and codec direction. It executes the original JavaScript module and requires Node.js.
- **Native Go IR:** string, number, int, bigint, boolean, null, date, array, set, map, object, enum, literal, union/XOR, optional, nullable, default, prefault, readonly, nonoptional, tuple, record, intersection, pipes, any/unknown, and lazy/recursive references. Callbacks and other schema types require module-backed export; the exporter rejects them explicitly.
- **Native wire forms:** legacy JSON uses date strings, arrays for sets, and string-key objects for maps. Typed transport preserves Date/Set/Map distinctions, arbitrary map keys, undefined, and non-finite numbers. Sets deduplicate values and maps replace duplicate keys.
- **Native defaults:** captured at export time; typed defaults preserve their types and are decoded afresh per parse. Dynamic factories execute per parse in module-backed mode.
- **Native errors and regex:** native validation provides Zod-compatible issue codes and paths, using Go regex rules plus format-specific validation. Use module-backed validation for exact JavaScript regex behavior and original messages, error maps, and nested issue metadata.
- **Evidence:** all 110 pinned upstream files / 1,444 tests pass through the Go runtime integration. Native replay matches 2,299 distinct fixtures with zero mismatches. Inventory gates cover 39 schema types, 15 check kinds, 26 named string formats, and 15 hash variants. See [verification details](specs/06-full-compatibility.md).

Run the complete conformance gate after initializing the submodule:

```sh
npm ci
npm run verify
```

The gate runs the upstream baseline, native replay, and full runtime lane, then
checks that every case requiring runtime execution is covered. Reports are written
to `/tmp/zod2go-{native,runtime}-fixture-reports`; CI retains them as artifacts.
Compatibility evidence applies to the pinned revision and tested Node/Go versions.

Contributions welcome — update the source inventory and both validation lanes together.
