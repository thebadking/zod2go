package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/thebadking/zod2go"
)

func main() {
	schemaPath := flag.String("schema", "", "path to exported zod2go schema JSON")
	flag.Parse()

	if *schemaPath == "" {
		fmt.Fprintln(os.Stderr, "usage: zod2go -schema schema.json [json-input]")
		os.Exit(2)
	}

	doc, err := zod2go.LoadDocument(*schemaPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	v := zod2go.NewValidator(doc)

	var input any
	if flag.NArg() > 0 {
		if err := json.Unmarshal([]byte(flag.Arg(0)), &input); err != nil {
			fmt.Fprintln(os.Stderr, "invalid json input:", err)
			os.Exit(1)
		}
	} else {
		if err := json.NewDecoder(os.Stdin).Decode(&input); err != nil {
			fmt.Fprintln(os.Stderr, "invalid json stdin:", err)
			os.Exit(1)
		}
	}

	out, err := v.Parse(input)
	if err != nil {
		if zerr, ok := err.(*zod2go.Error); ok {
			fmt.Println(zerr.JSON())
		} else {
			fmt.Fprintln(os.Stderr, err)
		}
		os.Exit(1)
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	_ = enc.Encode(out)
}
