import{defineConfig,mergeConfig}from"vitest/config";
import{resolve}from"node:path";
import upstream from"./vitest.upstream.config.js";
const helper=resolve(import.meta.dirname,"upstream-runtime.ts");
export default mergeConfig(upstream,defineConfig({
 plugins:[{name:"upstream-through-production-go-runtime",enforce:"pre",transform(code,id){
  if(!id.endsWith("/v4/core/parse.ts"))return;
  if ([...code.matchAll(/schema\._zod\.run\(\{ value, issues: \[\] \}, ctx\)/g)].length !== 4) throw new Error("Expected exactly four upstream parse hooks");
  const patched=code.replace(/schema\._zod\.run\(\{ value, issues: \[\] \}, ctx\)/g,"__runThroughGo(schema,value,ctx,()=>schema._zod.run({value,issues:[]},ctx))");
  if(patched===code)throw new Error("Upstream parse hook changed");
  return `import {runThroughGo as __runThroughGo} from ${JSON.stringify(helper)};\n`+patched;
 }}],
 test:{setupFiles:[helper],globalSetup:[resolve(import.meta.dirname,"upstream-runtime-setup.ts")]},
}));
