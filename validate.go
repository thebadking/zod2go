package zod2go

import (
	"encoding/json"
	"fmt"
	"math"
	"math/big"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode/utf16"
)

func (v *Validator) runString(input any, schema Schema, path []any) ([]Issue, any) {
	val := input
	if schema.Coerce {
		val = jsString(input)
	}
	s, ok := val.(string)
	if !ok {
		return []Issue{{
			Code: "invalid_type", Expected: "string", Path: path, Input: input,
			Message: "Invalid input: expected string, received " + typeName(input),
		}}, nil
	}

	rules := stringRules{format: schema.Format, pattern: schema.Pattern, alg: schema.Alg, hostname: schema.Hostname, protocol: schema.Protocol, normalize: schema.Normalize, uuid: schema.UUIDVer}
	if rules.format == "url" {
		normalized, valid := validateURL(s, rules)
		if !valid {
			return []Issue{{Code: "invalid_format", Format: "url", Path: path, Input: s, Message: "Invalid URL"}}, nil
		}
		s = normalized
	} else if !validateStringRules(s, rules) {
		return []Issue{{Code: "invalid_format", Format: schema.Format, Pattern: schema.Pattern, Path: path, Input: s, Message: "Invalid string format"}}, nil
	}

	for _, chk := range schema.Checks {
		if chk.Check == "string_format" && chk.Format == "url" {
			normalized, valid := validateURL(s, stringRules{format: chk.Format, pattern: chk.Pattern, alg: chk.Alg, hostname: chk.Hostname, protocol: chk.Protocol, normalize: chk.Normalize})
			if !valid {
				return []Issue{{Code: "invalid_format", Format: "url", Path: path, Input: s}}, nil
			}
			s = normalized
			continue
		}
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
		if len(utf16.Encode([]rune(s))) < min {
			return &Issue{
				Code: "too_small", Origin: "string", Minimum: min, Inclusive: true, Path: path, Input: s,
				Message: fmt.Sprintf("Too small: expected string to have >=%d characters", min),
			}
		}
	case "max_length":
		max := intFromAny(chk.Value)
		if len(utf16.Encode([]rune(s))) > max {
			return &Issue{
				Code: "too_big", Origin: "string", Maximum: max, Inclusive: true, Path: path, Input: s,
				Message: fmt.Sprintf("Too big: expected string to have <=%d characters", max),
			}
		}
	case "length_equals":
		n := intFromAny(chk.Value)
		if len(utf16.Encode([]rune(s))) != n {
			return &Issue{
				Code: "too_small", Origin: "string", Minimum: n, Exact: true, Path: path, Input: s,
				Message: fmt.Sprintf("Invalid length: expected %d", n),
			}
		}
	case "string_format":
		if chk.Format == "includes" {
			units := utf16.Encode([]rune(s))
			position := chk.Position
			if position < 0 {
				position = 0
			}
			if position > len(units) {
				position = len(units)
			}
			needle, _ := chk.Value.(string)
			if strings.Contains(string(utf16.Decode(units[position:])), needle) {
				return nil
			}
			return &Issue{Code: "invalid_format", Format: "includes", Path: path, Input: s}
		}
		version := intFromAny(chk.Value)
		if !validateStringRules(s, stringRules{format: chk.Format, pattern: chk.Pattern, alg: chk.Alg, uuid: &version}) {
			return &Issue{Code: "invalid_format", Format: chk.Format, Pattern: chk.Pattern, Path: path, Input: s}
		}

	}
	return nil
}

