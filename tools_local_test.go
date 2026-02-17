package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/modfin/bellman/tools"
)

func callTool(t *testing.T, tool tools.Tool, args any) string {
	t.Helper()
	data, err := json.Marshal(args)
	if err != nil {
		t.Fatalf("marshal args: %v", err)
	}
	result, err := tool.Function(context.Background(), tools.Call{Argument: data})
	if err != nil {
		t.Fatalf("tool %s returned error: %v", tool.Name, err)
	}
	return result
}

func writeTestFile(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("write test file: %v", err)
	}
	return path
}

// --- hashLineContent tests ---

func TestHashLineContent_Deterministic(t *testing.T) {
	h1 := hashLineContent("hello world")
	h2 := hashLineContent("hello world")
	if h1 != h2 {
		t.Errorf("same input produced different hashes: %s vs %s", h1, h2)
	}
}

func TestHashLineContent_TwoCharHex(t *testing.T) {
	h := hashLineContent("test")
	if len(h) != 2 {
		t.Errorf("expected 2-char hash, got %q (len %d)", h, len(h))
	}
	for _, c := range h {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')) {
			t.Errorf("hash contains non-hex character: %q", h)
		}
	}
}

func TestHashLineContent_DifferentInputs(t *testing.T) {
	// Different inputs should usually produce different hashes
	// (not guaranteed with 8-bit hash, but these specific strings should differ)
	h1 := hashLineContent("hello")
	h2 := hashLineContent("world")
	if h1 == h2 {
		t.Logf("warning: collision for 'hello' and 'world': both %s", h1)
	}
}

func TestHashLineContent_WhitespaceMatters(t *testing.T) {
	h1 := hashLineContent("  hello")
	h2 := hashLineContent("hello")
	if h1 == h2 {
		t.Logf("warning: collision for '  hello' and 'hello': both %s", h1)
	}
}

// --- readFileLines tests ---

func TestReadFileLines_Normal(t *testing.T) {
	dir := t.TempDir()
	path := writeTestFile(t, dir, "test.txt", "line1\nline2\nline3\n")

	lines, endsNL, err := readFileLines(path)
	if err != nil {
		t.Fatal(err)
	}
	if !endsNL {
		t.Error("expected endsWithNewline=true")
	}
	if len(lines) != 3 {
		t.Fatalf("expected 3 lines, got %d", len(lines))
	}
	if lines[0] != "line1" || lines[1] != "line2" || lines[2] != "line3" {
		t.Errorf("unexpected lines: %v", lines)
	}
}

func TestReadFileLines_NoTrailingNewline(t *testing.T) {
	dir := t.TempDir()
	path := writeTestFile(t, dir, "test.txt", "line1\nline2")

	lines, endsNL, err := readFileLines(path)
	if err != nil {
		t.Fatal(err)
	}
	if endsNL {
		t.Error("expected endsWithNewline=false")
	}
	if len(lines) != 2 {
		t.Fatalf("expected 2 lines, got %d", len(lines))
	}
}

func TestReadFileLines_Empty(t *testing.T) {
	dir := t.TempDir()
	path := writeTestFile(t, dir, "test.txt", "")

	lines, endsNL, err := readFileLines(path)
	if err != nil {
		t.Fatal(err)
	}
	if endsNL {
		t.Error("expected endsWithNewline=false")
	}
	if len(lines) != 0 {
		t.Errorf("expected 0 lines, got %d", len(lines))
	}
}

func TestReadFileLines_SingleLine(t *testing.T) {
	dir := t.TempDir()
	path := writeTestFile(t, dir, "test.txt", "hello\n")

	lines, endsNL, err := readFileLines(path)
	if err != nil {
		t.Fatal(err)
	}
	if !endsNL {
		t.Error("expected endsWithNewline=true")
	}
	if len(lines) != 1 || lines[0] != "hello" {
		t.Errorf("unexpected lines: %v", lines)
	}
}

func TestReadFileLines_NotFound(t *testing.T) {
	_, _, err := readFileLines("/nonexistent/file.txt")
	if err == nil {
		t.Error("expected error for nonexistent file")
	}
}

