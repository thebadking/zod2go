package zod2go

import (
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
	"time"
)

func (v *Validator) runString(input any, schema Schema, path []any) ([]Issue, any) {
	val := input
	if schema.Coerce {
		val = fmt.Sprint(input)
	}
	s, ok := val.(string)
	if !ok {
		return []Issue{{
			Code: "invalid_type", Expected: "string", Path: path, Input: input,
			Message: "Invalid input: expected string, received " + typeName(input),
		}}, nil
	}

	if schema.Format != "" {
		if re, ok := formatRegex(schema.Format, schema.UUIDVer); ok {
			if !re.MatchString(s) {
				return []Issue{{
					Code: "invalid_format", Format: schema.Format, Path: path, Input: s,
					Message: fmt.Sprintf("Invalid %s", schema.Format),
				}}, nil
			}
		}
	}
	if schema.Pattern != "" {
		re, err := regexp.Compile(schema.Pattern)
		if err == nil && !re.MatchString(s) {
			return []Issue{{
				Code: "invalid_format", Format: "regex", Pattern: schema.Pattern, Path: path, Input: s,
				Message: "Invalid string: must match pattern",
			}}, nil
		}
	}

	for _, chk := range schema.Checks {
		if iss := checkString(s, chk, path); iss != nil {
			return []Issue{*iss}, nil
		}
	}
	return nil, s
}

func checkString(s string, chk Check, path []any) *Issue {
	switch chk.Check {
	case "min_length":
		min := intFromAny(chk.Value)
		if len([]rune(s)) < min {
			return &Issue{
				Code: "too_small", Origin: "string", Minimum: min, Inclusive: true, Path: path, Input: s,
				Message: fmt.Sprintf("Too small: expected string to have >=%d characters", min),
			}
		}
	case "max_length":
		max := intFromAny(chk.Value)
		if len([]rune(s)) > max {
			return &Issue{
				Code: "too_big", Origin: "string", Maximum: max, Inclusive: true, Path: path, Input: s,
				Message: fmt.Sprintf("Too big: expected string to have <=%d characters", max),
			}
		}
	case "length_equals":
		n := intFromAny(chk.Value)
		if len([]rune(s)) != n {
			return &Issue{
				Code: "too_small", Origin: "string", Minimum: n, Exact: true, Path: path, Input: s,
				Message: fmt.Sprintf("Invalid length: expected %d", n),
			}
		}
	case "string_format":
		if re, ok := formatRegex(chk.Format, nil); ok && !re.MatchString(s) {
			return &Issue{
				Code: "invalid_format", Format: chk.Format, Path: path, Input: s,
				Message: fmt.Sprintf("Invalid %s", chk.Format),
			}
		}
		if chk.Pattern != "" {
			re, err := regexp.Compile(chk.Pattern)
			if err == nil && !re.MatchString(s) {
				return &Issue{Code: "invalid_format", Format: "regex", Pattern: chk.Pattern, Path: path, Input: s}
			}
		}
	}
	return nil
}

func (v *Validator) runNumber(input any, schema Schema, path []any) ([]Issue, any) {
	val := input
	if schema.Coerce {
		switch x := input.(type) {
		case string:
			if f, err := strconv.ParseFloat(x, 64); err == nil {
				val = f
			}
		case jsonNumber:
			if f, err := x.Float64(); err == nil {
				val = f
			}
		}
	}
	n, ok := toFloat64(val)
	if !ok {
		return []Issue{{
			Code: "invalid_type", Expected: "number", Path: path, Input: input,
			Message: "Invalid input: expected number, received " + typeName(input),
		}}, nil
	}
	if schema.Type == "int" && math.Trunc(n) != n {
		return []Issue{{
			Code: "invalid_type", Expected: "int", Path: path, Input: input,
			Message: "Invalid input: expected int",
		}}, nil
	}
	for _, chk := range schema.Checks {
		if iss := checkNumber(n, chk, path); iss != nil {
			return []Issue{*iss}, nil
		}
	}
	return nil, n
}

