package zod2go

import (
	"encoding/json"
	"fmt"
	"strings"
)

// Issue mirrors Zod v4 $ZodIssueBase and common subtypes.
type Issue struct {
	Code      string   `json:"code"`
	Message   string   `json:"message"`
	Path      []any    `json:"path"`
	Input     any      `json:"input,omitempty"`
	Expected  string   `json:"expected,omitempty"`
	Origin    string   `json:"origin,omitempty"`
	Minimum   any      `json:"minimum,omitempty"`
	Maximum   any      `json:"maximum,omitempty"`
	Inclusive bool     `json:"inclusive,omitempty"`
	Exact     bool     `json:"exact,omitempty"`
	Format    string   `json:"format,omitempty"`
	Pattern   string   `json:"pattern,omitempty"`
	Divisor   any      `json:"divisor,omitempty"`
	Keys      []string `json:"keys,omitempty"`
}

// Error is a Zod-compatible validation error (aggregate of issues).
type Error struct {
	Issues []Issue `json:"issues"`
}

func (e *Error) Error() string {
	if len(e.Issues) == 0 {
		return "validation failed"
	}
	var b strings.Builder
	for i, iss := range e.Issues {
		if i > 0 {
			b.WriteString("; ")
		}
		b.WriteString(formatIssue(iss))
	}
	return b.String()
}

func formatIssue(iss Issue) string {
	path := formatPath(iss.Path)
	if iss.Message != "" {
		if path != "" {
			return path + ": " + iss.Message
		}
		return iss.Message
	}
	switch iss.Code {
	case "invalid_type":
		return fmt.Sprintf("%s: expected %s", path, iss.Expected)
	case "too_small":
		return fmt.Sprintf("%s: too small", path)
	case "too_big":
		return fmt.Sprintf("%s: too big", path)
	case "invalid_format":
		return fmt.Sprintf("%s: invalid %s", path, iss.Format)
	default:
		return fmt.Sprintf("%s: %s", path, iss.Code)
	}
}

func formatPath(path []any) string {
	if len(path) == 0 {
		return ""
	}
	var b strings.Builder
	for i, seg := range path {
		switch v := seg.(type) {
		case string:
			if i > 0 {
				b.WriteByte('.')
			}
			b.WriteString(v)
		case float64:
			b.WriteByte('[')
			b.WriteString(fmt.Sprintf("%g", v))
			b.WriteByte(']')
		case int:
			b.WriteByte('[')
			b.WriteString(fmt.Sprintf("%d", v))
			b.WriteByte(']')
		default:
			b.WriteString(fmt.Sprintf("[%v]", v))
		}
	}
	return b.String()
}

// JSON returns a JSON representation matching Zod's error shape.
func (e *Error) JSON() string {
	out, _ := json.Marshal(map[string]any{"issues": e.Issues})
	return string(out)
}

func newIssue(code, message string, path []any, input any) Issue {
	return Issue{Code: code, Message: message, Path: append([]any{}, path...), Input: input}
}

func prefixPath(path []any, prefix any) []any {
	out := make([]any, 0, len(path)+1)
	out = append(out, path...)
	out = append(out, prefix)
	return out
}

func appendIssues(dst []Issue, src []Issue, prefix any) []Issue {
	for _, iss := range src {
		iss.Path = prefixPath(iss.Path, prefix)
		dst = append(dst, iss)
	}
	return dst
}

// SafeResult holds either parsed output or a validation error.
type SafeResult struct {
	Success bool
	Data    any
	Error   *Error
}

// Validator validates values against an exported Zod schema.
type Validator struct {
	schema Schema
	defs   map[string]Schema
	typed  bool
}

// NewValidator creates a validator from a schema document.
func NewValidator(doc *Document) *Validator {
	return &Validator{schema: doc.Schema, defs: doc.Definitions, typed: doc.Zod2go.Encoding == "tagged"}
}

// NewTypedValidator uses native Go value types corresponding to JavaScript values.
// JSON wire compatibility remains available through NewValidator's default mode.
func NewTypedValidator(doc *Document) *Validator { v := NewValidator(doc); v.typed = true; return v }

// NewValidatorFromSchema creates a validator from a schema node.
func NewValidatorFromSchema(schema Schema) *Validator {
	return &Validator{schema: schema}
}

// Parse validates input and returns the coerced/parsed value or an error.
func (v *Validator) Parse(input any) (any, error) {
	res := v.safeParse(input, v.schema, nil)
	if res.Success {
		return res.Data, nil
	}
	return nil, res.Error
}

// SafeParse validates without throwing; returns a SafeResult.
func (v *Validator) SafeParse(input any) SafeResult {
	return v.safeParse(input, v.schema, nil)
}

