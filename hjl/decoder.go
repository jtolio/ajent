package hjl

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/jtolio/ajent/private"
)

// Decoder will decode JSON objects from a Heredoc JSON Lines formatted stream.
type Decoder struct {
	lines *private.UnbufferedLineReader
	next  *string
}

// NewDecoder will create a Decoder from r.
func NewDecoder(r io.Reader) *Decoder {
	return &Decoder{
		lines: private.NewUnbufferedLineReader(r, -1),
	}
}

func (d *Decoder) readLine() (string, error) {
	if d.next != nil {
		// return peeked line if any
		line := *d.next
		d.next = nil
		return line, nil
	}
	for {
		line, err := d.lines.ReadLine()
		// skip comments
		if err != nil || !strings.HasPrefix(line, "#") {
			return line, err
		}
	}
}

// Decode will pull the next object off the stream, and use encoding/json's
// rules for writing the data to v.
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
		line, err := d.readLine()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return err
		}
		if !strings.HasPrefix(line, ".") {
			d.next = &line
			break
		}
		if err := d.readField(intermediate, line); err != nil {
			return err
		}
	}

	// TODO: like Encoder.Encode, this is gross. For performance we need to
	// fix this.
	encoded, err := json.Marshal(intermediate)
	if err != nil {
		return err
	}
	return json.Unmarshal(encoded, v)
}

func (d *Decoder) readField(m map[string]any, header string) error {
	parts := strings.SplitN(header[1:], "=", 2)
	if len(parts) != 2 {
		return fmt.Errorf("invalid field line: %s", header)
	}
	field, sep := strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1])
	if !strings.HasPrefix(sep, "<<") {
		return fmt.Errorf("invalid field line: %s", header)
	}
	sep = strings.TrimSpace(strings.TrimPrefix(sep, "<<"))

	var buf strings.Builder
	for {
		line, err := d.lines.ReadLine()
		if err != nil {
			return fmt.Errorf("reading value for %s: %w", field, err)
		}
		buf.WriteString(line)
		if s := buf.String(); strings.HasSuffix(s, sep+"\n") {
			return d.setField(m, field, s[:len(s)-len(sep)-1])
		}
	}
}

func (d *Decoder) setField(m map[string]any, field, val string) error {
	parts := strings.Split(field, ".")
	obj := m
	for len(parts) > 1 {
		var sub map[string]any
		lookup, ok := obj[parts[0]]
		if ok {
			sub, ok = lookup.(map[string]any)
			if !ok {
				return fmt.Errorf("invalid redefinition of %q", field)
			}
		} else {
			sub = map[string]any{}
			obj[parts[0]] = sub
		}
		obj = sub
		parts = parts[1:]
	}
	if _, exists := obj[parts[0]]; exists {
		return fmt.Errorf("redefinition of %q", field)
	}
	obj[parts[0]] = val
	return nil
}
