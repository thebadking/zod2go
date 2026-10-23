import type { z } from "zod";

/** Portable schema node matching the Go zod2go.Schema struct. */
export interface IRSchema {
  type: string;
  $ref?: string;
  coerce?: boolean;
  checks?: IRCheck[];
  element?: IRSchema;
  items?: IRSchema[];
  rest?: IRSchema;
  shape?: Record<string, IRSchema>;
  keyType?: IRSchema;
  valueType?: IRSchema;
  left?: IRSchema;
  right?: IRSchema;
  options?: IRSchema[];
  values?: unknown[];
  value?: unknown;
  inner?: IRSchema;
  catchall?: IRSchema;
  default?: unknown;
  format?: string;
  pattern?: string;
  uuidVersion?: number;
  strict?: boolean;
}

export interface IRCheck {
  check: string;
  value?: unknown;
  inclusive?: boolean;
  format?: string;
  pattern?: string;
  minimum?: unknown;
  maximum?: unknown;
  exact?: boolean;
}

export interface IRDocument {
  $zod2go: {
    version: number;
    zodVersion: string;
  };
  definitions?: Record<string, IRSchema>;
  schema: IRSchema;
}

type AnySchema = z.ZodTypeAny;

interface ExportContext {
  /** Fully serialized non-lazy schemas (deduplication). */
  seen: WeakMap<object, IRSchema>;
  /** Lazy targets currently being processed — maps schema object → assigned def name. */
  processing: WeakMap<object, string>;
  /** Lazy targets that are fully resolved — maps schema object → def name. */
  lazyNames: WeakMap<object, string>;
  /** Named definitions accumulated during serialization. */
  definitions: Map<string, IRSchema>;
  counter: number;
}

/** Export a Zod v4 schema to the zod2go intermediate representation. */
export function exportSchema(schema: AnySchema): IRDocument {
  const ctx: ExportContext = {
    seen: new WeakMap(),
    processing: new WeakMap(),
    lazyNames: new WeakMap(),
    definitions: new Map(),
    counter: 0,
  };

  const root = serializeSchema(schema, ctx);

  const doc: IRDocument = {
    $zod2go: {
      version: 1,
      zodVersion: readZodVersion(),
    },
    schema: root,
  };

  if (ctx.definitions.size > 0) {
    doc.definitions = Object.fromEntries(ctx.definitions);
  }

  return doc;
}

/** Serialize to pretty-printed JSON. */
export function exportSchemaJSON(schema: AnySchema, space = 2): string {
  return JSON.stringify(exportSchema(schema), null, space);
}

function readZodVersion(): string {
  try {
    const pkg = require("zod/package.json") as { version?: string };
    return pkg.version ?? "4.x";
  } catch {
    return "4.4.3";
  }
}

function serializeLazy(lazy: AnySchema, def: Record<string, unknown>, ctx: ExportContext): IRSchema {
  // Already fully resolved — return $ref.
  if (ctx.lazyNames.has(lazy)) {
    return { type: "$ref", $ref: ctx.lazyNames.get(lazy)! };
  }
  // Currently being processed (cycle) — emit $ref placeholder.
  if (ctx.processing.has(lazy)) {
    return { type: "$ref", $ref: ctx.processing.get(lazy)! };
  }
  // First encounter: assign a name, call getter ONCE, serialize inner.
  const name = `def_${ctx.counter++}`;
  ctx.processing.set(lazy, name);
  const inner = (def.getter as () => AnySchema)();
  const ir = serializeSchema(inner, ctx);
  ctx.processing.delete(lazy);
  ctx.definitions.set(name, ir);
  ctx.lazyNames.set(lazy, name);
  return { type: "$ref", $ref: name };
}

