import type { z } from "zod";
import { encodeValue, type TaggedValue } from "./wire.js";

/** Portable schema node matching the Go zod2go.Schema struct. */
export interface IRSchema {
  type: string;
  $ref?: string;
  coerce?: boolean;
  optionalInput?: boolean;
  optionalOutput?: boolean;
  exactOptional?: boolean;
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
  valueKind?: string;
  inner?: IRSchema;
  catchall?: IRSchema;
  default?: unknown;
  defaultEncoded?: TaggedValue;
  format?: string;
  pattern?: string;
  uuidVersion?: number;
  strict?: boolean;
  exclusive?: boolean;
  keyValues?: unknown[];
  partialKeys?: boolean;
  mode?: string;
  alg?: string;
  hostname?: string;
  protocol?: string;
  normalize?: boolean;
  position?: number;
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
  alg?: string;
  hostname?: string;
  protocol?: string;
  normalize?: boolean;
  position?: number;
}

export interface IRDocument {
  $zod2go: {
    version: number;
    encoding?: "json" | "tagged";
    zodVersion: string;
  };
  runtime?: { bundle?: string; modulePath?: string; loader?: string; coreModule?: string; exportName: string; format?: "cjs" | "esm" };
  definitions?: Record<string, IRSchema>;
  schema: IRSchema;
}

type AnySchema = z.core.$ZodType & { def?: z.core.$ZodTypeDef };

interface ExportContext {
  /** Fully serialized non-lazy schemas (deduplication). */
  seen: WeakMap<object, IRSchema>;
  /** Lazy targets currently being processed — maps schema object → assigned def name. */
  processing: WeakMap<object, string>;
  /** Lazy targets that are fully resolved — maps schema object → def name. */
  lazyNames: WeakMap<object, string>;
  /** Named definitions accumulated during serialization. */
  definitions: Map<string, IRSchema>;
  active: WeakSet<object>;
  recursiveNames: WeakMap<object, string>;
  counter: number;
  encoding?: "json" | "tagged";
}

/** Export a Zod v4 schema to the zod2go intermediate representation. */
export function exportSchema(schema: AnySchema,options: {encoding?: "json" | "tagged"} = {}): IRDocument {
  const ctx: ExportContext = {
    seen: new WeakMap(),
    processing: new WeakMap(),
    lazyNames: new WeakMap(),
    definitions: new Map(),
    active: new WeakSet(),
    recursiveNames: new WeakMap(),
    counter: 0,
    encoding: options.encoding,
  };

  const root = serializeSchema(schema, ctx);

  const doc: IRDocument = {
    $zod2go: {
      version: 1,
      zodVersion: readZodVersion(schema),
    },
    schema: root,
  };

  if(options.encoding) doc.$zod2go.encoding=options.encoding;
  if (ctx.definitions.size > 0) {
    doc.definitions = Object.fromEntries(ctx.definitions);
  }

  return doc;
}

/** Serialize to pretty-printed JSON. */
export function exportSchemaJSON(schema: AnySchema, space = 2,options:{encoding?:"json"|"tagged"} = {}): string {
  return JSON.stringify(exportSchema(schema,options), null, space);
}

function readZodVersion(schema: AnySchema): string {
  const version=schema._zod.version;
  return `${version.major}.${version.minor}.${version.patch}`;
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
  const inner = (lazy._zod as unknown as { innerType: AnySchema }).innerType;
  const ir = serializeSchema(inner, ctx);
  ctx.processing.delete(lazy);
  ctx.definitions.set(name, ir);
  ctx.lazyNames.set(lazy, name);
  return { type: "$ref", $ref: name };
}

