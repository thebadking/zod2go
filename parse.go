package zod2go

import (
	"encoding/json"
	"fmt"
	"os"
)

// Document is the root envelope written by the TypeScript exporter.
type Document struct {
	Zod2go struct {
		Version    int    `json:"version"`
		ZodVersion string `json:"zodVersion"`
	} `json:"$zod2go"`
	Schema Schema `json:"schema"`
}

// Schema is the portable Zod schema intermediate representation.
type Schema struct {
	Type      string            `json:"type"`
	Coerce    bool              `json:"coerce,omitempty"`
	Checks    []Check           `json:"checks,omitempty"`
	Element   *Schema           `json:"element,omitempty"`
	Items     []Schema          `json:"items,omitempty"`
	Rest      *Schema           `json:"rest,omitempty"`
	Shape     map[string]Schema `json:"shape,omitempty"`
	KeyType   *Schema           `json:"keyType,omitempty"`
	ValueType *Schema           `json:"valueType,omitempty"`
	Left      *Schema           `json:"left,omitempty"`
	Right     *Schema           `json:"right,omitempty"`
	Options   []Schema          `json:"options,omitempty"`
	Values    []any             `json:"values,omitempty"`
	Value     any               `json:"value,omitempty"`
	Inner     *Schema           `json:"inner,omitempty"`
	Catchall  *Schema           `json:"catchall,omitempty"`
	Default   any               `json:"default,omitempty"`
	Format    string            `json:"format,omitempty"`
	Pattern   string            `json:"pattern,omitempty"`
	UUIDVer   *int              `json:"uuidVersion,omitempty"`
	Strict    bool              `json:"strict,omitempty"`
}

// Check mirrors Zod v4 $ZodCheckDef entries.
type Check struct {
	Check     string  `json:"check"`
	Value     any     `json:"value,omitempty"`
	Inclusive bool    `json:"inclusive,omitempty"`
	Format    string  `json:"format,omitempty"`
	Pattern   string  `json:"pattern,omitempty"`
	Minimum   any     `json:"minimum,omitempty"`
	Maximum   any     `json:"maximum,omitempty"`
	Exact     bool    `json:"exact,omitempty"`
}

// LoadDocument reads a zod2go schema document from a JSON file.
func LoadDocument(path string) (*Document, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return ParseDocument(data)
}

// ParseDocument unmarshals a zod2go schema document from JSON bytes.
func ParseDocument(data []byte) (*Document, error) {
	var doc Document
	if err := json.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("zod2go: parse document: %w", err)
	}
	if doc.Zod2go.Version == 0 {
		return nil, fmt.Errorf("zod2go: missing $zod2go.version")
	}
	return &doc, nil
}

// MustLoadDocument loads a document or panics.
func MustLoadDocument(path string) *Document {
	doc, err := LoadDocument(path)
	if err != nil {
		panic(err)
	}
	return doc
}
