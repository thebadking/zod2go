package zod2go_test

import (
	"github.com/thebadking/zod2go"
	"math/big"
	"testing"
)

func TestNativeBigIntPrecision(t *testing.T) {
	n, _ := new(big.Int).SetString("900719925474099312345", 10)
	v := zod2go.NewValidatorFromSchema(zod2go.Schema{Type: "bigint"})
	for _, input := range []any{n, *n, n.String()} {
		result := v.SafeParse(input)
		if !result.Success || result.Data != n.String() {
			t.Fatalf("unexpected result: %+v", result)
		}
	}
	if n.String() != "900719925474099312345" {
		t.Fatal("input mutated")
	}
	for _, input := range []any{nil, 1.5, 42, true, "1.5", "garbage"} {
		if result := v.SafeParse(input); result.Success {
			t.Fatalf("accepted invalid bigint: %v", input)
		}
	}
}

func TestTypedBigIntRejectsDecimalString(t *testing.T) {
	doc := mustParseInlineSchema(t, `{"$zod2go":{"version":1,"encoding":"tagged"},"schema":{"type":"bigint"}}`)
	v := zod2go.NewTypedValidator(doc)
	if result := v.SafeParse("123"); result.Success {
		t.Fatal("typed bigint accepted a string")
	}
	result := v.SafeParse(big.NewInt(123))
	if !result.Success || result.Data.(*big.Int).Cmp(big.NewInt(123)) != 0 {
		t.Fatalf("typed bigint lost: %+v", result)
	}
	encoded, err := zod2go.EncodeTaggedValue(big.NewInt(123))
	if err != nil {
		t.Fatal(err)
	}
	value, err := v.ParseJSON(encoded)
	if err != nil || value.(*big.Int).Cmp(big.NewInt(123)) != 0 {
		t.Fatalf("tagged ParseJSON: %v %v", value, err)
	}
}