function serializeSchema(schema: AnySchema, ctx: ExportContext): IRSchema {
  if (ctx.seen.has(schema)) {
    if (ctx.active.has(schema)) {
      let name = ctx.recursiveNames.get(schema);
      if (!name) {
        name = `def_${ctx.counter++}`;
        ctx.recursiveNames.set(schema, name);
        ctx.definitions.set(name, ctx.seen.get(schema)!);
      }
      return { type: "$ref", $ref: name };
    }
    return ctx.seen.get(schema)!;
  }

  const def = (schema.def ?? schema._zod.def) as unknown as Record<string, unknown>;
  const type = def.type as string;

  // Lazy schemas get special treatment for cycle detection.
  // Key on the lazy schema object itself, NOT the inner result
  // (getter creates a new object each call, so inner identity is unstable).
  if (type === "lazy") {
    const ref = serializeLazy(schema, def, ctx);
    if (schema._zod.optin === "optional") ref.optionalInput = true;
    if(schema._zod.optout==="optional")ref.optionalOutput=true;
    return ref;
  }

  const ir: IRSchema = { type };
  if (schema._zod.optin === "optional") ir.optionalInput = true;
  if(schema._zod.optout==="optional")ir.optionalOutput=true;
  if(schema._zod.traits.has("$ZodExactOptional"))ir.exactOptional=true;
  ctx.seen.set(schema, ir);
  ctx.active.add(schema);

  if (def.coerce === true) ir.coerce = true;
  const ownChecks=serializeChecks(def.checks);
  if(ownChecks?.length){
    const allowed=nativeChecks[type]??[];
    if(ownChecks.some(check=>!allowed.includes(check.check)))throw new Error(`Checks on ${type} require exportModuleSchema (--runtime in the CLI)`);
    ir.checks=ownChecks;
  }

  switch (type) {
    case "string":
      ir.checks = ownChecks;
      if (typeof def.format === "string") ir.format = def.format;
      if (typeof def.fn==="function" && !(def.pattern instanceof RegExp)) throw new Error("JavaScript string formats require exportModuleSchema (--runtime in the CLI)");
      if (def.pattern instanceof RegExp) ir.pattern=regexSource(def.pattern); else if(typeof def.pattern==="string") ir.pattern=def.pattern;
      if(def.includes!==undefined)ir.value=def.includes;
    if(def.position!==undefined)ir.position=def.position as number;
    if(typeof def.alg==="string")ir.alg=def.alg;
      if(def.hostname instanceof RegExp)ir.hostname=regexSource(def.hostname);
      if(def.protocol instanceof RegExp)ir.protocol=regexSource(def.protocol);
      if(def.normalize===true)ir.normalize=true;
      if (typeof def.version === "string" && /^v[1-8]$/.test(def.version)) ir.uuidVersion = Number(def.version.slice(1));
      break;

    case "number":
    case "int":
    case "boolean":
    case "bigint":
      ir.checks = serializeChecks([...(Array.isArray(def.checks) ? def.checks : []), ...(def.check ? [schema] : [])]);
      if (type === "number" && hasIntFormat(ir.checks)) {
        ir.type = "int";
      }
      break;

    case "date":
      ir.checks = ownChecks;
      break;

    case "array":
      ir.element = serializeSchema(def.element as AnySchema, ctx);
      ir.checks = ownChecks;
      break;

    case "set":
      // Represent sets as arrays; uniqueness is not enforced in the IR.
      ir.type = "set";
      ir.element = serializeSchema(def.valueType as AnySchema, ctx);
      ir.checks = ownChecks;
      break;

    case "tuple": {
      const items = def.items as AnySchema[];
      ir.items = items.map((s) => serializeSchema(s, ctx));
      if (def.rest) ir.rest = serializeSchema(def.rest as AnySchema, ctx);
      ir.checks = ownChecks;
      break;
    }

    case "record":
      if (def.keyType) {
        const values=(def.keyType as AnySchema)._zod.values;
        if(values) ir.keyValues=[...values].map(jsonValue); else ir.partialKeys=true;
      }
      if(typeof def.mode==="string") ir.mode=def.mode;
      if (def.keyType) ir.keyType = serializeSchema(def.keyType as AnySchema, ctx);
      ir.valueType = serializeSchema(def.valueType as AnySchema, ctx);
      break;

    case "map":
      // JSON maps are serialized as objects; treat like record.
      ir.type = "map";
      ir.checks=ownChecks;
      if (def.keyType) ir.keyType = serializeSchema(def.keyType as AnySchema, ctx);
      ir.valueType = serializeSchema(def.valueType as AnySchema, ctx);
      break;

    case "intersection":
      ir.left = serializeSchema(def.left as AnySchema, ctx);
      ir.right = serializeSchema(def.right as AnySchema, ctx);
      break;

    case "object": {
      const shape = def.shape as Record<string, AnySchema>;
      ir.shape = Object.create(null);
      for (const [k, v] of Object.entries(shape)) {
        ir.shape![k] = serializeSchema(v, ctx);
      }
      if (def.catchall) ir.catchall = serializeSchema(def.catchall as AnySchema, ctx);
      if (def.catchall && (def.catchall as AnySchema)._zod.def.type === "never") {
        ir.strict = true;
        delete ir.catchall;
      }
      break;
    }

    case "enum": {
      const entries = def.entries as Record<string, string | number> | [string, ...string[]];
      if (Array.isArray(entries)) {
        ir.values = [...entries];
      } else {
        const numericValues = new Set(Object.values(entries).filter((v) => typeof v === "number"));
        ir.values = Object.entries(entries)
          .filter(([key]) => !numericValues.has(Number(key)))
          .map(([, value]) => value);
      }
      break;
    }

    case "literal": {
      // Zod v4 stores literal values as an array in def.values.
      const litVals = def.values as unknown[];
      if (Array.isArray(litVals) && litVals.length === 1) {
        Object.assign(ir,serializeLiteral(litVals[0]));
      } else if (Array.isArray(litVals) && litVals.length > 1) {
        // Multi-value literal: serialize as a union of literals.
        ir.type = "union";
        ir.options = litVals.map(serializeLiteral);
      } else if (def.value !== undefined) {
        ir.value = def.value;
      }
      break;
    }

    case "union":
      if (def.inclusive === false) ir.exclusive = true;
      ir.options = (def.options as AnySchema[]).map((s) => serializeSchema(s, ctx));
      break;

    case "optional":
    case "nullable":
    case "nonoptional":
    case "readonly":
      ir.type = type === "nonoptional" ? "nonoptional" : type;
      ir.inner = serializeSchema(innerSchema(def), ctx);
      break;

    case "prefault":
    case "default": {
      ir.inner = serializeSchema(innerSchema(def), ctx);
      const value = def.defaultValue;
      if (ctx.encoding === "tagged") ir.defaultEncoded = encodeValue(value);
      else {
        if (typeof value === "function") throw new Error("Function defaults require exportModuleSchema (--runtime in the CLI)");
        ir.default = jsonValue(value);
      }
      break;
    }

    case "pipe":
      if (typeof def.transform === "function" || typeof def.reverseTransform === "function") throw new Error("JavaScript codecs require exportModuleSchema (--runtime in the CLI)");
      ir.left = serializeSchema(def.in as AnySchema, ctx);
      ir.right = serializeSchema(def.out as AnySchema, ctx);
      break;

    case "transform":
    case "custom":
      throw new Error("JavaScript callbacks require exportModuleSchema (--runtime in the CLI)");

    case "any":
    case "unknown":
    case "null":
    case "undefined":
    case "void":
    case "never":
      break;

    default:
      throw new Error(`Unsupported schema type: ${type}; use exportModuleSchema (--runtime in the CLI)`);
  }

  ctx.active.delete(schema);
  return ir;
}

