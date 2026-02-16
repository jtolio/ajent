package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

type Decoder struct {
	lines *UnbufferedLineReader
	next  string
}

func NewDecoder(r io.Reader) *Decoder {
	return &Decoder{
		lines: NewUnbufferedLineReader(r, 10*1024*1024),
	}
}

func (d *Decoder) readLine() (string, error) {
	if d.next != "" {
		line := d.next
		d.next = ""
		return line, nil
	}
	return d.lines.ReadLine()
}

func (d *Decoder) Decode(v any) error {
	line, err := d.readLine()
	if err != nil {
		return err
	}

	intermediate := map[string]any{}
	if err := json.Unmarshal([]byte(line), &intermediate); err != nil {
		return err
	}

	for {
		line, err = d.readLine()
		if err != nil {
			if err == io.EOF {
				break
			}
			return err
		}
		if !strings.HasPrefix(line, ".") {
			d.next = line
			break
		}
		if err := d.readField(intermediate, line); err != nil {
			return err
		}
	}

	data, err := json.Marshal(intermediate)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, v)
}

func (d *Decoder) readField(m map[string]any, header string) error {
	header = strings.TrimSuffix(header, "\n")
	parts := strings.SplitN(header[1:], ": ", 2)
	if len(parts) != 2 {
		return fmt.Errorf("invalid field line: %s", header)
	}
	field, sep := parts[0], parts[1]

	var buf strings.Builder
	for {
		line, err := d.lines.ReadLine()
		if err != nil {
			return fmt.Errorf("reading value for %s: %w", field, err)
		}
		buf.WriteString(line)
		if s := buf.String(); strings.HasSuffix(s, sep+"\n") {
			d.setField(m, field, s[:len(s)-len(sep)-1])
			return nil
		}
	}
}

func (d *Decoder) setField(m map[string]any, field, val string) {
	parts := strings.SplitN(field, ".", 2)
	if len(parts) == 1 {
		m[field] = val
		return
	}
	sub, ok := m[parts[0]].(map[string]any)
	if !ok {
		sub = map[string]any{}
		m[parts[0]] = sub
	}
	if parts[0] == "tool_call" && parts[1] == "arguments" {
		sub[parts[1]] = base64.StdEncoding.EncodeToString([]byte(val))
	} else {
		sub[parts[1]] = val
	}
}
