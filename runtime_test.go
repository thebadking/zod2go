package zod2go_test

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"math/big"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/thebadking/zod2go"
)

func runtimeValidator(t *testing.T, source string) *zod2go.RuntimeValidator {
	t.Helper()
	zodPath, err := filepath.Abs("node_modules/zod")
	if err != nil {
		t.Fatal(err)
	}
	quoted, _ := json.Marshal(zodPath)
	doc := &zod2go.Document{Runtime: &zod2go.RuntimeModule{Bundle: "const {z}=require(" + string(quoted) + ");\n" + source, ExportName: "schema"}}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	t.Cleanup(cancel)
	v, err := zod2go.NewRuntimeValidator(ctx, doc, zod2go.RuntimeOptions{})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := v.Close(); err != nil {
			t.Error(err)
		}
	})
	return v
}

func TestRuntimeTransformsRefinementsAndClosures(t *testing.T) {
	v := runtimeValidator(t, `let calls=0; const prefix="captured"; exports.schema=z.string().min(2).refine(s=>s.startsWith("A"),"starts with A").transform(s=>prefix+":"+s+":"+(++calls)); console.log("module initialization log");`)
	for i, want := range []string{"captured:Ada:1", "captured:Ada:2"} {
		got, err := v.Parse("Ada")
		if err != nil || got != want {
			t.Fatalf("call %d: %v %v", i, got, err)
		}
	}
	_, err := v.Parse("Bob")
	zerr, ok := err.(*zod2go.RuntimeError)
	if !ok || len(zerr.Issues) != 1 {
		t.Fatalf("expected refinement error: %v", err)
	}
	issue := zerr.Issues[0].(map[string]any)
	if issue["code"] != "custom" || issue["message"] != "starts with A" {
		t.Fatalf("wrong issue: %+v", issue)
	}
}

func TestRuntimeAsyncAndDynamicCatch(t *testing.T) {
	v := runtimeValidator(t, `exports.schema=z.string().transform(async s=>s.toUpperCase()).pipe(z.string().min(4)).catch(ctx=>"caught:"+ctx.input);`)
	got, err := v.ParseAsync("hi")
	if err != nil || got != "caught:HI" {
		t.Fatalf("unexpected async catch: %v %v", got, err)
	}
	got, err = v.ParseAsync("hello")
	if err != nil || got != "HELLO" {
		t.Fatalf("unexpected async parse: %v %v", got, err)
	}
	if _, err = v.Parse("hello"); err == nil {
		t.Fatal("sync parse of async transform must fail")
	}
}

func TestRuntimeCodecDirections(t *testing.T) {
	v := runtimeValidator(t, `exports.schema=z.codec(z.string(),z.number(),{decode:Number,encode:String});`)
	got, err := v.Parse("42")
	if err != nil || got != float64(42) {
		t.Fatalf("decode: %v %v", got, err)
	}
	got, err = v.Encode(float64(42))
	if err != nil || got != "42" {
		t.Fatalf("encode: %v %v", got, err)
	}
}

func TestRuntimeValueRoundTrip(t *testing.T) {
	v := runtimeValidator(t, `exports.schema=z.unknown();`)
	huge, _ := new(big.Int).SetString("9007199254740993123456789", 10)
	values := []any{zod2go.Undefined, huge, time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), zod2go.RuntimeSet{1.0, "x"}, zod2go.RuntimeMap{{Key: 1.0, Value: "numeric key"}, {Key: "1", Value: "string key"}}, map[string]any{"kind": "bigint", "__proto__": "preserved"}, zod2go.InvalidDate{}}
	for _, input := range values {
		got, err := v.Parse(input)
		if err != nil || !reflect.DeepEqual(got, input) {
			t.Fatalf("roundtrip %T: %#v %v", input, got, err)
		}
	}
	for _, input := range []float64{math.NaN(), math.Inf(1), math.Inf(-1), math.Copysign(0, -1)} {
		got, err := v.Parse(input)
		if err != nil {
			t.Fatal(err)
		}
		n := got.(float64)
		if math.IsNaN(input) {
			if !math.IsNaN(n) {
				t.Fatal("lost NaN")
			}
		} else if input != n || math.Signbit(input) != math.Signbit(n) {
			t.Fatal("lost number identity")
		}
	}
}

