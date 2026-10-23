import { readFileSync } from "node:fs";
import { z } from "zod";

// Reference mode preserves import.meta.url and the application's file layout.
const data = JSON.parse(readFileSync(new URL("./runtime-data.json", import.meta.url), "utf8"));
export const SourceContextSchema = z.string().transform(value => `${data.prefix}:${value}`);
