import {readFileSync,readdirSync} from 'node:fs';
import {join,relative,resolve} from 'node:path';
const root=resolve(import.meta.dirname,'..');
const inventory=JSON.parse(readFileSync(join(root,'specs/upstream-inventory.json'),'utf8'));
function load(dir){
 const manifest=JSON.parse(readFileSync(join(dir,'manifest.json'),'utf8'));
 return readdirSync(dir).filter(f=>f.endsWith('.json')&&f!=='manifest.json').map(f=>JSON.parse(readFileSync(join(dir,f),'utf8'))).filter(r=>r.runId===manifest.runId);
}
const native=load(process.env.ZOD2GO_FIXTURE_REPORT_DIR??'/tmp/zod2go-native-fixture-reports');
const runtime=load(process.env.ZOD2GO_RUNTIME_FIXTURE_REPORT_DIR??'/tmp/zod2go-runtime-fixture-reports');
const failures=[];
for(const file of inventory.upstreamTestFiles){
 const n=native.find(r=>relative(root,r.file)===file),r=runtime.find(r=>relative(root,r.file)===file);
 if(!n||!r){failures.push({file,error:'Missing fresh native or runtime report'});continue;}
 if(n.mismatches.length)failures.push({file,error:'Native mismatches',details:n.mismatches});
 for(const [reason,entry] of Object.entries(n.unported))for(const test of entry.tests){
  if(!r.tests[test])failures.push({file,test,reason,error:'Runtime did not cover native-ineligible case'});
 }
}
const sum=(rows,key)=>rows.reduce((n,r)=>n+r[key],0);
const summary={files:inventory.upstreamTestFiles.length,nativeFixtures:sum(native,'distinctFixtures'),nativeCalls:sum(native,'calls'),runtimeCalls:sum(runtime,'calls'),nestedRuntimeCalls:sum(runtime,'nested'),runtimeExceptionPaths:sum(runtime,'exceptions'),failures};
if(!summary.runtimeCalls)failures.push({error:'No Go runtime calls recorded'});
console.log(JSON.stringify(summary,null,2));
if(failures.length)process.exitCode=1;
