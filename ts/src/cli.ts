#!/usr/bin/env node
import { writeFileSync } from "node:fs";
import { pathToFileURL } from "node:url";
import { tsImport } from "tsx/esm/api";
import { exportSchemaJSON } from "./export.js";

async function main(): Promise<void> {
  const [, , modulePath, exportName, outPath] = process.argv;
  if (!modulePath || !exportName || !outPath) {
    console.error("Usage: zod2go-export <schemas.ts> <ExportName> <output.json>");
    process.exit(1);
  }

  const mod = await tsImport(pathToFileURL(modulePath).href, import.meta.url);
  const schema = mod[exportName];
  if (!schema || typeof schema !== "object" || !("def" in schema)) {
    console.error(`Export "${exportName}" not found or is not a Zod schema in ${modulePath}`);
    process.exit(1);
  }

  writeFileSync(outPath, exportSchemaJSON(schema) + "\n", "utf8");
  console.error(`Exported ${exportName} -> ${outPath}`);
}

main().catch((err) => {
  console.error(err);
  process.exit(1);
});
