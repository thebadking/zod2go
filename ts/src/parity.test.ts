import { execFileSync } from "node:child_process";
import { existsSync, mkdtempSync, writeFileSync, rmSync } from "node:fs";
import { tmpdir } from "node:os";
import { join, resolve } from "node:path";
import { afterAll, beforeAll, describe, expect, it } from "vitest";
import { z } from "zod";
import { exportSchema, exportSchemaJSON } from "./export.js";

type Case = {
  name: string;
  schema: z.ZodTypeAny;
  input: unknown;
  success: boolean;
};

const repoRoot = resolve(import.meta.dirname, "../..");

// ─── Parity cases ─────────────────────────────────────────────────────────────

const cases: Case[] = [
  { name: "literal numeric string rejected", schema: z.literal(42), input: "42", success: false },
  { name: "literal boolean string rejected", schema: z.literal(true), input: "true", success: false },
  { name: "multi literal valid", schema: z.literal(["a", 2, true]), input: 2, success: true },
  { name: "multi literal invalid", schema: z.literal(["a", 2, true]), input: "2", success: false },
  { name: "optional rejects null", schema: z.string().optional(), input: null, success: false },
  { name: "default rejects null", schema: z.string().default("x"), input: null, success: false },
  { name: "default null preserved", schema: z.object({ x: z.null().default(null) }), input: {}, success: true },
  { name: "optional default preserved", schema: z.object({ x: z.string().default("x").optional() }), input: {}, success: true },
  { name: "unknown field required", schema: z.object({ x: z.unknown() }), input: {}, success: false },
  { name: "union undefined field required", schema: z.object({ x: z.union([z.string(), z.undefined()]) }), input: {}, success: false },
  { name: "nullable field required", schema: z.object({ x: z.string().nullable() }), input: {}, success: false },
  { name: "tuple extra rejected", schema: z.tuple([z.string()]), input: ["x", 2], success: false },
  { name: "tuple optional trailing", schema: z.tuple([z.string(), z.number().optional()]), input: ["x"], success: true },
  { name: "empty tuple rejects extra", schema: z.tuple([]), input: [1], success: false },
  { name: "regex accepts", schema: z.string().regex(/^a+$/), input: "aaa", success: true },
  { name: "regex rejects", schema: z.string().regex(/^a+$/), input: "bbb", success: false },
  { name: "UTF16 min", schema: z.string().min(2), input: "😀", success: true },
  { name: "UTF16 max", schema: z.string().max(1), input: "😀", success: false },
  { name: "safe integer overflow", schema: z.int(), input: 9007199254740992, success: false },
  { name: "boolean false string truthy", schema: z.coerce.boolean(), input: "false", success: true },
  { name: "boolean object truthy", schema: z.coerce.boolean(), input: {}, success: true },
  { name: "boolean null falsy", schema: z.coerce.boolean(), input: null, success: true },
  { name: "boolean empty string falsy", schema: z.coerce.boolean(), input: "", success: true },
  { name: "numeric enum reverse name rejected", schema: z.enum({ A: 0, B: 1, 0: "A", 1: "B" }), input: "A", success: false },
  { name: "number coerce null", schema: z.coerce.number(), input: null, success: true },
  { name: "number coerce boolean", schema: z.coerce.number(), input: true, success: true },
  { name: "number coerce whitespace", schema: z.coerce.number(), input: "  ", success: true },
  { name: "number coerce hex", schema: z.coerce.number(), input: "0xff", success: true },
  { name: "number coerce empty array", schema: z.coerce.number(), input: [], success: true },
  { name: "string coerce null", schema: z.coerce.string(), input: null, success: true },
  { name: "string coerce object", schema: z.coerce.string(), input: {}, success: true },
  { name: "string coerce array", schema: z.coerce.string(), input: [1, null, 2], success: true },
  { name: "decimal multiple valid", schema: z.number().multipleOf(0.1), input: 0.3, success: true },
  { name: "decimal multiple invalid", schema: z.number().multipleOf(0.1), input: 0.31, success: false },
  { name: "record enum exhaustive", schema: z.record(z.enum(["a", "b"]), z.number()), input: {a: 1}, success: false },
  { name: "record enum complete", schema: z.record(z.enum(["a", "b"]), z.number()), input: {a: 1, b: 2}, success: true },
  { name: "readonly object", schema: z.object({ a: z.string() }).readonly(), input: {a: "x"}, success: true },
  { name: "nonoptional rejects missing", schema: z.object({ a: z.string().optional().nonoptional() }), input: {}, success: false },
  { name: "prefault valid", schema: z.object({ a: z.string().min(2).prefault("yes") }), input: {}, success: true },
  { name: "prefault validates default", schema: z.object({ a: z.string().min(2).prefault("x") }), input: {}, success: false },
  { name: "date bound native", schema: z.date().min(new Date("2024-01-01Z")), input: new Date("2025-01-01Z"), success: true },
  { name: "set native", schema: z.set(z.string()).min(2), input: new Set(["a", "b"]), success: true },
  { name: "set native rejects element", schema: z.set(z.number()), input: new Set(["a"]), success: false },
  { name: "map native", schema: z.map(z.string(), z.number()), input: new Map([["a",1]]), success: true },
  { name: "intersection nested output merged", schema: z.intersection(z.object({x: z.object({a:z.string()})}), z.object({x:z.object({b:z.number()})})), input: {x:{a:"yes",b:1}}, success: true },
  { name: "regex flags", schema: z.string().regex(/^a$/i), input: "A", success: true },
  { name: "tiny nonmultiple", schema: z.number().multipleOf(0.1), input: 0.000000000001, success: false },
  { name: "lazy optional field absent", schema: z.object({ a:z.lazy(() => z.string().optional()) }), input: {}, success: true },
  { name: "recursive object root", schema: (() => { const root: z.ZodTypeAny = z.object({children:z.array(z.lazy(() => root))}); return root; })(), input: {children:[{children:[]}]}, success: true },
  { name: "mutual recursion", schema: (() => { const a: z.ZodTypeAny = z.lazy(() => z.object({b:b.optional()})); const b: z.ZodTypeAny = z.lazy(() => z.object({a:a.optional()})); return a; })(), input: {b:{a:{}}}, success: true },
  { name: "uuid v4 rejects v1", schema: z.uuidv4(), input: "550e8400-e29b-11d4-a716-446655440000", success: false },
  { name: "uuid method v4 rejects v1", schema: z.string().uuid({ version: "v4" }), input: "550e8400-e29b-11d4-a716-446655440000", success: false },
  { name: "set default wire form", schema: z.object({s:z.set(z.string()).default(new Set(["a"]))}), input: {}, success: true },
  { name: "map default wire form", schema: z.object({m:z.map(z.string(),z.number()).default(new Map([["a",1]]))}), input: {}, success: true },
  // ── Original 7 cases ──────────────────────────────────────────────────────

  {
    name: "valid user",
    schema: z.object({
      name: z.string().min(2),
      email: z.email(),
      role: z.enum(["admin", "user"]).default("user"),
    }),
    input: { name: "Ada", email: "ada@example.com" },
    success: true,
  },
  {
    name: "invalid email",
    schema: z.object({ name: z.string(), email: z.email() }),
    input: { name: "Ada", email: "nope" },
    success: false,
  },
  {
    name: "tuple type mismatch",
    schema: z.tuple([z.string(), z.number()]),
    input: ["x", "y"],
    success: false,
  },
  {
    name: "record valid",
    schema: z.record(z.string(), z.number()),
    input: { a: 1, b: 2 },
    success: true,
  },
  {
    name: "intersection valid",
    schema: z.intersection(z.object({ a: z.string() }), z.object({ b: z.number() })),
    input: { a: "x", b: 1 },
    success: true,
  },
  {
    name: "array min length",
    schema: z.array(z.string()).min(1),
    input: [],
    success: false,
  },
  {
    name: "strict object rejects extra keys",
    schema: z.object({ id: z.string() }).strict(),
    input: { id: "x", extra: true },
    success: false,
  },

  // ── Strings ──────────────────────────────────────────────────────────────

  {
    name: "string min too short",
    schema: z.string().min(5),
    input: "hi",
    success: false,
  },
  {
    name: "string max too long",
    schema: z.string().max(3),
    input: "toolong",
    success: false,
  },
  {
    name: "string min+max valid",
    schema: z.string().min(2).max(10),
    input: "hello",
    success: true,
  },
  {
    name: "string url valid",
    schema: z.string().url(),
    input: "https://example.com",
    success: true,
  },
  {
    name: "string url invalid",
    schema: z.string().url(),
    input: "not-a-url",
    success: false,
  },

  // ── Numbers ───────────────────────────────────────────────────────────────

  {
    name: "number min valid",
    schema: z.number().min(0),
    input: 5,
    success: true,
  },
  {
    name: "number min invalid",
    schema: z.number().min(10),
    input: 5,
    success: false,
  },
  {
    name: "number max invalid",
    schema: z.number().max(100),
    input: 150,
    success: false,
  },
  {
    name: "int rejects float",
    schema: z.number().int(),
    input: 3.14,
    success: false,
  },
  {
    name: "int accepts integer",
    schema: z.number().int(),
    input: 42,
    success: true,
  },

  // ── Coercion ─────────────────────────────────────────────────────────────

  {
    name: "coerce number from string",
    schema: z.coerce.number(),
    input: "42",
    success: true,
  },
  {
    name: "coerce boolean from string true",
    schema: z.coerce.boolean(),
    input: "true",
    success: true,
  },

  // ── Defaults and optionals ────────────────────────────────────────────────

  {
    name: "default fills missing field",
    schema: z.object({ x: z.string().default("hi") }),
    input: {},
    success: true,
  },
  {
    name: "optional field absent",
    schema: z.object({ name: z.string(), bio: z.string().optional() }),
    input: { name: "Ada" },
    success: true,
  },

  // ── Nullable ─────────────────────────────────────────────────────────────

  {
    name: "nullable accepts null",
    schema: z.string().nullable(),
    input: null,
    success: true,
  },
  {
    name: "nullable accepts string",
    schema: z.string().nullable(),
    input: "hello",
    success: true,
  },

  // ── Literal ──────────────────────────────────────────────────────────────

  {
    name: "literal match",
    schema: z.literal(42),
    input: 42,
    success: true,
  },
  {
    name: "literal mismatch",
    schema: z.literal(42),
    input: 43,
    success: false,
  },
  {
    name: "string literal match",
    schema: z.literal("ok"),
    input: "ok",
    success: true,
  },

  // ── Boolean ───────────────────────────────────────────────────────────────

  {
    name: "boolean valid",
    schema: z.boolean(),
    input: false,
    success: true,
  },
  {
    name: "boolean rejects string",
    schema: z.boolean(),
    input: "true",
    success: false,
  },

  // ── Nested objects ────────────────────────────────────────────────────────

  {
    name: "nested object valid",
    schema: z.object({ user: z.object({ name: z.string(), age: z.number() }) }),
    input: { user: { name: "Ada", age: 30 } },
    success: true,
  },
  {
    name: "nested object invalid inner",
    schema: z.object({ user: z.object({ name: z.string() }) }),
    input: { user: { name: 42 } },
    success: false,
  },

  // ── Array of objects ──────────────────────────────────────────────────────

  {
    name: "array of objects valid",
    schema: z.array(z.object({ id: z.number() })),
    input: [{ id: 1 }, { id: 2 }],
    success: true,
  },
  {
    name: "array of objects invalid element",
    schema: z.array(z.object({ id: z.number() })),
    input: [{ id: 1 }, { id: "nope" }],
    success: false,
  },

  // ── Union ─────────────────────────────────────────────────────────────────

  {
    name: "union matches first branch",
    schema: z.union([z.string(), z.number()]),
    input: "hello",
    success: true,
  },
  {
    name: "union matches second branch",
    schema: z.union([z.string(), z.number()]),
    input: 42,
    success: true,
  },
  {
    name: "union matches no branch",
    schema: z.union([z.string(), z.number()]),
    input: true,
    success: false,
  },

  // ── Enum ──────────────────────────────────────────────────────────────────

  {
    name: "enum valid",
    schema: z.enum(["a", "b", "c"]),
    input: "b",
    success: true,
  },
  {
    name: "enum invalid",
    schema: z.enum(["a", "b", "c"]),
    input: "d",
    success: false,
  },

  // ── Date ─────────────────────────────────────────────────────────────────
  // Native Date/Set/Map inputs above use their documented JSON wire forms.

  // ── Refinements (pass-through) ────────────────────────────────────────────

  {
    name: "refine base schema still validated",
    schema: z.string().min(2).refine((s) => s.startsWith("A")),
    // min(2) check applies; the refine predicate is skipped in Go (pass-through)
    input: "x",
    // Zod: fails min(2); Go: also fails min(2)
    success: false,
  },

  // ── Tuple with rest ───────────────────────────────────────────────────────

  {
    name: "tuple with rest valid",
    schema: z.tuple([z.string()]).rest(z.number()),
    input: ["x", 1, 2, 3],
    success: true,
  },
  {
    name: "tuple with rest invalid rest element",
    schema: z.tuple([z.string()]).rest(z.number()),
    input: ["x", 1, "bad"],
    success: false,
  },
];