function hasIntFormat(checks?: IRCheck[]): boolean {
  return checks?.some((c) => c.check === "number_format" && (c.format === "safeint" || c.format === "int32" || c.format === "uint32")) ?? false;
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
    if(typeof def.when==="function" && !builtinGuard(c,def))throw new Error("Custom check guards require exportModuleSchema (--runtime in the CLI)");
    if (def.check === "custom" || def.check === "overwrite") {
      throw new Error("JavaScript callbacks require exportModuleSchema (--runtime in the CLI)");
    }
    const ir: IRCheck = { check: def.check as string };
    if (def.value !== undefined) ir.value = jsonValue(def.value);
    if (def.length !== undefined) ir.value = def.length;
    if (def.size !== undefined) ir.value = def.size;
    if (def.minimum !== undefined) ir.value = jsonValue(def.minimum);
    if (def.maximum !== undefined) ir.value = jsonValue(def.maximum);
    if (def.inclusive !== undefined) ir.inclusive = def.inclusive as boolean;
    if (def.format !== undefined) ir.format = def.format as string;
    if (def.pattern instanceof RegExp) ir.pattern=regexSource(def.pattern);else if(def.pattern!==undefined)ir.pattern=String(def.pattern);
    if(def.includes!==undefined)ir.value=def.includes;
    if(def.position!==undefined)ir.position=def.position as number;
    if(typeof def.alg==="string")ir.alg=def.alg;
    if(def.hostname instanceof RegExp)ir.hostname=regexSource(def.hostname);
    if(def.protocol instanceof RegExp)ir.protocol=regexSource(def.protocol);
    if(def.normalize===true)ir.normalize=true;
    if (typeof def.version === "string" && /^v[1-8]$/.test(def.version)) ir.value = Number(def.version.slice(1));
    out.push(ir);
  }
  return out.length ? out : undefined;
}

