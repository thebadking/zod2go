import { spawn, type ChildProcessWithoutNullStreams } from "node:child_process";
import { readSync, writeSync, mkdirSync, writeFileSync } from "node:fs";
import { performance } from "node:perf_hooks";
import { createHash } from "node:crypto";
import { join } from "node:path";
import { afterAll, expect } from "vitest";

let child:ChildProcessWithoutNullStreams|undefined;
let inputFD:number,outputFD:number,buffer=Buffer.alloc(0);
const sleeper=new Int32Array(new SharedArrayBuffer(4));
const now=performance.now.bind(performance);
let depth=0,calls=0,nested=0,exceptions=0;
const cases=new Map<string,number>();
function start(){
 child=spawn(process.env.ZOD2GO_RUNTIME_FIXTURE_BINARY!,[],{stdio:["pipe","pipe","pipe"]});
 child.stderr.on("data",data=>process.stderr.write(data));
 inputFD=(child.stdin as any)._handle.fd;outputFD=(child.stdout as any)._handle.fd;
 (child.stdout as any)._handle.readStop();
}
function write(value:unknown){
 const data=Buffer.from(JSON.stringify(value)+"\n");let offset=0;
 while(offset<data.length){try{offset+=writeSync(inputFD,data,offset,data.length-offset)}catch(error){if((error as NodeJS.ErrnoException).code!=="EAGAIN")throw error;Atomics.wait(sleeper,0,0,1)}}
}
function read():any{
 const deadline=now()+30_000;
 for(;;){
  const newline=buffer.indexOf(10);if(newline>=0){const line=buffer.subarray(0,newline);buffer=buffer.subarray(newline+1);return JSON.parse(line.toString())}
  if(now()>deadline)throw new Error("Go runtime fixture transport timed out");
  const chunk=Buffer.allocUnsafe(8192);
  try{const count=readSync(outputFD,chunk,0,chunk.length,null);if(!count)throw new Error("Go runtime fixture transport closed");buffer=Buffer.concat([buffer,chunk.subarray(0,count)])}
  catch(error){if((error as NodeJS.ErrnoException).code!=="EAGAIN")throw error;Atomics.wait(sleeper,0,0,1)}
 }
}

export function runThroughGo(schema:any,input:any,params:any,local:()=>any):any{
 if(depth){nested++;return local()}
 if(!child)start();calls++;
 const name=expect.getState().currentTestName??"module setup";cases.set(name,(cases.get(name)??0)+1);
 const objects=new Map<number,any>();let nextID=1;
 const ref=(value:any)=>{const id=nextID++;objects.set(id,value);return ["handle",{id,kind:typeof value}]};
 const decode=(wire:any)=>{
  if(wire[0]==="handle"){if(!objects.has(wire[1].id))throw new Error("Unknown live fixture handle");return objects.get(wire[1].id)}
  if(wire[0]==="undefined")return undefined;if(wire[0]==="null")return null;
  throw new Error("Unexpected live fixture wire kind: "+wire[0]);
 };
 write({schema:ref(schema),input:ref(input),params:ref(params)});
 for(;;){
  const message=read();
  if(message.type==="result"){if(message.success)return decode(message.data);exceptions++;throw decode(message.exception)}
  if(message.operation!=="runReference")throw new Error("Unexpected runtime operation: "+message.operation);
  try{
   const target=objects.get(message.target);depth++;
   let result;try{result=target._zod.run({value:decode(message.value),issues:[]},decode(message.params))}finally{depth--}
   write({success:true,data:ref(result)});
  }catch(error){write({success:false,exceptionName:"HostThrownValue",exceptionMessage:"Original upstream callback threw",exceptionValue:ref(error)})}
 }
}

afterAll(()=>{
 child?.stdin.end();child?.kill();
 const file=expect.getState().testPath??"unknown";
 const dir=process.env.ZOD2GO_RUNTIME_FIXTURE_REPORT_DIR!;mkdirSync(dir,{recursive:true});
 writeFileSync(join(dir,createHash("sha256").update(file).digest("hex")+".json"),JSON.stringify({runId:process.env.ZOD2GO_RUNTIME_FIXTURE_RUN_ID,file,calls,nested,exceptions,tests:Object.fromEntries(cases)},null,2));
});