func TestRuntimeFunctionHandle(t *testing.T) {
	v := runtimeValidator(t, `let n=0; exports.schema=z.string().transform(prefix=>(suffix)=>prefix+suffix+(++n));`)
	output, err := v.Parse("hello")
	if err != nil {
		t.Fatal(err)
	}
	fn, ok := output.(*zod2go.RuntimeValue)
	if !ok {
		t.Fatalf("not a function handle: %T", output)
	}
	for _, want := range []string{"hello world1", "hello world2"} {
		got, err := fn.Call(" world")
		if err != nil || got != want {
			t.Fatalf("call: %v %v", got, err)
		}
	}
	if err := v.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := fn.Call(" world"); err == nil {
		t.Fatal("closed worker accepted call")
	}
}

func TestRuntimeOriginalNestedIssues(t *testing.T) {
	v := runtimeValidator(t, `exports.schema=z.object({x:z.union([z.string(),z.number()])});`)
	_, err := v.Parse(map[string]any{"x": true})
	zerr, ok := err.(*zod2go.RuntimeError)
	if !ok {
		t.Fatalf("wrong error: %v", err)
	}
	issue := zerr.Issues[0].(map[string]any)
	if issue["code"] != "invalid_union" || issue["errors"] == nil || !reflect.DeepEqual(issue["path"], []any{"x"}) {
		t.Fatalf("lost issue metadata: %#v", issue)
	}
}

func TestRuntimeInitializationFailure(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	_, err := zod2go.NewRuntimeValidator(ctx, &zod2go.Document{Runtime: &zod2go.RuntimeModule{Bundle: "throw Error('broken module')", ExportName: "schema"}}, zod2go.RuntimeOptions{})
	if err == nil || !strings.Contains(err.Error(), "broken module") {
		t.Fatalf("missing initialization error: %v", err)
	}
}

func TestRuntimeFileAndClassMethods(t *testing.T) {
	v := runtimeValidator(t, `exports.schema=z.file().mime("text/plain").min(2);`)
	output, err := v.Parse(zod2go.RuntimeFile{Data: []byte("hello"), Name: "test.txt", MIMEType: "text/plain", LastModified: 1234})
	if err != nil {
		t.Fatal(err)
	}
	file := output.(*zod2go.RuntimeValue)
	name, err := file.Get("name")
	if err != nil || name != "test.txt" {
		t.Fatalf("name: %v %v", name, err)
	}
	text, err := file.Invoke("text")
	if err != nil || text != "hello" {
		t.Fatalf("text: %v %v", text, err)
	}
	if err := file.Release(); err != nil {
		t.Fatal(err)
	}
	if _, err := file.Get("name"); err == nil {
		t.Fatal("released handle remained accessible")
	}
}

func TestRuntimeSymbolIdentity(t *testing.T) {
	v := runtimeValidator(t, `const key=Symbol("key");exports.key=key;exports.schema=z.custom(v=>v===key);`)
	key, err := v.ExportValue("key")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := v.Parse(key); err != nil {
		t.Fatal(err)
	}
	if _, err := v.Parse("key"); err == nil {
		t.Fatal("symbol identity lost")
	}
}

func TestRuntimePromise(t *testing.T) {
	v := runtimeValidator(t, `exports.schema=z.promise(z.string().min(2));`)
	got, err := v.ParseAsync(zod2go.RuntimePromise{Value: "hello"})
	if err != nil || got != "hello" {
		t.Fatalf("promise: %v %v", got, err)
	}
}