func checkNumber(n float64, chk Check, path []any) *Issue {
	switch chk.Check {
	case "greater_than":
		min := floatFromAny(chk.Value)
		if chk.Inclusive {
			if n < min {
				return &Issue{Code: "too_small", Origin: "number", Minimum: min, Inclusive: true, Path: path, Input: n}
			}
		} else if n <= min {
			return &Issue{Code: "too_small", Origin: "number", Minimum: min, Inclusive: false, Path: path, Input: n}
		}
	case "less_than":
		max := floatFromAny(chk.Value)
		if chk.Inclusive {
			if n > max {
				return &Issue{Code: "too_big", Origin: "number", Maximum: max, Inclusive: true, Path: path, Input: n}
			}
		} else if n >= max {
			return &Issue{Code: "too_big", Origin: "number", Maximum: max, Inclusive: false, Path: path, Input: n}
		}
	case "multiple_of":
		div := floatFromAny(chk.Value)
		if div != 0 && math.Mod(n, div) != 0 {
			return &Issue{Code: "not_multiple_of", Divisor: div, Path: path, Input: n}
		}
	case "number_format":
		// Exported metadata; int typing handled at schema level.
		return nil
	}
	return nil
}

// runDate validates ISO 8601 date/datetime strings (JSON representation of JS Date).
func (v *Validator) runDate(input any, schema Schema, path []any) ([]Issue, any) {
	s, ok := input.(string)
	if !ok {
		return []Issue{{
			Code: "invalid_type", Expected: "date", Path: path, Input: input,
			Message: "Invalid input: expected ISO date string, received " + typeName(input),
		}}, nil
	}
	formats := []string{
		time.RFC3339Nano,
		time.RFC3339,
		"2006-01-02T15:04:05.000Z",
		"2006-01-02",
	}
	var t time.Time
	var parsed bool
	for _, f := range formats {
		if pt, err := time.Parse(f, s); err == nil {
			t = pt
			parsed = true
			break
		}
	}
	if !parsed {
		return []Issue{{
			Code: "invalid_format", Format: "date", Path: path, Input: s,
			Message: "Invalid date: expected ISO 8601 date string",
		}}, nil
	}
	for _, chk := range schema.Checks {
		if iss := checkDate(t, chk, path, s); iss != nil {
			return []Issue{*iss}, nil
		}
	}
	return nil, s
}

func checkDate(t time.Time, chk Check, path []any, input string) *Issue {
	parseDate := func(v any) (time.Time, bool) {
		s, ok := v.(string)
		if !ok {
			return time.Time{}, false
		}
		for _, f := range []string{time.RFC3339Nano, time.RFC3339, "2006-01-02T15:04:05.000Z", "2006-01-02"} {
			if pt, err := time.Parse(f, s); err == nil {
				return pt, true
			}
		}
		return time.Time{}, false
	}
	switch chk.Check {
	case "greater_than":
		ref, ok := parseDate(chk.Value)
		if !ok {
			return nil
		}
		if chk.Inclusive {
			if t.Before(ref) {
				return &Issue{Code: "too_small", Origin: "date", Path: path, Input: input,
					Message: "Date too early"}
			}
		} else if !t.After(ref) {
			return &Issue{Code: "too_small", Origin: "date", Path: path, Input: input,
				Message: "Date too early"}
		}
	case "less_than":
		ref, ok := parseDate(chk.Value)
		if !ok {
			return nil
		}
		if chk.Inclusive {
			if t.After(ref) {
				return &Issue{Code: "too_big", Origin: "date", Path: path, Input: input,
					Message: "Date too late"}
			}
		} else if !t.Before(ref) {
			return &Issue{Code: "too_big", Origin: "date", Path: path, Input: input,
				Message: "Date too late"}
		}
	}
	return nil
}

