#!/usr/bin/env node
import { writeFileSync } from "node:fs";
import { pathToFileURL } from "node:url";
import { tsImport } from "tsx/esm/api";
import { exportSchemaJSON } from "./export.js";

async function main(): Promise<void> {
  const args = process.argv.slice(2);
  let reference=false,runtime=false,tagged=false;
  while(args[0]?.startsWith("--")){
    const flag=args.shift();
    if(flag==="--runtime")runtime=true;
    else if(flag==="--runtime-reference"){runtime=true;reference=true}
    else if(flag==="--tagged")tagged=true;
    else throw new Error(`Unknown option: ${flag}`);
  }
  const [modulePath, exportName, outPath] = args;
  if (!modulePath || !exportName || !outPath) {
    console.error("Usage: zod2go-export [--runtime|--runtime-reference] [--tagged] <schemas.ts> <ExportName> <output.json>");
    process.exit(1);
  }

  if (runtime) {
    const { exportModuleSchema } = await import("./module.js");
    const doc=await exportModuleSchema(modulePath,exportName,{bundle:!reference});
    if(tagged)doc.$zod2go.encoding="tagged";
    writeFileSync(outPath,JSON.stringify(doc,null,2)+"\n","utf8");
    console.error(`Exported runtime ${exportName} -> ${outPath}`);
    return;
  }

  const mod = await tsImport(pathToFileURL(modulePath).href, import.meta.url);
  const schema = mod[exportName];
  if (!schema || typeof schema !== "object" || !("_zod" in schema && schema._zod?.def)) {
    console.error(`Export "${exportName}" not found or is not a Zod schema in ${modulePath}`);
    process.exit(1);
  }

  writeFileSync(outPath, exportSchemaJSON(schema,2,tagged?{encoding:"tagged"}:{}) + "\n", "utf8");
  console.error(`Exported ${exportName} -> ${outPath}`);
}

main().catch((err) => {
  console.error(err);
  process.exit(1);
});
