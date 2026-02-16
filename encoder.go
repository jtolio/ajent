package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

type Encoder struct {
	w io.Writer
}

func NewEncoder(w io.Writer) *Encoder {
	return &Encoder{w: w}
}

func (e *Encoder) Encode(v any) error {
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(v); err != nil {
		return err
	}
	intermediate := map[string]any{}
	if err := json.NewDecoder(&buf).Decode(&intermediate); err != nil {
		return err
	}

	for _, key := range []string{"text", "system_prompt"} {
		if val, exists := intermediate[key].(string); exists {
			delete(intermediate, key)
			return e.write(intermediate, key, val)
		}
	}

	if toolResp, exists := intermediate["tool_response"].(map[string]any); exists {
		if content, exists := toolResp["content"].(string); exists {
			delete(toolResp, "content")
			return e.write(intermediate, "tool_response.content", content)
		}
	}

	if toolCall, exists := intermediate["tool_call"].(map[string]any); exists {
		if args, exists := toolCall["arguments"].(string); exists {
			if binargs, err := base64.StdEncoding.DecodeString(args); err == nil {
				delete(toolCall, "arguments")
				return e.write(intermediate, "tool_call.arguments", string(binargs))
			}
		}
	}

	return json.NewEncoder(e.w).Encode(intermediate)
}

func (e *Encoder) newSep(val string) string {
	i := 0
	for {
		sep := fmt.Sprintf("--%d--", i)
		if !strings.Contains(val, sep) {
			return sep
		}
		i++
	}
}

func (e *Encoder) write(rest map[string]any, field, val string) error {
	if err := json.NewEncoder(e.w).Encode(rest); err != nil {
		return err
	}
	sep := e.newSep(val)
	_, err := fmt.Fprintf(e.w, ".%s: %s\n%s%s\n", field, sep, val, sep)
	return err
}
