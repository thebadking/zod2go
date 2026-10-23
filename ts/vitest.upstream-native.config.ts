import { defineConfig, mergeConfig } from "vitest/config";
import upstream from "./vitest.upstream.config.js";
import { resolve } from "node:path";
const helper=resolve(import.meta.dirname,"upstream-capture.ts");
export default mergeConfig(upstream,defineConfig({
 plugins:[{
  name:"capture-upstream-validation-fixtures",
  enforce:"pre",
  transform(code,id){
   if(!id.endsWith("/v4/core/parse.ts"))return;
   if ([...code.matchAll(/schema\._zod\.run\(\{ value, issues: \[\] \}, ctx\)/g)].length !== 4) throw new Error("Expected exactly four upstream parse hooks");
   const patched=code.replace(/schema\._zod\.run\(\{ value, issues: \[\] \}, ctx\)/g,"__captureRun(schema,value,ctx,()=>schema._zod.run({value,issues:[]},ctx))");
   if(patched===code)throw new Error("Upstream parse hook changed; fixture capture must be updated");
   return `import { captureRun as __captureRun } from ${JSON.stringify(helper)};\n`+patched;
  }
 }],
 test:{setupFiles:[helper],globalSetup:[resolve(import.meta.dirname,"upstream-capture-setup.ts")]},
}));