func (v *Validator) runBoolean(input any, schema Schema, path []any) ([]Issue, any) {
	val := input
	if schema.Coerce {
		switch x := input.(type) {
		case string:
			switch strings.ToLower(strings.TrimSpace(x)) {
			case "true", "1", "yes", "on":
				val = true
			case "false", "0", "no", "off", "":
				val = false
			}
		case float64:
			val = x != 0
		}
	}
	b, ok := val.(bool)
	if !ok {
		return []Issue{{
			Code: "invalid_type", Expected: "boolean", Path: path, Input: input,
			Message: "Invalid input: expected boolean, received " + typeName(input),
		}}, nil
	}
	return nil, b
}

func (v *Validator) runNull(input any, path []any) ([]Issue, any) {
	if input != nil {
		return []Issue{{
			Code: "invalid_type", Expected: "null", Path: path, Input: input,
			Message: "Invalid input: expected null",
		}}, nil
	}
	return nil, nil
}

func (v *Validator) runUndefined(input any, path []any) ([]Issue, any) {
	if input != nil {
		return []Issue{{
			Code: "invalid_type", Expected: "undefined", Path: path, Input: input,
			Message: "Invalid input: expected undefined",
		}}, nil
	}
	return nil, nil
}

func (v *Validator) runArray(input any, schema Schema, path []any) ([]Issue, any) {
	arr, ok := input.([]any)
	if !ok {
		return []Issue{{
			Code: "invalid_type", Expected: "array", Path: path, Input: input,
			Message: "Invalid input: expected array, received " + typeName(input),
		}}, nil
	}
	for _, chk := range schema.Checks {
		if iss := checkSize(len(arr), chk, "array", path); iss != nil {
			return []Issue{*iss}, nil
		}
	}
	if schema.Element == nil {
		return nil, arr
	}
	out := make([]any, len(arr))
	var issues []Issue
	for i, item := range arr {
		res := v.safeParse(item, *schema.Element, prefixPath(path, float64(i)))
		if !res.Success {
			issues = append(issues, res.Error.Issues...)
		} else {
			out[i] = res.Data
		}
	}
	if len(issues) > 0 {
		return issues, nil
	}
	return nil, out
}

func (v *Validator) runObject(input any, schema Schema, path []any) ([]Issue, any) {
	obj, ok := input.(map[string]any)
	if !ok {
		return []Issue{{
			Code: "invalid_type", Expected: "object", Path: path, Input: input,
			Message: "Invalid input: expected object, received " + typeName(input),
		}}, nil
	}
	out := make(map[string]any)
	var issues []Issue

	if schema.Strict {
		allowed := make(map[string]struct{}, len(schema.Shape))
		for k := range schema.Shape {
			allowed[k] = struct{}{}
		}
		var unknown []string
		for k := range obj {
			if _, ok := allowed[k]; !ok {
				unknown = append(unknown, k)
			}
		}
		if len(unknown) > 0 {
			return []Issue{{
				Code: "unrecognized_keys", Keys: unknown, Path: path, Input: obj,
				Message: "Unrecognized key(s) in object: " + strings.Join(unknown, ", "),
			}}, nil
		}
	}

	for key, child := range schema.Shape {
		val, present := obj[key]
		if !present {
			if child.Type == "optional" || child.Type == "default" {
				_, parsed := v.run(nil, child, prefixPath(path, key))
				if parsed != nil {
					out[key] = parsed
				}
				continue
			}
			issues = append(issues, Issue{
				Code: "invalid_type", Expected: "nonoptional", Path: prefixPath(path, key), Input: nil,
				Message: "Required",
			})
			continue
		}
		res := v.safeParse(val, child, prefixPath(path, key))
		if !res.Success {
			issues = append(issues, res.Error.Issues...)
		} else if res.Data != nil || present {
			out[key] = res.Data
		}
	}

	if schema.Catchall != nil {
		for k, val := range obj {
			if _, defined := schema.Shape[k]; defined {
				continue
			}
			res := v.safeParse(val, *schema.Catchall, prefixPath(path, k))
			if !res.Success {
				issues = append(issues, res.Error.Issues...)
			} else {
				out[k] = res.Data
			}
		}
	}

	if len(issues) > 0 {
		return issues, nil
	}
	return nil, out
}

