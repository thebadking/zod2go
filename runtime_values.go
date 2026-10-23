package zod2go

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math"
	"math/big"
	"reflect"
	"time"
)

// RuntimeSet and RuntimeMap preserve JavaScript collection types and map key types.
type RuntimeSet []any
type RuntimeMap []RuntimeMapEntry

// RuntimeObject preserves object properties whose keys are symbols or opaque strings.
type RuntimeObject []RuntimeMapEntry
type RuntimeMapEntry struct {
	Key   any
	Value any
}
type InvalidDate struct{}

// RuntimeFile constructs a JavaScript File, preserving bytes and metadata.
type RuntimeFile struct {
	Data         []byte
	Name         string
	MIMEType     string
	LastModified int64
}
type RuntimeRegExp struct {
	Source string
	Flags  string
}
type RuntimePromise struct{ Value any }

// RuntimeValue refers to a value retained by a running JavaScript worker.
// Handles preserve symbol identity, functions, cyclic references, and class instances.
type RuntimeValue struct {
	ID      int
	Kind    string
	runtime *RuntimeValidator
}

func (value *RuntimeValue) Call(args ...any) (any, error) {
	if value == nil || value.runtime == nil {
		return nil, fmt.Errorf("zod2go: detached runtime value")
	}
	return value.runtime.request("call", args, value.ID)
}

func (value *RuntimeValue) Get(property string) (any, error) {
	if value == nil || value.runtime == nil {
		return nil, fmt.Errorf("zod2go: detached runtime value")
	}
	return value.runtime.request("get", []any{property}, value.ID)
}
func (value *RuntimeValue) Invoke(method string, args ...any) (any, error) {
	if value == nil || value.runtime == nil {
		return nil, fmt.Errorf("zod2go: detached runtime value")
	}
	return value.runtime.request("invoke", append([]any{method}, args...), value.ID)
}
func (value *RuntimeValue) Release() error {
	if value == nil || value.runtime == nil {
		return nil
	}
	_, err := value.runtime.request("release", nil, value.ID)
	return err
}

func encodeRuntimeValue(input any, v *RuntimeValidator) (any, error) {
	return encodeRuntimeDepth(input, v, 0, &runtimeEncodeState{seen: make(map[runtimeIdentity]int)})
}

type runtimeIdentity struct {
	kind    string
	pointer uintptr
}
type runtimeEncodeState struct {
	seen map[runtimeIdentity]int
	next int
}

func encodeRuntimeDepth(input any, v *RuntimeValidator, depth int, state *runtimeEncodeState) (any, error) {
	var identity runtimeIdentity
	switch x := input.(type) {
	case map[string]any:
		identity = runtimeIdentity{"object", uintptr(reflect.ValueOf(x).UnsafePointer())}
	case []any:
		if len(x) > 0 {
			identity = runtimeIdentity{"array", reflect.ValueOf(x).Pointer()}
		}
	case RuntimeSet:
		if len(x) > 0 {
			identity = runtimeIdentity{"set", reflect.ValueOf(x).Pointer()}
		}
	case RuntimeObject:
		if len(x) > 0 {
			identity = runtimeIdentity{"keyedObject", reflect.ValueOf(x).Pointer()}
		}
	case RuntimeMap:
		if len(x) > 0 {
			identity = runtimeIdentity{"map", reflect.ValueOf(x).Pointer()}
		}
	}
	if identity.pointer != 0 {
		if id, ok := state.seen[identity]; ok {
			return []any{"ref", id}, nil
		}
		state.next++
		id := state.next
		state.seen[identity] = id
		body, err := encodeRuntimeBody(input, v, depth, state)
		if err != nil {
			return nil, err
		}
		return []any{"id", map[string]any{"id": id, "value": body}}, nil
	}
	return encodeRuntimeBody(input, v, depth, state)
}
func encodeRuntimeBody(input any, v *RuntimeValidator, depth int, state *runtimeEncodeState) (any, error) {
	if depth > 1000 {
		return nil, fmt.Errorf("zod2go: cyclic or excessively deep runtime input")
	}
	tag := func(kind string, value any) (any, error) { return []any{kind, value}, nil }
	encode := func(value any) (any, error) { return encodeRuntimeDepth(value, v, depth+1, state) }
	sequence := func(values []any) ([]any, error) {
		out := make([]any, len(values))
		for i, item := range values {
			wire, err := encode(item)
			if err != nil {
				return nil, err
			}
			out[i] = wire
		}
		return out, nil
	}
	switch x := input.(type) {
	case nil:
		return []any{"null"}, nil
	case undefinedValue:
		return []any{"undefined"}, nil
	case string:
		return tag("string", x)
	case bool:
		return tag("boolean", x)
	case *big.Int:
		if x == nil {
			return nil, fmt.Errorf("zod2go: nil bigint")
		}
		return tag("bigint", x.String())
	case big.Int:
		return tag("bigint", x.String())
	case time.Time:
		return tag("date", x.UTC().Format(time.RFC3339Nano))
	case RuntimeFile:
		return tag("file", map[string]any{"data": base64.StdEncoding.EncodeToString(x.Data), "name": x.Name, "type": x.MIMEType, "lastModified": x.LastModified})
	case RuntimeRegExp:
		return tag("regexp", map[string]any{"source": x.Source, "flags": x.Flags})
	case RuntimePromise:
		wire, err := encode(x.Value)
		if err != nil {
			return nil, err
		}
		return tag("promise", wire)
	case InvalidDate:
		return tag("date", "invalid")
	case *RuntimeValue:
		if x == nil || x.runtime != v {
			return nil, fmt.Errorf("zod2go: runtime value belongs to a different worker")
		}
		return tag("handle", map[string]any{"id": x.ID, "kind": x.Kind})
	case RuntimeSet:
		values, err := sequence([]any(x))
		if err != nil {
			return nil, err
		}
		return tag("set", values)
	case RuntimeMap:
		values := make([]any, len(x))
		for i, entry := range x {
			key, err := encode(entry.Key)
			if err != nil {
				return nil, err
			}
			val, err := encode(entry.Value)
			if err != nil {
				return nil, err
			}
			values[i] = []any{key, val}
		}
		return tag("map", values)
	case RuntimeObject:
		values := make([]any, len(x))
		for i, entry := range x {
			key, err := encode(entry.Key)
			if err != nil {
				return nil, err
			}
			val, err := encode(entry.Value)
			if err != nil {
				return nil, err
			}
			values[i] = []any{key, val}
		}
		return tag("keyedObject", values)
	case []any:
		values, err := sequence(x)
		if err != nil {
			return nil, err
		}
		return tag("array", values)
	case map[string]any:
		values := make([]any, 0, len(x))
		for key, item := range x {
			wire, err := encode(item)
			if err != nil {
				return nil, err
			}
			values = append(values, []any{key, wire})
		}
		return tag("object", values)
	default:
		if n, ok := toFloat64(input); ok {
			switch {
			case math.IsNaN(n):
				return tag("number", "NaN")
			case math.IsInf(n, 1):
				return tag("number", "Infinity")
			case math.IsInf(n, -1):
				return tag("number", "-Infinity")
			case n == 0 && math.Signbit(n):
				return tag("number", "-0")
			default:
				return tag("number", n)
			}
		}
	}
	return nil, fmt.Errorf("zod2go: unsupported runtime input %T", input)
}

