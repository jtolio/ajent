// Package hjl implements Encoding and Decoding support for the "Heredoc JSON
// Lines" format.
//
// Heredoc JSON Lines is an extension to the JSON Lines (jsonl) format that adds
// support for field definitions via heredocs.
//
// JSON Lines (jsonl) is defined as data where individual JSON objects are
// newline delimited.
// https://en.wikipedia.org/wiki/JSON_streaming#Newline-delimited_JSON
//
// Heredocs are most commonly seen in bash and other shell scripting languages,
// where a multiline value can be specified inline, via `<< EOF` or similar.
// https://en.wikipedia.org/wiki/Here_document
//
// An "hjl" file is thus defined as a collection of zero or more JSON objects,
// newline delimited, where each JSON object is followed by zero or more
// additional Heredoc style field definitions.
//
// Here is an example:
//
//	{"type": "object1"}
//	.text = <<END
//	Here is
//	some
//	multiline
//	data
//	END
//	{"type": "object2"}
//
// This document parses to two JSON objects, the first being:
//
//	{"type": "object1", "text": "Here is\nsome\nmultiline\ndata\n"}
//
// Subobjects can also be defined:
//
//	{"type": "object", "sub": {}}
//	.sub.text = <<END
//	value
//	END
//
// A note about newlines: it is expected that the delimiting identifier ends
// with a newline. So here is the value "":
//
//	{}
//	.text = <<END
//	END
//	{}
//
// Here is the value "\n":
//
//	{}
//	.text = <<END
//
//	END
//	{}
//
// Here is the value "Hello, world!":
//
//	{}
//	.text = <<END
//	Hello, world!END
//	{}
//
// Note that the above is intended as a way to preserve exact storage regarding
// trailing newlines.
//
// Encoding with the Go library is straightforward, but which fields are
// translated to heredoc style definition does require specification. If the
// specified heredoc fields are missing in the source object, they are skipped.
//
//	fh, err := os.Create("file")
//	if err != nil {
//	    return err
//	}
//	enc := hjl.NewEncoder(fh)
//
//	err = enc.Encode(obj1, "text")
//	if err != nil {
//	    return err
//	}
//	err = enc.Encode(obj2, "text")
//	if err != nil {
//	    return err
//	}
//
//	return fh.Close()
//
// Decoding is similar:
//
//	fh, err := os.Open("file")
//	if err != nil {
//	    return err
//	}
//	defer fh.Close()
//
//	dec := hjl.NewDecoder(fh)
//
//	var obj1 Object
//	err = dec.Decode(&obj1)
//	if err != nil {
//	    return err
//	}
//	var obj2 Object
//	err = dec.Decode(&obj2)
//	if err != nil {
//	    return err
//	}
package hjl
