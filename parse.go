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
		Encoding   string `json:"encoding,omitempty"`
		ZodVersion string `json:"zodVersion"`
	} `json:"$zod2go"`
	Runtime     *RuntimeModule    `json:"runtime,omitempty"`
	Definitions map[string]Schema `json:"definitions,omitempty"`
	Schema      Schema            `json:"schema"`
}

// Schema is the portable Zod schema intermediate representation.
type Schema struct {
	Type           string            `json:"type"`
	Ref            string            `json:"$ref,omitempty"`
	OptionalOutput bool              `json:"optionalOutput,omitempty"`
	ExactOptional  bool              `json:"exactOptional,omitempty"`
	OptionalInput  bool              `json:"optionalInput,omitempty"`
	Coerce         bool              `json:"coerce,omitempty"`
	Checks         []Check           `json:"checks,omitempty"`
	Element        *Schema           `json:"element,omitempty"`
	Items          []Schema          `json:"items,omitempty"`
	Rest           *Schema           `json:"rest,omitempty"`
	Shape          map[string]Schema `json:"shape,omitempty"`
	KeyType        *Schema           `json:"keyType,omitempty"`
	ValueType      *Schema           `json:"valueType,omitempty"`
	Left           *Schema           `json:"left,omitempty"`
	Right          *Schema           `json:"right,omitempty"`
	Options        []Schema          `json:"options,omitempty"`
	Values         []any             `json:"values,omitempty"`
	ValueKind      string            `json:"valueKind,omitempty"`
	Value          any               `json:"value,omitempty"`
	Inner          *Schema           `json:"inner,omitempty"`
	Catchall       *Schema           `json:"catchall,omitempty"`
	DefaultEncoded json.RawMessage   `json:"defaultEncoded,omitempty"`
	Default        any               `json:"default,omitempty"`
	Format         string            `json:"format,omitempty"`
	Alg            string            `json:"alg,omitempty"`
	Hostname       string            `json:"hostname,omitempty"`
	Protocol       string            `json:"protocol,omitempty"`
	Normalize      bool              `json:"normalize,omitempty"`
	Pattern        string            `json:"pattern,omitempty"`
	UUIDVer        *int              `json:"uuidVersion,omitempty"`
	KeyValues      []any             `json:"keyValues,omitempty"`
	PartialKeys    bool              `json:"partialKeys,omitempty"`
	Mode           string            `json:"mode,omitempty"`
	Exclusive      bool              `json:"exclusive,omitempty"`
	Strict         bool              `json:"strict,omitempty"`
}

// Check mirrors Zod v4 $ZodCheckDef entries.
type Check struct {
	Check     string `json:"check"`
	Value     any    `json:"value,omitempty"`
	Inclusive bool   `json:"inclusive,omitempty"`
	Format    string `json:"format,omitempty"`
	Alg       string `json:"alg,omitempty"`
	Hostname  string `json:"hostname,omitempty"`
	Protocol  string `json:"protocol,omitempty"`
	Normalize bool   `json:"normalize,omitempty"`
	Pattern   string `json:"pattern,omitempty"`
	Minimum   any    `json:"minimum,omitempty"`
	Maximum   any    `json:"maximum,omitempty"`
	Position  int    `json:"position,omitempty"`
	Exact     bool   `json:"exact,omitempty"`
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
	if doc.Zod2go.Version != Version {
		return nil, fmt.Errorf("zod2go: unsupported or missing $zod2go.version: %d", doc.Zod2go.Version)
	}
	if doc.Zod2go.Encoding != "" && doc.Zod2go.Encoding != "json" && doc.Zod2go.Encoding != "tagged" {
		return nil, fmt.Errorf("zod2go: unsupported encoding %q", doc.Zod2go.Encoding)
	}
	if doc.Schema.Type == "" && doc.Schema.Ref == "" {
		return nil, fmt.Errorf("zod2go: missing schema")
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

// RuntimeModule contains the original bundled schema module for exact Zod execution.
// It is executed only by the explicit NewRuntimeValidator API.
type RuntimeModule struct {
	CoreModule string `json:"coreModule,omitempty"`
	ModulePath string `json:"modulePath,omitempty"`
	Loader     string `json:"loader,omitempty"`
	Format     string `json:"format,omitempty"`
	Bundle     string `json:"bundle"`
	ExportName string `json:"exportName"`
}