func TestRuntimeCloseInterruptsPendingCall(t *testing.T) {
	v := runtimeValidator(t, `exports.schema=z.string().transform(async ()=>new Promise(()=>{}));`)
	finished := make(chan error, 1)
	go func() { _, err := v.ParseAsync("x"); finished <- err }()
	if err := v.Close(); err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-finished:
		if err == nil {
			t.Fatal("pending operation succeeded after close")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("close did not interrupt pending operation")
	}
}

func TestRuntimeGraphIdentityAndCycles(t *testing.T) {
	v := runtimeValidator(t, `exports.schema=z.custom(v=>v.a===v.b && v.self===v);`)
	shared := map[string]any{"name": "shared"}
	input := map[string]any{"a": shared, "b": shared}
	input["self"] = input
	out, err := v.Parse(input)
	if err != nil {
		t.Fatal(err)
	}
	parsed := out.(map[string]any)
	parsed["a"].(map[string]any)["changed"] = true
	if parsed["b"].(map[string]any)["changed"] != true {
		t.Fatal("shared reference lost")
	}
	if parsed["self"].(map[string]any)["a"].(map[string]any)["changed"] != true {
		t.Fatal("cycle lost")
	}
	if _, err := v.Parse(parsed); err != nil {
		t.Fatal("roundtrip identity lost:", err)
	}
}

func TestRuntimeExportedObjectIdentity(t *testing.T) {
	v := runtimeValidator(t, `exports.input={value:1};exports.schema=z.custom(v=>v===exports.input);`)
	input, err := v.ExportValue("input")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := v.Parse(input); err != nil {
		t.Fatal(err)
	}
}

func TestRuntimeParseParamsAndCustomMessages(t *testing.T) {
	v := runtimeValidator(t, `exports.errorMap=issue=>"custom:"+issue.code;exports.schema=z.string();`)
	errorMap, err := v.ExportValue("errorMap")
	if err != nil {
		t.Fatal(err)
	}
	_, err = v.ParseWithParams(42, map[string]any{"error": errorMap, "reportInput": true})
	zerr, ok := err.(*zod2go.RuntimeError)
	if !ok {
		t.Fatalf("missing validation error: %v", err)
	}
	issue := zerr.Issues[0].(map[string]any)
	if issue["message"] != "custom:invalid_type" || issue["input"] != float64(42) {
		t.Fatalf("params lost: %#v", issue)
	}
}

func TestRuntimeFunctionValidationError(t *testing.T) {
	v := runtimeValidator(t, `exports.input=s=>s.length;exports.schema=z.function({input:[z.string()],output:z.number()});`)
	input, err := v.ExportValue("input")
	if err != nil {
		t.Fatal(err)
	}
	fn, err := v.Parse(input)
	if err != nil {
		t.Fatal(err)
	}
	_, err = fn.(*zod2go.RuntimeValue).Call(42)
	if _, ok := err.(*zod2go.RuntimeError); !ok {
		t.Fatalf("function validation issues lost: %T %v", err, err)
	}
}

func TestRuntimeExceptionIdentity(t *testing.T) {
	v := runtimeValidator(t, `exports.schema=z.string().transform(()=>{throw new TypeError("intentional")});`)
	_, err := v.Parse("x")
	exception, ok := err.(*zod2go.RuntimeException)
	if !ok || exception.Name != "TypeError" || exception.Message != "intentional" || exception.Stack == "" {
		t.Fatalf("exception metadata lost: %#v", err)
	}
}