// --- formatHashlines tests ---

func TestFormatHashlines_Basic(t *testing.T) {
	lines := []string{"hello", "world"}
	result := formatHashlines(lines, 0)

	h1 := hashLineContent("hello")
	h2 := hashLineContent("world")
	expected := "1:" + h1 + "|hello\n2:" + h2 + "|world\n"
	if result != expected {
		t.Errorf("expected:\n%s\ngot:\n%s", expected, result)
	}
}

func TestFormatHashlines_WithOffset(t *testing.T) {
	lines := []string{"third", "fourth"}
	result := formatHashlines(lines, 2)

	// Should start at line 3 (offset 2 + 1)
	if !strings.HasPrefix(result, "3:") {
		t.Errorf("expected to start with '3:', got %q", result[:10])
	}
	if !strings.Contains(result, "4:") {
		t.Error("expected line 4 in output")
	}
}

func TestFormatHashlines_Empty(t *testing.T) {
	result := formatHashlines(nil, 0)
	if result != "" {
		t.Errorf("expected empty string, got %q", result)
	}
}

// --- parseHashlineRef tests ---

func TestParseHashlineRef_Valid(t *testing.T) {
	lineNum, hash, err := parseHashlineRef("5:a3")
	if err != nil {
		t.Fatal(err)
	}
	if lineNum != 5 {
		t.Errorf("expected lineNum=5, got %d", lineNum)
	}
	if hash != "a3" {
		t.Errorf("expected hash='a3', got %q", hash)
	}
}

func TestParseHashlineRef_InvalidFormat(t *testing.T) {
	_, _, err := parseHashlineRef("invalid")
	if err == nil {
		t.Error("expected error for invalid format")
	}
}

func TestParseHashlineRef_InvalidLineNumber(t *testing.T) {
	_, _, err := parseHashlineRef("abc:ff")
	if err == nil {
		t.Error("expected error for non-numeric line number")
	}
}

func TestParseHashlineRef_ZeroLine(t *testing.T) {
	_, _, err := parseHashlineRef("0:ff")
	if err == nil {
		t.Error("expected error for line number 0")
	}
}

func TestParseHashlineRef_NegativeLine(t *testing.T) {
	_, _, err := parseHashlineRef("-1:ff")
	if err == nil {
		t.Error("expected error for negative line number")
	}
}

// --- validateHashlineRef tests ---

