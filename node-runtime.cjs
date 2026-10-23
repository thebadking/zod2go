const fs = require('node:fs');
const readline = require('node:readline');
const {types}=require('node:util');
const handles = new Map();
const ids = new Map();
let nextId = 1;
function handle(value, kind = typeof value) {
  if (!ids.has(value)) { ids.set(value, nextId); handles.set(nextId++, value); }
  return ['handle', { id: ids.get(value), kind }];
}
function wellFormed(value) {
  for (const char of value) {const cp=char.codePointAt(0);if(cp>=0xd800 && cp<=0xdfff)return false;}
  return true;
}
function reference(value) {
  if(value!==null && (typeof value==='object'||typeof value==='function'||typeof value==='symbol'||typeof value==='string'&&!wellFormed(value)))return handle(value,typeof value==='object' ? value.constructor?.name ?? 'object' : typeof value);
  return encode(value);
}
function encode(value, context = {ids:new Map(), next:1}) {
  if (value === undefined) return ['undefined'];
  if (value === null) return ['null'];
  if (typeof value === 'bigint') return ['bigint', String(value)];
  if (typeof value === 'number') {
    if (Number.isNaN(value)) return ['number', 'NaN'];
    if (value === Infinity) return ['number', 'Infinity'];
    if (value === -Infinity) return ['number', '-Infinity'];
    if (Object.is(value, -0)) return ['number', '-0'];
    return ['number', value];
  }
  if (typeof value === 'string') return wellFormed(value)?['string',value]:handle(value,'string');
  if (typeof value === 'boolean') return ['boolean',value];
  if (typeof value === 'symbol' || typeof value === 'function') return handle(value);
  if (value instanceof Date) return ['date', Number.isNaN(value.getTime()) ? 'invalid' : value.toISOString()];
  if (!(value instanceof Set) && !(value instanceof Map) && !Array.isArray(value) && Object.getPrototypeOf(value) !== Object.prototype && Object.getPrototypeOf(value) !== null) return handle(value, value.constructor?.name ?? 'object');
  if (context.ids.has(value)) return ['ref',context.ids.get(value)];
  const id = context.next++; context.ids.set(value,id);
  let body;
  if (value instanceof Set) body = ['set', [...value].map(x => encode(x,context))];
  else if (value instanceof Map) body = ['map', [...value].map(([k,v]) => [encode(k,context),encode(v,context)])];
  else if (Array.isArray(value)) body = ['array',value.map(x => encode(x,context))];
  else {
    const keys=Reflect.ownKeys(value).filter(key=>Object.prototype.propertyIsEnumerable.call(value,key));
    if(keys.some(key=>typeof key==='symbol'||!wellFormed(key)))body=['keyedObject',keys.map(key=>[encode(key,context),encode(value[key],context)])];
    else body=['object',keys.map(key=>[key,encode(value[key],context)])];
  }
  return ['id',{id,value:body}];
}