function serializeSchema(schema: AnySchema, ctx: ExportContext): IRSchema {
  if (ctx.seen.has(schema)) {
    return ctx.seen.get(schema)!;
  }

  const def = schema.def as unknown as Record<string, unknown>;
  const type = def.type as string;

  // Lazy schemas get special treatment for cycle detection.
  // Key on the lazy schema object itself, NOT the inner result
  // (getter creates a new object each call, so inner identity is unstable).
  if (type === "lazy") {
    return serializeLazy(schema, def, ctx);
  }

  const ir: IRSchema = { type };
  ctx.seen.set(schema, ir);

  if (def.coerce === true) ir.coerce = true;

  switch (type) {
    case "string":
      ir.checks = serializeChecks(def.checks);
      if (typeof def.format === "string") ir.format = def.format;
      if (typeof def.pattern === "string") ir.pattern = def.pattern;
      break;

    case "number":
    case "int":
    case "boolean":
    case "bigint":
      ir.checks = serializeChecks(def.checks);
      if (type === "number" && hasIntFormat(ir.checks)) {
        ir.type = "int";
      }
      break;

    case "date":
      ir.checks = serializeChecks(def.checks);
      break;

    case "array":
      ir.element = serializeSchema(def.element as AnySchema, ctx);
      ir.checks = serializeChecks(def.checks);
      break;

    case "set":
      // Represent sets as arrays; uniqueness is not enforced in the IR.
      ir.type = "set";
      ir.element = serializeSchema(def.valueType as AnySchema, ctx);
      ir.checks = serializeChecks(def.checks);
      break;

    case "tuple": {
      const items = def.items as AnySchema[];
      ir.items = items.map((s) => serializeSchema(s, ctx));
      if (def.rest) ir.rest = serializeSchema(def.rest as AnySchema, ctx);
      ir.checks = serializeChecks(def.checks);
      break;
    }

    case "record":
      if (def.keyType) ir.keyType = serializeSchema(def.keyType as AnySchema, ctx);
      ir.valueType = serializeSchema(def.valueType as AnySchema, ctx);
      break;

    case "map":
      // JSON maps are serialized as objects; treat like record.
      ir.type = "map";
      if (def.keyType) ir.keyType = serializeSchema(def.keyType as AnySchema, ctx);
      ir.valueType = serializeSchema(def.valueType as AnySchema, ctx);
      break;

    case "intersection":
      ir.left = serializeSchema(def.left as AnySchema, ctx);
      ir.right = serializeSchema(def.right as AnySchema, ctx);
      break;

    case "object": {
      const shape = def.shape as Record<string, AnySchema>;
      ir.shape = {};
      for (const [k, v] of Object.entries(shape)) {
        ir.shape[k] = serializeSchema(v, ctx);
      }
      if (def.catchall) ir.catchall = serializeSchema(def.catchall as AnySchema, ctx);
      if (def.catchall && (def.catchall as AnySchema).def?.type === "never") {
        ir.strict = true;
        delete ir.catchall;
      }
      break;
    }

    case "enum": {
      const entries = def.entries as Record<string, string> | [string, ...string[]];
      if (Array.isArray(entries)) {
        ir.values = [...entries];
      } else {
        ir.values = Object.values(entries);
      }
      break;
    }

    case "literal": {
      // Zod v4 stores literal values as an array in def.values.
      const litVals = def.values as unknown[];
      if (Array.isArray(litVals) && litVals.length === 1) {
        ir.value = litVals[0];
      } else if (Array.isArray(litVals) && litVals.length > 1) {
        // Multi-value literal: serialize as a union of literals.
        ir.type = "union";
        ir.options = litVals.map((v) => ({ type: "literal", value: v }));
      } else if (def.value !== undefined) {
        ir.value = def.value;
      }
      break;
    }

    case "union":
      ir.options = (def.options as AnySchema[]).map((s) => serializeSchema(s, ctx));
      break;

    case "optional":
    case "nullable":
    case "nonoptional":
    case "readonly":
    case "catch":
    case "prefault":
      ir.type = type === "nonoptional" ? "nonoptional" : type;
      ir.inner = serializeSchema(innerSchema(def), ctx);
      break;

    case "default":
      ir.inner = serializeSchema(innerSchema(def), ctx);
      ir.default = typeof def.defaultValue === "function" ? (def.defaultValue as () => unknown)() : def.defaultValue;
      break;

    case "pipe":
      // Use the input (pre-transform) schema; covers .transform() and .refine().
      return serializeSchema(def.in as AnySchema, ctx);

    case "transform":
    case "custom":
      // Runtime-only; fall back to unknown.
      return { type: "unknown" };

    default:
      break;
  }

  return ir;
}

function hasIntFormat(checks?: IRCheck[]): boolean {
  return checks?.some((c) => c.check === "number_format" && (c.format === "safeint" || c.format === "int32")) ?? false;
}

function innerSchema(def: Record<string, unknown>): AnySchema {
  return (def.innerType ?? def.inner) as AnySchema;
}

function serializeChecks(checks: unknown): IRCheck[] | undefined {
  if (!Array.isArray(checks) || checks.length === 0) return undefined;
  const out: IRCheck[] = [];
  for (const chk of checks) {
    const c = chk as { _zod?: { def?: Record<string, unknown> }; def?: Record<string, unknown> };
    const def = (c._zod?.def ?? c.def) as Record<string, unknown> | undefined;
    if (!def?.check) continue;
    // Skip custom (refinement) checks — predicate cannot be serialized.
    if (def.check === "custom") continue;
    const ir: IRCheck = { check: def.check as string };
    if (def.value !== undefined) ir.value = def.value;
    if (def.minimum !== undefined) ir.value = def.minimum;
    if (def.maximum !== undefined) ir.value = def.maximum;
    if (def.inclusive !== undefined) ir.inclusive = def.inclusive as boolean;
    if (def.format !== undefined) ir.format = def.format as string;
    if (def.pattern !== undefined) ir.pattern = String(def.pattern);
    if (def.version !== undefined) ir.value = def.version;
    out.push(ir);
  }
  return out.length ? out : undefined;
}

export { serializeSchema };
