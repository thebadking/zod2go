package zod2go

import (
	"fmt"
	"math"
	"math/big"
	"strings"
	"time"
)

// Big integers use decimal strings on the JSON wire so precision is preserved.
// Native Go callers may also provide big.Int or *big.Int values.
func (v *Validator) runBigInt(input any, schema Schema, path []any) ([]Issue, any) {
	if v.typed && !schema.Coerce {
		switch input.(type) {
		case *big.Int, big.Int:
		default:
			return []Issue{{Code: "invalid_type", Expected: "bigint", Path: path, Input: input}}, nil
		}
	}
	n, ok := parseBigInt(input, schema.Coerce)
	if !ok {
		return []Issue{{Code: "invalid_type", Expected: "bigint", Path: path, Input: input, Message: "Invalid input: expected bigint"}}, nil
	}
	for _, chk := range schema.Checks {
		if chk.Check == "bigint_format" {
			min, max := new(big.Int), new(big.Int)
			switch chk.Format {
			case "int64":
				min.SetString("-9223372036854775808", 10)
				max.SetString("9223372036854775807", 10)
			case "uint64":
				min.SetInt64(0)
				max.SetString("18446744073709551615", 10)
			default:
				return []Issue{newIssue("custom", "Unknown bigint format: "+chk.Format, path, input)}, nil
			}
			if n.Cmp(min) < 0 {
				return []Issue{{Code: "too_small", Origin: "bigint", Minimum: min.String(), Inclusive: true, Path: path, Input: input}}, nil
			}
			if n.Cmp(max) > 0 {
				return []Issue{{Code: "too_big", Origin: "bigint", Maximum: max.String(), Inclusive: true, Path: path, Input: input}}, nil
			}
			continue
		}
		bound, valid := parseBigInt(chk.Value, false)
		if !valid {
			return []Issue{newIssue("custom", "Invalid bigint check value", path, input)}, nil
		}
		cmp := n.Cmp(bound)
		switch chk.Check {
		case "greater_than":
			if cmp < 0 || (cmp == 0 && !chk.Inclusive) {
				return []Issue{{Code: "too_small", Origin: "bigint", Minimum: bound.String(), Inclusive: chk.Inclusive, Path: path, Input: input}}, nil
			}
		case "less_than":
			if cmp > 0 || (cmp == 0 && !chk.Inclusive) {
				return []Issue{{Code: "too_big", Origin: "bigint", Maximum: bound.String(), Inclusive: chk.Inclusive, Path: path, Input: input}}, nil
			}
		case "multiple_of":
			if bound.Sign() == 0 || new(big.Int).Rem(n, bound).Sign() != 0 {
				return []Issue{{Code: "not_multiple_of", Origin: "bigint", Divisor: bound.String(), Path: path, Input: input}}, nil
			}
		default:
			return []Issue{newIssue("custom", fmt.Sprintf("Unsupported bigint check %q", chk.Check), path, input)}, nil
		}
	}
	if v.typed {
		return nil, n
	}
	return nil, n.String()
}

func parseBigInt(input any, coerce bool) (*big.Int, bool) {
	switch x := input.(type) {
	case *big.Int:
		if x == nil {
			return nil, false
		}
		return new(big.Int).Set(x), true
	case big.Int:
		return new(big.Int).Set(&x), true
	case string:
		s := x
		if coerce {
			s = jsTrim(s)
			if s == "" {
				return new(big.Int), true
			}
		}
		base := 10
		if coerce && len(s) > 2 && s[0] == '0' && strings.ContainsAny(s[1:2], "xXoObB") {
			base = 0
		}
		return new(big.Int).SetString(s, base)
	case []any:
		if coerce {
			return parseBigInt(jsString(x), true)
		}
	case time.Time:
		if coerce {
			return big.NewInt(x.UnixMilli()), true
		}
	case *time.Time:
		if coerce && x != nil {
			return big.NewInt(x.UnixMilli()), true
		}
	case bool:
		if coerce {
			if x {
				return big.NewInt(1), true
			}
			return new(big.Int), true
		}
	default:
		if coerce {
			if n, ok := toFloat64(input); ok && !math.IsNaN(n) && !math.IsInf(n, 0) && math.Trunc(n) == n {
				result, _ := new(big.Float).SetFloat64(n).Int(nil)
				return result, true
			}
		}
	}
	return nil, false
}