func decodeRuntimeValue(data []byte, v *RuntimeValidator) (any, error) {
	var wire any
	if err := json.Unmarshal(data, &wire); err != nil {
		return nil, err
	}
	return decodeRuntimeNode(wire, v, make(map[int]any))
}
func decodeRuntimeNode(wire any, v *RuntimeValidator, refs map[int]any) (any, error) {
	node, ok := wire.([]any)
	if !ok || len(node) < 1 {
		return nil, fmt.Errorf("zod2go: malformed tagged runtime value")
	}
	kind, ok := node[0].(string)
	if !ok {
		return nil, fmt.Errorf("zod2go: missing runtime value kind")
	}
	if kind == "null" {
		return nil, nil
	}
	if kind == "undefined" {
		return Undefined, nil
	}
	if len(node) != 2 {
		return nil, fmt.Errorf("zod2go: missing runtime value payload")
	}
	value := node[1]
	decode := func(item any) (any, error) { return decodeRuntimeNode(item, v, refs) }
	switch kind {
	case "ref":
		id, ok := value.(float64)
		if ok {
			if result, exists := refs[int(id)]; exists {
				return result, nil
			}
		}
	case "id":
		payload, ok := value.(map[string]any)
		if !ok {
			break
		}
		id, ok := payload["id"].(float64)
		if !ok {
			break
		}
		body, ok := payload["value"].([]any)
		if !ok || len(body) != 2 {
			break
		}
		items, ok := body[1].([]any)
		if !ok {
			break
		}
		switch body[0] {
		case "object":
			out := make(map[string]any, len(items))
			refs[int(id)] = out
			for _, item := range items {
				pair, ok := item.([]any)
				if !ok || len(pair) != 2 {
					return nil, fmt.Errorf("invalid graph object")
				}
				key, ok := pair[0].(string)
				if !ok {
					return nil, fmt.Errorf("invalid graph object key")
				}
				parsed, err := decode(pair[1])
				if err != nil {
					return nil, err
				}
				out[key] = parsed
			}
			return out, nil
		case "array", "set":
			out := make([]any, len(items))
			if body[0] == "set" {
				refs[int(id)] = RuntimeSet(out)
			} else {
				refs[int(id)] = out
			}
			for i, item := range items {
				parsed, err := decode(item)
				if err != nil {
					return nil, err
				}
				out[i] = parsed
			}
			return refs[int(id)], nil
		case "map", "keyedObject":
			out := make(RuntimeMap, len(items))
			refs[int(id)] = out
			if body[0] == "keyedObject" {
				refs[int(id)] = RuntimeObject(out)
			}
			for i, item := range items {
				pair, ok := item.([]any)
				if !ok || len(pair) != 2 {
					return nil, fmt.Errorf("invalid graph map")
				}
				key, err := decode(pair[0])
				if err != nil {
					return nil, err
				}
				parsed, err := decode(pair[1])
				if err != nil {
					return nil, err
				}
				out[i] = RuntimeMapEntry{Key: key, Value: parsed}
			}
			return refs[int(id)], nil
		}
	case "string":
		if x, ok := value.(string); ok {
			return x, nil
		}
	case "boolean":
		if x, ok := value.(bool); ok {
			return x, nil
		}
	case "number":
		if x, ok := value.(float64); ok {
			return x, nil
		}
		switch value {
		case "NaN":
			return math.NaN(), nil
		case "Infinity":
			return math.Inf(1), nil
		case "-Infinity":
			return math.Inf(-1), nil
		case "-0":
			return math.Copysign(0, -1), nil
		}
	case "bigint":
		if s, ok := value.(string); ok {
			if x, valid := new(big.Int).SetString(s, 10); valid {
				return x, nil
			}
		}
	case "date":
		if s, ok := value.(string); ok {
			if s == "invalid" {
				return InvalidDate{}, nil
			}
			return time.Parse(time.RFC3339Nano, s)
		}
	case "handle":
		if v == nil {
			return nil, fmt.Errorf("zod2go: remote handles require a runtime validator")
		}
		if payload, ok := value.(map[string]any); ok {
			id, valid := payload["id"].(float64)
			handleKind, validKind := payload["kind"].(string)
			if valid && validKind {
				return &RuntimeValue{ID: int(id), Kind: handleKind, runtime: v}, nil
			}
		}
	case "array", "set":
		if values, ok := value.([]any); ok {
			out := make([]any, len(values))
			for i, item := range values {
				parsed, err := decode(item)
				if err != nil {
					return nil, err
				}
				out[i] = parsed
			}
			if kind == "set" {
				return RuntimeSet(out), nil
			}
			return out, nil
		}
	case "object", "map", "keyedObject":
		if values, ok := value.([]any); ok {
			obj := make(map[string]any, len(values))
			entries := make(RuntimeMap, 0, len(values))
			for _, entry := range values {
				pair, ok := entry.([]any)
				if !ok || len(pair) != 2 {
					return nil, fmt.Errorf("zod2go: malformed runtime entry")
				}
				val, err := decode(pair[1])
				if err != nil {
					return nil, err
				}
				if kind == "object" {
					key, ok := pair[0].(string)
					if !ok {
						return nil, fmt.Errorf("zod2go: invalid object key")
					}
					obj[key] = val
				} else {
					key, err := decode(pair[0])
					if err != nil {
						return nil, err
					}
					entries = append(entries, RuntimeMapEntry{Key: key, Value: val})
				}
			}
			if kind == "object" {
				return obj, nil
			}
			if kind == "keyedObject" {
				return RuntimeObject(entries), nil
			}
			return entries, nil
		}
	}
	return nil, fmt.Errorf("zod2go: invalid %s runtime payload", kind)
}

