package zod2go

import (
	"bufio"
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
)

//go:embed node-runtime.cjs
var nodeRuntime string

// RuntimeOptions explicitly opts into executing a bundled JavaScript schema.
type RuntimeOptions struct {
	WorkingDirectory string
	NodeBinary       string
	Stdout           io.Writer
	Stderr           io.Writer
}

// RuntimeValidator preserves the original Zod module, closures, and runtime state.
// Close it to release the worker process and temporary module.
type RuntimeValidator struct {
	mu          sync.Mutex
	writeMu     sync.Mutex
	multiplex   bool
	nextRequest uint64
	pending     map[string]chan runtimeReply
	terminal    error
	cmd         *exec.Cmd
	cancel      context.CancelFunc
	stdin       io.WriteCloser
	results     io.Closer
	reader      *bufio.Reader
	dir         string
	closed      bool
}

// RuntimeError preserves all original Zod issue metadata, including nested issues.
type RuntimeError struct {
	Issues  []any
	Message string
	Name    string
	Thrown  bool
	Value   any
	wire    json.RawMessage
}

func (e *RuntimeError) Error() string {
	if e.Message != "" {
		return e.Message
	}
	data, err := json.Marshal(e.Issues)
	if err != nil {
		return string(e.wire)
	}
	return string(data)
}

// RuntimeException preserves a JavaScript exception separately from validation issues.
type RuntimeException struct {
	Name, Message, Stack string
	Value                any
}

func (e *RuntimeException) Error() string { return e.Name + ": " + e.Message }

type runtimeResponse struct {
	ID               string          `json:"id"`
	Success          bool            `json:"success"`
	Data             json.RawMessage `json:"data"`
	Issues           json.RawMessage `json:"issues"`
	Exception        string          `json:"exception"`
	ExceptionName    string          `json:"exceptionName"`
	ExceptionMessage string          `json:"exceptionMessage"`
	Message          string          `json:"message"`
	ErrorName        string          `json:"errorName"`
	ExceptionValue   json.RawMessage `json:"exceptionValue"`
	Thrown           bool            `json:"thrown"`
}

func NewRuntimeValidator(ctx context.Context, doc *Document, options RuntimeOptions) (*RuntimeValidator, error) {
	if doc == nil || doc.Runtime == nil || (doc.Runtime.Bundle == "" && doc.Runtime.ModulePath == "") || doc.Runtime.ExportName == "" {
		return nil, fmt.Errorf("zod2go: missing runtime module")
	}
	dir, err := os.MkdirTemp("", "zod2go-runtime-")
	if err != nil {
		return nil, err
	}
	format := doc.Runtime.Format
	if format == "" {
		format = "cjs"
	}
	if format != "cjs" && format != "esm" {
		os.RemoveAll(dir)
		return nil, fmt.Errorf("zod2go: unsupported runtime module format %q", format)
	}
	filename := "schema.cjs"
	if format == "esm" {
		filename = "schema.mjs"
	}
	bundlePath := filepath.Join(dir, filename)
	if doc.Runtime.ModulePath != "" {
		bundlePath, err = filepath.Abs(doc.Runtime.ModulePath)
		if err != nil {
			os.RemoveAll(dir)
			return nil, err
		}
	} else if err = os.WriteFile(bundlePath, []byte(doc.Runtime.Bundle), 0600); err != nil {
		os.RemoveAll(dir)
		return nil, err
	}
	resultRead, resultWrite, err := os.Pipe()
	if err != nil {
		os.RemoveAll(dir)
		return nil, err
	}
	workerContext, cancel := context.WithCancel(ctx)
	node := options.NodeBinary
	if node == "" {
		node = "node"
	}
	cmd := exec.CommandContext(workerContext, node, "-e", nodeRuntime, bundlePath, doc.Runtime.ExportName, format, doc.Runtime.Loader, doc.Runtime.CoreModule)
	cmd.Dir = options.WorkingDirectory
	cmd.ExtraFiles = []*os.File{resultWrite}
	cmd.Stdout = options.Stdout
	cmd.Stderr = options.Stderr
	stdin, err := cmd.StdinPipe()
	if err != nil {
		cancel()
		resultRead.Close()
		resultWrite.Close()
		os.RemoveAll(dir)
		return nil, err
	}
	v := &RuntimeValidator{cmd: cmd, cancel: cancel, stdin: stdin, results: resultRead, reader: bufio.NewReader(resultRead), dir: dir}
	if err = cmd.Start(); err != nil {
		stdin.Close()
		cancel()
		resultRead.Close()
		resultWrite.Close()
		os.RemoveAll(dir)
		return nil, err
	}
	resultWrite.Close()
	response, err := v.readResponse()
	if err != nil {
		v.Close()
		return nil, err
	}
	if !response.Success {
		v.Close()
		return nil, fmt.Errorf("zod2go: initialize runtime: %s", response.Exception)
	}
	v.multiplex = true
	v.pending = make(map[string]chan runtimeReply)
	go v.readResponses()
	return v, nil
}