export { serializeSchema };


function jsonValue(value: unknown): unknown {
  if (typeof value === "bigint") return value.toString();
  if (value instanceof Date) return value.toISOString();
  if (value instanceof Set) return [...value].map(jsonValue);
  if (value instanceof Map) {
    if ([...value.keys()].some(key => typeof key !== "string")) throw new Error("Map defaults require string keys");
    return Object.fromEntries([...value].map(([key, val]) => [key, jsonValue(val)]));
  }
  if (Array.isArray(value)) return value.map(jsonValue);
  if (value && typeof value === "object") return Object.fromEntries(Object.entries(value).map(([key, val]) => [key, jsonValue(val)]));
  return value;
}


function regexSource(pattern:RegExp):string {
 if(/[^imsu]/.test(pattern.flags))throw new Error(`Unsupported regex flags: ${pattern.flags}`);
 const flags=pattern.flags.replace(/u/g,"");
 return (flags?`(?${flags})`:"")+pattern.source;
}


function serializeLiteral(value:unknown):IRSchema{
 if(typeof value==="bigint")return {type:"literal",value:value.toString(),valueKind:"bigint"};
 if(value===undefined)return {type:"literal",valueKind:"undefined"};
 if(typeof value==="number" && !Number.isFinite(value))return {type:"literal",value:String(value),valueKind:"nonfinite"};
 return {type:"literal",value};
}


const nativeChecks:Record<string,string[]>={
 string:["min_length","max_length","length_equals","string_format"],
 number:["greater_than","less_than","multiple_of","number_format"],
 int:["greater_than","less_than","multiple_of","number_format"],
 bigint:["greater_than","less_than","multiple_of","bigint_format"],
 date:["greater_than","less_than"],
 array:["min_length","max_length","length_equals"],
 tuple:["min_length","max_length","length_equals"],
 set:["min_size","max_size","size_equals"],
 map:["min_size","max_size","size_equals"],
};


const guardSources=new WeakMap<object,string|undefined>();
function builtinGuard(check:{_zod?:{def?:Record<string,unknown>}},def:Record<string,unknown>):boolean{
 const internal=check._zod as {constr?:new(def:Record<string,unknown>)=>{_zod:{def:Record<string,unknown>}}}|undefined;
 const Constructor=internal?.constr;
 if(!Constructor || !/^\$ZodCheck(?:MinLength|MaxLength|LengthEquals|MinSize|MaxSize|SizeEquals)$/.test(Constructor.name))return false;
 if(!guardSources.has(Constructor)){
  const stock=new Constructor({...def,when:undefined});
  const when=stock._zod.def.when;
  guardSources.set(Constructor,typeof when==="function"?Function.prototype.toString.call(when):undefined);
 }
 return guardSources.get(Constructor)===Function.prototype.toString.call(def.when);
}
