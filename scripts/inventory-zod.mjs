import { readFileSync, writeFileSync, readdirSync } from 'node:fs';
import { resolve, relative, join } from 'node:path';
import { fileURLToPath } from 'node:url';
import { createHash } from 'node:crypto';

const root = resolve(fileURLToPath(new URL('..', import.meta.url)));
const sourceRoot = join(root, 'vendor/zod/packages/zod/src/v4');
const schemas = readFileSync(join(sourceRoot, 'core/schemas.ts'), 'utf8');
const util = readFileSync(join(sourceRoot, "core/util.ts"), "utf8");
const checks = readFileSync(join(sourceRoot, 'core/checks.ts'), 'utf8');
const unique = values => [...new Set(values)].sort();
function files(dir) {
  return readdirSync(dir, { withFileTypes: true }).flatMap(entry => {
    const path = join(dir, entry.name);
    return entry.isDirectory() ? files(path) : [path];
  });
}
const inventory = {
  zodVersion: JSON.parse(readFileSync(join(root, 'vendor/zod/packages/zod/package.json'), 'utf8')).version,
  schemaSourceSHA256: createHash('sha256').update(schemas).digest('hex'),
  checkSourceSHA256: createHash('sha256').update(checks).digest('hex'),
  schemaTypes: unique([...schemas.matchAll(/^\s*type: "([^"]+)";/gm)].map(match => match[1])),
  checkKinds: unique([...checks.matchAll(/^\s*check: "([^"]+)";/gm)].map(match => match[1])),
  stringFormats: unique([...schemas.matchAll(/\$Zod(?:Custom)?StringFormat(?:Def)?<"([^"]+)"/g)].map(match => match[1])),
  hashFormats: [...util.match(/export type HashAlgorithm = ([^;]+);/)[1].matchAll(/"([^"]+)"/g)].flatMap(match => [...util.match(/export type HashEncoding = ([^;]+);/)[1].matchAll(/"([^"]+)"/g)].map(enc => `${match[1]}_${enc[1]}`)).sort(),
  upstreamTestFiles: files(sourceRoot).filter(path => path.endsWith('.test.ts')).map(path => relative(root, path)).sort(),
};
const target = join(root, 'specs/upstream-inventory.json');
const content = JSON.stringify(inventory, null, 2) + '\n';
if (process.argv.includes('--check')) {
  if (readFileSync(target, 'utf8') !== content) throw new Error('Upstream inventory is stale; run npm run inventory:zod');
} else writeFileSync(target, content);
console.log(`${inventory.schemaTypes.length} schema types, ${inventory.checkKinds.length} check kinds, ${inventory.stringFormats.length} named string formats, ${inventory.upstreamTestFiles.length} upstream test files`);