// EncodeTaggedValue and DecodeTaggedValue provide the typed transport without
// starting Node. Remote handles require a live RuntimeValidator instead.
func EncodeTaggedValue(value any) ([]byte, error) {
	wire, err := encodeRuntimeValue(value, nil)
	if err != nil {
		return nil, err
	}
	return json.Marshal(wire)
}
func DecodeTaggedValue(data []byte) (any, error) { return decodeRuntimeValue(data, nil) }

func (value *RuntimeValue) GetKey(key any) (any, error) {
	if value == nil || value.runtime == nil {
		return nil, fmt.Errorf("zod2go: detached runtime value")
	}
	return value.runtime.request("get", []any{key}, value.ID)
}
func (value *RuntimeValue) GetReference(key any) (any, error) {
	if value == nil || value.runtime == nil {
		return nil, fmt.Errorf("zod2go: detached runtime value")
	}
	return value.runtime.request("getReference", []any{key}, value.ID)
}
func (value *RuntimeValue) InvokeKey(key any, args ...any) (any, error) {
	if value == nil || value.runtime == nil {
		return nil, fmt.Errorf("zod2go: detached runtime value")
	}
	return value.runtime.request("invoke", append([]any{key}, args...), value.ID)
}
func (value *RuntimeValue) CallReference(args ...any) (any, error) {
	if value == nil || value.runtime == nil {
		return nil, fmt.Errorf("zod2go: detached runtime value")
	}
	return value.runtime.request("callReference", args, value.ID)
}

func (value *RuntimeValue) Await() (any, error) {
	if value == nil || value.runtime == nil {
		return nil, fmt.Errorf("zod2go: detached runtime value")
	}
	return value.runtime.request("await", nil, value.ID)
}

func (value *RuntimeValue) Construct(args ...any) (any, error) {
	if value == nil || value.runtime == nil {
		return nil, fmt.Errorf("zod2go: detached runtime value")
	}
	return value.runtime.request("construct", args, value.ID)
}
