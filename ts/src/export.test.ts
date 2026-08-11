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
});
