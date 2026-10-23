package zod2go_test

import (
	"testing"

	"github.com/thebadking/zod2go"
)

// ─── Date ────────────────────────────────────────────────────────────────────

func TestDateValid(t *testing.T) {
	doc := mustParseInlineSchema(t, `{
		"$zod2go": {"version": 1, "zodVersion": "4.x"},
		"schema": {"type": "date"}
	}`)
	v := zod2go.NewValidator(doc)
	for _, s := range []string{
		"2024-01-15T10:30:00.000Z",
		"2024-01-15T10:30:00Z",
		"2024-01-15",
	} {
		if _, err := v.Parse(s); err != nil {
			t.Errorf("expected valid date for %q, got %v", s, err)
		}
	}
}

func TestDateInvalid(t *testing.T) {
	doc := mustParseInlineSchema(t, `{
		"$zod2go": {"version": 1, "zodVersion": "4.x"},
		"schema": {"type": "date"}
	}`)
	v := zod2go.NewValidator(doc)
	for _, input := range []any{"not-a-date", "2024-99-99", 42} {
		if _, err := v.Parse(input); err == nil {
			t.Errorf("expected error for %v", input)
		}
	}
}

func TestDateChecks(t *testing.T) {
	doc := mustParseInlineSchema(t, `{
		"$zod2go": {"version": 1, "zodVersion": "4.x"},
		"schema": {
			"type": "date",
			"checks": [{"check": "greater_than", "value": "2020-01-01T00:00:00.000Z", "inclusive": true}]
		}
	}`)
	v := zod2go.NewValidator(doc)
	if _, err := v.Parse("2024-06-01T00:00:00Z"); err != nil {
		t.Fatalf("expected valid future date, got %v", err)
	}
	if _, err := v.Parse("2019-01-01T00:00:00Z"); err == nil {
		t.Fatal("expected error for past date")
	}
}

// ─── Lazy / $ref ──────────────────────────────────────────────────────────────

func TestLazyRecursiveValid(t *testing.T) {
	doc := mustParseInlineSchema(t, `{
		"$zod2go": {"version": 1, "zodVersion": "4.x"},
		"definitions": {
			"def_0": {
				"type": "object",
				"shape": {
					"val":      {"type": "number"},
					"children": {"type": "array", "element": {"type": "$ref", "$ref": "def_0"}}
				}
			}
		},
		"schema": {"type": "$ref", "$ref": "def_0"}
	}`)
	v := zod2go.NewValidator(doc)

	_, err := v.Parse(map[string]any{
		"val": float64(1),
		"children": []any{
			map[string]any{"val": float64(2), "children": []any{}},
		},
	})
	if err != nil {
		t.Fatalf("expected valid recursive input, got %v", err)
	}
}

func TestLazyRecursiveInvalid(t *testing.T) {
	doc := mustParseInlineSchema(t, `{
		"$zod2go": {"version": 1, "zodVersion": "4.x"},
		"definitions": {
			"def_0": {
				"type": "object",
				"shape": {
					"val":      {"type": "number"},
					"children": {"type": "array", "element": {"type": "$ref", "$ref": "def_0"}}
				}
			}
		},
		"schema": {"type": "$ref", "$ref": "def_0"}
	}`)
	v := zod2go.NewValidator(doc)

	_, err := v.Parse(map[string]any{
		"val": "not-a-number",
		"children": []any{},
	})
	if err == nil {
		t.Fatal("expected error for invalid val type")
	}
}

// ─── Set ─────────────────────────────────────────────────────────────────────

func TestSetValid(t *testing.T) {
	doc := mustParseInlineSchema(t, `{
		"$zod2go": {"version": 1, "zodVersion": "4.x"},
		"schema": {"type": "set", "element": {"type": "string"}}
	}`)
	v := zod2go.NewValidator(doc)
	if _, err := v.Parse([]any{"a", "b", "c"}); err != nil {
		t.Fatalf("expected valid set, got %v", err)
	}
}

func TestSetInvalidElement(t *testing.T) {
	doc := mustParseInlineSchema(t, `{
		"$zod2go": {"version": 1, "zodVersion": "4.x"},
		"schema": {"type": "set", "element": {"type": "number"}}
	}`)
	v := zod2go.NewValidator(doc)
	if _, err := v.Parse([]any{1, "not-a-number"}); err == nil {
		t.Fatal("expected error for invalid element type")
	}
}

// ─── Map ─────────────────────────────────────────────────────────────────────

func TestMapValid(t *testing.T) {
	doc := mustParseInlineSchema(t, `{
		"$zod2go": {"version": 1, "zodVersion": "4.x"},
		"schema": {"type": "map", "keyType": {"type": "string"}, "valueType": {"type": "number"}}
	}`)
	v := zod2go.NewValidator(doc)
	if _, err := v.Parse(map[string]any{"a": float64(1), "b": float64(2)}); err != nil {
		t.Fatalf("expected valid map, got %v", err)
	}
}

// ─── Literal (updated Zod v4 single-value) ───────────────────────────────────

func TestLiteralNumber(t *testing.T) {
	doc := mustParseInlineSchema(t, `{
		"$zod2go": {"version": 1, "zodVersion": "4.x"},
		"schema": {"type": "literal", "value": 42}
	}`)
	v := zod2go.NewValidator(doc)
	if _, err := v.Parse(float64(42)); err != nil {
		t.Fatalf("expected literal match, got %v", err)
	}
	if _, err := v.Parse(float64(43)); err == nil {
		t.Fatal("expected literal mismatch error")
	}
}

// ─── helpers ─────────────────────────────────────────────────────────────────

func mustParseInlineSchema(t *testing.T, raw string) *zod2go.Document {
	t.Helper()
	doc, err := zod2go.ParseDocument([]byte(raw))
	if err != nil {
		t.Fatalf("ParseDocument: %v", err)
	}
	return doc
}
