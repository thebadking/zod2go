package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/thebadking/zod2go"
)

func main() {
	doc, err := zod2go.LoadDocument("testdata/user.schema.json")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	v := zod2go.NewValidator(doc)

	out, err := v.Parse(map[string]any{
		"name":  "Grace Hopper",
		"email": "grace@example.com",
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	_ = enc.Encode(out)
}
