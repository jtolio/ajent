package hjl

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

// Encoder will write zero or more objects to an outgoing stream encoded in
// Heredoc JSON Lines format.
type Encoder struct {
	w io.Writer
}

// NewEncoder creates an Encoder that will write to w.
func NewEncoder(w io.Writer) *Encoder {
	return &Encoder{w: w}
}

// Encode will add another object v to the output stream with the provided
// heredocFields (if they exist) encoded in heredoc style.
func (e *Encoder) Encode(v any, heredocFields ...string) error {

	// TODO: this is gross, to encode and then parse JSON to get normalized
	// fields, but it's easy to reason about. For performance we need to fix
	// this.
	encoded, err := json.Marshal(v)
	if err != nil {
		return err
	}
	intermediate := map[string]any{}
	if err = json.Unmarshal(encoded, &intermediate); err != nil {
		return err
	}

	heredocs := map[string]string{}

	for _, field := range heredocFields {
		parts := strings.Split(field, ".")
		obj := intermediate

		// traverse parent objects to get to the direct owning object
		for len(parts) > 1 {
			if subobj, ok := obj[parts[0]].(map[string]any); ok {
				obj = subobj
				parts = parts[1:]
			} else {
				break
			}
		}

		if len(parts) == 1 {
			// we found it, set it in the heredocs map and delete
			// it from the intermediate.
			if val, ok := obj[parts[0]].(string); ok {
				heredocs[field] = val
				delete(obj, parts[0])
			}
		}
	}

	return e.write(intermediate, heredocs)
}

func (e *Encoder) write(obj map[string]any, heredocs map[string]string) error {
	if err := json.NewEncoder(e.w).Encode(obj); err != nil {
		return err
	}
	for field, val := range heredocs {
		sep := e.newSep(val)
		_, err := fmt.Fprintf(e.w, ".%s = <<%s\n%s%s\n", field, sep, val, sep)
		if err != nil {
			return err
		}
	}
	return nil
}

func (e *Encoder) newSep(val string) string {
	i := 0
	for {
		sep := fmt.Sprintf("END%d", i)
		if !strings.Contains(val, sep) {
			return sep
		}
		i++
	}
}
