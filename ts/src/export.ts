import type { z } from "zod";

/** Portable schema node matching the Go zod2go.Schema struct. */
export interface IRSchema {
  type: string;
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
  schema: IRSchema;
}

type AnySchema = z.ZodTypeAny;

/** Export a Zod v4 schema to the zod2go intermediate representation. */
export function exportSchema(schema: AnySchema): IRDocument {
  const seen = new WeakMap<object, IRSchema>();
  return {
    $zod2go: {
      version: 1,
      zodVersion: readZodVersion(),
    },
    schema: serializeSchema(schema, seen),
  };
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

function serializeSchema(schema: AnySchema, seen: WeakMap<object, IRSchema>): IRSchema {
  if (seen.has(schema)) {
    return seen.get(schema)!;
  }

  const def = schema.def as unknown as Record<string, unknown>;
  const type = def.type as string;
  const ir: IRSchema = { type };
  seen.set(schema, ir);

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

    case "array":
      ir.element = serializeSchema(def.element as AnySchema, seen);
      ir.checks = serializeChecks(def.checks);
      break;

    case "tuple": {
      const items = def.items as AnySchema[];
      ir.items = items.map((s) => serializeSchema(s, seen));
      if (def.rest) ir.rest = serializeSchema(def.rest as AnySchema, seen);
      ir.checks = serializeChecks(def.checks);
      break;
    }

    case "record":
      if (def.keyType) ir.keyType = serializeSchema(def.keyType as AnySchema, seen);
      ir.valueType = serializeSchema(def.valueType as AnySchema, seen);
      break;

    case "intersection":
      ir.left = serializeSchema(def.left as AnySchema, seen);
      ir.right = serializeSchema(def.right as AnySchema, seen);
      break;

    case "object": {
      const shape = def.shape as Record<string, AnySchema>;
      ir.shape = {};
      for (const [k, v] of Object.entries(shape)) {
        ir.shape[k] = serializeSchema(v, seen);
      }
      if (def.catchall) ir.catchall = serializeSchema(def.catchall as AnySchema, seen);
      // Zod object strictness is encoded via catchall never/unknown; we mark strict when catchall is never.
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

    case "literal":
      ir.value = def.value;
      break;

    case "union":
      ir.options = (def.options as AnySchema[]).map((s) => serializeSchema(s, seen));
      break;

    case "optional":
    case "nullable":
    case "nonoptional":
    case "readonly":
    case "catch":
    case "prefault":
      ir.type = type === "nonoptional" ? "nonoptional" : type;
      ir.inner = serializeSchema(innerSchema(def), seen);
      break;

    case "default":
      ir.inner = serializeSchema(innerSchema(def), seen);
      ir.default = typeof def.defaultValue === "function" ? (def.defaultValue as () => unknown)() : def.defaultValue;
      break;

    case "pipe":
      // validate using input schema (pre-transform)
      return serializeSchema(def.in as AnySchema, seen);

    case "transform":
    case "custom":
    case "lazy":
      // runtime-only; fall back to unknown
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