func TestValidateHashlineRef_Valid(t *testing.T) {
	lines := []string{"hello", "world"}
	hash := hashLineContent("hello")
	err := validateHashlineRef(lines, 1, hash)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestValidateHashlineRef_Mismatch(t *testing.T) {
	lines := []string{"hello", "world"}
	err := validateHashlineRef(lines, 1, "zz")
	if err == nil {
		t.Error("expected error for hash mismatch")
	}
	if !strings.Contains(err.Error(), "hash mismatch") {
		t.Errorf("expected 'hash mismatch' in error, got: %v", err)
	}
}

func TestValidateHashlineRef_OutOfBounds(t *testing.T) {
	lines := []string{"hello"}
	err := validateHashlineRef(lines, 5, "ff")
	if err == nil {
		t.Error("expected error for out of bounds line")
	}
	if !strings.Contains(err.Error(), "does not exist") {
		t.Errorf("expected 'does not exist' in error, got: %v", err)
	}
}

// --- read_file tool tests ---

func TestReadFileTool_BasicFile(t *testing.T) {
	dir := t.TempDir()
	path := writeTestFile(t, dir, "test.txt", "hello\nworld\n")

	result := callTool(t, readFileTool, readFileArgs{Path: path})

	if !strings.Contains(result, "[page 1/1") {
		t.Errorf("expected page header, got: %s", result)
	}
	if !strings.Contains(result, "|hello") {
		t.Errorf("expected 'hello' in output, got: %s", result)
	}
	if !strings.Contains(result, "|world") {
		t.Errorf("expected 'world' in output, got: %s", result)
	}
}

func TestReadFileTool_EmptyFile(t *testing.T) {
	dir := t.TempDir()
	path := writeTestFile(t, dir, "test.txt", "")

	result := callTool(t, readFileTool, readFileArgs{Path: path})

	if !strings.Contains(result, "(empty file)") {
		t.Errorf("expected '(empty file)', got: %s", result)
	}
}

func TestReadFileTool_Pagination(t *testing.T) {
	dir := t.TempDir()
	var sb strings.Builder
	for i := 0; i < 250; i++ {
		sb.WriteString("line\n")
	}
	path := writeTestFile(t, dir, "test.txt", sb.String())

	// Page 1
	result := callTool(t, readFileTool, readFileArgs{Path: path, Page: 1})
	if !strings.Contains(result, "[page 1/3") {
		t.Errorf("expected page 1/3, got: %s", strings.SplitN(result, "\n", 2)[0])
	}
	if !strings.Contains(result, "lines 1-100 of 250") {
		t.Errorf("expected 'lines 1-100 of 250', got: %s", strings.SplitN(result, "\n", 2)[0])
	}

	// Page 3
	result = callTool(t, readFileTool, readFileArgs{Path: path, Page: 3})
	if !strings.Contains(result, "[page 3/3") {
		t.Errorf("expected page 3/3, got: %s", strings.SplitN(result, "\n", 2)[0])
	}
	if !strings.Contains(result, "lines 201-250 of 250") {
		t.Errorf("expected 'lines 201-250 of 250', got: %s", strings.SplitN(result, "\n", 2)[0])
	}

	// Page out of range
	result = callTool(t, readFileTool, readFileArgs{Path: path, Page: 4})
	if !strings.Contains(result, "error:") {
		t.Errorf("expected error for page 4, got: %s", result)
	}
}

func TestReadFileTool_DefaultPage(t *testing.T) {
	dir := t.TempDir()
	path := writeTestFile(t, dir, "test.txt", "hello\n")

	// Page 0 should default to page 1
	result := callTool(t, readFileTool, readFileArgs{Path: path, Page: 0})
	if !strings.Contains(result, "[page 1/1") {
		t.Errorf("expected page 1, got: %s", result)
	}
}

func TestReadFileTool_NonexistentFile(t *testing.T) {
	result := callTool(t, readFileTool, readFileArgs{Path: "/nonexistent/file.txt"})
	if !strings.Contains(result, "error:") {
		t.Errorf("expected error, got: %s", result)
	}
}

func TestReadFileTool_MissingPath(t *testing.T) {
	result := callTool(t, readFileTool, readFileArgs{})
	if !strings.Contains(result, "error:") {
		t.Errorf("expected error for missing path, got: %s", result)
	}
}

func TestReadFileTool_HashlineFormat(t *testing.T) {
	dir := t.TempDir()
	path := writeTestFile(t, dir, "test.txt", "first line\nsecond line\n")

	result := callTool(t, readFileTool, readFileArgs{Path: path})

	// Check each hashline has the right format: linenum:hash|content
	lines := strings.Split(strings.TrimSpace(result), "\n")
	for _, line := range lines[1:] { // skip header
		parts := strings.SplitN(line, "|", 2)
		if len(parts) != 2 {
			t.Errorf("expected pipe separator in hashline, got: %s", line)
			continue
		}
		ref := parts[0]
		colonParts := strings.SplitN(ref, ":", 2)
		if len(colonParts) != 2 {
			t.Errorf("expected colon in hashline ref, got: %s", ref)
		}
	}
}

// --- list_directory tool tests ---

func TestListDirTool_Basic(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, dir, "a.txt", "hello")
	writeTestFile(t, dir, "b.txt", "world")

	result := callTool(t, listDirTool, listDirArgs{Path: dir})

	if !strings.Contains(result, "a.txt") {
		t.Errorf("expected 'a.txt' in output, got: %s", result)
	}
	if !strings.Contains(result, "b.txt") {
		t.Errorf("expected 'b.txt' in output, got: %s", result)
	}
}

func TestListDirTool_EmptyDir(t *testing.T) {
	dir := t.TempDir()
	result := callTool(t, listDirTool, listDirArgs{Path: dir})
	if !strings.Contains(result, "(empty directory)") {
		t.Errorf("expected '(empty directory)', got: %s", result)
	}
}