func (v *RuntimeValidator) Close() error {
	if v.cancel != nil {
		v.cancel()
	}
	if v.results != nil {
		v.results.Close()
	}
	v.mu.Lock()
	defer v.mu.Unlock()
	if v.closed {
		return nil
	}
	v.closed = true
	v.failPendingLocked(fmt.Errorf("zod2go: runtime validator is closed"))
	if v.stdin != nil {
		v.stdin.Close()
	}
	if v.cancel != nil {
		v.cancel()
	}
	if v.results != nil {
		v.results.Close()
	}
	if v.cmd != nil {
		v.cmd.Wait()
	}
	return os.RemoveAll(v.dir)
}

func (v *RuntimeValidator) readResponse() (runtimeResponse, error) {
	var response runtimeResponse
	line, err := v.reader.ReadBytes('\n')
	if err != nil {
		return response, fmt.Errorf("zod2go: runtime protocol: %w", err)
	}
	if err = json.Unmarshal(line, &response); err != nil {
		return response, fmt.Errorf("zod2go: runtime response: %w", err)
	}
	return response, nil
}

func (v *RuntimeValidator) request(operation string, input any, target int) (any, error) {
	return v.requestWithParams(operation, input, target, nil)
}
func (v *RuntimeValidator) requestWithParams(operation string, input any, target int, params any) (any, error) {
	if v.multiplex {
		return v.requestConcurrent(operation, input, target, params)
	}
	v.mu.Lock()
	defer v.mu.Unlock()
	if v.closed {
		return nil, fmt.Errorf("zod2go: runtime validator is closed")
	}
	wire, err := encodeRuntimeValue(input, v)
	if err != nil {
		return nil, err
	}
	request := map[string]any{"operation": operation, "value": wire, "target": target}
	if params != nil {
		encoded, err := encodeRuntimeValue(params, v)
		if err != nil {
			return nil, err
		}
		request["params"] = encoded
	}
	message, err := json.Marshal(request)
	if err != nil {
		return nil, err
	}
	if _, err = v.stdin.Write(append(message, '\n')); err != nil {
		return nil, err
	}
	response, err := v.readResponse()
	if err != nil {
		return nil, err
	}
	return v.responseResult(response)
}
func (v *RuntimeValidator) responseResult(response runtimeResponse) (any, error) {
	var err error
	if response.Exception != "" || response.ExceptionName != "" {
		var value any
		if len(response.ExceptionValue) > 0 {
			value, err = decodeRuntimeValue(response.ExceptionValue, v)
			if err != nil {
				return nil, err
			}
		}
		return nil, &RuntimeException{Name: response.ExceptionName, Message: response.ExceptionMessage, Stack: response.Exception, Value: value}
	}
	if !response.Success {
		issues, err := decodeRuntimeValue(response.Issues, v)
		if err != nil {
			return nil, err
		}
		list, ok := issues.([]any)
		if !ok {
			return nil, fmt.Errorf("zod2go: malformed runtime issues")
		}
		var thrown any
		if len(response.ExceptionValue) > 0 {
			thrown, err = decodeRuntimeValue(response.ExceptionValue, v)
			if err != nil {
				return nil, err
			}
		}
		return nil, &RuntimeError{Issues: list, wire: response.Issues, Message: response.Message, Name: response.ErrorName, Thrown: response.Thrown, Value: thrown}
	}
	return decodeRuntimeValue(response.Data, v)
}