// ─── Go subprocess harness ────────────────────────────────────────────────────

let goBin: string;
const tempDirs: string[] = [];
afterAll(() => { for (const dir of tempDirs) rmSync(dir, { recursive: true, force: true }); });

beforeAll(() => {
  const binDir = mkdtempSync(join(tmpdir(), "zod2go-bin-"));
  tempDirs.push(binDir);
  goBin = join(binDir, "zod2go");
  execFileSync(
    process.env.GO_BINARY ?? (existsSync("/usr/local/go/bin/go") ? "/usr/local/go/bin/go" : "go"),
    ["build", "-o", goBin, "./cmd/zod2go"],
    {
      cwd: repoRoot,
      encoding: "utf8",
      env: { ...process.env, GOTOOLCHAIN: "go1.23.6" },
    },
  );
}, 60_000);

function goValidate(schemaJson: string, input: unknown): { success: boolean; data?: unknown; issues?: unknown } {
  const dir = mkdtempSync(join(tmpdir(), "zod2go-parity-"));
  tempDirs.push(dir);
  const schemaPath = join(dir, "schema.json");
  writeFileSync(schemaPath, schemaJson, "utf8");

  try {
    const stdout = execFileSync(goBin, ["-schema", schemaPath], {
      input: JSON.stringify(jsonValue(input)),
      encoding: "utf8",
    });
    return { success: true, data: JSON.parse(stdout) };
  } catch (error) {
    const failure = error as { status?: number; stdout?: string; stderr?: string };
    if (failure.status !== 1 || !failure.stdout) throw error;
    const result = JSON.parse(failure.stdout);
    if (!Array.isArray(result.issues)) throw error;
    return { success: false, issues: result.issues };
  }
}

