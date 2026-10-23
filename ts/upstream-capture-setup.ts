import { execFileSync } from "node:child_process";
import { existsSync, mkdirSync, writeFileSync } from "node:fs";
import {randomUUID} from "node:crypto";
import { resolve,join } from "node:path";
export default function setup(){
 const root=resolve(import.meta.dirname,"..");
 const binary=process.env.ZOD2GO_FIXTURE_BINARY??"/tmp/zod2go-upstream-fixtures";
 process.env.ZOD2GO_FIXTURE_BINARY=binary;
 process.env.ZOD2GO_FIXTURE_REPORT_DIR??="/tmp/zod2go-native-fixture-reports";
 mkdirSync(process.env.ZOD2GO_FIXTURE_REPORT_DIR,{recursive:true});
 const runId=randomUUID();process.env.ZOD2GO_FIXTURE_RUN_ID=runId;
 writeFileSync(join(process.env.ZOD2GO_FIXTURE_REPORT_DIR,"manifest.json"),JSON.stringify({runId,reportDir:process.env.ZOD2GO_FIXTURE_REPORT_DIR,startedAt:new Date().toISOString()}));
 console.log(`Native fixture report: ${process.env.ZOD2GO_FIXTURE_REPORT_DIR} (run ${runId})`);
 execFileSync(process.env.GO_BINARY??(existsSync("/usr/local/go/bin/go")?"/usr/local/go/bin/go":"go"),["build","-o",binary,"./cmd/zod2go-fixtures"],{cwd:root,env:{...process.env,GOTOOLCHAIN:"go1.23.6"}});
}
