package zod2go_test

import (
	"encoding/json"
	"math"
	"math/big"
	"os"
	"reflect"
	"sort"
	"testing"
	"time"

	"github.com/thebadking/zod2go"
)

// Every upstream schema type crosses the Go runtime transport. This is a
// feature gate, separate from the still-required exhaustive upstream fixture port.
func TestRuntimeEveryUpstreamSchemaType(t *testing.T) {
	type schemaCase struct {
		name, schema  string
		input         any
		want          any
		failure       bool
		exportedInput string
		async         bool
	}
	cases := []schemaCase{
		{name: "any", schema: `z.any()`, input: "x", want: "x"},
		{name: "array", schema: `z.array(z.string())`, input: []any{"x"}, want: []any{"x"}},
		{name: "bigint", schema: `z.bigint()`, input: big.NewInt(123), want: big.NewInt(123)},
		{name: "boolean", schema: `z.boolean()`, input: true, want: true},
		{name: "catch", schema: `z.string().catch("fallback")`, input: 42, want: "fallback"},
		{name: "custom", schema: `z.custom(x=>x===42)`, input: 42, want: float64(42)},
		{name: "date", schema: `z.date()`, input: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), want: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)},
		{name: "default", schema: `z.string().default("fallback")`, input: zod2go.Undefined, want: "fallback"},
		{name: "enum", schema: `z.enum(["a","b"])`, input: "a", want: "a"},
		{name: "file", schema: `z.file()`, input: zod2go.RuntimeFile{Data: []byte("hello"), Name: "a.txt"}},
		{name: "function", schema: `z.function({input:[z.string()],output:z.number()})`, exportedInput: `s=>s.length`},
		{name: "intersection", schema: `z.intersection(z.object({a:z.string()}),z.object({b:z.number()}))`, input: map[string]any{"a": "x", "b": float64(1)}, want: map[string]any{"a": "x", "b": float64(1)}},
		{name: "lazy", schema: `z.lazy(()=>z.string())`, input: "x", want: "x"},
		{name: "literal", schema: `z.literal("x")`, input: "x", want: "x"},
		{name: "map", schema: `z.map(z.number(),z.string())`, input: zod2go.RuntimeMap{{Key: float64(1), Value: "x"}}, want: zod2go.RuntimeMap{{Key: float64(1), Value: "x"}}},
		{name: "nan", schema: `z.nan()`, input: math.NaN(), want: math.NaN()},
		{name: "never", schema: `z.never()`, input: "x", failure: true},
		{name: "nonoptional", schema: `z.string().optional().nonoptional()`, input: "x", want: "x"},
		{name: "null", schema: `z.null()`, input: nil},
		{name: "nullable", schema: `z.string().nullable()`, input: nil},
		{name: "number", schema: `z.number()`, input: float64(42), want: float64(42)},
		{name: "object", schema: `z.object({x:z.string()})`, input: map[string]any{"x": "y"}, want: map[string]any{"x": "y"}},
		{name: "optional", schema: `z.string().optional()`, input: zod2go.Undefined, want: zod2go.Undefined},
		{name: "pipe", schema: `z.string().pipe(z.string().min(2))`, input: "hi", want: "hi"},
		{name: "prefault", schema: `z.string().prefault("fallback")`, input: zod2go.Undefined, want: "fallback"},
		{name: "promise", schema: `z.promise(z.string())`, input: zod2go.RuntimePromise{Value: "x"}, want: "x", async: true},
		{name: "readonly", schema: `z.object({x:z.string()}).readonly()`, input: map[string]any{"x": "y"}, want: map[string]any{"x": "y"}},
		{name: "record", schema: `z.record(z.string(),z.number())`, input: map[string]any{"x": float64(1)}, want: map[string]any{"x": float64(1)}},
		{name: "set", schema: `z.set(z.string())`, input: zod2go.RuntimeSet{"x"}, want: zod2go.RuntimeSet{"x"}},
		{name: "string", schema: `z.string()`, input: "x", want: "x"},
		{name: "success", schema: `z.success(z.string())`, input: "x", want: true},
		{name: "symbol", schema: `z.symbol()`, exportedInput: `Symbol("x")`},
		{name: "template_literal", schema: `z.templateLiteral([z.number(),"px"])`, input: "12px", want: "12px"},
		{name: "transform", schema: `z.transform(x=>String(x))`, input: 42, want: "42"},
		{name: "tuple", schema: `z.tuple([z.string(),z.number()])`, input: []any{"x", float64(1)}, want: []any{"x", float64(1)}},
		{name: "undefined", schema: `z.undefined()`, input: zod2go.Undefined, want: zod2go.Undefined},
		{name: "union", schema: `z.union([z.string(),z.number()])`, input: "x", want: "x"},
		{name: "unknown", schema: `z.unknown()`, input: float64(1), want: float64(1)},
		{name: "void", schema: `z.void()`, input: zod2go.Undefined, want: zod2go.Undefined},
	}
	data, err := os.ReadFile("specs/upstream-inventory.json")
	if err != nil {
		t.Fatal(err)
	}
	var inventory struct {
		SchemaTypes []string `json:"schemaTypes"`
	}
	if err := json.Unmarshal(data, &inventory); err != nil {
		t.Fatal(err)
	}
	covered := make([]string, 0, len(cases))
	for _, c := range cases {
		covered = append(covered, c.name)
	}
	sort.Strings(covered)
	if !reflect.DeepEqual(covered, inventory.SchemaTypes) {
		t.Fatalf("upstream type coverage mismatch: %v", covered)
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			source := `exports.schema=` + c.schema + `;`
			if c.exportedInput != "" {
				source += `exports.input=` + c.exportedInput + `;`
			}
			v := runtimeValidator(t, source)
			input := c.input
			if c.exportedInput != "" {
				var err error
				input, err = v.ExportValue("input")
				if err != nil {
					t.Fatal(err)
				}
			}
			var got any
			var err error
			if c.async {
				got, err = v.ParseAsync(input)
			} else {
				got, err = v.Parse(input)
			}
			if c.failure {
				if err == nil {
					t.Fatal("invalid input accepted")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if c.name == "nan" {
				if !math.IsNaN(got.(float64)) {
					t.Fatal("NaN lost")
				}
				return
			}
			if c.name == "file" || c.name == "function" || c.name == "symbol" {
				if _, ok := got.(*zod2go.RuntimeValue); !ok {
					t.Fatalf("missing runtime handle: %T", got)
				}
				return
			}
			if !reflect.DeepEqual(got, c.want) {
				t.Fatalf("got %#v; want %#v", got, c.want)
			}
		})
	}
}