function decode(wire, refs = new Map()) {
  const [kind, value] = wire;
  switch (kind) {
    case 'ref': {
      if (!refs.has(value)) throw new Error('Unknown graph reference');
      return refs.get(value);
    }
    case 'id': {
      const [type, items] = value.value;
      let result;
      if (type === 'object' || type === 'keyedObject') result = {};
      else if (type === 'array') result = [];
      else if (type === 'set') result = new Set();
      else if (type === 'map') result = new Map();
      else throw new Error('Invalid graph node type');
      refs.set(value.id,result);
      if (type === 'keyedObject') for(const [k,v]of items)Object.defineProperty(result,decode(k,refs),{value:decode(v,refs),enumerable:true,writable:true,configurable:true});
      if (type === 'object') for (const [k,v] of items) Object.defineProperty(result,k,{value:decode(v,refs),enumerable:true,writable:true,configurable:true});
      if (type === 'array') for (const item of items) result.push(decode(item,refs));
      if (type === 'set') for (const item of items) result.add(decode(item,refs));
      if (type === 'map') for (const [k,v] of items) result.set(decode(k,refs),decode(v,refs));
      return result;
    }
    case 'undefined': return undefined;
    case 'null': return null;
    case 'string': case 'boolean': return value;
    case 'number': return typeof value === 'number' ? value : ({NaN:NaN,Infinity:Infinity,'-Infinity':-Infinity,'-0':-0})[value];
    case 'bigint': return BigInt(value);
    case 'date': return new Date(value === 'invalid' ? NaN : value);
    case 'array': return value.map(x => decode(x,refs));
    case 'set': return new Set(value.map(x => decode(x,refs)));
    case 'map': return new Map(value.map(([k,v]) => [decode(k,refs),decode(v,refs)]));
    case 'keyedObject': return Object.fromEntries(value.map(([k,v])=>[decode(k,refs),decode(v,refs)]));
    case 'object': return Object.fromEntries(value.map(([k,v]) => [k,decode(v,refs)]));
    case 'file': return new File([Buffer.from(value.data, 'base64')], value.name, {type:value.type,lastModified:value.lastModified});
    case 'regexp': return new RegExp(value.source,value.flags);
    case 'promise': return Promise.resolve(decode(value,refs));
    case 'handle': {
      if (!handles.has(value.id)) throw new Error(`Unknown runtime handle ${value.id}`);
      return handles.get(value.id);
    }
    default: throw new Error(`Unknown wire kind ${kind}`);
  }
}
function send(response) {
  const data = Buffer.from(JSON.stringify(response)+'\n');
  let offset = 0;
  while (offset < data.length) offset += fs.writeSync(3, data, offset, data.length-offset);
}
function failure(error) {
  const raw=handle(error,'thrown');
  if(types.isNativeError(error)){
    let name='Error',message='JavaScript error',stack='';
    try{name=String(error.name);message=String(error.message);stack=String(error.stack??'')}catch{}
    const ownIssues=Object.getOwnPropertyDescriptor(error,'issues');
    if(Array.isArray(ownIssues?.value) && (name==='ZodError'||name==='$ZodError'))return {success:false,issues:encode(ownIssues.value),message,errorName:name,thrown:true,exceptionValue:raw};
    return {success:false,exception:stack,exceptionName:name,exceptionMessage:message,exceptionValue:raw};
  }
  const primitive=error===null||!['object','function'].includes(typeof error);
  return {success:false,exceptionName:'ThrownValue',exceptionMessage:primitive?String(error):'JavaScript threw an object',exceptionValue:raw};
}
async function main() {
  const module = process.argv[4] ? await (await import(process.argv[4])).tsImport(process.argv[1], require("node:url").pathToFileURL(process.argv[1]).href) : await import(require("node:url").pathToFileURL(process.argv[1]).href);
  const namespace = process.argv[3] === "esm" ? module : module.default;
  const exports=namespace.__zod2goModule??namespace;
  const core=exports.__zod2goCore??namespace.__zod2goCore??(process.argv[5]?await import(process.argv[5]):undefined);
  const schema = exports[process.argv[2]];
  if (!schema || typeof schema.safeParse !== 'function' && !schema._zod?.run) throw new Error('Selected export is not a Zod schema');
  send({success:true});
  const jobs=new Set();
  async function dispatch(request){
    const reply=response=>send({...response,id:request.id});
    try {
      if(request.operation==='symbol'||request.operation==='globalSymbol'){
        const value=decode(request.value);reply({success:true,data:encode(request.operation==='symbol'?Symbol(value):Symbol.for(value))});return;
      }
      if(request.operation==='utf16'){
        const units=decode(request.value);let value='';for(let i=0;i<units.length;i+=32768)value+=String.fromCharCode(...units.slice(i,i+32768));reply({success:true,data:reference(value)});return;
      }
      if (request.operation === 'export') {
        const name = decode(request.value);
        if (!Object.prototype.hasOwnProperty.call(exports,name)) throw new Error('Unknown module export: '+name);
        const value=exports[name];
        reply({success:true,data:reference(value)});
        return;
      }
      if (['get','invoke','getReference','invokeReference'].includes(request.operation)) {
        if (!handles.has(request.target)) throw new Error('Unknown runtime handle');
        const target = handles.get(request.target);
        const args = decode(request.value);
        const result = request.operation.startsWith('get') ? target[args[0]] : await target[args[0]](...args.slice(1));
        reply({success:true,data:request.operation.endsWith('Reference')?reference(result):encode(result)});
        return;
      }
      if(request.operation==='await'){if(!handles.has(request.target))throw new Error('Unknown runtime handle');reply({success:true,data:encode(await handles.get(request.target))});return;}
      if (request.operation === 'release') {
        if(handles.has(request.target)){const value=handles.get(request.target);handles.delete(request.target);ids.delete(value);}
        reply({success:true,data:encode(undefined)});
        return;
      }
      if(request.operation==='construct'){const Constructor=handles.get(request.target);if(typeof Constructor!=='function')throw new Error('Runtime handle is not a constructor');reply({success:true,data:reference(Reflect.construct(Constructor,decode(request.value)))});return;}
      if (request.operation === 'call' || request.operation === 'callReference') {
        const fn = handles.get(request.target);
        if (typeof fn !== 'function') throw new Error('Runtime handle is not callable');
        const result=fn(...decode(request.value));reply({success:true,data:request.operation==='callReference'?reference(result):encode(await result)});
        return;
      }
      if(request.operation==='runReference'){
        const target=handles.get(request.target);if(!target?._zod?.run)throw new Error('Handle is not a core Zod schema');
        reply({success:true,data:reference(target._zod.run({value:decode(request.value),issues:[]},request.params?decode(request.params):{}))});return;
      }
      const methods = {parse:'safeParse',parseAsync:'safeParseAsync',encode:'safeEncode',encodeAsync:'safeEncodeAsync',parseReference:'safeParse',parseAsyncReference:'safeParseAsync',encodeReference:'safeEncode',encodeAsyncReference:'safeEncodeAsync'};
      const method = methods[request.operation];
      if(!method)throw new Error('Unsupported operation: '+request.operation);
      const value=decode(request.value),params=request.params?decode(request.params):undefined;
      let result;
      if(typeof schema[method]==='function')result=await schema[method](value,params);
      else if(typeof core?.[method]==='function')result=await core[method](schema,value,params);
      else throw new Error('Bare core schemas require the matching core module namespace');
      reply(result.success ? {success:true,data:request.operation.endsWith('Reference')?reference(result.data):encode(result.data)} : {success:false,issues:encode(result.error.issues),message:result.error.message,errorName:result.error.name});
    } catch(error) {reply(failure(error));
    }
  }
  for await(const line of readline.createInterface({input:process.stdin,crlfDelay:Infinity})){
    let request;try{request=JSON.parse(line)}catch(error){send(failure(error));continue;}
    const job=dispatch(request);jobs.add(job);job.finally(()=>jobs.delete(job));
  }
  await Promise.allSettled(jobs);
}
main().catch(error => {send(failure(error));process.exitCode=1;});
