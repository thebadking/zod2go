import { execFileSync } from "node:child_process";
import { mkdtempSync, writeFileSync } from "node:fs";
import { tmpdir } from "node:os";
import { join, resolve } from "node:path";
import { describe, expect, it } from "vitest";
import { z } from "zod";
import { exportSchema, exportSchemaJSON } from "./export.js";

type Case = {
  name: string;
  schema: z.ZodTypeAny;
  input: unknown;
  success: boolean;
};

const repoRoot = resolve(import.meta.dirname, "../..");

const cases: Case[] = [
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
];

function goValidate(schemaJson: string, input: unknown): boolean {
  const dir = mkdtempSync(join(tmpdir(), "zod2go-parity-"));
  const schemaPath = join(dir, "schema.json");
  writeFileSync(schemaPath, schemaJson, "utf8");

  try {
    execFileSync(
      "/usr/local/go/bin/go",
      ["run", "./cmd/zod2go", "-schema", schemaPath],
      {
        cwd: repoRoot,
        input: JSON.stringify(input),
        encoding: "utf8",
        env: { ...process.env, GOTOOLCHAIN: "go1.23.6" },
      },
    );
    return true;
  } catch {
    return false;
  }
}

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
