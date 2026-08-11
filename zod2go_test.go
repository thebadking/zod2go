package zod2go_test

import (
	"encoding/json"
	"path/filepath"
	"testing"

	"github.com/thebadking/zod2go"
)

func TestUserSchemaValid(t *testing.T) {
	doc := loadFixture(t, "user.schema.json")
	v := zod2go.NewValidator(doc)

	input := map[string]any{
		"name":  "Jane Doe",
		"email": "jane@example.com",
		"role":  "admin",
		"tags":  []any{"go", "zod"},
	}

	out, err := v.Parse(input)
	if err != nil {
		t.Fatalf("expected valid user, got %v", err)
	}
	m := out.(map[string]any)
	if m["name"] != "Jane Doe" {
		t.Fatalf("unexpected name: %v", m["name"])
	}
}

func TestUserSchemaInvalidEmail(t *testing.T) {
	doc := loadFixture(t, "user.schema.json")
	v := zod2go.NewValidator(doc)

	_, err := v.Parse(map[string]any{
		"name":  "Jane",
		"email": "not-an-email",
	})
	if err == nil {
		t.Fatal("expected validation error")
	}
	zerr, ok := err.(*zod2go.Error)
	if !ok {
		t.Fatalf("expected *zod2go.Error, got %T", err)
	}
	if len(zerr.Issues) == 0 {
		t.Fatal("expected issues")
	}
}

func TestUserSchemaMissingRequired(t *testing.T) {
	doc := loadFixture(t, "user.schema.json")
	v := zod2go.NewValidator(doc)

	res := v.SafeParse(map[string]any{"email": "a@b.com"})
	if res.Success {
		t.Fatal("expected failure for missing name")
	}
}

func TestParseJSON(t *testing.T) {
	doc := loadFixture(t, "user.schema.json")
	v := zod2go.NewValidator(doc)

	raw := `{"name":"Bob","email":"bob@example.com"}`
	out, err := v.ParseJSON([]byte(raw))
	if err != nil {
		t.Fatal(err)
	}
	m := out.(map[string]any)
	if m["role"] != "user" {
		t.Fatalf("expected default role user, got %v", m["role"])
	}
}

func TestDocumentRoundTrip(t *testing.T) {
	doc := loadFixture(t, "user.schema.json")
	data, err := json.Marshal(doc)
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := zod2go.ParseDocument(data)
	if err != nil {
		t.Fatal(err)
	}
	if parsed.Schema.Type != "object" {
		t.Fatalf("expected object schema, got %s", parsed.Schema.Type)
	}
}

func loadFixture(t *testing.T, name string) *zod2go.Document {
	t.Helper()
	path := filepath.Join("testdata", name)
	doc, err := zod2go.LoadDocument(path)
	if err != nil {
		t.Fatal(err)
	}
	return doc
}
