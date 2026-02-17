package hjl

import (
	"bytes"
	"io"
	"strings"
	"testing"
)

// helper types for decoding
type basicObj struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
}

type nestedObj struct {
	Type string `json:"type"`
	Sub  subObj `json:"sub"`
}

type subObj struct {
	Text string `json:"text,omitempty"`
	More string `json:"more,omitempty"`
}

type byteFieldObj struct {
	Type string `json:"type"`
	Data []byte `json:"data,omitempty"`
}

type nestedByteObj struct {
	Type string     `json:"type"`
	Sub  byteSubObj `json:"sub"`
}

type byteSubObj struct {
	Data []byte `json:"data,omitempty"`
}

// --- Encoder tests ---

func TestEncodeBasicNoHeredoc(t *testing.T) {
	var buf bytes.Buffer
	enc := NewEncoder(&buf)
	err := enc.Encode(basicObj{Type: "hello"})
	if err != nil {
		t.Fatal(err)
	}
	got := buf.String()
	want := `{"type":"hello"}` + "\n"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestEncodeWithHeredocField(t *testing.T) {
	var buf bytes.Buffer
	enc := NewEncoder(&buf)
	err := enc.Encode(basicObj{Type: "obj", Text: "line1\nline2\n"}, "text")
	if err != nil {
		t.Fatal(err)
	}
	got := buf.String()
	// JSON line should not contain "text"
	if strings.Contains(strings.Split(got, "\n")[0], `"text"`) {
		t.Errorf("JSON line should not contain heredoc field: %s", got)
	}
	// should contain heredoc definition
	if !strings.Contains(got, ".text = <<") {
		t.Errorf("expected heredoc definition, got: %s", got)
	}
	if !strings.Contains(got, "line1\nline2\n") {
		t.Errorf("expected heredoc value, got: %s", got)
	}
}

func TestEncodeHeredocFieldMissing(t *testing.T) {
	var buf bytes.Buffer
	enc := NewEncoder(&buf)
	// "text" is empty, should be skipped
	err := enc.Encode(basicObj{Type: "obj"}, "text")
	if err != nil {
		t.Fatal(err)
	}
	got := buf.String()
	if strings.Contains(got, ".text") {
		t.Errorf("missing heredoc field should be skipped, got: %s", got)
	}
}

func TestEncodeNestedHeredocField(t *testing.T) {
	var buf bytes.Buffer
	enc := NewEncoder(&buf)
	obj := nestedObj{
		Type: "obj",
		Sub:  subObj{Text: "nested value\n"},
	}
	err := enc.Encode(obj, "sub.text")
	if err != nil {
		t.Fatal(err)
	}
	got := buf.String()
	if !strings.Contains(got, ".sub.text = <<") {
		t.Errorf("expected nested heredoc, got: %s", got)
	}
}

func TestEncodeSeparatorAvoidance(t *testing.T) {
	var buf bytes.Buffer
	enc := NewEncoder(&buf)
	// value contains END0, so separator should be END1 or higher
	obj := basicObj{Type: "obj", Text: "has END0 in it\n"}
	err := enc.Encode(obj, "text")
	if err != nil {
		t.Fatal(err)
	}
	got := buf.String()
	if strings.Contains(got, "<<END0") {
		t.Errorf("separator should not be END0 when value contains it, got: %s", got)
	}
}

func TestEncodeMultipleObjects(t *testing.T) {
	var buf bytes.Buffer
	enc := NewEncoder(&buf)
	if err := enc.Encode(basicObj{Type: "first", Text: "a\n"}, "text"); err != nil {
		t.Fatal(err)
	}
	if err := enc.Encode(basicObj{Type: "second", Text: "b\n"}, "text"); err != nil {
		t.Fatal(err)
	}
	got := buf.String()
	if strings.Count(got, `"type"`) != 2 {
		t.Errorf("expected two JSON objects, got: %s", got)
	}
}

// --- Decoder tests ---

func TestDecodeBasicJSON(t *testing.T) {
	input := `{"type":"hello"}` + "\n"
	dec := NewDecoder(strings.NewReader(input))
	var obj basicObj
	if err := dec.Decode(&obj); err != nil {
		t.Fatal(err)
	}
	if obj.Type != "hello" {
		t.Errorf("got type %q, want %q", obj.Type, "hello")
	}
}

func TestDecodeWithHeredoc(t *testing.T) {
	input := `{"type":"obj"}
.text = <<END0
Hello
World
END0
`
	dec := NewDecoder(strings.NewReader(input))
	var obj basicObj
	if err := dec.Decode(&obj); err != nil {
		t.Fatal(err)
	}
	if obj.Type != "obj" {
		t.Errorf("got type %q, want %q", obj.Type, "obj")
	}
	if obj.Text != "Hello\nWorld\n" {
		t.Errorf("got text %q, want %q", obj.Text, "Hello\nWorld\n")
	}
}

func TestDecodeEmptyHeredoc(t *testing.T) {
	// Per docs: empty value ""
	input := `{}
.text = <<END0
END0
`
	dec := NewDecoder(strings.NewReader(input))
	var obj basicObj
	if err := dec.Decode(&obj); err != nil {
		t.Fatal(err)
	}
	if obj.Text != "" {
		t.Errorf("got text %q, want %q", obj.Text, "")
	}
}

func TestDecodeHeredocSingleNewline(t *testing.T) {
	// Per docs: value "\n"
	input := "{}\n.text = <<END0\n\nEND0\n"
	dec := NewDecoder(strings.NewReader(input))
	var obj basicObj
	if err := dec.Decode(&obj); err != nil {
		t.Fatal(err)
	}
	if obj.Text != "\n" {
		t.Errorf("got text %q, want %q", obj.Text, "\n")
	}
}

func TestDecodeHeredocNoTrailingNewline(t *testing.T) {
	// Per docs: value "Hello, world!" (no trailing newline)
	input := "{}\n.text = <<END0\nHello, world!END0\n"
	dec := NewDecoder(strings.NewReader(input))
	var obj basicObj
	if err := dec.Decode(&obj); err != nil {
		t.Fatal(err)
	}
	if obj.Text != "Hello, world!" {
		t.Errorf("got text %q, want %q", obj.Text, "Hello, world!")
	}
}

func TestDecodeMultipleObjects(t *testing.T) {
	input := `{"type":"first"}
.text = <<END0
aaa
END0
{"type":"second"}
.text = <<END0
bbb
END0
`
	dec := NewDecoder(strings.NewReader(input))

	var obj1 basicObj
	if err := dec.Decode(&obj1); err != nil {
		t.Fatal(err)
	}
	if obj1.Type != "first" || obj1.Text != "aaa\n" {
		t.Errorf("obj1: got %+v", obj1)
	}

	var obj2 basicObj
	if err := dec.Decode(&obj2); err != nil {
		t.Fatal(err)
	}
	if obj2.Type != "second" || obj2.Text != "bbb\n" {
		t.Errorf("obj2: got %+v", obj2)
	}
}

func TestDecodeEOFAfterLastObject(t *testing.T) {
	input := `{"type":"only"}` + "\n"
	dec := NewDecoder(strings.NewReader(input))
	var obj basicObj
	if err := dec.Decode(&obj); err != nil {
		t.Fatal(err)
	}
	var obj2 basicObj
	err := dec.Decode(&obj2)
	if err != io.EOF {
		t.Errorf("expected io.EOF, got %v", err)
	}
}

func TestDecodeCommentSkipping(t *testing.T) {
	input := `# this is a comment
{"type":"obj"}
# another comment
{"type":"second"}
`
	dec := NewDecoder(strings.NewReader(input))

	var obj1 basicObj
	if err := dec.Decode(&obj1); err != nil {
		t.Fatal(err)
	}
	if obj1.Type != "obj" {
		t.Errorf("got type %q, want %q", obj1.Type, "obj")
	}

	var obj2 basicObj
	if err := dec.Decode(&obj2); err != nil {
		t.Fatal(err)
	}
	if obj2.Type != "second" {
		t.Errorf("got type %q, want %q", obj2.Type, "second")
	}
}

func TestDecodeNestedHeredoc(t *testing.T) {
	input := `{"type":"obj","sub":{}}
.sub.text = <<END0
nested value
END0
`
	dec := NewDecoder(strings.NewReader(input))
	var obj nestedObj
	if err := dec.Decode(&obj); err != nil {
		t.Fatal(err)
	}
	if obj.Sub.Text != "nested value\n" {
		t.Errorf("got sub.text %q, want %q", obj.Sub.Text, "nested value\n")
	}
}

func TestDecodeMultipleHeredocFields(t *testing.T) {
	input := `{"type":"obj","sub":{}}
.sub.text = <<END0
first
END0
.sub.more = <<END0
second
END0
`
	dec := NewDecoder(strings.NewReader(input))
	var obj nestedObj
	if err := dec.Decode(&obj); err != nil {
		t.Fatal(err)
	}
	if obj.Sub.Text != "first\n" {
		t.Errorf("got sub.text %q, want %q", obj.Sub.Text, "first\n")
	}
	if obj.Sub.More != "second\n" {
		t.Errorf("got sub.more %q, want %q", obj.Sub.More, "second\n")
	}
}

func TestDecodeInvalidFieldLine(t *testing.T) {
	input := `{"type":"obj"}
.text INVALID
`
	dec := NewDecoder(strings.NewReader(input))
	var obj basicObj
	err := dec.Decode(&obj)
	if err == nil {
		t.Fatal("expected error for invalid field line")
	}
}

func TestDecodeInvalidHeredocSyntax(t *testing.T) {
	input := `{"type":"obj"}
.text = NOHEREDOC
`
	dec := NewDecoder(strings.NewReader(input))
	var obj basicObj
	err := dec.Decode(&obj)
	if err == nil {
		t.Fatal("expected error for missing << prefix")
	}
}

func TestDecodeRedefinition(t *testing.T) {
	input := `{"type":"obj","text":"existing"}
.text = <<END0
duplicate
END0
`
	dec := NewDecoder(strings.NewReader(input))
	var obj basicObj
	err := dec.Decode(&obj)
	if err == nil {
		t.Fatal("expected error for field redefinition")
	}
}

func TestDecodeEOFDuringHeredoc(t *testing.T) {
	input := `{"type":"obj"}
.text = <<END0
unclosed heredoc`
	dec := NewDecoder(strings.NewReader(input))
	var obj basicObj
	err := dec.Decode(&obj)
	if err == nil {
		t.Fatal("expected error for unterminated heredoc")
	}
}

// --- Round-trip tests ---

func TestRoundTripSimple(t *testing.T) {
	original := basicObj{Type: "test", Text: "hello world\n"}

	var buf bytes.Buffer
	enc := NewEncoder(&buf)
	if err := enc.Encode(original, "text"); err != nil {
		t.Fatal(err)
	}

	dec := NewDecoder(&buf)
	var decoded basicObj
	if err := dec.Decode(&decoded); err != nil {
		t.Fatal(err)
	}

	if decoded != original {
		t.Errorf("round trip failed: got %+v, want %+v", decoded, original)
	}
}

func TestRoundTripNoTrailingNewline(t *testing.T) {
	original := basicObj{Type: "test", Text: "no newline at end"}

	var buf bytes.Buffer
	enc := NewEncoder(&buf)
	if err := enc.Encode(original, "text"); err != nil {
		t.Fatal(err)
	}

	dec := NewDecoder(&buf)
	var decoded basicObj
	if err := dec.Decode(&decoded); err != nil {
		t.Fatal(err)
	}

	if decoded != original {
		t.Errorf("round trip failed: got %+v, want %+v", decoded, original)
	}
}

func TestRoundTripEmptyString(t *testing.T) {
	original := basicObj{Type: "test", Text: ""}

	var buf bytes.Buffer
	enc := NewEncoder(&buf)
	if err := enc.Encode(original, "text"); err != nil {
		t.Fatal(err)
	}

	dec := NewDecoder(&buf)
	var decoded basicObj
	if err := dec.Decode(&decoded); err != nil {
		t.Fatal(err)
	}

	if decoded != original {
		t.Errorf("round trip failed: got %+v, want %+v", decoded, original)
	}
}

func TestRoundTripMultipleNewlines(t *testing.T) {
	original := basicObj{Type: "test", Text: "\n\n\n"}

	var buf bytes.Buffer
	enc := NewEncoder(&buf)
	if err := enc.Encode(original, "text"); err != nil {
		t.Fatal(err)
	}

	dec := NewDecoder(&buf)
	var decoded basicObj
	if err := dec.Decode(&decoded); err != nil {
		t.Fatal(err)
	}

	if decoded != original {
		t.Errorf("round trip failed: got %+v, want %+v", decoded, original)
	}
}

func TestRoundTripSeparatorInValue(t *testing.T) {
	original := basicObj{Type: "test", Text: "has END0 and END1 inside\n"}

	var buf bytes.Buffer
	enc := NewEncoder(&buf)
	if err := enc.Encode(original, "text"); err != nil {
		t.Fatal(err)
	}

	dec := NewDecoder(&buf)
	var decoded basicObj
	if err := dec.Decode(&decoded); err != nil {
		t.Fatal(err)
	}

	if decoded != original {
		t.Errorf("round trip failed: got %+v, want %+v", decoded, original)
	}
}

func TestRoundTripNested(t *testing.T) {
	original := nestedObj{
		Type: "test",
		Sub:  subObj{Text: "nested content\n"},
	}

	var buf bytes.Buffer
	enc := NewEncoder(&buf)
	if err := enc.Encode(original, "sub.text"); err != nil {
		t.Fatal(err)
	}

	dec := NewDecoder(&buf)
	var decoded nestedObj
	if err := dec.Decode(&decoded); err != nil {
		t.Fatal(err)
	}

	if decoded != original {
		t.Errorf("round trip failed: got %+v, want %+v", decoded, original)
	}
}

func TestRoundTripMultipleObjects(t *testing.T) {
	objects := []basicObj{
		{Type: "first", Text: "one\n"},
		{Type: "second", Text: "two\n"},
		{Type: "third", Text: "three\n"},
	}

	var buf bytes.Buffer
	enc := NewEncoder(&buf)
	for _, obj := range objects {
		if err := enc.Encode(obj, "text"); err != nil {
			t.Fatal(err)
		}
	}

	dec := NewDecoder(&buf)
	for i, want := range objects {
		var got basicObj
		if err := dec.Decode(&got); err != nil {
			t.Fatalf("object %d: %v", i, err)
		}
		if got != want {
			t.Errorf("object %d: got %+v, want %+v", i, got, want)
		}
	}

	var extra basicObj
	if err := dec.Decode(&extra); err != io.EOF {
		t.Errorf("expected EOF after all objects, got %v", err)
	}
}

func TestRoundTripNoHeredocFields(t *testing.T) {
	original := basicObj{Type: "plain"}

	var buf bytes.Buffer
	enc := NewEncoder(&buf)
	if err := enc.Encode(original); err != nil {
		t.Fatal(err)
	}

	dec := NewDecoder(&buf)
	var decoded basicObj
	if err := dec.Decode(&decoded); err != nil {
		t.Fatal(err)
	}

	if decoded != original {
		t.Errorf("round trip failed: got %+v, want %+v", decoded, original)
	}
}

func TestRoundTripLargeValue(t *testing.T) {
	large := strings.Repeat("abcdefghij\n", 1000)
	original := basicObj{Type: "big", Text: large}

	var buf bytes.Buffer
	enc := NewEncoder(&buf)
	if err := enc.Encode(original, "text"); err != nil {
		t.Fatal(err)
	}

	dec := NewDecoder(&buf)
	var decoded basicObj
	if err := dec.Decode(&decoded); err != nil {
		t.Fatal(err)
	}

	if decoded != original {
		t.Errorf("round trip failed for large value: lengths got %d, want %d", len(decoded.Text), len(original.Text))
	}
}

// --- Base64 field override tests ---

func TestEncodeBase64Field(t *testing.T) {
	var buf bytes.Buffer
	enc := NewEncoder(&buf)
	obj := byteFieldObj{Type: "test", Data: []byte(`{"query":"hello"}`)}
	if err := enc.Encode(obj, "data:base64"); err != nil {
		t.Fatal(err)
	}
	got := buf.String()
	jsonLine := strings.Split(got, "\n")[0]
	if strings.Contains(jsonLine, `"data"`) {
		t.Errorf("JSON line should not contain base64 field: %s", jsonLine)
	}
	if !strings.Contains(got, `.data = <<`) {
		t.Errorf("expected heredoc for data, got: %s", got)
	}
	if !strings.Contains(got, `{"query":"hello"}`) {
		t.Errorf("expected readable content in heredoc, got: %s", got)
	}
}

func TestEncodeBase64FieldNilSkipped(t *testing.T) {
	var buf bytes.Buffer
	enc := NewEncoder(&buf)
	obj := byteFieldObj{Type: "test"} // Data is nil
	if err := enc.Encode(obj, "data:base64"); err != nil {
		t.Fatal(err)
	}
	got := buf.String()
	if strings.Contains(got, ".data") {
		t.Errorf("nil base64 field should be skipped, got: %s", got)
	}
}

func TestDecodeBase64Override(t *testing.T) {
	input := `{"type":"test"}
.data = <<END0
{"query":"hello"}END0
`
	dec := NewDecoder(strings.NewReader(input))
	var obj byteFieldObj
	if err := dec.Decode(&obj, "data:base64"); err != nil {
		t.Fatal(err)
	}
	if obj.Type != "test" {
		t.Errorf("got type %q, want %q", obj.Type, "test")
	}
	if string(obj.Data) != `{"query":"hello"}` {
		t.Errorf("got data %q, want %q", string(obj.Data), `{"query":"hello"}`)
	}
}

func TestDecodeBase64OverrideInlineUnaffected(t *testing.T) {
	// "aGVsbG8=" is base64 of "hello"
	input := `{"type":"test","data":"aGVsbG8="}` + "\n"
	dec := NewDecoder(strings.NewReader(input))
	var obj byteFieldObj
	if err := dec.Decode(&obj, "data:base64"); err != nil {
		t.Fatal(err)
	}
	if string(obj.Data) != "hello" {
		t.Errorf("got data %q, want %q", string(obj.Data), "hello")
	}
}

func TestRoundTripBase64Field(t *testing.T) {
	original := byteFieldObj{Type: "test", Data: []byte(`{"query":"hello world"}`)}

	var buf bytes.Buffer
	enc := NewEncoder(&buf)
	if err := enc.Encode(original, "data:base64"); err != nil {
		t.Fatal(err)
	}

	dec := NewDecoder(&buf)
	var decoded byteFieldObj
	if err := dec.Decode(&decoded, "data:base64"); err != nil {
		t.Fatal(err)
	}

	if decoded.Type != original.Type {
		t.Errorf("type: got %q, want %q", decoded.Type, original.Type)
	}
	if string(decoded.Data) != string(original.Data) {
		t.Errorf("data: got %q, want %q", string(decoded.Data), string(original.Data))
	}
}

func TestRoundTripBase64WithStringHeredoc(t *testing.T) {
	type mixedObj struct {
		Type string `json:"type"`
		Text string `json:"text,omitempty"`
		Data []byte `json:"data,omitempty"`
	}

	original := mixedObj{
		Type: "mixed",
		Text: "hello\nworld\n",
		Data: []byte(`{"key":"value"}`),
	}

	var buf bytes.Buffer
	enc := NewEncoder(&buf)
	if err := enc.Encode(original, "text", "data:base64"); err != nil {
		t.Fatal(err)
	}

	dec := NewDecoder(&buf)
	var decoded mixedObj
	if err := dec.Decode(&decoded, "data:base64"); err != nil {
		t.Fatal(err)
	}

	if decoded.Type != original.Type {
		t.Errorf("type: got %q, want %q", decoded.Type, original.Type)
	}
	if decoded.Text != original.Text {
		t.Errorf("text: got %q, want %q", decoded.Text, original.Text)
	}
	if string(decoded.Data) != string(original.Data) {
		t.Errorf("data: got %q, want %q", string(decoded.Data), string(original.Data))
	}
}

func TestRoundTripBase64NestedField(t *testing.T) {
	original := nestedByteObj{
		Type: "test",
		Sub:  byteSubObj{Data: []byte(`{"nested":"value"}`)},
	}

	var buf bytes.Buffer
	enc := NewEncoder(&buf)
	if err := enc.Encode(original, "sub.data:base64"); err != nil {
		t.Fatal(err)
	}

	dec := NewDecoder(&buf)
	var decoded nestedByteObj
	if err := dec.Decode(&decoded, "sub.data:base64"); err != nil {
		t.Fatal(err)
	}

	if string(decoded.Sub.Data) != string(original.Sub.Data) {
		t.Errorf("sub.data: got %q, want %q", string(decoded.Sub.Data), string(original.Sub.Data))
	}
}

func TestRoundTripBase64MultipleObjects(t *testing.T) {
	type obj struct {
		Type string `json:"type"`
		Text string `json:"text,omitempty"`
		Data []byte `json:"data,omitempty"`
	}

	objects := []obj{
		{Type: "text-only", Text: "hello\n"},
		{Type: "with-data", Data: []byte(`{"action":"search"}`)},
		{Type: "both", Text: "response\n", Data: []byte(`{"result":42}`)},
	}

	var buf bytes.Buffer
	enc := NewEncoder(&buf)
	for _, o := range objects {
		if err := enc.Encode(o, "text", "data:base64"); err != nil {
			t.Fatal(err)
		}
	}

	dec := NewDecoder(&buf)
	for i, want := range objects {
		var got obj
		if err := dec.Decode(&got, "data:base64"); err != nil {
			t.Fatalf("object %d: %v", i, err)
		}
		if got.Type != want.Type {
			t.Errorf("object %d type: got %q, want %q", i, got.Type, want.Type)
		}
		if got.Text != want.Text {
			t.Errorf("object %d text: got %q, want %q", i, got.Text, want.Text)
		}
		if string(got.Data) != string(want.Data) {
			t.Errorf("object %d data: got %q, want %q", i, string(got.Data), string(want.Data))
		}
	}
}
