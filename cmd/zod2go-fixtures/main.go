// zod2go-fixtures validates a batch captured from the upstream fixture suite.
package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/thebadking/zod2go"
)

func main() {
	var requests []struct {
		Document zod2go.Document `json:"document"`
		Input    json.RawMessage `json:"input"`
	}
	if err := json.NewDecoder(os.Stdin).Decode(&requests); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	results := make([]map[string]any, len(requests))
	for i, request := range requests {
		input, err := zod2go.DecodeTaggedValue(request.Input)
		if err != nil {
			results[i] = map[string]any{"success": false, "transportError": err.Error()}
			continue
		}
		result := zod2go.NewTypedValidator(&request.Document).SafeParse(input)
		results[i] = map[string]any{"success": result.Success}
		if result.Success {
			encoded, err := zod2go.EncodeTaggedValue(result.Data)
			if err != nil {
				results[i]["success"] = false
				results[i]["transportError"] = err.Error()
			} else {
				results[i]["data"] = json.RawMessage(encoded)
			}
		} else {
			results[i]["error"] = result.Error.Error()
		}
	}
	if err := json.NewEncoder(os.Stdout).Encode(results); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
}
