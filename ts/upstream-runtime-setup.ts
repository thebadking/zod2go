import { execFileSync } from "node:child_process";
import { existsSync, mkdirSync, writeFileSync } from "node:fs";
import {randomUUID} from "node:crypto";
import {resolve,join}from"node:path";
export default function setup(){
 const binary="/tmp/zod2go-runtime-fixtures";
 process.env.ZOD2GO_RUNTIME_FIXTURE_BINARY=binary;
 process.env.ZOD2GO_RUNTIME_FIXTURE_REPORT_DIR="/tmp/zod2go-runtime-fixture-reports";
 const dir=process.env.ZOD2GO_RUNTIME_FIXTURE_REPORT_DIR;
 mkdirSync(dir,{recursive:true});
 const runId=randomUUID();process.env.ZOD2GO_RUNTIME_FIXTURE_RUN_ID=runId;
 writeFileSync(join(dir,"manifest.json"),JSON.stringify({runId,startedAt:new Date().toISOString()}));
 execFileSync(process.env.GO_BINARY??(existsSync("/usr/local/go/bin/go")?"/usr/local/go/bin/go":"go"),["build","-o",binary,"./cmd/zod2go-runtime-fixtures"],{cwd:resolve(import.meta.dirname,".."),env:{...process.env,GOTOOLCHAIN:"go1.23.6"}});
}