func TestListDirTool_NonexistentDir(t *testing.T) {
	result := callTool(t, listDirTool, listDirArgs{Path: "/nonexistent/dir"})
	if !strings.Contains(result, "error:") {
		t.Errorf("expected error, got: %s", result)
	}
}

func TestListDirTool_Pagination(t *testing.T) {
	dir := t.TempDir()
	for i := 0; i < 150; i++ {
		writeTestFile(t, dir, fmt.Sprintf("file_%04d.txt", i), "")
	}

	// Verify there are at least 150 entries
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) < 150 {
		t.Fatalf("expected at least 150 entries, got %d", len(entries))
	}

	result := callTool(t, listDirTool, listDirArgs{Path: dir, Page: 1})
	if !strings.Contains(result, "[page 1/2") {
		t.Errorf("expected page 1/2, got first line: %s", strings.SplitN(result, "\n", 2)[0])
	}

	result = callTool(t, listDirTool, listDirArgs{Path: dir, Page: 2})
	if !strings.Contains(result, "[page 2/2") {
		t.Errorf("expected page 2/2, got first line: %s", strings.SplitN(result, "\n", 2)[0])
	}
}

func TestListDirTool_ShowsPermissions(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, dir, "test.txt", "hello")

	result := callTool(t, listDirTool, listDirArgs{Path: dir})

	// Should contain permission string like "-rw-"
	if !strings.Contains(result, "rw") {
		t.Errorf("expected permissions in output, got: %s", result)
	}
}

// --- edit_file tool tests ---

func getHash(t *testing.T, content string) string {
	t.Helper()
	return hashLineContent(content)
}

