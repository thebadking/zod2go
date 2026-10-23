import { afterAll, beforeAll, describe, expect, it } from "vitest";
import { execFileSync } from "node:child_process";
import { mkdtempSync, writeFileSync, rmSync, existsSync } from "node:fs";
import { join, resolve } from "node:path";
import { tmpdir } from "node:os";
import { exportModuleSchema } from "./module.js";

const root = resolve(import.meta.dirname, "../..");
const dir = mkdtempSync(join(tmpdir(), "zod2go-module-test-"));
const binary = join(dir, "zod2go");
beforeAll(() => {
  execFileSync(process.env.GO_BINARY ?? (existsSync("/usr/local/go/bin/go") ? "/usr/local/go/bin/go" : "go"),
    ["build", "-o", binary, "./cmd/zod2go"], { cwd: root, env: { ...process.env, GOTOOLCHAIN: "go1.23.6" } });
}, 60_000);
afterAll(() => rmSync(dir, { recursive: true, force: true }));

describe("bundled module through Go runtime CLI", () => {
  for (const c of [
    {name:"CoreStringSchema",input:"core",output:"core",args:[]},
    {name:"MiniStringSchema",input:"mini",output:"mini",args:[]},
    { name: "TopLevelAwaitSchema", input: "hello", output: "top-level:hello", args: [] },
    { name: "ClosureSchema", input: "Ada", output: "captured:Ada:1", args: [] },
    { name: "AsyncSchema", input: "hello", output: "HELLO", args: ["-op", "parseAsync"] },
    { name: "CodecSchema", input: 42, output: "42", args: ["-op", "encode"] },
    { name: "BigIntSchema", input: ["bigint", "900719925474099312345"], output: ["bigint", "900719925474099312345"], args: ["-tagged"] },
    { name: "NumericMapSchema", input: ["map", [[["number",1],["string","x"]]]], output: ["id", {id:1,value:["map", [[["number",1],["string","x"]]]]}], args: ["-tagged"] },
  ]) {
    it(c.name, async () => {
      const doc = await exportModuleSchema(join(root,"examples/runtime-schemas.ts"), c.name);
      const path = join(dir,c.name+".json");
      writeFileSync(path, JSON.stringify(doc));
      const output = execFileSync(binary,["-schema",path,"-runtime",...c.args],{ input:JSON.stringify(c.input),encoding:"utf8" });
      expect(JSON.parse(output)).toEqual(c.output);
    });
  }
});


it("keeps function handles alive across streaming CLI requests", async () => {
  const doc = await exportModuleSchema(join(root,"examples/runtime-schemas.ts"),"FunctionSchema");
  const path = join(dir,"stream.json"); writeFileSync(path,JSON.stringify(doc));
  const requests = [
    {operation:"parse",value:["string","prefix"]},
    {operation:"call",target:1,value:["array",[["string","first"]]]},
    {operation:"call",target:1,value:["array",[["string","second"]]]},
  ];
  const output = execFileSync(binary,["-schema",path,"-runtime","-stream"],{input:requests.map(r=>JSON.stringify(r)).join("\n")+"\n",encoding:"utf8"});
  const responses = output.trim().split("\n").map(line=>JSON.parse(line));
  expect(responses[0].data).toEqual(["handle",{id:1,kind:"function"}]);
  expect(responses[1].data).toEqual(["string","prefix:first:1"]);
  expect(responses[2].data).toEqual(["string","prefix:second:2"]);
});

it("preserves source-module file context in reference mode",async()=>{
  const doc=await exportModuleSchema(join(root,"examples/runtime-reference.ts"),"SourceContextSchema",{bundle:false});
  const path=join(dir,"reference.json");writeFileSync(path,JSON.stringify(doc));
  const output=execFileSync(binary,["-schema",path,"-runtime"],{input:JSON.stringify("hello"),encoding:"utf8"});
  expect(JSON.parse(output)).toBe("source-asset:hello");
});

it("honors tagged document encoding without a CLI flag",async()=>{
 const doc=await exportModuleSchema(join(root,"examples/runtime-schemas.ts"),"BigIntSchema");
 doc.$zod2go.encoding="tagged";
 const path=join(dir,"tagged-header.json");writeFileSync(path,JSON.stringify(doc));
 const output=execFileSync(binary,["-schema",path,"-runtime"],{input:JSON.stringify(["bigint","900719925474099312345"]),encoding:"utf8"});
 expect(JSON.parse(output)).toEqual(["bigint","900719925474099312345"]);
});