func (v *Validator) runEnum(input any, schema Schema, path []any) ([]Issue, any) {
	for _, opt := range schema.Values {
		if valuesEqual(input, opt) {
			return nil, input
		}
	}
	return []Issue{{
		Code: "invalid_value", Path: path, Input: input,
		Message: "Invalid enum value",
	}}, nil
}

func (v *Validator) runLiteral(input any, schema Schema, path []any) ([]Issue, any) {
	if valuesEqual(input, schema.Value) {
		return nil, input
	}
	return []Issue{{
		Code: "invalid_value", Path: path, Input: input,
		Message: fmt.Sprintf("Invalid literal value, expected %v", schema.Value),
	}}, nil
}

func (v *Validator) runUnion(input any, schema Schema, path []any) ([]Issue, any) {
	var allIssues []Issue
	for _, opt := range schema.Options {
		res := v.safeParse(input, opt, path)
		if res.Success {
			return nil, res.Data
		}
		allIssues = append(allIssues, res.Error.Issues...)
	}
	return []Issue{{
		Code: "invalid_union", Path: path, Input: input,
		Message: "Invalid input",
	}}, nil
}

func (v *Validator) runOptional(input any, schema Schema, path []any) ([]Issue, any) {
	if input == nil {
		return nil, nil
	}
	if schema.Inner == nil {
		return nil, input
	}
	return v.run(input, *schema.Inner, path)
}

func (v *Validator) runNullable(input any, schema Schema, path []any) ([]Issue, any) {
	if input == nil {
		return nil, nil
	}
	if schema.Inner == nil {
		return nil, input
	}
	return v.run(input, *schema.Inner, path)
}

func (v *Validator) runDefault(input any, schema Schema, path []any) ([]Issue, any) {
	if input == nil && schema.Default != nil {
		return nil, schema.Default
	}
	if schema.Inner == nil {
		return nil, input
	}
	return v.run(input, *schema.Inner, path)
}

func checkSize(n int, chk Check, origin string, path []any) *Issue {
	switch chk.Check {
	case "min_size", "min_length":
		min := intFromCheck(chk, true)
		if n < min {
			return &Issue{Code: "too_small", Origin: origin, Minimum: min, Inclusive: true, Path: path}
		}
	case "max_size", "max_length":
		max := intFromCheck(chk, false)
		if n > max {
			return &Issue{Code: "too_big", Origin: origin, Maximum: max, Inclusive: true, Path: path}
		}
	case "size_equals", "length_equals":
		eq := intFromCheck(chk, true)
		if n != eq {
			return &Issue{Code: "too_small", Origin: origin, Minimum: eq, Exact: true, Path: path}
		}
	}
	return nil
}

func intFromCheck(chk Check, preferMin bool) int {
	if chk.Value != nil {
		return intFromAny(chk.Value)
	}
	if preferMin && chk.Minimum != nil {
		return intFromAny(chk.Minimum)
	}
	if !preferMin && chk.Maximum != nil {
		return intFromAny(chk.Maximum)
	}
	return 0
}