func (v *RuntimeValidator) Parse(input any) (any, error) { return v.request("parse", input, 0) }
func (v *RuntimeValidator) ParseAsync(input any) (any, error) {
	return v.request("parseAsync", input, 0)
}
func (v *RuntimeValidator) Encode(input any) (any, error) { return v.request("encode", input, 0) }
func (v *RuntimeValidator) EncodeAsync(input any) (any, error) {
	return v.request("encodeAsync", input, 0)
}

// DecodeValue reads the tagged transport, preserving values JSON cannot represent.
func (v *RuntimeValidator) DecodeValue(data []byte) (any, error) { return decodeRuntimeValue(data, v) }

// EncodeValue writes the tagged transport, including handles owned by this worker.
func (v *RuntimeValidator) EncodeValue(value any) ([]byte, error) {
	wire, err := encodeRuntimeValue(value, v)
	if err != nil {
		return nil, err
	}
	return json.Marshal(wire)
}

// ExportValue retrieves values from the original module, preserving their identity.
func (v *RuntimeValidator) ExportValue(name string) (any, error) { return v.request("export", name, 0) }

func (v *RuntimeValidator) ParseWithParams(input any, params map[string]any) (any, error) {
	return v.requestWithParams("parse", input, 0, params)
}
func (v *RuntimeValidator) ParseAsyncWithParams(input any, params map[string]any) (any, error) {
	return v.requestWithParams("parseAsync", input, 0, params)
}
func (v *RuntimeValidator) EncodeWithParams(input any, params map[string]any) (any, error) {
	return v.requestWithParams("encode", input, 0, params)
}
func (v *RuntimeValidator) EncodeAsyncWithParams(input any, params map[string]any) (any, error) {
	return v.requestWithParams("encodeAsync", input, 0, params)
}

// ExecuteJSON handles one tagged RPC request. It supports persistent CLI clients,
// including handles returned by prior requests to this validator.
func (v *RuntimeValidator) ExecuteJSON(data []byte) ([]byte, error) {
	var request struct {
		Operation string          `json:"operation"`
		Value     json.RawMessage `json:"value"`
		Params    json.RawMessage `json:"params"`
		Target    int             `json:"target"`
	}
	if err := json.Unmarshal(data, &request); err != nil {
		return nil, err
	}
	var input any = Undefined
	var params any
	var err error
	if len(request.Value) > 0 {
		input, err = v.DecodeValue(request.Value)
		if err != nil {
			return nil, err
		}
	}
	if len(request.Params) > 0 {
		params, err = v.DecodeValue(request.Params)
		if err != nil {
			return nil, err
		}
	}
	output, err := v.requestWithParams(request.Operation, input, request.Target, params)
	response := map[string]any{"success": err == nil}
	if err == nil {
		wire, encodeErr := encodeRuntimeValue(output, v)
		if encodeErr != nil {
			return nil, encodeErr
		}
		response["data"] = wire
	} else {
		switch e := err.(type) {
		case *RuntimeError:
			wire, encodeErr := encodeRuntimeValue(e.Issues, v)
			if encodeErr != nil {
				return nil, encodeErr
			}
			response["issues"] = wire
			response["message"] = e.Message
			response["errorName"] = e.Name
			response["thrown"] = e.Thrown
		case *RuntimeException:
			response["exception"] = map[string]any{"name": e.Name, "message": e.Message, "stack": e.Stack}
		default:
			return nil, err
		}
	}
	return json.Marshal(response)
}

