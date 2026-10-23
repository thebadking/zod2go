import { execFileSync } from "node:child_process";
import { mkdirSync, writeFileSync } from "node:fs";
import { createHash } from "node:crypto";
import { join } from "node:path";
import { isDeepStrictEqual } from "node:util";
import { afterAll, expect } from "vitest";
import {encodeValue,decodeValue,type TaggedValue} from "./src/wire.js";
import { exportSchema } from "./src/export.js";

type Fixture = { document: unknown; input: unknown; expected: { success: boolean; data?: unknown }; tests: Set<string>; calls: number };
const fixtures = new Map<string,Fixture>();
const unavailable = new Map<string,{calls:number;tests:Set<string>}>();
const schemaCache = new WeakMap<object,unknown>();
function unavailableCase(reason:string){
 const entry=unavailable.get(reason)??{calls:0,tests:new Set<string>()};entry.calls++;entry.tests.add(expect.getState().currentTestName??"module setup");unavailable.set(reason,entry);
}

function portableSchema(schema:any,seen=new Set<object>()):boolean {
  if (!schema || typeof schema!=="object") return false;
  if (seen.has(schema)) return false;
  seen.add(schema);
  const def=schema._zod?.def;
  if (!def) return false;
  if (typeof def.transform === "function" || typeof def.reverseTransform === "function") return false;
  if (["lazy","default","prefault","catch","transform","custom","promise","function","symbol","file","success","template_literal","nan"].includes(def.type)) return false;
  if (def.checks?.some((check:any)=>["custom","overwrite"].includes(check._zod?.def?.check))) return false;
  const children:any[]=[];
  if (def.shape) children.push(...Object.values(def.shape));
  for(const key of ["innerType","element","valueType","keyType","left","right","in","out","catchall","rest"]) if(def[key]) children.push(def[key]);
  for(const key of ["items","options"]) if(Array.isArray(def[key])) children.push(...def[key]);
  for(const child of children) if(!portableSchema(child,new Set(seen))) return false;
  return true;
}

function capture(schema:any,input:any,result:any,context:any){
  if(context?.direction==="backward"){unavailableCase("encode uses module runtime");return}
  try{
    if(!portableSchema(schema)){unavailableCase("schema requires module runtime");return}
    let document=schemaCache.get(schema);
    if(!document){document=exportSchema(schema,{encoding:"tagged"});schemaCache.set(schema,document)}
    const expected=result.issues.length?{success:false}:{success:true,data:encodeValue(result.value)};
    const value=encodeValue(input);
    const key=JSON.stringify([document,value,expected]);
    const name=expect.getState().currentTestName??"module setup";
    const existing=fixtures.get(key);
    if(existing){existing.tests.add(name);existing.calls++;return}
    fixtures.set(key,{document,input:value,expected,tests:new Set([name]),calls:1});
  }catch(error){unavailableCase(`module runtime transport required: ${(error as Error).message}`)}
}
export function captureRun(schema:any,input:any,context:any,run:()=>any):any {
 let result;
 try{result=run()}catch(error){unavailableCase("original validation exception uses module runtime");throw error}
  if(result instanceof Promise)result.then(value=>capture(schema,input,value,context),()=>unavailableCase("async exception uses module runtime"));
  else capture(schema,input,result,context);
  return result;
}

afterAll(()=>{
  const captured=[...fixtures.values()];
  const requests=captured.map(({document,input})=>({document,input}));
  const actual=requests.length?JSON.parse(execFileSync(process.env.ZOD2GO_FIXTURE_BINARY!,[],{input:JSON.stringify(requests),encoding:"utf8",maxBuffer:256*1024*1024})):[];
  const mismatches=captured.flatMap((fixture,index)=>{
    const result=actual[index];
    if(result.success===fixture.expected.success && (!result.success || isDeepStrictEqual(decodeValue(result.data),decodeValue(fixture.expected.data as TaggedValue))))return [];
    return [{...fixture,tests:[...fixture.tests],actual:result}];
  });
  const file=expect.getState().testPath??"unknown";
  const report={runId:process.env.ZOD2GO_FIXTURE_RUN_ID,file,distinctFixtures:captured.length,calls:captured.reduce((n,f)=>n+f.calls,0),unported:Object.fromEntries([...unavailable].map(([reason,value])=>[reason,{calls:value.calls,tests:[...value.tests]}])),mismatches};
  mkdirSync(process.env.ZOD2GO_FIXTURE_REPORT_DIR!,{recursive:true});
  writeFileSync(join(process.env.ZOD2GO_FIXTURE_REPORT_DIR!,createHash("sha256").update(file).digest("hex")+".json"),JSON.stringify(report,null,2));
  if(mismatches.length)throw new Error(`${mismatches.length} native Go parity mismatches in ${file}; see ${process.env.ZOD2GO_FIXTURE_REPORT_DIR}`);
});