func (v *Validator) runNumber(input any, schema Schema, path []any) ([]Issue, any) {
	val := input
	if schema.Coerce {
		val = jsNumber(input)
	}
	n, ok := toFloat64(val)
	if !ok || math.IsNaN(n) || math.IsInf(n, 0) {
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
		ratio := n / div
		if div == 0 || math.Abs(ratio-math.Round(ratio)) >= (math.Nextafter(1, 2)-1)*math.Max(math.Abs(ratio), 1) {
			return &Issue{Code: "not_multiple_of", Divisor: div, Path: path, Input: n}
		}
	case "number_format":
		min, max := -math.MaxFloat64, math.MaxFloat64
		isInt := strings.Contains(chk.Format, "int")
		if isInt {
			if math.Trunc(n) != n {
				return &Issue{Code: "invalid_type", Expected: "int", Path: path, Input: n}
			}
			if n < -9007199254740991 {
				return &Issue{Code: "too_small", Origin: "int", Minimum: float64(-9007199254740991), Inclusive: true, Path: path, Input: n}
			}
			if n > 9007199254740991 {
				return &Issue{Code: "too_big", Origin: "int", Maximum: float64(9007199254740991), Inclusive: true, Path: path, Input: n}
			}
		}
		switch chk.Format {
		case "safeint":
			min, max = -9007199254740991, 9007199254740991
		case "int32":
			min, max = -2147483648, 2147483647
		case "uint32":
			min, max = 0, 4294967295
		case "float32":
			min, max = -math.MaxFloat32, math.MaxFloat32
		case "float64", "":
		default:
			return &Issue{Code: "custom", Message: "Unknown number format: " + chk.Format, Path: path, Input: n}
		}
		if n < min {
			return &Issue{Code: "too_small", Origin: "number", Minimum: min, Inclusive: true, Path: path, Input: n}
		}
		if n > max {
			return &Issue{Code: "too_big", Origin: "number", Maximum: max, Inclusive: true, Path: path, Input: n}
		}
	}
	return nil
}

