package zod2go_test

import (
	"reflect"
	"testing"

	"github.com/thebadking/zod2go"
)

func TestReferenceChainsAndCycles(t *testing.T) {
	for _, tc := range []struct {
		name        string
		definitions string
		success     bool
	}{
		{"chain", `{"a":{"type":"$ref","$ref":"b"},"b":{"type":"number"}}`, true},
		{"cycle", `{"a":{"type":"$ref","$ref":"b"},"b":{"type":"$ref","$ref":"a"}}`, false},
		{"missing", `{}`, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			doc := mustParseInlineSchema(t, `{"$zod2go":{"version":1},"definitions":`+tc.definitions+`,"schema":{"type":"$ref","$ref":"a"}}`)
			if got := zod2go.NewValidator(doc).SafeParse(42); got.Success != tc.success {
				t.Fatalf("unexpected result: %+v", got)
			}
		})
	}
}

func TestNestedIssuePath(t *testing.T) {
	doc := mustParseInlineSchema(t, `{"$zod2go":{"version":1},"schema":{"type":"object","shape":{"children":{"type":"array","element":{"type":"object","shape":{"value":{"type":"number"}}}}}}}`)
	result := zod2go.NewValidator(doc).SafeParse(map[string]any{"children": []any{map[string]any{"value": "bad"}}})
	if result.Success || !reflect.DeepEqual(result.Error.Issues[0].Path, []any{"children", float64(0), "value"}) {
		t.Fatalf("unexpected result: %+v", result)
	}
}

func TestUndefinedDistinctFromNull(t *testing.T) {
	for _, tc := range []struct {
		schema  string
		input   any
		success bool
		data    any
	}{
		{`{"type":"optional","inner":{"type":"string"}}`, nil, false, nil},
		{`{"type":"optional","inner":{"type":"string"}}`, zod2go.Undefined, true, zod2go.Undefined},
		{`{"type":"default","inner":{"type":"string"},"default":"x"}`, zod2go.Undefined, true, "x"},
		{`{"type":"default","inner":{"type":"string"},"default":"x"}`, nil, false, nil},
	} {
		doc := mustParseInlineSchema(t, `{"$zod2go":{"version":1},"schema":`+tc.schema+`}`)
		result := zod2go.NewValidator(doc).SafeParse(tc.input)
		if result.Success != tc.success || (tc.success && !reflect.DeepEqual(result.Data, tc.data)) {
			t.Fatalf("schema %s: %+v", tc.schema, result)
		}
	}
}

func TestDeepRecursiveValidation(t *testing.T) {
	doc := mustParseInlineSchema(t, `{"$zod2go":{"version":1},"definitions":{"node":{"type":"object","shape":{"value":{"type":"number"},"children":{"type":"array","element":{"type":"$ref","$ref":"node"}}}}},"schema":{"type":"$ref","$ref":"node"}}`)
	leaf := map[string]any{"value": 1, "children": []any{}}
	var input any = leaf
	for i := 0; i < 256; i++ {
		input = map[string]any{"value": i, "children": []any{input}}
	}
	v := zod2go.NewValidator(doc)
	if result := v.SafeParse(input); !result.Success {
		t.Fatal(result.Error)
	}
	leaf["value"] = "bad"
	result := v.SafeParse(input)
	if result.Success || len(result.Error.Issues[0].Path) != 513 {
		t.Fatalf("unexpected deep validation result: %+v", result)
	}
}

func TestDocumentEnvelopeValidation(t *testing.T) {
	for _, raw := range []string{`{}`, `{"$zod2go":{"version":2},"schema":{"type":"string"}}`, `{"$zod2go":{"version":1}}`} {
		if _, err := zod2go.ParseDocument([]byte(raw)); err == nil {
			t.Fatalf("accepted invalid document: %s", raw)
		}
	}
}

func TestInvalidRegexFailsValidation(t *testing.T) {
	schema := zod2go.Schema{Type: "string", Pattern: `(?=a)a`}
	if result := zod2go.NewValidatorFromSchema(schema).SafeParse("b"); result.Success {
		t.Fatal("unsupported RE2 pattern must fail")
	}
}

func TestExactOptionalTupleTail(t *testing.T) {
	doc := mustParseInlineSchema(t, `{"$zod2go":{"version":1,"encoding":"tagged"},"schema":{"type":"tuple","items":[{"type":"string"},{"type":"optional","exactOptional":true,"optionalInput":true,"optionalOutput":true,"inner":{"type":"string"}},{"type":"optional","optionalInput":true,"optionalOutput":true,"inner":{"type":"string"}}]}}`)
	v := zod2go.NewTypedValidator(doc)
	result := v.SafeParse([]any{"alpha"})
	if !result.Success || !reflect.DeepEqual(result.Data, []any{"alpha"}) {
		t.Fatalf("optional tail not truncated: %+v", result)
	}
	if v.SafeParse([]any{"alpha", zod2go.Undefined}).Success {
		t.Fatal("exact optional accepted explicit undefined")
	}
}
