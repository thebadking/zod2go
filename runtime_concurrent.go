package zod2go

import (
	"encoding/json"
	"fmt"
	"strconv"
)

type runtimeReply struct {
	response runtimeResponse
	err      error
}

func (v *RuntimeValidator) failPendingLocked(err error) {
	if v.terminal == nil {
		v.terminal = err
	}
	for id, waiter := range v.pending {
		delete(v.pending, id)
		waiter <- runtimeReply{err: err}
	}
}
func (v *RuntimeValidator) readResponses() {
	for {
		response, err := v.readResponse()
		v.mu.Lock()
		if err != nil {
			v.failPendingLocked(err)
			v.mu.Unlock()
			return
		}
		if response.ID == "" {
			v.failPendingLocked(fmt.Errorf("zod2go: missing runtime response id"))
			v.mu.Unlock()
			return
		}
		waiter := v.pending[response.ID]
		delete(v.pending, response.ID)
		v.mu.Unlock()
		if waiter != nil {
			waiter <- runtimeReply{response: response}
		}
	}
}
func (v *RuntimeValidator) requestConcurrent(operation string, input any, target int, params any) (any, error) {
	value, err := encodeRuntimeValue(input, v)
	if err != nil {
		return nil, err
	}
	request := map[string]any{"operation": operation, "value": value, "target": target}
	if params != nil {
		encoded, err := encodeRuntimeValue(params, v)
		if err != nil {
			return nil, err
		}
		request["params"] = encoded
	}
	v.mu.Lock()
	if v.closed {
		v.mu.Unlock()
		return nil, fmt.Errorf("zod2go: runtime validator is closed")
	}
	if v.terminal != nil {
		err := v.terminal
		v.mu.Unlock()
		return nil, err
	}
	v.nextRequest++
	id := strconv.FormatUint(v.nextRequest, 10)
	request["id"] = id
	waiter := make(chan runtimeReply, 1)
	v.pending[id] = waiter
	v.mu.Unlock()
	message, err := json.Marshal(request)
	if err == nil {
		v.writeMu.Lock()
		_, err = v.stdin.Write(append(message, '\n'))
		v.writeMu.Unlock()
	}
	if err != nil {
		v.mu.Lock()
		delete(v.pending, id)
		v.mu.Unlock()
		return nil, err
	}
	reply := <-waiter
	if reply.err != nil {
		return nil, reply.err
	}
	return v.responseResult(reply.response)
}