func (v *Validator) runTuple(input any, schema Schema, path []any) ([]Issue, any) {
	arr, ok := input.([]any)
	if !ok {
		return []Issue{{
			Code: "invalid_type", Expected: "array", Path: path, Input: input,
			Message: "Invalid input: expected tuple (array), received " + typeName(input),
		}}, nil
	}
	for _, chk := range schema.Checks {
		if iss := checkSize(len(arr), chk, "array", path); iss != nil {
			return []Issue{*iss}, nil
		}
	}
	if len(schema.Items) > 0 && len(arr) < len(schema.Items) {
		return []Issue{{
			Code: "too_small", Origin: "array", Minimum: len(schema.Items), Inclusive: true, Path: path, Input: input,
			Message: fmt.Sprintf("Too small: expected array to have >=%d items", len(schema.Items)),
		}}, nil
	}
	out := make([]any, len(arr))
	var issues []Issue
	for i, item := range arr {
		if i >= len(schema.Items) {
			if schema.Rest != nil {
				res := v.safeParse(item, *schema.Rest, prefixPath(path, float64(i)))
				if !res.Success {
					issues = append(issues, res.Error.Issues...)
				} else {
					out[i] = res.Data
				}
			}
			continue
		}
		res := v.safeParse(item, schema.Items[i], prefixPath(path, float64(i)))
		if !res.Success {
			issues = append(issues, res.Error.Issues...)
		} else {
			out[i] = res.Data
		}
	}
	if len(issues) > 0 {
		return issues, nil
	}
	return nil, out
}

func (v *Validator) runRecord(input any, schema Schema, path []any) ([]Issue, any) {
	obj, ok := input.(map[string]any)
	if !ok {
		return []Issue{{
			Code: "invalid_type", Expected: "object", Path: path, Input: input,
			Message: "Invalid input: expected object, received " + typeName(input),
		}}, nil
	}
	if schema.ValueType == nil {
		return nil, obj
	}
	out := make(map[string]any)
	var issues []Issue
	for k, val := range obj {
		if schema.KeyType != nil {
			res := v.safeParse(k, *schema.KeyType, prefixPath(path, k))
			if !res.Success {
				issues = append(issues, res.Error.Issues...)
				continue
			}
		}
		res := v.safeParse(val, *schema.ValueType, prefixPath(path, k))
		if !res.Success {
			issues = append(issues, res.Error.Issues...)
		} else {
			out[k] = res.Data
		}
	}
	if len(issues) > 0 {
		return issues, nil
	}
	return nil, out
}

func (v *Validator) runIntersection(input any, schema Schema, path []any) ([]Issue, any) {
	if schema.Left == nil || schema.Right == nil {
		return nil, input
	}
	leftIssues, leftOut := v.run(input, *schema.Left, path)
	if len(leftIssues) > 0 {
		return leftIssues, nil
	}
	rightIssues, rightOut := v.run(input, *schema.Right, path)
	if len(rightIssues) > 0 {
		return rightIssues, nil
	}
	return nil, mergeOutputs(leftOut, rightOut)
}

func mergeOutputs(a, b any) any {
	am, aok := a.(map[string]any)
	bm, bok := b.(map[string]any)
	if aok && bok {
		out := make(map[string]any, len(am)+len(bm))
		for k, v := range am {
			out[k] = v
		}
		for k, v := range bm {
			out[k] = v
		}
		return out
	}
	if b != nil {
		return b
	}
	return a
}

type jsonNumber interface {
	Float64() (float64, error)
}

func toFloat64(v any) (float64, bool) {
	switch x := v.(type) {
	case float64:
		return x, true
	case float32:
		return float64(x), true
	case int:
		return float64(x), true
	case int64:
		return float64(x), true
	case jsonNumber:
		f, err := x.Float64()
		return f, err == nil
	default:
		return 0, false
	}
}

func floatFromAny(v any) float64 {
	f, _ := toFloat64(v)
	return f
}

func intFromAny(v any) int {
	switch x := v.(type) {
	case float64:
		return int(x)
	case int:
		return x
	case int64:
		return int(x)
	default:
		return 0
	}
}

func typeName(v any) string {
	if v == nil {
		return "null"
	}
	switch v.(type) {
	case string:
		return "string"
	case float64, float32, int, int64:
		return "number"
	case bool:
		return "boolean"
	case []any:
		return "array"
	case map[string]any:
		return "object"
	default:
		return fmt.Sprintf("%T", v)
	}
}

func valuesEqual(a, b any) bool {
	if af, ok := toFloat64(a); ok {
		if bf, ok := toFloat64(b); ok {
			return af == bf
		}
	}
	return fmt.Sprint(a) == fmt.Sprint(b)
}
