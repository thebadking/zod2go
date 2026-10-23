// zod2go-runtime-fixtures drives original upstream core schemas through the
// production Go runtime client over a live, bidirectional test transport.
package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/thebadking/zod2go"
)

func main() {
	reader := bufio.NewReader(os.Stdin)
	client := zod2go.NewRuntimeConnection(os.Stdout, reader)
	defer client.Close()
	for {
		line, err := reader.ReadBytes('\n')
		if err == io.EOF {
			return
		}
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(2)
		}
		var request struct {
			Schema json.RawMessage `json:"schema"`
			Input  json.RawMessage `json:"input"`
			Params json.RawMessage `json:"params"`
		}
		if err := json.Unmarshal(line, &request); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(2)
		}
		schema, err := client.DecodeValue(request.Schema)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(2)
		}
		input, err := client.DecodeValue(request.Input)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(2)
		}
		params, err := client.DecodeValue(request.Params)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(2)
		}
		output, err := client.RunReference(schema.(*zod2go.RuntimeValue), input, params)
		response := map[string]any{"type": "result", "success": err == nil}
		if err == nil {
			data, encodeErr := client.EncodeValue(output)
			if encodeErr != nil {
				fmt.Fprintln(os.Stderr, encodeErr)
				os.Exit(2)
			}
			response["data"] = json.RawMessage(data)
		} else {
			exception, ok := err.(*zod2go.RuntimeException)
			if !ok {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(2)
			}
			value, encodeErr := client.EncodeValue(exception.Value)
			if encodeErr != nil {
				fmt.Fprintln(os.Stderr, encodeErr)
				os.Exit(2)
			}
			response["exception"] = json.RawMessage(value)
		}
		if err := json.NewEncoder(os.Stdout).Encode(response); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(2)
		}
	}
}
