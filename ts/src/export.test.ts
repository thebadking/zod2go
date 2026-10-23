import { describe, expect, it } from "vitest";
import { z } from "zod";
import { exportSchema } from "./export.js";

describe("exportSchema", () => {
  it("exports a user object schema", () => {
    const UserSchema = z.object({
      name: z.string().min(2),
      email: z.email(),
      age: z.number().int().min(0).optional(),
      role: z.enum(["admin", "user"]).default("user"),
    });

    const doc = exportSchema(UserSchema);
    expect(doc.$zod2go.version).toBe(1);
    expect(doc.schema.type).toBe("object");
    expect(doc.schema.shape?.name.type).toBe("string");
    expect(doc.schema.shape?.email.format ?? doc.schema.shape?.email.checks?.[0]?.format).toBeTruthy();
  });

  it("exports unions and arrays", () => {
    const S = z.array(z.union([z.string(), z.number()]));
    const doc = exportSchema(S);
    expect(doc.schema.type).toBe("array");
    expect(doc.schema.element?.type).toBe("union");
    expect(doc.schema.element?.options).toHaveLength(2);
  });

  it("exports tuple, record, and intersection", () => {
    const S = z.object({
      pair: z.tuple([z.string(), z.number()]),
      scores: z.record(z.string(), z.number()),
      merged: z.intersection(z.object({ a: z.string() }), z.object({ b: z.number() })),
    });

    const doc = exportSchema(S);
    expect(doc.schema.shape?.pair.type).toBe("tuple");
    expect(doc.schema.shape?.pair.items).toHaveLength(2);
    expect(doc.schema.shape?.scores.type).toBe("record");
    expect(doc.schema.shape?.scores.valueType?.type).toBe("number");
    expect(doc.schema.shape?.merged.type).toBe("intersection");
    expect(doc.schema.shape?.merged.left?.shape?.a.type).toBe("string");
    expect(doc.schema.shape?.merged.right?.shape?.b.type).toBe("number");
  });

  it("exports array min_length checks from Zod v4", () => {
    const S = z.array(z.string()).min(1);
    const doc = exportSchema(S);
    expect(doc.schema.checks?.[0]?.check).toBe("min_length");
    expect(doc.schema.checks?.[0]?.value).toBe(1);
  });

  it("exports date type", () => {
    const S = z.date();
    const doc = exportSchema(S);
    expect(doc.schema.type).toBe("date");
  });

  it("exports set as set type with element", () => {
    const S = z.set(z.string());
    const doc = exportSchema(S);
    expect(doc.schema.type).toBe("set");
    expect(doc.schema.element?.type).toBe("string");
  });

  it("exports map as map type with keyType and valueType", () => {
    const S = z.map(z.string(), z.number());
    const doc = exportSchema(S);
    expect(doc.schema.type).toBe("map");
    expect(doc.schema.keyType?.type).toBe("string");
    expect(doc.schema.valueType?.type).toBe("number");
  });

  it("exports lazy recursive schema with $ref and definitions", () => {
    type Tree = { val: number; children: Tree[] };
    const TreeSchema: z.ZodType<Tree> = z.lazy(() =>
      z.object({ val: z.number(), children: z.array(TreeSchema) })
    );
    const doc = exportSchema(TreeSchema);
    // Top-level schema is a $ref
    expect(doc.schema.type).toBe("$ref");
    expect(doc.schema["$ref"]).toBeTruthy();
    // definitions contains the actual object schema
    expect(doc.definitions).toBeDefined();
    const defName = doc.schema["$ref"]!;
    expect(doc.definitions![defName].type).toBe("object");
    // children element is a $ref back to the same def
    const childrenEl = doc.definitions![defName].shape?.children.element;
    expect(childrenEl?.type).toBe("$ref");
    expect(childrenEl?.["$ref"]).toBe(defName);
  });

  it("exports refinements as pass-through (skips custom checks)", () => {
    const S = z.string().min(3).refine((s) => s.includes("@"), "must contain @");
    const doc = exportSchema(S);
    // Should be a string with min_length check but no custom check
    expect(doc.schema.type).toBe("string");
    const checks = doc.schema.checks ?? [];
    expect(checks.some((c) => c.check === "custom")).toBe(false);
    expect(checks.some((c) => c.check === "min_length")).toBe(true);
  });

  it("exports transforms as input schema (pipe pass-through)", () => {
    const S = z.string().transform((s) => s.toUpperCase());
    const doc = exportSchema(S);
    expect(doc.schema.type).toBe("string");
  });

  it("exports coerced number", () => {
    const S = z.coerce.number();
    const doc = exportSchema(S);
    expect(doc.schema.type).toBe("number");
    expect(doc.schema.coerce).toBe(true);
  });
});


describe("recursive serialization regressions", () => {
  it("serializes an object root referenced by a lazy child", () => {
    const root: z.ZodTypeAny = z.object({ children: z.array(z.lazy(() => root)) });
    const doc = JSON.parse(JSON.stringify(exportSchema(root)));
    expect(doc.schema.type).toBe("object");
    expect(Object.keys(doc.definitions).length).toBeGreaterThan(0);
  });

  it("preserves prototype-named fields", () => {
    const schema = z.object(Object.fromEntries([["__proto__", z.string()]]));
    const doc = JSON.parse(JSON.stringify(exportSchema(schema)));
    expect(doc.schema.shape.__proto__.type).toBe("string");
  });

  it("reads the installed Zod version", () => {
    expect(exportSchema(z.string()).$zod2go.zodVersion).toBe("4.4.3");
  });
});


describe("export boundaries", () => {
  it("resolves a lazy getter once", () => {
    let calls = 0;
    const lazy = z.lazy(() => { calls++; return z.string().optional(); });
    const doc = exportSchema(z.object({ a: lazy, b: lazy }));
    expect(calls).toBe(1);
    expect(doc.schema.shape?.a.optionalInput).toBe(true);
  });
  it("rejects unsupported types explicitly", () => {
    expect(() => exportSchema(z.bigint())).toThrow("Unsupported schema type");
    expect(() => exportSchema(z.string().catch("fallback"))).toThrow("Unsupported schema type");
  });
  it("preserves numeric string enum keys without reverse mappings", () => {
    expect(exportSchema(z.enum({ "0": "zero" })).schema.values).toEqual(["zero"]);
  });
});