// runDate validates ISO 8601 date/datetime strings (JSON representation of JS Date).
func (v *Validator) runDate(input any, schema Schema, path []any) ([]Issue, any) {
	if schema.Coerce {
		parsed, valid := coerceDate(input)
		if !valid {
			return []Issue{{Code: "invalid_type", Expected: "date", Path: path, Input: input, Message: "Invalid date"}}, nil
		}
		text := parsed.UTC().Format("2006-01-02T15:04:05.000Z")
		for _, check := range schema.Checks {
			if issue := checkDate(parsed, check, path, text); issue != nil {
				return []Issue{*issue}, nil
			}
		}
		if v.typed {
			return nil, parsed
		}
		return nil, text
	}

	if v.typed {
		var nativeTime time.Time
		valid := false
		switch value := input.(type) {
		case time.Time:
			nativeTime = value
			valid = true
		case *time.Time:
			if value != nil {
				nativeTime = *value
				valid = true
			}
		}
		if valid {
			text := nativeTime.UTC().Format(time.RFC3339Nano)
			for _, check := range schema.Checks {
				if issue := checkDate(nativeTime, check, path, text); issue != nil {
					return []Issue{*issue}, nil
				}
			}
			return nil, nativeTime
		}
		if !schema.Coerce {
			return []Issue{{Code: "invalid_type", Expected: "date", Path: path, Input: input}}, nil
		}
	}
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
	if v.typed {
		return nil, t
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
				return &Issue{Code: "too_small", Origin: "date", Minimum: chk.Value, Inclusive: chk.Inclusive, Path: path, Input: input,
					Message: "Date too early"}
			}
		} else if !t.After(ref) {
			return &Issue{Code: "too_small", Origin: "date", Minimum: chk.Value, Inclusive: chk.Inclusive, Path: path, Input: input,
				Message: "Date too early"}
		}
	case "less_than":
		ref, ok := parseDate(chk.Value)
		if !ok {
			return nil
		}
		if chk.Inclusive {
			if t.After(ref) {
				return &Issue{Code: "too_big", Origin: "date", Maximum: chk.Value, Inclusive: chk.Inclusive, Path: path, Input: input,
					Message: "Date too late"}
			}
		} else if !t.Before(ref) {
			return &Issue{Code: "too_big", Origin: "date", Maximum: chk.Value, Inclusive: chk.Inclusive, Path: path, Input: input,
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
			val = x != ""
		case bool:
			val = x
		case nil, undefinedValue:
			val = false
		default:
			if n, ok := toFloat64(input); ok {
				val = n != 0 && !math.IsNaN(n)
			} else {
				val = true
			}
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
	if !isUndefined(input) {
		return []Issue{{
			Code: "invalid_type", Expected: "undefined", Path: path, Input: input,
			Message: "Invalid input: expected undefined",
		}}, nil
	}
	return nil, Undefined
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
			if k == "__proto__" {
				continue
			}
			if _, ok := allowed[k]; !ok {
				unknown = append(unknown, k)
			}
		}
		if len(unknown) > 0 {
			issues = append(issues, Issue{Code: "unrecognized_keys", Keys: unknown, Path: path, Input: obj, Message: "Unrecognized key(s) in object: " + strings.Join(unknown, ", ")})
		}
	}

	for key, child := range schema.Shape {
		val, present := obj[key]
		if !present {
			val = Undefined
		}
		res := v.safeParse(val, child, prefixPath(path, key))
		if !res.Success {
			if !present && (child.OptionalInput && child.OptionalOutput || child.Type == "optional") {
				continue
			}
			issues = append(issues, res.Error.Issues...)
		} else if !present && isUndefined(res.Data) && !child.OptionalInput && child.Type != "optional" {
			issues = append(issues, Issue{Code: "invalid_type", Expected: "nonoptional", Path: prefixPath(path, key), Message: "Required"})
		} else if !isUndefined(res.Data) || (v.typed && present) {
			out[key] = res.Data
		}
	}

	if schema.Catchall != nil {
		for k, val := range obj {
			if k == "__proto__" {
				continue
			}
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
		return issues, out
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
	if schema.ValueKind != "" {
		matches := false
		switch schema.ValueKind {
		case "undefined":
			matches = isUndefined(input)
		case "bigint":
			validType := !v.typed
			switch input.(type) {
			case *big.Int, big.Int:
				validType = true
			}
			if validType {
				actual, ok := parseBigInt(input, false)
				expected, valid := parseBigInt(schema.Value, false)
				matches = ok && valid && actual.Cmp(expected) == 0
			}
		case "nonfinite":
			if number, ok := toFloat64(input); ok {
				matches = (schema.Value == "NaN" && math.IsNaN(number)) || (schema.Value == "Infinity" && math.IsInf(number, 1)) || (schema.Value == "-Infinity" && math.IsInf(number, -1))
			}
		}
		if matches {
			return nil, input
		}
		return []Issue{{Code: "invalid_value", Path: path, Input: input, Message: "Invalid literal value"}}, nil
	}
	if valuesEqual(input, schema.Value) {
		return nil, input
	}
	return []Issue{{
		Code: "invalid_value", Path: path, Input: input,
		Message: fmt.Sprintf("Invalid literal value, expected %v", schema.Value),
	}}, nil
}

func (v *Validator) runUnion(input any, schema Schema, path []any) ([]Issue, any) {
	var matches int
	var parsed any
	for _, opt := range schema.Options {
		res := v.safeParse(input, opt, path)
		if res.Success {
			if !schema.Exclusive {
				return nil, res.Data
			}
			matches++
			parsed = res.Data
		}
	}
	if matches == 1 {
		return nil, parsed
	}
	return []Issue{{Code: "invalid_union", Path: path, Input: input, Message: "Invalid input"}}, nil
}

func (v *Validator) runOptional(input any, schema Schema, path []any) ([]Issue, any) {
	if schema.ExactOptional {
		if schema.Inner == nil {
			return nil, input
		}
		return v.run(input, *schema.Inner, path)
	}
	if isUndefined(input) {
		if schema.Inner != nil && (schema.Inner.OptionalInput || schema.Inner.Type == "default" || schema.Inner.Type == "prefault") {
			issues, parsed := v.run(input, *schema.Inner, path)
			if len(issues) == 0 {
				return nil, parsed
			}
		}
		return nil, Undefined
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
	if isUndefined(input) {
		return nil, schema.defaultValue()
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
		return []Issue{{Code: "invalid_type", Expected: "tuple", Path: path, Input: input}}, nil
	}
	for _, check := range schema.Checks {
		if issue := checkSize(len(arr), check, "array", path); issue != nil {
			return []Issue{*issue}, nil
		}
	}
	optionalIn := func(item Schema) bool {
		return item.OptionalInput || item.Type == "optional" || item.Type == "default" || item.Type == "prefault"
	}
	optionalOut := func(item Schema) bool { return item.OptionalOutput || item.Type == "optional" }
	optinStart, optoutStart := 0, 0
	for i := len(schema.Items) - 1; i >= 0; i-- {
		if !optionalIn(schema.Items[i]) {
			optinStart = i + 1
			break
		}
	}
	for i := len(schema.Items) - 1; i >= 0; i-- {
		if !optionalOut(schema.Items[i]) {
			optoutStart = i + 1
			break
		}
	}
	var issues []Issue
	if schema.Rest == nil {
		if len(arr) < optinStart {
			return []Issue{{Code: "too_small", Origin: "array", Minimum: optinStart, Inclusive: true, Path: path}}, nil
		}
		if len(arr) > len(schema.Items) {
			issues = append(issues, Issue{Code: "too_big", Origin: "array", Maximum: len(schema.Items), Inclusive: true, Path: path})
		}
	}
	length := len(schema.Items)
	if schema.Rest != nil && len(arr) > length {
		length = len(arr)
	}
	out := make([]any, length)
	for i, itemSchema := range schema.Items {
		var value any = Undefined
		present := i < len(arr)
		if present {
			value = arr[i]
		}
		childIssues, parsed := v.run(value, itemSchema, prefixPath(path, float64(i)))
		if len(childIssues) > 0 {
			if !present && i >= optoutStart {
				out = out[:i]
				break
			}
			issues = append(issues, childIssues...)
		}
		out[i] = parsed
	}
	if schema.Rest != nil {
		for i := len(schema.Items); i < len(arr); i++ {
			childIssues, parsed := v.run(arr[i], *schema.Rest, prefixPath(path, float64(i)))
			issues = append(issues, childIssues...)
			out[i] = parsed
		}
	}
	for i := len(out) - 1; i >= len(arr); i-- {
		if i < len(schema.Items) && optionalOut(schema.Items[i]) && isUndefined(out[i]) {
			out = out[:i]
		} else {
			break
		}
	}
	if len(issues) > 0 {
		return issues, out
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
	if schema.Type == "map" {
		for _, check := range schema.Checks {
			if issue := checkSize(len(obj), check, "map", path); issue != nil {
				return []Issue{*issue}, nil
			}
		}
	}
	if schema.ValueType == nil {
		return nil, obj
	}
	out := make(map[string]any)
	var issues []Issue
	if schema.Type == "record" && schema.KeyType != nil && !schema.PartialKeys {
		values := schema.KeyValues
		finite := values != nil
		if !finite {
			values, finite = finiteKeys(*schema.KeyType)
		}
		if finite {
			allowed := make(map[string]bool, len(values))
			for _, key := range values {
				_, isString := key.(string)
				_, isNumber := toFloat64(key)
				if !isString && !isNumber {
					continue
				}
				name := jsString(key)
				allowed[name] = true
				value, present := obj[name]
				if !present {
					value = Undefined
				}
				result := v.safeParse(value, *schema.ValueType, prefixPath(path, name))
				if !result.Success {
					issues = append(issues, result.Error.Issues...)
				} else if !isUndefined(result.Data) {
					out[name] = result.Data
				}
			}
			var unknown []string
			for key := range obj {
				if !allowed[key] {
					unknown = append(unknown, key)
				}
			}
			if len(unknown) > 0 {
				issues = append(issues, Issue{Code: "unrecognized_keys", Keys: unknown, Path: path})
			}
			if len(issues) > 0 {
				return issues, nil
			}
			return nil, out
		}
	}
	for k, val := range obj {
		if k == "__proto__" {
			continue
		}
		if schema.KeyType != nil {
			res := v.safeParse(k, *schema.KeyType, prefixPath(path, k))
			if !res.Success && schema.Type == "record" && reNumericKey.MatchString(k) {
				if number, err := strconv.ParseFloat(k, 64); err == nil {
					res = v.safeParse(number, *schema.KeyType, prefixPath(path, k))
				}
			}
			if !res.Success {
				if schema.Mode == "loose" {
					out[k] = val
					continue
				}
				issues = append(issues, res.Error.Issues...)
				continue
			}
			k = jsString(res.Data)
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
	rightIssues, rightOut := v.run(input, *schema.Right, path)
	var issues []Issue
	leftKeys := make(map[string]bool)
	rightKeys := make(map[string]bool)
	for _, issue := range leftIssues {
		if issue.Code == "unrecognized_keys" {
			for _, key := range issue.Keys {
				leftKeys[key] = true
			}
		} else {
			issues = append(issues, issue)
		}
	}
	for _, issue := range rightIssues {
		if issue.Code == "unrecognized_keys" {
			for _, key := range issue.Keys {
				rightKeys[key] = true
			}
		} else {
			issues = append(issues, issue)
		}
	}
	var unknown []string
	for key := range leftKeys {
		if rightKeys[key] {
			unknown = append(unknown, key)
		}
	}
	if len(unknown) > 0 {
		issues = append(issues, Issue{Code: "unrecognized_keys", Keys: unknown, Path: path})
	}
	if len(issues) > 0 {
		return issues, nil
	}
	out, ok := mergeOutputs(leftOut, rightOut)
	if !ok {
		return []Issue{newIssue("custom", "Intersection outputs could not be merged", path, input)}, nil
	}
	return nil, out
}

func mergeOutputs(a, b any) (any, bool) {
	if reflect.DeepEqual(a, b) {
		return a, true
	}
	am, aok := a.(map[string]any)
	bm, bok := b.(map[string]any)
	if aok && bok {
		out := make(map[string]any, len(am)+len(bm))
		for k, val := range am {
			out[k] = val
		}
		for k, val := range bm {
			if prev, exists := out[k]; exists {
				merged, ok := mergeOutputs(prev, val)
				if !ok {
					return nil, false
				}
				out[k] = merged
			} else {
				out[k] = val
			}
		}
		return out, true
	}
	aa, aok := a.([]any)
	ba, bok := b.([]any)
	if aok && bok && len(aa) == len(ba) {
		out := make([]any, len(aa))
		for i := range aa {
			merged, ok := mergeOutputs(aa[i], ba[i])
			if !ok {
				return nil, false
			}
			out[i] = merged
		}
		return out, true
	}
	return nil, false
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
	return reflect.DeepEqual(a, b)
}

// jsString and jsNumber implement JavaScript coercion for JSON values.
func jsString(input any) string {
	switch x := input.(type) {
	case nil:
		return "null"
	case undefinedValue:
		return "undefined"
	case string:
		return x
	case []any:
		parts := make([]string, len(x))
		for i, item := range x {
			if item != nil && !isUndefined(item) {
				parts[i] = jsString(item)
			}
		}
		return strings.Join(parts, ",")
	case RuntimeSet:
		return "[object Set]"
	case RuntimeMap:
		return "[object Map]"
	case InvalidDate:
		return "Invalid Date"
	case time.Time:
		return jsDateString(x)
	case *time.Time:
		if x != nil {
			return jsDateString(*x)
		}
		return "null"
	case map[string]any:
		return "[object Object]"
	default:
		if n, ok := toFloat64(input); ok {
			return jsNumberString(n)
		}
		return fmt.Sprint(input)
	}
}

func jsNumber(input any) float64 {
	if n, ok := toFloat64(input); ok {
		return n
	}
	switch x := input.(type) {
	case *big.Int:
		if x != nil {
			f, _ := new(big.Float).SetInt(x).Float64()
			return f
		}
	case big.Int:
		f, _ := new(big.Float).SetInt(&x).Float64()
		return f
	case time.Time:
		return float64(x.UnixMilli())
	case *time.Time:
		if x != nil {
			return float64(x.UnixMilli())
		}
	case nil:
		return 0
	case bool:
		if x {
			return 1
		}
		return 0
	case []any:
		return jsNumber(jsString(x))
	case string:
		s := jsTrim(x)
		if s == "" {
			return 0
		}
		if len(s) > 2 && s[0] == '0' && strings.ContainsAny(s[1:2], "xXbBoO") {
			if n, err := strconv.ParseUint(s, 0, 64); err == nil {
				return float64(n)
			}
			return math.NaN()
		}
		if n, err := strconv.ParseFloat(s, 64); err == nil {
			return n
		}
	}
	return math.NaN()
}

var reNumericKey = regexp.MustCompile(`^-?\d+(?:\.\d+)?$`)

func finiteKeys(schema Schema) ([]any, bool) {
	switch schema.Type {
	case "enum":
		return schema.Values, true
	case "literal":
		return []any{schema.Value}, true
	case "union":
		var values []any
		for _, option := range schema.Options {
			keys, finite := finiteKeys(option)
			if !finite {
				return nil, false
			}
			values = append(values, keys...)
		}
		return values, true
	case "readonly":
		if schema.Inner != nil {
			return finiteKeys(*schema.Inner)
		}
	}
	return nil, false
}

func jsNumberString(number float64) string {
	if number == 0 {
		return "0"
	}
	if math.IsNaN(number) {
		return "NaN"
	}
	if math.IsInf(number, 1) {
		return "Infinity"
	}
	if math.IsInf(number, -1) {
		return "-Infinity"
	}
	if math.Abs(number) >= 1e-6 && math.Abs(number) < 1e21 {
		return strconv.FormatFloat(number, 'f', -1, 64)
	}
	result := strconv.FormatFloat(number, 'e', -1, 64)
	result = strings.Replace(result, "e-0", "e-", 1)
	result = strings.Replace(result, "e+0", "e+", 1)
	return result
}
func jsDateString(value time.Time) string {
	local := value.In(time.Local)
	name, _ := local.Zone()
	if name == "UTC" || name == "GMT" {
		name = "Coordinated Universal Time"
	}
	return local.Format("Mon Jan 02 2006 15:04:05 GMT-0700") + " (" + name + ")"
}

func nativeCheckSupported(kind, check string) bool {
	switch kind {
	case "string":
		return check == "min_length" || check == "max_length" || check == "length_equals" || check == "string_format"
	case "number", "int":
		return check == "greater_than" || check == "less_than" || check == "multiple_of" || check == "number_format"
	case "bigint":
		return check == "greater_than" || check == "less_than" || check == "multiple_of" || check == "bigint_format"
	case "date":
		return check == "greater_than" || check == "less_than"
	case "array", "tuple":
		return check == "min_length" || check == "max_length" || check == "length_equals"
	case "set", "map":
		return check == "min_size" || check == "max_size" || check == "size_equals"
	}
	return false
}

// Decode per parse so mutable defaults are never shared between callers.
func (schema Schema) defaultValue() any {
	if len(schema.DefaultEncoded) != 0 {
		value, err := DecodeTaggedValue(schema.DefaultEncoded)
		if err == nil {
			return value
		}
	}
	if schema.Default != nil {
		data, err := json.Marshal(schema.Default)
		if err == nil {
			var value any
			if json.Unmarshal(data, &value) == nil {
				return value
			}
		}
	}
	return schema.Default
}
