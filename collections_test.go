package zod2go_test

import (
	"reflect"
	"testing"

	"github.com/thebadking/zod2go"
)

func TestTypedCollectionsPreserveTypes(t *testing.T) {
	setDoc := mustParseInlineSchema(t, `{"$zod2go":{"version":1,"encoding":"tagged"},"schema":{"type":"set","element":{"type":"string"},"checks":[{"check":"size_equals","value":2}]}}`)
	set := zod2go.NewTypedValidator(setDoc)
	if set.SafeParse([]any{"a", "b"}).Success {
		t.Fatal("typed set accepted array")
	}
	result := set.SafeParse(zod2go.RuntimeSet{"a", "a", "b"})
	if !result.Success || !reflect.DeepEqual(result.Data, zod2go.RuntimeSet{"a", "b"}) {
		t.Fatalf("set semantics: %+v", result)
	}
	mapDoc := mustParseInlineSchema(t, `{"$zod2go":{"version":1,"encoding":"tagged"},"schema":{"type":"map","keyType":{"type":"number"},"valueType":{"type":"string"}}}`)
	m := zod2go.NewTypedValidator(mapDoc)
	if m.SafeParse(map[string]any{"1": "a"}).Success {
		t.Fatal("typed map accepted object")
	}
	result = m.SafeParse(zod2go.RuntimeMap{{Key: float64(1), Value: "first"}, {Key: float64(1), Value: "second"}})
	if !result.Success || !reflect.DeepEqual(result.Data, zod2go.RuntimeMap{{Key: float64(1), Value: "second"}}) {
		t.Fatalf("map semantics: %+v", result)
	}
}

func TestTypedDefaultsAreFresh(t *testing.T) {
	doc := mustParseInlineSchema(t, `{"$zod2go":{"version":1,"encoding":"tagged"},"schema":{"type":"default","inner":{"type":"any"},"defaultEncoded":["id",{"id":1,"value":["object",[["x",["number",1]]]]}]}}`)
	v := zod2go.NewTypedValidator(doc)
	first := v.SafeParse(zod2go.Undefined)
	if !first.Success {
		t.Fatal(first)
	}
	first.Data.(map[string]any)["x"] = float64(2)
	second := v.SafeParse(zod2go.Undefined)
	if second.Data.(map[string]any)["x"] != float64(1) {
		t.Fatal("default leaked mutation", second)
	}
	date := mustParseInlineSchema(t, `{"$zod2go":{"version":1,"encoding":"tagged"},"schema":{"type":"prefault","inner":{"type":"date"},"defaultEncoded":["date","2026-01-01T00:00:00.000Z"]}}`)
	if result := zod2go.NewTypedValidator(date).SafeParse(zod2go.Undefined); !result.Success {
		t.Fatal(result)
	}
}
