package zod2go_test

import (
	"encoding/base64"
	"encoding/json"
	"math/big"
	"os"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/thebadking/zod2go"
)

func assertInventoryCoverage(t *testing.T, field string, covered []string) {
	t.Helper()
	data, err := os.ReadFile("specs/upstream-inventory.json")
	if err != nil {
		t.Fatal(err)
	}
	var coveredInventory []string
	// Other inventory fields are scalars, so decode only the requested field.
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(raw[field], &coveredInventory); err != nil {
		t.Fatal(err)
	}
	sort.Strings(covered)
	if !reflect.DeepEqual(covered, coveredInventory) {
		t.Fatalf("%s coverage mismatch: %v", field, covered)
	}
}

func TestRuntimeEveryUpstreamCheckKind(t *testing.T) {
	type checkCase struct {
		name, schema   string
		valid, invalid any
		want           any
	}
	limit, _ := new(big.Int).SetString("9223372036854775808", 10)
	file := func(mime string) any { return zod2go.RuntimeFile{Data: []byte("hello"), Name: "a.txt", MIMEType: mime} }
	cases := []checkCase{
		{"bigint_format", `z.int64()`, big.NewInt(1), limit, nil},
		{"greater_than", `z.number().gt(0)`, float64(1), float64(0), nil},
		{"length_equals", `z.string().length(2)`, "hi", "x", nil},
		{"less_than", `z.number().lt(2)`, float64(1), float64(2), nil},
		{"max_length", `z.string().max(2)`, "hi", "long", nil},
		{"max_size", `z.set(z.string()).max(1)`, zod2go.RuntimeSet{"x"}, zod2go.RuntimeSet{"x", "y"}, nil},
		{"mime_type", `z.file().mime("text/plain")`, file("text/plain"), file("application/json"), nil},
		{"min_length", `z.string().min(2)`, "hi", "x", nil},
		{"min_size", `z.set(z.string()).min(2)`, zod2go.RuntimeSet{"x", "y"}, zod2go.RuntimeSet{"x"}, nil},
		{"multiple_of", `z.number().multipleOf(2)`, float64(4), float64(3), nil},
		{"number_format", `z.int32()`, float64(1), float64(2147483648), nil},
		{"overwrite", `z.string().trim()`, " hello ", 42, "hello"},
		{"property", `z.object({x:z.string()}).check(z.property("x",z.string().min(2)))`, map[string]any{"x": "hi"}, map[string]any{"x": "x"}, nil},
		{"size_equals", `z.set(z.string()).size(2)`, zod2go.RuntimeSet{"x", "y"}, zod2go.RuntimeSet{"x"}, nil},
		{"string_format", `z.uuid()`, "550e8400-e29b-41d4-a716-446655440000", "invalid", nil},
	}
	covered := make([]string, 0, len(cases))
	for _, c := range cases {
		covered = append(covered, c.name)
	}
	assertInventoryCoverage(t, "checkKinds", covered)
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			v := runtimeValidator(t, `exports.schema=`+c.schema+`;`)
			got, err := v.Parse(c.valid)
			if err != nil {
				t.Fatalf("valid: %v", err)
			}
			if c.want != nil && !reflect.DeepEqual(got, c.want) {
				t.Fatalf("got %#v; want %#v", got, c.want)
			}
			if _, err := v.Parse(c.invalid); err == nil {
				t.Fatal("invalid input accepted")
			}
		})
	}
}

func TestRuntimeEveryNamedStringFormat(t *testing.T) {
	cases := []struct{ name, schema, valid string }{
		{"base64", `z.base64()`, "aGVsbG8="},
		{"base64url", `z.base64url()`, "aGVsbG8"},
		{"cidrv4", `z.cidrv4()`, "192.168.0.0/24"},
		{"cidrv6", `z.cidrv6()`, "2001:db8::/32"},
		{"cuid", `z.cuid()`, "cjld2cyuq0000t3rmniod1foy"},
		{"cuid2", `z.cuid2()`, "abc123"},
		{"date", `z.iso.date()`, "2024-02-29"},
		{"datetime", `z.iso.datetime()`, "2024-01-01T12:34:56Z"},
		{"duration", `z.iso.duration()`, "P1DT2H"},
		{"e164", `z.e164()`, "+14155552671"},
		{"email", `z.email()`, "ada@example.com"},
		{"emoji", `z.emoji()`, "😀"},
		{"hex", `z.hex()`, "abcdef0123"},
		{"hostname", `z.hostname()`, "example.com"},
		{"guid", `z.guid()`, "550e8400-e29b-41d4-a716-446655440000"},
		{"ipv4", `z.ipv4()`, "192.168.1.1"},
		{"ipv6", `z.ipv6()`, "2001:db8::1"},
		{"jwt", `z.jwt()`, "eyJhbGciOiJIUzI1NiJ9.eyJzdWIiOiIxMjMifQ.c2lnbmF0dXJl"},
		{"ksuid", `z.ksuid()`, "000000000000000000000000000"},
		{"mac", `z.mac()`, "00:1A:2B:3C:4D:5E"},
		{"nanoid", `z.nanoid()`, "V1StGXR8_Z5jdHi6B-myT"},
		{"time", `z.iso.time()`, "12:34:56"},
		{"ulid", `z.ulid()`, "01ARZ3NDEKTSV4RRFFQ69G5FAV"},
		{"url", `z.url()`, "mailto:ada@example.com"},
		{"uuid", `z.uuid()`, "550e8400-e29b-41d4-a716-446655440000"},
		{"xid", `z.xid()`, "00000000000000000000"},
	}
	covered := make([]string, 0, len(cases))
	for _, c := range cases {
		covered = append(covered, c.name)
	}
	assertInventoryCoverage(t, "stringFormats", covered)
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			v := runtimeValidator(t, `exports.schema=`+c.schema+`;`)
			got, err := v.Parse(c.valid)
			if err != nil || got != c.valid {
				t.Fatalf("valid: %v %v", got, err)
			}
			if _, err := v.Parse("not valid!!"); err == nil {
				t.Fatal("invalid format accepted")
			}
		})
	}
}

func TestRuntimeEveryHashFormat(t *testing.T) {
	var covered []string
	for _, alg := range []struct {
		name  string
		bytes int
	}{{"md5", 16}, {"sha1", 20}, {"sha256", 32}, {"sha384", 48}, {"sha512", 64}} {
		for _, encoding := range []string{"hex", "base64", "base64url"} {
			name := alg.name + "_" + encoding
			covered = append(covered, name)
			t.Run(name, func(t *testing.T) {
				value := strings.Repeat("0", alg.bytes*2)
				if encoding == "base64" {
					value = base64.StdEncoding.EncodeToString(make([]byte, alg.bytes))
				}
				if encoding == "base64url" {
					value = base64.RawURLEncoding.EncodeToString(make([]byte, alg.bytes))
				}
				v := runtimeValidator(t, `exports.schema=z.hash("`+alg.name+`",{enc:"`+encoding+`"});`)
				if _, err := v.Parse(value); err != nil {
					t.Fatal(err)
				}
				if _, err := v.Parse("invalid!!"); err == nil {
					t.Fatal("invalid hash accepted")
				}
			})
		}
	}
	assertInventoryCoverage(t, "hashFormats", covered)
}
