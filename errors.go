package zod2go

import (
	"encoding/json"
	"fmt"
	"strings"
)

// Issue mirrors Zod v4 $ZodIssueBase and common subtypes.
type Issue struct {
	Code     string `json:"code"`
	Message  string `json:"message"`
	Path     []any  `json:"path"`
	Input    any    `json:"input,omitempty"`
	Expected string `json:"expected,omitempty"`
	Origin   string `json:"origin,omitempty"`
	Minimum  any    `json:"minimum,omitempty"`
	Maximum  any    `json:"maximum,omitempty"`
	Inclusive bool  `json:"inclusive,omitempty"`
	Exact    bool   `json:"exact,omitempty"`
	Format   string `json:"format,omitempty"`
	Pattern  string `json:"pattern,omitempty"`
	Divisor  any    `json:"divisor,omitempty"`
	Keys     []string `json:"keys,omitempty"`
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
	return Issue{Code: code, Message: message, Path: append([]any(nil), path...), Input: input}
}

func prefixPath(path []any, prefix any) []any {
	out := make([]any, 0, len(path)+1)
	out = append(out, prefix)
	out = append(out, path...)
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
}

// NewValidator creates a validator from a schema document.
func NewValidator(doc *Document) *Validator {
	return &Validator{schema: doc.Schema, defs: doc.Definitions}
}

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
	if err := json.Unmarshal(data, &input); err != nil {
		return nil, fmt.Errorf("zod2go: invalid json: %w", err)
	}
	return v.Parse(input)
}

func (v *Validator) safeParse(input any, schema Schema, path []any) SafeResult {
	issues, out := v.run(input, schema, path)
	if len(issues) > 0 {
		return SafeResult{Success: false, Error: &Error{Issues: issues}}
	}
	return SafeResult{Success: true, Data: out}
}

func (v *Validator) resolveRef(schema Schema) (Schema, bool) {
	if schema.Ref == "" {
		return schema, false
	}
	if v.defs != nil {
		if def, ok := v.defs[schema.Ref]; ok {
			return def, true
		}
	}
	return schema, false
}

func (v *Validator) run(input any, schema Schema, path []any) ([]Issue, any) {
	if resolved, ok := v.resolveRef(schema); ok {
		schema = resolved
	}
	switch schema.Type {
	case "string":
		return v.runString(input, schema, path)
	case "number", "int":
		return v.runNumber(input, schema, path)
	case "boolean":
		return v.runBoolean(input, schema, path)
	case "date":
		return v.runDate(input, schema, path)
	case "null":
		return v.runNull(input, path)
	case "undefined", "void":
		return v.runUndefined(input, path)
	case "array", "set":
		return v.runArray(input, schema, path)
	case "tuple":
		return v.runTuple(input, schema, path)
	case "record", "map":
		return v.runRecord(input, schema, path)
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
	case "default":
		return v.runDefault(input, schema, path)
	case "any", "unknown":
		return nil, input
	case "never":
		return []Issue{newIssue("invalid_type", "Invalid input", path, input)}, nil
	default:
		return []Issue{newIssue("custom", fmt.Sprintf("unsupported schema type %q", schema.Type), path, input)}, input
	}
}