func TestEditFileTool_ReplaceSingleLine(t *testing.T) {
	dir := t.TempDir()
	path := writeTestFile(t, dir, "test.txt", "line1\nline2\nline3\n")

	hash := getHash(t, "line2")
	result := callTool(t, editFileTool, editFileArgs{
		Path:      path,
		Operation: "replace",
		Start:     "2:" + hash,
		Content:   "replaced",
	})

	if !strings.Contains(result, "ok:") {
		t.Fatalf("expected ok, got: %s", result)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "line1\nreplaced\nline3\n" {
		t.Errorf("unexpected file content: %q", string(data))
	}
}

func TestEditFileTool_ReplaceRange(t *testing.T) {
	dir := t.TempDir()
	path := writeTestFile(t, dir, "test.txt", "a\nb\nc\nd\ne\n")

	hashB := getHash(t, "b")
	hashD := getHash(t, "d")
	result := callTool(t, editFileTool, editFileArgs{
		Path:      path,
		Operation: "replace",
		Start:     "2:" + hashB,
		End:       "4:" + hashD,
		Content:   "new1\nnew2",
	})

	if !strings.Contains(result, "ok:") {
		t.Fatalf("expected ok, got: %s", result)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "a\nnew1\nnew2\ne\n" {
		t.Errorf("unexpected file content: %q", string(data))
	}
}

func TestEditFileTool_DeleteLines(t *testing.T) {
	dir := t.TempDir()
	path := writeTestFile(t, dir, "test.txt", "keep\ndelete\nkeep2\n")

	hash := getHash(t, "delete")
	result := callTool(t, editFileTool, editFileArgs{
		Path:      path,
		Operation: "replace",
		Start:     "2:" + hash,
		Content:   "",
	})

	if !strings.Contains(result, "ok:") {
		t.Fatalf("expected ok, got: %s", result)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "keep\nkeep2\n" {
		t.Errorf("unexpected file content: %q", string(data))
	}
}

func TestEditFileTool_InsertAfter(t *testing.T) {
	dir := t.TempDir()
	path := writeTestFile(t, dir, "test.txt", "line1\nline2\nline3\n")

	hash := getHash(t, "line1")
	result := callTool(t, editFileTool, editFileArgs{
		Path:      path,
		Operation: "insert_after",
		Start:     "1:" + hash,
		Content:   "inserted",
	})

	if !strings.Contains(result, "ok:") {
		t.Fatalf("expected ok, got: %s", result)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "line1\ninserted\nline2\nline3\n" {
		t.Errorf("unexpected file content: %q", string(data))
	}
}

func TestEditFileTool_InsertMultipleLines(t *testing.T) {
	dir := t.TempDir()
	path := writeTestFile(t, dir, "test.txt", "a\nb\n")

	hash := getHash(t, "a")
	result := callTool(t, editFileTool, editFileArgs{
		Path:      path,
		Operation: "insert_after",
		Start:     "1:" + hash,
		Content:   "x\ny\nz",
	})

	if !strings.Contains(result, "ok:") {
		t.Fatalf("expected ok, got: %s", result)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "a\nx\ny\nz\nb\n" {
		t.Errorf("unexpected file content: %q", string(data))
	}
}

func TestEditFileTool_HashMismatch(t *testing.T) {
	dir := t.TempDir()
	path := writeTestFile(t, dir, "test.txt", "hello\n")

	result := callTool(t, editFileTool, editFileArgs{
		Path:      path,
		Operation: "replace",
		Start:     "1:zz",
		Content:   "replaced",
	})

	if !strings.Contains(result, "hash mismatch") {
		t.Errorf("expected hash mismatch error, got: %s", result)
	}

	// File should be unchanged
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "hello\n" {
		t.Errorf("file should be unchanged, got: %q", string(data))
	}
}

func TestEditFileTool_LineOutOfBounds(t *testing.T) {
	dir := t.TempDir()
	path := writeTestFile(t, dir, "test.txt", "hello\n")

	hash := getHash(t, "hello")
	result := callTool(t, editFileTool, editFileArgs{
		Path:      path,
		Operation: "replace",
		Start:     "5:" + hash,
		Content:   "replaced",
	})

	if !strings.Contains(result, "does not exist") {
		t.Errorf("expected 'does not exist' error, got: %s", result)
	}
}

func TestEditFileTool_EndBeforeStart(t *testing.T) {
	dir := t.TempDir()
	path := writeTestFile(t, dir, "test.txt", "a\nb\nc\n")

	hashA := getHash(t, "a")
	hashC := getHash(t, "c")
	result := callTool(t, editFileTool, editFileArgs{
		Path:      path,
		Operation: "replace",
		Start:     "3:" + hashC,
		End:       "1:" + hashA,
		Content:   "x",
	})

	if !strings.Contains(result, "before start") {
		t.Errorf("expected 'before start' error, got: %s", result)
	}
}

func TestEditFileTool_MissingPath(t *testing.T) {
	result := callTool(t, editFileTool, editFileArgs{
		Operation: "replace",
		Start:     "1:ff",
		Content:   "x",
	})
	if !strings.Contains(result, "error:") {
		t.Errorf("expected error for missing path, got: %s", result)
	}
}

func TestEditFileTool_NonexistentFile(t *testing.T) {
	result := callTool(t, editFileTool, editFileArgs{
		Path:      "/nonexistent/file.txt",
		Operation: "replace",
		Start:     "1:ff",
		Content:   "x",
	})
	if !strings.Contains(result, "error:") {
		t.Errorf("expected error, got: %s", result)
	}
}

func TestEditFileTool_ReplaceFirstLine(t *testing.T) {
	dir := t.TempDir()
	path := writeTestFile(t, dir, "test.txt", "old\nsecond\n")

	hash := getHash(t, "old")
	callTool(t, editFileTool, editFileArgs{
		Path:      path,
		Operation: "replace",
		Start:     "1:" + hash,
		Content:   "new",
	})

	data, _ := os.ReadFile(path)
	if string(data) != "new\nsecond\n" {
		t.Errorf("unexpected: %q", string(data))
	}
}

func TestEditFileTool_ReplaceLastLine(t *testing.T) {
	dir := t.TempDir()
	path := writeTestFile(t, dir, "test.txt", "first\nlast\n")

	hash := getHash(t, "last")
	callTool(t, editFileTool, editFileArgs{
		Path:      path,
		Operation: "replace",
		Start:     "2:" + hash,
		Content:   "new last",
	})

	data, _ := os.ReadFile(path)
	if string(data) != "first\nnew last\n" {
		t.Errorf("unexpected: %q", string(data))
	}
}

func TestEditFileTool_InsertAfterLastLine(t *testing.T) {
	dir := t.TempDir()
	path := writeTestFile(t, dir, "test.txt", "first\nlast\n")

	hash := getHash(t, "last")
	callTool(t, editFileTool, editFileArgs{
		Path:      path,
		Operation: "insert_after",
		Start:     "2:" + hash,
		Content:   "appended",
	})

	data, _ := os.ReadFile(path)
	if string(data) != "first\nlast\nappended\n" {
		t.Errorf("unexpected: %q", string(data))
	}
}

// --- bash tool tests ---

func TestBashTool_SimpleCommand(t *testing.T) {
	result := callTool(t, bashTool, bashArgs{Command: "echo hello"})
	if strings.TrimSpace(result) != "hello" {
		t.Errorf("expected 'hello', got: %q", result)
	}
}

func TestBashTool_ExitStatus(t *testing.T) {
	result := callTool(t, bashTool, bashArgs{Command: "exit 1"})
	if !strings.Contains(result, "exit status") {
		t.Errorf("expected exit status error, got: %s", result)
	}
}

func TestBashTool_Stderr(t *testing.T) {
	result := callTool(t, bashTool, bashArgs{Command: "echo error >&2"})
	if !strings.Contains(result, "error") {
		t.Errorf("expected stderr in output, got: %s", result)
	}
}

func TestBashTool_MissingCommand(t *testing.T) {
	result := callTool(t, bashTool, bashArgs{})
	if !strings.Contains(result, "error:") {
		t.Errorf("expected error for missing command, got: %s", result)
	}
}

func TestBashTool_OutputTruncation(t *testing.T) {
	// Generate output larger than bashMaxOutput
	result := callTool(t, bashTool, bashArgs{
		Command: "yes 'this is a long line of text for testing truncation' | head -n 100000",
	})
	if !strings.Contains(result, "... (output truncated)") {
		t.Error("expected truncation notice in output")
	}
	// Should contain the tail (last lines)
	if !strings.Contains(result, "this is a long line") {
		t.Error("expected tail content to be preserved")
	}
}

func TestBashTool_MultilineOutput(t *testing.T) {
	result := callTool(t, bashTool, bashArgs{Command: "echo line1; echo line2; echo line3"})
	if !strings.Contains(result, "line1") || !strings.Contains(result, "line3") {
		t.Errorf("expected multiline output, got: %s", result)
	}
}

// --- Integration: read then edit ---

func TestIntegration_ReadThenEdit(t *testing.T) {
	dir := t.TempDir()
	path := writeTestFile(t, dir, "test.go", "package main\n\nfunc hello() {\n\treturn \"world\"\n}\n")

	// Read the file
	readResult := callTool(t, readFileTool, readFileArgs{Path: path})

	// Parse hashlines to get references
	lines := strings.Split(readResult, "\n")
	var returnLineRef string
	for _, line := range lines {
		if strings.Contains(line, "return") {
			parts := strings.SplitN(line, "|", 2)
			if len(parts) == 2 {
				returnLineRef = parts[0]
			}
		}
	}
	if returnLineRef == "" {
		t.Fatal("could not find return line in read output")
	}

	// Edit using the hashline reference
	editResult := callTool(t, editFileTool, editFileArgs{
		Path:      path,
		Operation: "replace",
		Start:     returnLineRef,
		Content:   "\treturn \"universe\"",
	})
	if !strings.Contains(editResult, "ok:") {
		t.Fatalf("edit failed: %s", editResult)
	}

	// Verify
	data, _ := os.ReadFile(path)
	if !strings.Contains(string(data), "universe") {
		t.Errorf("expected 'universe' in edited file, got: %s", string(data))
	}
	if strings.Contains(string(data), "world") {
		t.Errorf("did not expect 'world' in edited file")
	}
}