// ParseJSON validates JSON bytes.
func (v *Validator) ParseJSON(data []byte) (any, error) {
	var input any
	if v.typed {
		value, err := DecodeTaggedValue(data)
		if err != nil {
			return nil, err
		}
		return v.Parse(value)
	}
	if err := json.Unmarshal(data, &input); err != nil {
		return nil, fmt.Errorf("zod2go: invalid json: %w", err)
	}
	return v.Parse(input)
}

func (v *Validator) safeParse(input any, schema Schema, path []any) SafeResult {
	if path == nil {
		path = []any{}
	}
	issues, out := v.run(input, schema, path)
	if len(issues) > 0 {
		return SafeResult{Success: false, Error: &Error{Issues: issues}}
	}
	return SafeResult{Success: true, Data: out}
}

// Undefined represents an absent value, distinct from JSON null (nil).
// Use it for top-level optional/default parsing; object fields use it internally.
var Undefined = undefinedValue{}

type undefinedValue struct{}

func isUndefined(input any) bool {
	_, ok := input.(undefinedValue)
	return ok
}

func (v *Validator) run(input any, schema Schema, path []any) ([]Issue, any) {
	seen := make(map[string]bool)
	for schema.Ref != "" || schema.Type == "$ref" {
		if schema.Ref == "" || seen[schema.Ref] {
			return []Issue{newIssue("custom", "cyclic or empty schema reference", path, input)}, nil
		}
		seen[schema.Ref] = true
		resolved, ok := v.defs[schema.Ref]
		if !ok {
			return []Issue{newIssue("custom", "unresolved schema reference: "+schema.Ref, path, input)}, nil
		}
		schema = resolved
	}
	for _, check := range schema.Checks {
		if !nativeCheckSupported(schema.Type, check.Check) {
			return []Issue{newIssue("custom", "check requires module-backed runtime: "+check.Check, path, input)}, nil
		}
	}
	switch schema.Type {
	case "runtime":
		return []Issue{newIssue("custom", "runtime-backed schema requires NewRuntimeValidator or CLI -runtime", path, input)}, nil
	case "string":
		return v.runString(input, schema, path)
	case "number", "int":
		return v.runNumber(input, schema, path)
	case "bigint":
		return v.runBigInt(input, schema, path)
	case "boolean":
		return v.runBoolean(input, schema, path)
	case "date":
		return v.runDate(input, schema, path)
	case "null":
		return v.runNull(input, path)
	case "undefined", "void":
		return v.runUndefined(input, path)
	case "set":
		return v.runSet(input, schema, path)
	case "array":
		return v.runArray(input, schema, path)
	case "tuple":
		return v.runTuple(input, schema, path)
	case "map":
		if v.typed {
			return v.runTypedMap(input, schema, path)
		}
		return v.runRecord(input, schema, path)
	case "record":
		return v.runRecord(input, schema, path)
	case "pipe":
		if schema.Left == nil || schema.Right == nil {
			return []Issue{newIssue("custom", "missing pipe input/output", path, input)}, nil
		}
		issues, parsed := v.run(input, *schema.Left, path)
		if len(issues) > 0 {
			return issues, nil
		}
		return v.run(parsed, *schema.Right, path)
	case "intersection":
		return v.runIntersection(input, schema, path)
	case "object":
		return v.runObject(input, schema, path)
	case "enum":
		return v.runEnum(input, schema, path)
	case "literal":
		return v.runLiteral(input, schema, path)
	case "union":
		return v.runUnion(input, schema, path)
	case "optional":
		return v.runOptional(input, schema, path)
	case "nullable":
		return v.runNullable(input, schema, path)
	case "readonly":
		if schema.Inner != nil {
			return v.run(input, *schema.Inner, path)
		}
	case "nonoptional":
		if !isUndefined(input) && schema.Inner != nil {
			return v.run(input, *schema.Inner, path)
		}
		return []Issue{newIssue("invalid_type", "Required", path, input)}, nil
	case "prefault":
		if isUndefined(input) {
			input = schema.defaultValue()
		}
		if schema.Inner != nil {
			return v.run(input, *schema.Inner, path)
		}
	case "default":
		return v.runDefault(input, schema, path)
	case "any", "unknown":
		return nil, input
	case "never":
		return []Issue{newIssue("invalid_type", "Invalid input", path, input)}, nil
	default:
		return []Issue{newIssue("custom", fmt.Sprintf("unsupported schema type %q", schema.Type), path, input)}, input
	}
	return []Issue{newIssue("custom", "missing inner schema", path, input)}, nil
}
