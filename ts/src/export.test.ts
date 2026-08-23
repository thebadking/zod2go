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
});