// ─── Special parity cases where Zod and Go diverge intentionally ─────────────
// Refinement predicates are skipped in Go — only the base schema is validated.
// For these we only assert Go's outcome, not Zod == Go.
const goOnlyCases: { name: string; schema: z.ZodTypeAny; input: unknown; goSuccess: boolean }[] = [
  { name: "date minimum rejects earlier", schema: z.date().min(new Date("2024-01-01Z")), input: "2023-01-01T00:00:00Z", goSuccess: false },
  { name: "date minimum equality", schema: z.date().min(new Date("2024-01-01Z")), input: "2024-01-01T00:00:00Z", goSuccess: true },
  { name: "date maximum rejects later", schema: z.date().max(new Date("2024-01-01Z")), input: "2025-01-01T00:00:00Z", goSuccess: false },
  { name: "set min rejected", schema: z.set(z.string()).min(2), input: ["a"], goSuccess: false },
  { name: "set max rejected", schema: z.set(z.string()).max(1), input: ["a", "b"], goSuccess: false },
  { name: "map key rejected", schema: z.map(z.string().min(2), z.number()), input: { x: 1 }, goSuccess: false },
  { name: "map value rejected", schema: z.map(z.string(), z.number()), input: { x: "bad" }, goSuccess: false },
  { name: "lazy alias chain", schema: z.lazy(() => z.lazy(() => z.number())), input: 42, goSuccess: true },
  { name: "transform shared schema preserves input", schema: (() => { const s = z.string().transform(s => s.length); return z.object({ a: s, b: s }); })(), input: { a: "x", b: "y" }, goSuccess: true },
  {
    name: "refine predicate skipped in Go — valid base",
    schema: z.string().min(1).refine((s) => s === "magic"),
    input: "anything",
    goSuccess: true, // Go only validates string + min(1), skips the predicate
  },
  {
    name: "date valid ISO string in Go",
    schema: z.date(),
    input: "2024-01-15T10:30:00.000Z",
    goSuccess: true,
  },
  {
    name: "date invalid string in Go",
    schema: z.date(),
    input: "not-a-date",
    goSuccess: false,
  },
  {
    name: "lazy recursive schema — single node",
    schema: (() => {
      type Tree = { val: number; children: Tree[] };
      const TreeSchema: z.ZodType<Tree> = z.lazy(() =>
        z.object({ val: z.number(), children: z.array(TreeSchema) })
      );
      return TreeSchema;
    })(),
    input: { val: 1, children: [] },
    goSuccess: true,
  },
  {
    name: "lazy recursive schema — nested",
    schema: (() => {
      type Tree = { val: number; children: Tree[] };
      const TreeSchema: z.ZodType<Tree> = z.lazy(() =>
        z.object({ val: z.number(), children: z.array(TreeSchema) })
      );
      return TreeSchema;
    })(),
    input: { val: 1, children: [{ val: 2, children: [] }] },
    goSuccess: true,
  },
  {
    name: "lazy recursive schema — invalid inner",
    schema: (() => {
      type Tree = { val: number; children: Tree[] };
      const TreeSchema: z.ZodType<Tree> = z.lazy(() =>
        z.object({ val: z.number(), children: z.array(TreeSchema) })
      );
      return TreeSchema;
    })(),
    input: { val: "not-a-number", children: [] },
    goSuccess: false,
  },
];

