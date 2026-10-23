package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/thebadking/zod2go"
)

func main() { os.Exit(run()) }

func run() int {
	schemaPath := flag.String("schema", "", "path to exported zod2go schema JSON")
	stream := flag.Bool("stream", false, "serve tagged runtime RPC requests from stdin as JSON lines")
	runtimeMode := flag.Bool("runtime", false, "execute the bundled original Zod schema in Node.js")
	tagged := flag.Bool("tagged", false, "read and write tagged runtime values")
	operation := flag.String("op", "parse", "runtime operation: parse, parseAsync, encode, encodeAsync")
	timeout := flag.Duration("timeout", 30*time.Second, "runtime timeout (0 disables timeout)")
	flag.Parse()

	if *schemaPath == "" {
		fmt.Fprintln(os.Stderr, "usage: zod2go -schema schema.json [json-input]")
		return 2
	}

	doc, err := zod2go.LoadDocument(*schemaPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	v := zod2go.NewValidator(doc)
	var runtimeValidator *zod2go.RuntimeValidator
	if !*runtimeMode && (*stream || (*operation != "parse" && *operation != "parseAsync")) {
		fmt.Fprintln(os.Stderr, "stream and encode operations require -runtime")
		return 2
	}
	nativeTagged := *tagged || doc.Zod2go.Encoding == "tagged"
	if nativeTagged && !*runtimeMode {
		v = zod2go.NewTypedValidator(doc)
	}
	if *runtimeMode {
		ctx := context.Background()
		if *timeout > 0 {
			var cancel context.CancelFunc
			ctx, cancel = context.WithTimeout(ctx, *timeout)
			defer cancel()
		}
		runtimeValidator, err = zod2go.NewRuntimeValidator(ctx, doc, zod2go.RuntimeOptions{Stderr: os.Stderr})
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		defer runtimeValidator.Close()
	}

	if *stream {
		if flag.NArg() > 0 {
			fmt.Fprintln(os.Stderr, "-stream accepts stdin only")
			return 2
		}
		reader := bufio.NewReader(os.Stdin)
		for {
			line, readErr := reader.ReadBytes('\n')
			if len(bytes.TrimSpace(line)) > 0 {
				response, err := runtimeValidator.ExecuteJSON(line)
				if err != nil {
					response, _ = json.Marshal(map[string]any{"success": false, "error": err.Error()})
				}
				if _, err := fmt.Fprintln(os.Stdout, string(response)); err != nil {
					fmt.Fprintln(os.Stderr, err)
					return 1
				}
			}
			if readErr != nil {
				if readErr == io.EOF {
					return 0
				}
				fmt.Fprintln(os.Stderr, readErr)
				return 1
			}
		}
	}
	var input any
	if flag.NArg() > 0 {
		if err := json.Unmarshal([]byte(flag.Arg(0)), &input); err != nil {
			fmt.Fprintln(os.Stderr, "invalid json input:", err)
			return 1
		}
	} else {
		if err := json.NewDecoder(os.Stdin).Decode(&input); err != nil {
			fmt.Fprintln(os.Stderr, "invalid json stdin:", err)
			return 1
		}
	}

	var out any
	if runtimeValidator != nil {
		if nativeTagged {
			data, marshalErr := json.Marshal(input)
			if marshalErr != nil {
				err = marshalErr
			} else {
				input, err = runtimeValidator.DecodeValue(data)
			}
		}
		if err == nil {
			switch *operation {
			case "parse":
				out, err = runtimeValidator.Parse(input)
			case "parseAsync":
				out, err = runtimeValidator.ParseAsync(input)
			case "encode":
				out, err = runtimeValidator.Encode(input)
			case "encodeAsync":
				out, err = runtimeValidator.EncodeAsync(input)
			default:
				err = fmt.Errorf("unknown runtime operation %q", *operation)
			}
		}
		if err == nil && nativeTagged {
			var data []byte
			data, err = runtimeValidator.EncodeValue(out)
			out = json.RawMessage(data)
		}
	} else {
		if nativeTagged {
			var data []byte
			data, err = json.Marshal(input)
			if err == nil {
				input, err = zod2go.DecodeTaggedValue(data)
			}
		}
		if err == nil {
			out, err = v.Parse(input)
		}
		if err == nil && nativeTagged {
			var data []byte
			data, err = zod2go.EncodeTaggedValue(out)
			out = json.RawMessage(data)
		}
	}
	if err != nil {
		if zerr, ok := err.(*zod2go.Error); ok {
			fmt.Println(zerr.JSON())
		} else if zerr, ok := err.(*zod2go.RuntimeError); ok {
			var data []byte
			if nativeTagged {
				data, _ = runtimeValidator.EncodeValue(map[string]any{"issues": zerr.Issues})
			} else {
				data, _ = json.Marshal(map[string]any{"issues": zerr.Issues})
			}
			fmt.Println(string(data))
		} else {
			fmt.Fprintln(os.Stderr, err)
		}
		return 1
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(out); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	return 0
}
