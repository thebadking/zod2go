import { z } from "zod";

const prefix = "captured";
let count = 0;
export const ClosureSchema = z.string().refine(s => s.startsWith("A"), "starts with A")
  .transform(value => `${prefix}:${value}:${++count}`);
export const AsyncSchema = z.string().transform(async value => value.toUpperCase());
export const CodecSchema = z.codec(z.string(), z.number(), { decode: Number, encode: String });
export const BigIntSchema = z.bigint().min(9007199254740993n);
export const NumericMapSchema = z.map(z.number(), z.string());

const topLevelPrefix = await Promise.resolve("top-level");
export const TopLevelAwaitSchema = z.string().transform(value => `${topLevelPrefix}:${value}`);

export const FunctionSchema = z.string().transform(prefix => (suffix: string) => `${prefix}:${suffix}:${++count}`);

import * as core from "zod/v4/core";
import * as mini from "zod/mini";
export const CoreStringSchema = new core.$ZodString({type:"string"});
export const MiniStringSchema = mini.string();