// ─── Test runner ──────────────────────────────────────────────────────────────

describe("Zod parity (export + safeParse agreement)", () => {
  for (const c of cases) {
    it(`${c.name} — export shape`, () => {
      const doc = exportSchema(c.schema);
      expect(doc.$zod2go.version).toBe(1);
      expect(doc.schema.type).toBeTruthy();
    });

    it(`${c.name} — Zod vs Go`, () => {
      const zodResult = c.schema.safeParse(c.input);
      expect(zodResult.success).toBe(c.success);

      const goSuccess = goValidate(exportSchemaJSON(c.schema), c.input);
      expect(goSuccess.success).toBe(c.success);
      if (zodResult.success) expect(goSuccess.data).toEqual(jsonValue(zodResult.data));
    });
  }
});

describe("Go-only validation cases", () => {
  for (const c of goOnlyCases) {
    it(`${c.name}`, () => {
      const goSuccess = goValidate(exportSchemaJSON(c.schema), c.input);
      expect(goSuccess.success).toBe(c.goSuccess);
    });
  }
});


function jsonValue(value: unknown): unknown {
  if (value instanceof Date) return value.toISOString();
  if (value instanceof Set) return [...value].map(jsonValue);
  if (value instanceof Map) return Object.fromEntries([...value].map(([key, val]) => [key, jsonValue(val)]));
  if (Array.isArray(value)) return value.map(jsonValue);
  if (value && typeof value === "object") return Object.fromEntries(Object.entries(value).map(([key, val]) => [key, jsonValue(val)]));
  return value;
}
