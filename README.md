# zod2go

Validate data in Go using **real [Zod](https://zod.dev) v4 schemas** defined in TypeScript.

Unlike reimplemented Zod-like APIs (see `inspiration/zod-go`), zod2go exports your existing Zod schema definitions to a portable intermediate representation (IR) and validates JSON in Go with Zod-compatible error codes and paths.

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

Zod is vendored as a git submodule at `vendor/zod` (from `git@github.com:colinhacks/zod.git`):

```bash
# Initialize / update submodule
git submodule update --init --recursive vendor/zod

# Pull latest upstream Zod
npm run sync:zod
# or: ./scripts/sync-zod.sh
```

The TypeScript exporter (`@zod2go/export`) depends on this submodule so schema serialization tracks Zod v4 internals.

## Quick start

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

See [`specs/`](specs/) for agent guidelines, implementation tracker, and per-feature specs.

```bash
# Dev server (Control Plane / Traefik)
npm run dev -- --host 0.0.0.0 --port 4020

# Regenerate IR fixtures from examples/schemas.ts
npm run export:fixtures

# Tests
npm test          # TS + Go
go test ./...
npm run test:ts -w @zod2go/export
```

## Inspiration

The Go validation patterns and error ergonomics are informed by [aymaneallaoui/zod-go](inspiration/zod-go) (also MIT), but zod2go’s goal is **direct compatibility with Zod TypeScript schemas**, not a separate Go builder API.

## Compatibility status

- **Target:** Zod v4.4.x (see `vendor/zod`)
- **Supported IR types:** string, number, int, boolean, null, date, array, set, map, object, enum, literal, union, optional, nullable, default, tuple, record, intersection, any/unknown, lazy/recursive (`$ref` + definitions)
- **Refinements/transforms:** base schema validated; custom predicates and transform functions are skipped (cannot be serialized to IR)
- **Remaining:** exhaustive upstream Zod fixture parity

Contributions welcome — keep `vendor/zod` updated and extend the IR exporter + Go runtime together.
