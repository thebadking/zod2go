package zod2go

import (
	"math"
	"reflect"
)

func sameValueZero(a, b any) bool {
	if an, ok := toFloat64(a); ok {
		if bn, ok := toFloat64(b); ok {
			return an == bn || (math.IsNaN(an) && math.IsNaN(bn))
		}
	}
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	av, bv := reflect.ValueOf(a), reflect.ValueOf(b)
	if av.Type() != bv.Type() {
		return false
	}
	switch av.Kind() {
	case reflect.Map:
		return av.UnsafePointer() == bv.UnsafePointer()
	case reflect.Slice:
		return av.Pointer() == bv.Pointer() && av.Len() == bv.Len()
	case reflect.Pointer:
		return av.Pointer() == bv.Pointer()
	default:
		if av.Type().Comparable() {
			return av.Interface() == bv.Interface()
		}
	}
	return false
}

func (v *Validator) runSet(input any, schema Schema, path []any) ([]Issue, any) {
	var values []any
	if v.typed {
		set, ok := input.(RuntimeSet)
		if !ok {
			return []Issue{{Code: "invalid_type", Expected: "set", Path: path, Input: input}}, nil
		}
		values = []any(set)
	} else {
		array, ok := input.([]any)
		if !ok {
			return []Issue{{Code: "invalid_type", Expected: "set", Path: path, Input: input}}, nil
		}
		values = array
	}
	unique := make([]any, 0, len(values))
	for _, item := range values {
		found := false
		for _, prev := range unique {
			if sameValueZero(prev, item) {
				found = true
				break
			}
		}
		if !found {
			unique = append(unique, item)
		}
	}
	for _, check := range schema.Checks {
		if issue := checkSize(len(unique), check, "set", path); issue != nil {
			return []Issue{*issue}, nil
		}
	}
	var issues []Issue
	out := make(RuntimeSet, 0, len(unique))
	for _, item := range unique {
		parsed := item
		if schema.Element != nil {
			result := v.safeParse(item, *schema.Element, path)
			if !result.Success {
				issues = append(issues, result.Error.Issues...)
				continue
			}
			parsed = result.Data
		}
		found := false
		for _, prev := range out {
			if sameValueZero(prev, parsed) {
				found = true
				break
			}
		}
		if !found {
			out = append(out, parsed)
		}
	}
	if len(issues) > 0 {
		return issues, nil
	}
	if v.typed {
		return nil, out
	}
	return nil, []any(out)
}

func (v *Validator) runTypedMap(input any, schema Schema, path []any) ([]Issue, any) {
	entries, ok := input.(RuntimeMap)
	if !ok {
		return []Issue{{Code: "invalid_type", Expected: "map", Path: path, Input: input}}, nil
	}
	var issues []Issue
	out := make(RuntimeMap, 0, len(entries))
	// A JS Map collapses duplicate input keys before validation.
	unique := make(RuntimeMap, 0, len(entries))
	for _, entry := range entries {
		index := -1
		for i, prev := range unique {
			if sameValueZero(prev.Key, entry.Key) {
				index = i
				break
			}
		}
		if index < 0 {
			unique = append(unique, entry)
		} else {
			unique[index] = entry
		}
	}
	for _, check := range schema.Checks {
		if issue := checkSize(len(unique), check, "map", path); issue != nil {
			return []Issue{*issue}, nil
		}
	}
	for _, entry := range unique {
		key, value := entry.Key, entry.Value
		if schema.KeyType != nil {
			result := v.safeParse(key, *schema.KeyType, path)
			if !result.Success {
				issues = append(issues, result.Error.Issues...)
				continue
			}
			key = result.Data
		}
		if schema.ValueType != nil {
			result := v.safeParse(value, *schema.ValueType, path)
			if !result.Success {
				issues = append(issues, result.Error.Issues...)
				continue
			}
			value = result.Data
		}
		index := -1
		for i, prev := range out {
			if sameValueZero(prev.Key, key) {
				index = i
				break
			}
		}
		if index < 0 {
			out = append(out, RuntimeMapEntry{Key: key, Value: value})
		} else {
			out[index] = RuntimeMapEntry{Key: key, Value: value}
		}
	}
	if len(issues) > 0 {
		return issues, nil
	}
	return nil, out
}
