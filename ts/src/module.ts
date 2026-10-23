import { build } from "esbuild";
import { resolve,dirname } from "node:path";
import {createRequire}from"node:module";
import {pathToFileURL}from"node:url";
import { z } from "zod";
import type { IRDocument } from "./export.js";

/** Bundle the actual schema and its dependency graph, preserving lexical closures.
 * Use the /module entry point in Node; the normal exporter remains browser-safe.
 */
export async function exportModuleSchema(modulePath: string, exportName: string, options: { bundle?: boolean } = {}): Promise<IRDocument> {
  if (!exportName) throw new Error("A schema export name is required");
  const version = z.core.version;
  if (options.bundle === false) {
    return {
      $zod2go: {version:1,zodVersion:`${version.major}.${version.minor}.${version.patch}`},
      schema:{type:"runtime"},
      runtime:{modulePath:resolve(modulePath),loader:import.meta.resolve("tsx/esm/api"),coreModule:resolveCore(modulePath),exportName,format:"esm"},
    };
  }
  const result = await build({
    stdin:{
      contents:`import * as user from ${JSON.stringify(resolve(modulePath))}; import * as core from "zod/v4/core"; export {user as __zod2goModule,core as __zod2goCore};`,
      resolveDir:dirname(resolve(modulePath)),sourcefile:"zod2go-module-entry.ts",loader:"ts",
    },
    bundle: true,
    platform: "node",
    target: "node20",
    format: "esm",
    banner: { js: 'import { createRequire as __zod2goCreateRequire } from "node:module"; const require = __zod2goCreateRequire(import.meta.url);' },
    write: false,
    sourcemap: "inline",
  });
  return {
    $zod2go: { version: 1, zodVersion: `${version.major}.${version.minor}.${version.patch}` },
    schema: { type: "runtime" },
    runtime: { bundle: result.outputFiles[0].text, exportName, format: "esm" },
  };
}


function resolveCore(modulePath:string):string{
 const require=createRequire(pathToFileURL(resolve(modulePath)));
 const resolved=require.resolve("zod/v4/core");
 return pathToFileURL(resolved.endsWith(".cjs")?resolved.slice(0,-4)+".js":resolved).href;
}
