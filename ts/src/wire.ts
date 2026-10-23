/** Typed transport shared by fixture runners and standalone native Go clients. */
export type TaggedValue = [string, unknown?];
export function encodeValue(value: unknown): TaggedValue {
  const seen = new Map<object,number>();
  function encode(value:unknown):TaggedValue {
    if(value===undefined)return ["undefined"];
    if(value===null)return ["null"];
    if(typeof value==="string") {
      for(const char of value){const cp=char.codePointAt(0)!;if(cp>=0xd800&&cp<=0xdfff)throw new Error("Malformed UTF-16 strings require a runtime handle");}
      return ["string",value];
    }
    if(typeof value==="boolean")return ["boolean",value];
    if(typeof value==="bigint")return ["bigint",String(value)];
    if(typeof value==="number")return ["number",Number.isNaN(value)?"NaN":value===Infinity?"Infinity":value===-Infinity?"-Infinity":Object.is(value,-0)?"-0":value];
    if(typeof value!=="object")throw new Error(`Typed transport requires a runtime handle for ${typeof value}`);
    if(value instanceof Date)return ["date",Number.isNaN(value.getTime())?"invalid":value.toISOString()];
    if(seen.has(value))return ["ref",seen.get(value)];
    const id=seen.size+1;seen.set(value,id);
    let body:TaggedValue;
    if(Array.isArray(value))body=["array",value.map(encode)];
    else if(value instanceof Set)body=["set",[...value].map(encode)];
    else if(value instanceof Map)body=["map",[...value].map(([k,v])=>[encode(k),encode(v)])];
    else {
      if(Object.getPrototypeOf(value)!==Object.prototype && Object.getPrototypeOf(value)!==null)throw new Error("Class instances require a runtime handle");
      if(Reflect.ownKeys(value).some(key=>typeof key==="symbol"&&Object.prototype.propertyIsEnumerable.call(value,key)))throw new Error("Symbol keys require a runtime handle");
      for(const key of Object.keys(value))encode(key);
      body=["object",Object.entries(value).map(([k,v])=>[k,encode(v)])];
    }
    return ["id",{id,value:body}];
  }
  return encode(value);
}

export function decodeValue(wire:TaggedValue):unknown {
  const refs=new Map<number,any>();
  function decode(wire:TaggedValue):any {
    const [kind,value]=wire;
    switch(kind){
      case "undefined":return undefined;
      case "null":return null;
      case "string":case "boolean":return value;
      case "number":return typeof value==="number"?value:({NaN:NaN,Infinity:Infinity,"-Infinity":-Infinity,"-0":-0} as Record<string,number>)[value as string];
      case "bigint":return BigInt(value as string);
      case "date":return new Date(value==="invalid"?NaN:value as string);
      case "ref":if(!refs.has(value as number))throw new Error("Unknown graph reference");return refs.get(value as number);
      case "id":{
        const payload=value as {id:number;value:TaggedValue};
        const [type,items]=payload.value;
        const result=type==="array"?[]:type==="set"?new Set():type==="map"?new Map():{};
        refs.set(payload.id,result);fill(type,items,result);return result;
      }
      case "array":{const result:any[]=[];fill(kind,value,result);return result;}
      case "object":{const result={};fill(kind,value,result);return result;}
      case "set":{const result=new Set();fill(kind,value,result);return result;}
      case "map":{const result=new Map();fill(kind,value,result);return result;}
      default:throw new Error(`Unsupported native wire kind: ${kind}`);
    }
  }
  function fill(kind:string,items:any,result:any){
    if(kind==="array")for(const item of items)result.push(decode(item));
    else if(kind==="set")for(const item of items)result.add(decode(item));
    else if(kind==="map")for(const [key,value]of items)result.set(decode(key),decode(value));
    else if(kind==="object")for(const [key,value]of items)Object.defineProperty(result,key,{value:decode(value),enumerable:true,writable:true,configurable:true});
    else throw new Error(`Invalid graph node kind: ${kind}`);
  }
  return decode(wire);
}