func TestRuntimeSymbolPropertyKeys(t *testing.T) {
	v := runtimeValidator(t, `exports.schema=z.record(z.symbol(),z.number());`)
	symbol, err := v.NewSymbol("key")
	if err != nil {
		t.Fatal(err)
	}
	input := zod2go.RuntimeObject{{Key: symbol, Value: float64(42)}}
	out, err := v.Parse(input)
	if err != nil {
		t.Fatal(err)
	}
	properties, ok := out.(zod2go.RuntimeObject)
	if !ok || len(properties) != 1 {
		t.Fatalf("symbol property lost: %T %#v", out, out)
	}
	key := properties[0].Key.(*zod2go.RuntimeValue)
	if key.ID != symbol.ID || properties[0].Value != float64(42) {
		t.Fatalf("key identity lost: %#v", properties)
	}
	reference, err := v.ParseReference(input)
	if err != nil {
		t.Fatal(err)
	}
	got, err := reference.(*zod2go.RuntimeValue).GetKey(symbol)
	if err != nil || got != float64(42) {
		t.Fatalf("symbol lookup: %v %v", got, err)
	}
}

func TestRuntimeReferencePreservesReadonlyAndGetters(t *testing.T) {
	v := runtimeValidator(t, `let reads=0;const nested=Object.freeze({x:1});const object={nested,get value(){reads++;return reads}};exports.reads=()=>reads;exports.isFrozen=Object.isFrozen;exports.schema=z.unknown().transform(()=>object);`)
	output, err := v.ParseReference(nil)
	if err != nil {
		t.Fatal(err)
	}
	reference := output.(*zod2go.RuntimeValue)
	reads, _ := v.ExportValue("reads")
	count, err := reads.(*zod2go.RuntimeValue).Call()
	if err != nil || count != float64(0) {
		t.Fatalf("serialization invoked getter: %v %v", count, err)
	}
	nested, err := reference.GetReference("nested")
	if err != nil {
		t.Fatal(err)
	}
	frozen, _ := v.ExportValue("isFrozen")
	isFrozen, err := frozen.(*zod2go.RuntimeValue).Call(nested)
	if err != nil || isFrozen != true {
		t.Fatalf("readonly identity lost: %v %v", isFrozen, err)
	}
}

func TestRuntimeUnpairedSurrogates(t *testing.T) {
	v := runtimeValidator(t, `exports.schema=z.string().refine(s=>s.length===1 && s.charCodeAt(0)===0xd800);`)
	input, err := v.UTF16String([]uint16{0xd800})
	if err != nil {
		t.Fatal(err)
	}
	output, err := v.Parse(input)
	if err != nil {
		t.Fatal(err)
	}
	code, err := output.(*zod2go.RuntimeValue).Invoke("charCodeAt", float64(0))
	if err != nil || code != float64(0xd800) {
		t.Fatalf("UTF-16 value lost: %v %v", code, err)
	}
}

func TestRuntimeRawAsyncFunctionHandle(t *testing.T) {
	v := runtimeValidator(t, `exports.input=async()=>Object.freeze({answer:42});exports.schema=z.unknown();`)
	fn, err := v.ExportValue("input")
	if err != nil {
		t.Fatal(err)
	}
	promise, err := fn.(*zod2go.RuntimeValue).CallReference()
	if err != nil {
		t.Fatal(err)
	}
	got, err := promise.(*zod2go.RuntimeValue).Await()
	if err != nil || got.(map[string]any)["answer"] != float64(42) {
		t.Fatalf("promise handle: %v %v", got, err)
	}
}

func TestRuntimeClassInstancesAndMultipleSchemas(t *testing.T) {
	v := runtimeValidator(t, `class Item{constructor(value){this.value=value}read(){return this.value}};exports.Item=Item;exports.schema=z.instanceof(Item);exports.other=z.custom(v=>v instanceof Item && v.value===42);`)
	constructor, err := v.ExportValue("Item")
	if err != nil {
		t.Fatal(err)
	}
	item, err := constructor.(*zod2go.RuntimeValue).Construct(float64(42))
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := v.Parse(item)
	if err != nil {
		t.Fatal(err)
	}
	value, err := parsed.(*zod2go.RuntimeValue).Invoke("read")
	if err != nil || value != float64(42) {
		t.Fatalf("class identity: %v %v", value, err)
	}
	other, err := v.ExportValue("other")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := other.(*zod2go.RuntimeValue).Invoke("parse", item); err != nil {
		t.Fatal("cross-schema class identity lost:", err)
	}
}

