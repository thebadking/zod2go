import { execFileSync } from "node:child_process";
import { mkdtempSync, writeFileSync } from "node:fs";
import { tmpdir } from "node:os";
import { join, resolve } from "node:path";
import { beforeAll, describe, expect, it } from "vitest";
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
  // Note: Zod z.date() parses JS Date objects; for JSON parity we test the
  // Go side only via the schema export (not Zod safeParse on a raw string).

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

beforeAll(() => {
  const binDir = mkdtempSync(join(tmpdir(), "zod2go-bin-"));
  goBin = join(binDir, "zod2go");
  execFileSync(
    "/usr/local/go/bin/go",
    ["build", "-o", goBin, "./cmd/zod2go"],
    {
      cwd: repoRoot,
      encoding: "utf8",
      env: { ...process.env, GOTOOLCHAIN: "go1.23.6" },
    },
  );
}, 60_000);

function goValidate(schemaJson: string, input: unknown): boolean {
  const dir = mkdtempSync(join(tmpdir(), "zod2go-parity-"));
  const schemaPath = join(dir, "schema.json");
  writeFileSync(schemaPath, schemaJson, "utf8");

  try {
    execFileSync(goBin, ["-schema", schemaPath], {
      input: JSON.stringify(input),
      encoding: "utf8",
    });
    return true;
  } catch {
    return false;
  }
}

// ─── Special parity cases where Zod and Go diverge intentionally ─────────────
// Refinement predicates are skipped in Go — only the base schema is validated.
// For these we only assert Go's outcome, not Zod == Go.
const goOnlyCases: { name: string; schema: z.ZodTypeAny; input: unknown; goSuccess: boolean }[] = [
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
      expect(goSuccess).toBe(c.success);
    });
  }
});

describe("Go-only validation cases", () => {
  for (const c of goOnlyCases) {
    it(`${c.name}`, () => {
      const goSuccess = goValidate(exportSchemaJSON(c.schema), c.input);
      expect(goSuccess).toBe(c.goSuccess);
    });
  }
});
