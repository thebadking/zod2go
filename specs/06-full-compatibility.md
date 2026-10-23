# 06 — Full upstream compatibility

Every inventoried feature has an execution path. Native Go IR handles portable
schemas; the original-module Node runtime preserves arbitrary JavaScript closures,
custom predicates, transforms, async callbacks, codecs, functions, files, symbols,
and other JavaScript semantics. Complete semantics require Node.js. Native IR does
not claim to implement every JavaScript behavior or exact upstream error message.

## Source and inventory

- npm exporter dependency: `zod@4.4.3`.
- Upstream conformance source: submodule `2d90846af918af9602e088812d63a035d47cdbe4`,
  19 commits after v4.4.3. These revisions are deliberately recorded separately;
  module-backed execution uses the dependency actually imported by the module.
- `upstream-inventory.json` records source hashes, 39 schema types, 15 check kinds,
  26 named string formats, 15 hash variants, and all 110 upstream v4 test files.
- `npm run check:inventory` rejects stale inventory. Go integration tests assert
  complete inventory membership for types, checks, formats, and hashes.

## Verification lanes

| Lane | Command | Verified result |
| --- | --- | --- |
| Upstream baseline | `npm run test:upstream` | 110 files / 1,444 tests |
| Native Go replay | `npm run test:upstream:native` | 110 files / 1,444 tests; 2,299 distinct portable fixtures, 2,507 calls, zero acceptance/output mismatches |
| Full Go runtime | `npm run test:upstream:runtime` | 110 files / 1,444 tests; 3,727 root calls through Go, 2 nested calls, 28 original exception paths |
| Combined coverage | `npm run check:upstream:coverage` | Zero missing files or uncovered native-ineligible named cases |
| Project tests | `npm test` | 291 TypeScript tests and all Go tests pass |

The native lane runs upstream tests unchanged and captures portable parse inputs
and results for replay through the native Go validator. It records every case that
requires runtime execution, including callbacks, encoding, stateful factories,
JavaScript-only values, and original exceptions. It compares acceptance and parsed
outputs; exact error metadata is checked in the full runtime lane.

The full runtime lane hooks all four upstream core parse entry points. Live schemas,
inputs, contexts, and results pass through the production Go `NewRuntimeConnection`
and `RunReference` APIs. Original JavaScript objects remain in their realm, retaining
closures, prototypes, getters, identity, promises, and thrown values. The original
upstream assertions then inspect results and errors. Nested operations execute
inside the original validation call and are counted separately. No test files are
excluded. This proves the Node-backed Go integration, not native IR equivalence for
all JavaScript features.

Each lane writes reports tagged with a fresh run ID. The combined gate requires a
report for every inventory file, zero native mismatches, and runtime coverage for
every named case unavailable to native replay. CI retains detailed reports. Tests
of the ordinary bundled/reference module APIs separately exercise by-value typed
transport, handles, module loading, async operations, both codec directions, and
all inventoried types/checks/formats.

## Implemented behavior

- Native bigint precision, numeric formats, coercion, date bounds, set/map
  semantics, finite/partial/loose records, XOR, exact optional properties, tuple
  defaults, intersections, recursive references, and exported format patterns.
- Callback/unsupported native exports fail explicitly with module-runtime guidance.
- Module bundles preserve closures and imports; reference mode preserves original
  paths, local assets, and dynamic dependency resolution.
- Typed graph transport preserves undefined, non-finite numbers, bigint, dates,
  maps, sets, cyclic/shared structures, files, and original object/function/symbol
  handles. Reference APIs retain getters and property descriptors. Opaque handles
  support malformed UTF-16 strings without lossy JSON replacement.
- Sync/async parse and encode, per-call parameters/error maps, complete Zod issues,
  thrown values, safe-result APIs, concurrent requests, cancellation, and cleanup.
- Native defaults are freshly decoded per parse; runtime factories execute in the
  original module. Native JSON wire conventions remain supported.

## Reproduction and maintenance

Use Node 24 and Go 1.23.6; initialize the pinned submodule, then run:

```sh
npm ci
npm run verify
GOTOOLCHAIN=go1.23.6 go vet ./...
GOTOOLCHAIN=go1.23.6 go test -race ./...
npm run export:fixtures
npm exec vite build -- --config ts/vite.config.ts
npm pack -w @zod2go/export --pack-destination /tmp
```

The exporter package is additionally installed and exercised from a separate
consumer project, including native/core CLI export, bundled/reference runtime
export, and execution by the Go CLI. The browser playground is built and exercised
with valid input, invalid schema data, and invalid JSON. CI repeats the conformance,
race, fixture, package, and demo build checks. Coverage evidence is bounded to the
pinned source and tested platforms; updating upstream requires rerunning the gate.