func (v *RuntimeValidator) NewSymbol(description string) (*RuntimeValue, error) {
	value, err := v.request("symbol", description, 0)
	if err != nil {
		return nil, err
	}
	handle, ok := value.(*RuntimeValue)
	if !ok {
		return nil, fmt.Errorf("zod2go: runtime did not return a symbol")
	}
	return handle, nil
}
func (v *RuntimeValidator) GlobalSymbol(key string) (*RuntimeValue, error) {
	value, err := v.request("globalSymbol", key, 0)
	if err != nil {
		return nil, err
	}
	handle, ok := value.(*RuntimeValue)
	if !ok {
		return nil, fmt.Errorf("zod2go: runtime did not return a symbol")
	}
	return handle, nil
}
func (v *RuntimeValidator) UTF16String(units []uint16) (any, error) {
	values := make([]any, len(units))
	for i, unit := range units {
		values[i] = float64(unit)
	}
	return v.request("utf16", values, 0)
}
func (v *RuntimeValidator) ParseReference(input any) (any, error) {
	return v.request("parseReference", input, 0)
}
func (v *RuntimeValidator) ParseAsyncReference(input any) (any, error) {
	return v.request("parseAsyncReference", input, 0)
}
func (v *RuntimeValidator) EncodeReference(input any) (any, error) {
	return v.request("encodeReference", input, 0)
}
func (v *RuntimeValidator) EncodeAsyncReference(input any) (any, error) {
	return v.request("encodeAsyncReference", input, 0)
}

// NewRuntimeConnection attaches the same typed client to an existing runtime
// transport. The peer owns the JavaScript schemas and their lexical context.
func NewRuntimeConnection(input io.WriteCloser, output io.Reader) *RuntimeValidator {
	reader, ok := output.(*bufio.Reader)
	if !ok {
		reader = bufio.NewReader(output)
	}
	var closer io.Closer
	if candidate, ok := output.(io.Closer); ok {
		closer = candidate
	}
	return &RuntimeValidator{stdin: input, reader: reader, results: closer}
}

// RunReference executes a live core schema and returns its original parse payload.
func (v *RuntimeValidator) RunReference(schema *RuntimeValue, input any, params any) (any, error) {
	if schema == nil || schema.runtime != v {
		return nil, fmt.Errorf("zod2go: schema belongs to a different runtime")
	}
	return v.requestWithParams("runReference", input, schema.ID, params)
}

// RuntimeSafeResult distinguishes ordinary validation failures from exceptions
// thrown by callbacks, matching Zod's safe-parse contract without Go panics.
type RuntimeSafeResult struct {
	Success   bool
	Data      any
	Error     *RuntimeError
	Exception error
}

func runtimeSafeResult(data any, err error) RuntimeSafeResult {
	if err == nil {
		return RuntimeSafeResult{Success: true, Data: data}
	}
	if failure, ok := err.(*RuntimeError); ok && !failure.Thrown {
		return RuntimeSafeResult{Error: failure}
	}
	return RuntimeSafeResult{Exception: err}
}
func (v *RuntimeValidator) SafeParse(input any) RuntimeSafeResult {
	data, err := v.Parse(input)
	return runtimeSafeResult(data, err)
}
func (v *RuntimeValidator) SafeParseAsync(input any) RuntimeSafeResult {
	data, err := v.ParseAsync(input)
	return runtimeSafeResult(data, err)
}
func (v *RuntimeValidator) SafeEncode(input any) RuntimeSafeResult {
	data, err := v.Encode(input)
	return runtimeSafeResult(data, err)
}
func (v *RuntimeValidator) SafeEncodeAsync(input any) RuntimeSafeResult {
	data, err := v.EncodeAsync(input)
	return runtimeSafeResult(data, err)
}