func TestRuntimeSafeParseDistinguishesThrownErrors(t *testing.T) {
	v := runtimeValidator(t, `exports.schema=z.string().transform(s=>{if(s==="throw")throw new z.ZodError([{code:"custom",message:"thrown",path:[]}]);return s});`)
	ordinary := v.SafeParse(42)
	if ordinary.Error == nil || ordinary.Exception != nil {
		t.Fatalf("ordinary failure: %+v", ordinary)
	}
	thrown := v.SafeParse("throw")
	if thrown.Exception == nil || thrown.Error != nil {
		t.Fatalf("thrown failure: %+v", thrown)
	}
	good := v.SafeParse("ok")
	if !good.Success || good.Data != "ok" {
		t.Fatalf("success: %+v", good)
	}
}

func TestRuntimeArbitraryThrownValues(t *testing.T) {
	v := runtimeValidator(t, `exports.thrown=Symbol("failure");exports.schema=z.string().transform(()=>{throw exports.thrown});`)
	expected, err := v.ExportValue("thrown")
	if err != nil {
		t.Fatal(err)
	}
	_, err = v.Parse("x")
	exception, ok := err.(*zod2go.RuntimeException)
	if !ok {
		t.Fatalf("wrong exception: %T %v", err, err)
	}
	actual := exception.Value.(*zod2go.RuntimeValue)
	if actual.ID != expected.(*zod2go.RuntimeValue).ID {
		t.Fatal("thrown symbol identity lost")
	}
}

func TestRuntimeThrownProxyDoesNotInvokeGetters(t *testing.T) {
	v := runtimeValidator(t, `let reads=0;exports.reads=()=>reads;exports.thrown=new Proxy({},{get(){reads++;throw Error("getter")}});exports.schema=z.string().transform(()=>{throw exports.thrown});`)
	result := v.SafeParse("x")
	if result.Exception == nil {
		t.Fatal("missing exception")
	}
	reads, _ := v.ExportValue("reads")
	count, err := reads.(*zod2go.RuntimeValue).Call()
	if err != nil || count != float64(0) {
		t.Fatalf("exception serialization touched proxy: %v %v", count, err)
	}
}

func TestRuntimeConcurrentAsyncCallbacks(t *testing.T) {
	v := runtimeValidator(t, `let release,started=false;exports.started=()=>started;exports.schema=z.string().transform(async value=>{if(value==="wait"){started=true;await new Promise(resolve=>{release=resolve});return "done"}release();return "released"});`)
	started, err := v.ExportValue("started")
	if err != nil {
		t.Fatal(err)
	}
	completed := make(chan error, 1)
	go func() {
		value, err := v.ParseAsync("wait")
		if err == nil && value != "done" {
			err = fmt.Errorf("wrong waiting result: %v", value)
		}
		completed <- err
	}()
	for {
		ready, err := started.(*zod2go.RuntimeValue).Call()
		if err != nil {
			t.Fatal(err)
		}
		if ready == true {
			break
		}
		time.Sleep(time.Millisecond)
	}
	value, err := v.ParseAsync("release")
	if err != nil || value != "released" {
		t.Fatalf("release: %v %v", value, err)
	}
	if err := <-completed; err != nil {
		t.Fatal(err)
	}
}

func TestRuntimeCustomCheckGuard(t *testing.T) {
	v := runtimeValidator(t, `exports.schema=z.string().min(3,{when:()=>false});`)
	value, err := v.Parse("x")
	if err != nil || value != "x" {
		t.Fatalf("custom guard lost: %v %v", value, err)
	}
}
