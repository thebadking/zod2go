# @zod2go/export

Export real Zod v4 schemas for the [zod2go Go validator](https://github.com/thebadking/zod2go).
Requires Node.js 22.12 or newer.

```ts
import { exportSchema } from "@zod2go/export";
import { exportModuleSchema } from "@zod2go/export/module";

// Portable native Go IR. Callback schemas require the original module runtime.
const native = exportSchema(schema, { encoding: "tagged" });
// Preserves closures, refinements, transforms, async checks, and codecs.
const runtime = await exportModuleSchema("./schemas.ts", "MySchema");
```

```sh
zod2go-export schemas.ts MySchema schema.json
zod2go-export --runtime schemas.ts MySchema schema.runtime.json
zod2go-export --runtime-reference schemas.ts MySchema schema.reference.json
```

Module bundles include dependencies. Reference exports retain the original module
location and installed dependencies, for modules that load local assets or dynamic
imports. Both require explicit `NewRuntimeValidator` / CLI `-runtime` execution in Go.
See the repository README for typed transport, async validation, encoding, and errors.
