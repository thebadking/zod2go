package zod2go_test

import (
	"testing"

	"github.com/thebadking/zod2go"
)

func TestExtendedSchemaValid(t *testing.T) {
	doc := loadFixture(t, "extended.schema.json")
	v := zod2go.NewValidator(doc)

	out, err := v.Parse(map[string]any{
		"pair":   []any{"x", float64(1)},
		"scores": map[string]any{"math": float64(95)},
		"merged": map[string]any{"a": "ok", "b": float64(2)},
		"tags":   []any{"go"},
	})
	if err != nil {
		t.Fatalf("expected valid input, got %v", err)
	}
	m := out.(map[string]any)
	if m["pair"].([]any)[0] != "x" {
		t.Fatalf("unexpected pair: %v", m["pair"])
	}
}

func TestExtendedSchemaTupleInvalid(t *testing.T) {
	doc := loadFixture(t, "extended.schema.json")
	v := zod2go.NewValidator(doc)

	_, err := v.Parse(map[string]any{
		"pair":   []any{"x", "not-a-number"},
		"scores": map[string]any{},
		"merged": map[string]any{"a": "ok", "b": float64(2)},
		"tags":   []any{"go"},
	})
	if err == nil {
		t.Fatal("expected tuple type error")
	}
}

func TestExtendedSchemaArrayMinLength(t *testing.T) {
	doc := loadFixture(t, "extended.schema.json")
	v := zod2go.NewValidator(doc)

	_, err := v.Parse(map[string]any{
		"pair":   []any{"x", float64(1)},
		"scores": map[string]any{},
		"merged": map[string]any{"a": "ok", "b": float64(2)},
		"tags":   []any{},
	})
	if err == nil {
		t.Fatal("expected array min_length error for empty tags")
	}
}

func TestExtendedSchemaStrictKeys(t *testing.T) {
	doc := loadFixture(t, "extended.schema.json")
	v := zod2go.NewValidator(doc)

	_, err := v.Parse(map[string]any{
		"pair":      []any{"x", float64(1)},
		"scores":    map[string]any{},
		"merged":    map[string]any{"a": "ok", "b": float64(2)},
		"tags":      []any{"go"},
		"extra_key": true,
	})
	if err == nil {
		t.Fatal("expected unrecognized_keys error")
	}
}

func TestUserSchemaTagsMinLength(t *testing.T) {
	doc := loadFixture(t, "user.schema.json")
	v := zod2go.NewValidator(doc)

	_, err := v.Parse(map[string]any{
		"name":  "Jane",
		"email": "jane@example.com",
		"tags":  []any{},
	})
	if err == nil {
		t.Fatal("expected min_length error for empty tags array")
	}
}
