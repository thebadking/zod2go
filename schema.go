// Package zod2go validates data using schemas exported from Zod (TypeScript).
//
// Schemas are serialized to the zod2go IR format by the @zod2go/export TypeScript
// package, which walks Zod v4 schema definitions from the vendor/zod submodule.
package zod2go

// Version tracks the zod2go IR format version.
const Version = 1

// ZodVersion is the upstream Zod semver this release targets (from vendor/zod).
const ZodVersion = "4.4.3"
