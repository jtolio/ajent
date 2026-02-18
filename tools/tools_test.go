package tools

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

	result := callTool(t, ReadFileTool, readFileArgs{Path: path})

	if !strings.Contains(result, "[lines 1-2 of 2]") {
		t.Errorf("expected line range header, got: %s", result)
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

	result := callTool(t, ReadFileTool, readFileArgs{Path: path})

	if !strings.Contains(result, "(empty file)") {
		t.Errorf("expected '(empty file)', got: %s", result)
	}
}

func TestReadFileTool_LineRange(t *testing.T) {
	dir := t.TempDir()
	var sb strings.Builder
	for i := 0; i < 250; i++ {
		fmt.Fprintf(&sb, "line %d\n", i+1)
	}
	path := writeTestFile(t, dir, "test.txt", sb.String())

	// Default: first 200 lines
	result := callTool(t, ReadFileTool, readFileArgs{Path: path})
	if !strings.Contains(result, "[lines 1-200 of 250]") {
		t.Errorf("expected [lines 1-200 of 250], got: %s", strings.SplitN(result, "\n", 2)[0])
	}

	// Specific range
	result = callTool(t, ReadFileTool, readFileArgs{Path: path, StartLine: 201, EndLine: 250})
	if !strings.Contains(result, "[lines 201-250 of 250]") {
		t.Errorf("expected [lines 201-250 of 250], got: %s", strings.SplitN(result, "\n", 2)[0])
	}

	// Cap at 200 lines
	result = callTool(t, ReadFileTool, readFileArgs{Path: path, StartLine: 1, EndLine: 250})
	if !strings.Contains(result, "[lines 1-200 of 250]") {
		t.Errorf("expected capped at 200 lines, got: %s", strings.SplitN(result, "\n", 2)[0])
	}
}

func TestReadFileTool_StartLineBeyondFile(t *testing.T) {
	dir := t.TempDir()
	path := writeTestFile(t, dir, "test.txt", "hello\nworld\n")

	result := callTool(t, ReadFileTool, readFileArgs{Path: path, StartLine: 100})
	if !strings.Contains(result, "error:") {
		t.Errorf("expected error for start_line beyond file, got: %s", result)
	}
}

func TestReadFileTool_SmallFile(t *testing.T) {
	dir := t.TempDir()
	path := writeTestFile(t, dir, "test.txt", "hello\n")

	// File with <= 200 lines and no params: return entire file
	result := callTool(t, ReadFileTool, readFileArgs{Path: path})
	if !strings.Contains(result, "[lines 1-1 of 1]") {
		t.Errorf("expected [lines 1-1 of 1], got: %s", result)
	}
}

func TestReadFileTool_NonexistentFile(t *testing.T) {
	result := callTool(t, ReadFileTool, readFileArgs{Path: "/nonexistent/file.txt"})
	if !strings.Contains(result, "error:") {
		t.Errorf("expected error, got: %s", result)
	}
}

func TestReadFileTool_MissingPath(t *testing.T) {
	result := callTool(t, ReadFileTool, readFileArgs{})
	if !strings.Contains(result, "error:") {
		t.Errorf("expected error for missing path, got: %s", result)
	}
}

func TestReadFileTool_HashlineFormat(t *testing.T) {
	dir := t.TempDir()
	path := writeTestFile(t, dir, "test.txt", "first line\nsecond line\n")

	result := callTool(t, ReadFileTool, readFileArgs{Path: path})

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

func TestReadFileTool_EndLineClampedToFile(t *testing.T) {
	dir := t.TempDir()
	path := writeTestFile(t, dir, "test.txt", "a\nb\nc\n")

	result := callTool(t, ReadFileTool, readFileArgs{Path: path, StartLine: 2, EndLine: 100})
	if !strings.Contains(result, "[lines 2-3 of 3]") {
		t.Errorf("expected end_line clamped, got: %s", strings.SplitN(result, "\n", 2)[0])
	}
}

// --- list_directory tool tests ---

func TestListDirTool_Basic(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, dir, "a.txt", "hello")
	writeTestFile(t, dir, "b.txt", "world")

	result := callTool(t, ListDirTool, listDirArgs{Path: dir})

	if !strings.Contains(result, "a.txt") {
		t.Errorf("expected 'a.txt' in output, got: %s", result)
	}
	if !strings.Contains(result, "b.txt") {
		t.Errorf("expected 'b.txt' in output, got: %s", result)
	}
}

func TestListDirTool_EmptyDir(t *testing.T) {
	dir := t.TempDir()
	result := callTool(t, ListDirTool, listDirArgs{Path: dir})
	if !strings.Contains(result, "(empty directory)") {
		t.Errorf("expected '(empty directory)', got: %s", result)
	}
}

func TestListDirTool_NonexistentDir(t *testing.T) {
	result := callTool(t, ListDirTool, listDirArgs{Path: "/nonexistent/dir"})
	if !strings.Contains(result, "error:") {
		t.Errorf("expected error, got: %s", result)
	}
}

func TestListDirTool_OffsetLimit(t *testing.T) {
	dir := t.TempDir()
	for i := 0; i < 300; i++ {
		writeTestFile(t, dir, fmt.Sprintf("file_%04d.txt", i), "")
	}

	// Default: first 200 entries
	result := callTool(t, ListDirTool, listDirArgs{Path: dir})
	if !strings.Contains(result, "[entries 1-200 of 300]") {
		t.Errorf("expected [entries 1-200 of 300], got first line: %s", strings.SplitN(result, "\n", 2)[0])
	}

	// Offset 201
	result = callTool(t, ListDirTool, listDirArgs{Path: dir, Offset: 201})
	if !strings.Contains(result, "[entries 201-300 of 300]") {
		t.Errorf("expected [entries 201-300 of 300], got first line: %s", strings.SplitN(result, "\n", 2)[0])
	}

	// Custom limit
	result = callTool(t, ListDirTool, listDirArgs{Path: dir, Offset: 1, Limit: 50})
	if !strings.Contains(result, "[entries 1-50 of 300]") {
		t.Errorf("expected [entries 1-50 of 300], got first line: %s", strings.SplitN(result, "\n", 2)[0])
	}
}

func TestListDirTool_OffsetBeyondEntries(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, dir, "a.txt", "")

	result := callTool(t, ListDirTool, listDirArgs{Path: dir, Offset: 100})
	if !strings.Contains(result, "error:") {
		t.Errorf("expected error for offset beyond entries, got: %s", result)
	}
}

func TestListDirTool_LimitCapped(t *testing.T) {
	dir := t.TempDir()
	for i := 0; i < 10; i++ {
		writeTestFile(t, dir, fmt.Sprintf("f%d.txt", i), "")
	}

	// Limit > 500 should be capped to 500
	result := callTool(t, ListDirTool, listDirArgs{Path: dir, Limit: 1000})
	// Should still work, just returns all 10 entries
	if !strings.Contains(result, "[entries 1-10 of 10]") {
		t.Errorf("expected all entries returned, got: %s", strings.SplitN(result, "\n", 2)[0])
	}
}

func TestListDirTool_ShowsPermissions(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, dir, "test.txt", "hello")

	result := callTool(t, ListDirTool, listDirArgs{Path: dir})

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
	result := callTool(t, EditFileTool, editFileArgs{
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
	result := callTool(t, EditFileTool, editFileArgs{
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
	result := callTool(t, EditFileTool, editFileArgs{
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
	result := callTool(t, EditFileTool, editFileArgs{
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
	result := callTool(t, EditFileTool, editFileArgs{
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

	result := callTool(t, EditFileTool, editFileArgs{
		Path:      path,
		Operation: "replace",
		Start:     "1:zz",
		Content:   "replaced",
	})

	if !strings.Contains(result, "hash mismatch") {
		t.Errorf("expected hash mismatch error, got: %s", result)
	}
	// Should include context lines
	if !strings.Contains(result, "Current content around line") {
		t.Errorf("expected context in error, got: %s", result)
	}
	if !strings.Contains(result, "|hello") {
		t.Errorf("expected actual line content in error, got: %s", result)
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
	result := callTool(t, EditFileTool, editFileArgs{
		Path:      path,
		Operation: "replace",
		Start:     "5:" + hash,
		Content:   "replaced",
	})

	if !strings.Contains(result, "does not exist") {
		t.Errorf("expected 'does not exist' error, got: %s", result)
	}
	// Should include end-of-file context
	if !strings.Contains(result, "End of file:") {
		t.Errorf("expected end-of-file context in error, got: %s", result)
	}
	if !strings.Contains(result, "|hello") {
		t.Errorf("expected file content in error, got: %s", result)
	}
}

func TestEditFileTool_EndBeforeStart(t *testing.T) {
	dir := t.TempDir()
	path := writeTestFile(t, dir, "test.txt", "a\nb\nc\n")

	hashA := getHash(t, "a")
	hashC := getHash(t, "c")
	result := callTool(t, EditFileTool, editFileArgs{
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
	result := callTool(t, EditFileTool, editFileArgs{
		Operation: "replace",
		Start:     "1:ff",
		Content:   "x",
	})
	if !strings.Contains(result, "error:") {
		t.Errorf("expected error for missing path, got: %s", result)
	}
}

func TestEditFileTool_NonexistentFile(t *testing.T) {
	result := callTool(t, EditFileTool, editFileArgs{
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
	callTool(t, EditFileTool, editFileArgs{
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
	callTool(t, EditFileTool, editFileArgs{
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
	callTool(t, EditFileTool, editFileArgs{
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
	result := callTool(t, BashTool, bashArgs{Command: "echo hello"})
	if strings.TrimSpace(result) != "hello" {
		t.Errorf("expected 'hello', got: %q", result)
	}
}

func TestBashTool_ExitStatus(t *testing.T) {
	result := callTool(t, BashTool, bashArgs{Command: "exit 1"})
	if !strings.Contains(result, "exit status") {
		t.Errorf("expected exit status error, got: %s", result)
	}
}

func TestBashTool_Stderr(t *testing.T) {
	result := callTool(t, BashTool, bashArgs{Command: "echo error >&2"})
	if !strings.Contains(result, "error") {
		t.Errorf("expected stderr in output, got: %s", result)
	}
}

func TestBashTool_MissingCommand(t *testing.T) {
	result := callTool(t, BashTool, bashArgs{})
	if !strings.Contains(result, "error:") {
		t.Errorf("expected error for missing command, got: %s", result)
	}
}

func TestBashTool_OutputTruncation(t *testing.T) {
	result := callTool(t, BashTool, bashArgs{
		Command: "yes 'this is a long line of text for testing truncation' | head -n 100000",
	})
	if !strings.Contains(result, "... (output truncated)") {
		t.Error("expected truncation notice in output")
	}
	if !strings.Contains(result, "this is a long line") {
		t.Error("expected tail content to be preserved")
	}
}

func TestBashTool_MultilineOutput(t *testing.T) {
	result := callTool(t, BashTool, bashArgs{Command: "echo line1; echo line2; echo line3"})
	if !strings.Contains(result, "line1") || !strings.Contains(result, "line3") {
		t.Errorf("expected multiline output, got: %s", result)
	}
}

// --- create_file tool tests ---

func TestCreateFileTool_Basic(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "new.txt")

	result := callTool(t, CreateFileTool, createFileArgs{Path: path, Content: "hello\nworld\n"})
	if !strings.Contains(result, "ok: created") {
		t.Fatalf("expected ok, got: %s", result)
	}
	if !strings.Contains(result, "|hello") {
		t.Errorf("expected hashline content in output, got: %s", result)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "hello\nworld\n" {
		t.Errorf("unexpected file content: %q", string(data))
	}
}

func TestCreateFileTool_RefusesOverwrite(t *testing.T) {
	dir := t.TempDir()
	path := writeTestFile(t, dir, "existing.txt", "original")

	result := callTool(t, CreateFileTool, createFileArgs{Path: path, Content: "overwrite"})
	if !strings.Contains(result, "error: file already exists") {
		t.Errorf("expected overwrite refusal, got: %s", result)
	}

	data, _ := os.ReadFile(path)
	if string(data) != "original" {
		t.Errorf("file should be unchanged, got: %q", string(data))
	}
}

func TestCreateFileTool_EmptyContent(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "empty.txt")

	result := callTool(t, CreateFileTool, createFileArgs{Path: path, Content: ""})
	if !strings.Contains(result, "ok: created") {
		t.Fatalf("expected ok, got: %s", result)
	}
	if !strings.Contains(result, "empty file") {
		t.Errorf("expected empty file note, got: %s", result)
	}
}

func TestCreateFileTool_MissingPath(t *testing.T) {
	result := callTool(t, CreateFileTool, createFileArgs{Content: "hello"})
	if !strings.Contains(result, "error:") {
		t.Errorf("expected error for missing path, got: %s", result)
	}
}

func TestCreateFileTool_NoParentDir(t *testing.T) {
	result := callTool(t, CreateFileTool, createFileArgs{Path: "/nonexistent/dir/file.txt", Content: "hello"})
	if !strings.Contains(result, "error:") {
		t.Errorf("expected error for missing parent dir, got: %s", result)
	}
}

// --- grep_file tool tests ---

func TestGrepFileTool_Basic(t *testing.T) {
	dir := t.TempDir()
	path := writeTestFile(t, dir, "test.txt", "hello world\nfoo bar\nhello again\n")

	result := callTool(t, GrepFileTool, grepFileArgs{Path: path, Query: "hello"})
	if !strings.Contains(result, "2 matches") {
		t.Errorf("expected 2 matches, got: %s", result)
	}
	if !strings.Contains(result, "|hello world") {
		t.Errorf("expected 'hello world' match, got: %s", result)
	}
	if !strings.Contains(result, "|hello again") {
		t.Errorf("expected 'hello again' match, got: %s", result)
	}
}

func TestGrepFileTool_NoMatches(t *testing.T) {
	dir := t.TempDir()
	path := writeTestFile(t, dir, "test.txt", "hello world\n")

	result := callTool(t, GrepFileTool, grepFileArgs{Path: path, Query: "xyz"})
	if !strings.Contains(result, "no matches") {
		t.Errorf("expected no matches, got: %s", result)
	}
}

func TestGrepFileTool_NoPageAnnotation(t *testing.T) {
	dir := t.TempDir()
	var sb strings.Builder
	for i := 0; i < 150; i++ {
		fmt.Fprintf(&sb, "line %d\n", i+1)
	}
	path := writeTestFile(t, dir, "test.txt", sb.String())

	result := callTool(t, GrepFileTool, grepFileArgs{Path: path, Query: "line 150"})
	// Should NOT contain page annotations
	if strings.Contains(result, "[page") {
		t.Errorf("expected no page annotations, got: %s", result)
	}
	if !strings.Contains(result, "1 match") {
		t.Errorf("expected 1 match, got: %s", result)
	}
}

func TestGrepFileTool_ContextLines(t *testing.T) {
	dir := t.TempDir()
	path := writeTestFile(t, dir, "test.txt", "aaa\nbbb\nccc\nddd\neee\n")

	result := callTool(t, GrepFileTool, grepFileArgs{Path: path, Query: "ccc", ContextLines: 1})
	if !strings.Contains(result, "1 match") {
		t.Errorf("expected 1 match, got: %s", result)
	}
	// Match line should be prefixed with "> "
	if !strings.Contains(result, "> ") {
		t.Errorf("expected '> ' prefix for match line, got: %s", result)
	}
	// Context lines should be prefixed with "  "
	if !strings.Contains(result, "  2:") {
		t.Errorf("expected context line for bbb (line 2), got: %s", result)
	}
	if !strings.Contains(result, "  4:") {
		t.Errorf("expected context line for ddd (line 4), got: %s", result)
	}
}

func TestGrepFileTool_ContextLinesMerge(t *testing.T) {
	dir := t.TempDir()
	path := writeTestFile(t, dir, "test.txt", "aaa\nbbb\nccc\nddd\neee\n")

	// Both bbb (line 2) and ddd (line 4) match, with context 1 their windows overlap
	result := callTool(t, GrepFileTool, grepFileArgs{Path: path, Query: "b\n", ContextLines: 1})
	// This specific query only matches "bbb" since we look for "b\n" substring within a line
	// Let me use a better query
	result = callTool(t, GrepFileTool, grepFileArgs{Path: path, Query: "bb", ContextLines: 2})
	// bbb is at line 2, context 2 means lines 1-4
	if !strings.Contains(result, "> ") {
		t.Errorf("expected match prefix, got: %s", result)
	}
}

func TestGrepFileTool_ContextMergeAdjacentMatches(t *testing.T) {
	dir := t.TempDir()
	// Lines where match1 context and match2 context overlap
	path := writeTestFile(t, dir, "test.txt", "x1\nmatch1\nx2\nmatch2\nx3\n")

	result := callTool(t, GrepFileTool, grepFileArgs{Path: path, Query: "match", ContextLines: 1})
	if !strings.Contains(result, "2 matches") {
		t.Errorf("expected 2 matches, got: %s", result)
	}
	// Both matches and their context should be in one merged group (no blank line separator)
	lines := strings.Split(strings.TrimSpace(result), "\n")
	blankCount := 0
	for _, l := range lines[1:] { // skip "2 matches:" header
		if l == "" {
			blankCount++
		}
	}
	if blankCount > 0 {
		t.Errorf("expected merged group (no blank separators), got %d blank lines", blankCount)
	}
}

func TestGrepFileTool_ContextSeparateGroups(t *testing.T) {
	dir := t.TempDir()
	// Matches far enough apart that context windows don't overlap
	path := writeTestFile(t, dir, "test.txt", "match1\na\nb\nc\nd\ne\nf\nmatch2\n")

	result := callTool(t, GrepFileTool, grepFileArgs{Path: path, Query: "match", ContextLines: 1})
	if !strings.Contains(result, "2 matches") {
		t.Errorf("expected 2 matches, got: %s", result)
	}
	// Should have a blank line between non-contiguous groups
	parts := strings.Split(result, "\n\n")
	if len(parts) < 2 {
		t.Errorf("expected separate groups with blank line, got: %s", result)
	}
}

func TestGrepFileTool_CaseSensitive(t *testing.T) {
	dir := t.TempDir()
	path := writeTestFile(t, dir, "test.txt", "Hello\nhello\nHELLO\n")

	result := callTool(t, GrepFileTool, grepFileArgs{Path: path, Query: "hello"})
	if !strings.Contains(result, "1 match") {
		t.Errorf("expected exactly 1 match (case-sensitive), got: %s", result)
	}
}

func TestGrepFileTool_MaxMatches(t *testing.T) {
	dir := t.TempDir()
	var sb strings.Builder
	for i := 0; i < 150; i++ {
		sb.WriteString("match\n")
	}
	path := writeTestFile(t, dir, "test.txt", sb.String())

	result := callTool(t, GrepFileTool, grepFileArgs{Path: path, Query: "match"})
	if !strings.Contains(result, "100 matches") {
		t.Errorf("expected 100 matches (capped), got: %s", result)
	}
	if !strings.Contains(result, "truncated") {
		t.Errorf("expected truncation note, got: %s", result)
	}
}

func TestGrepFileTool_MissingArgs(t *testing.T) {
	result := callTool(t, GrepFileTool, grepFileArgs{Path: "/tmp/x"})
	if !strings.Contains(result, "error:") {
		t.Errorf("expected error for missing query, got: %s", result)
	}

	result = callTool(t, GrepFileTool, grepFileArgs{Query: "x"})
	if !strings.Contains(result, "error:") {
		t.Errorf("expected error for missing path, got: %s", result)
	}
}

// --- tree tool tests ---

func TestTreeTool_Basic(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "subdir"), 0755)
	writeTestFile(t, dir, "file1.txt", "hello")
	writeTestFile(t, filepath.Join(dir, "subdir"), "file2.txt", "world")

	result := callTool(t, TreeTool, treeArgs{Path: dir})
	if !strings.Contains(result, "subdir") {
		t.Errorf("expected 'subdir' in output, got: %s", result)
	}
	if !strings.Contains(result, "file1.txt") {
		t.Errorf("expected 'file1.txt' in output, got: %s", result)
	}
	if !strings.Contains(result, "file2.txt") {
		t.Errorf("expected 'file2.txt' in output, got: %s", result)
	}
}

func TestTreeTool_DirsBeforeFiles(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "adir"), 0755)
	writeTestFile(t, dir, "bfile.txt", "")

	result := callTool(t, TreeTool, treeArgs{Path: dir})
	adirIdx := strings.Index(result, "adir")
	bfileIdx := strings.Index(result, "bfile.txt")
	if adirIdx < 0 || bfileIdx < 0 {
		t.Fatalf("expected both entries, got: %s", result)
	}
	if adirIdx > bfileIdx {
		t.Errorf("expected directory before file, got: %s", result)
	}
}

func TestTreeTool_HiddenFilesDefault(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, dir, ".hidden", "")
	writeTestFile(t, dir, "visible", "")

	result := callTool(t, TreeTool, treeArgs{Path: dir})
	if strings.Contains(result, ".hidden") {
		t.Errorf("expected hidden file to be excluded, got: %s", result)
	}
	if !strings.Contains(result, "visible") {
		t.Errorf("expected visible file in output, got: %s", result)
	}
}

func TestTreeTool_ShowHidden(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, dir, ".hidden", "")
	writeTestFile(t, dir, "visible", "")

	result := callTool(t, TreeTool, treeArgs{Path: dir, ShowHidden: true})
	if !strings.Contains(result, ".hidden") {
		t.Errorf("expected hidden file with show_hidden, got: %s", result)
	}
}

func TestTreeTool_DepthLimit(t *testing.T) {
	dir := t.TempDir()
	deep := filepath.Join(dir, "a", "b", "c")
	os.MkdirAll(deep, 0755)
	writeTestFile(t, deep, "deep.txt", "")

	result := callTool(t, TreeTool, treeArgs{Path: dir, Depth: 1})
	if !strings.Contains(result, "a") {
		t.Errorf("expected 'a' at depth 1, got: %s", result)
	}
	if strings.Contains(result, "b") {
		t.Errorf("expected 'b' to be excluded at depth 1, got: %s", result)
	}
}

func TestTreeTool_OffsetLimit(t *testing.T) {
	dir := t.TempDir()
	for i := 0; i < 250; i++ {
		writeTestFile(t, dir, fmt.Sprintf("file_%04d.txt", i), "")
	}

	// Default: first 200 lines
	result := callTool(t, TreeTool, treeArgs{Path: dir})
	header := strings.SplitN(result, "\n", 2)[0]
	if !strings.Contains(header, "[lines 1-200 of") {
		t.Errorf("expected [lines 1-200 of ...], got: %s", header)
	}

	// Offset beyond first page
	result = callTool(t, TreeTool, treeArgs{Path: dir, Offset: 201})
	header = strings.SplitN(result, "\n", 2)[0]
	if !strings.Contains(header, "[lines 201-") {
		t.Errorf("expected [lines 201-...], got: %s", header)
	}

	// Custom limit
	result = callTool(t, TreeTool, treeArgs{Path: dir, Offset: 1, Limit: 50})
	header = strings.SplitN(result, "\n", 2)[0]
	if !strings.Contains(header, "[lines 1-50 of") {
		t.Errorf("expected [lines 1-50 of ...], got: %s", header)
	}
}

func TestTreeTool_OffsetBeyondLines(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, dir, "a.txt", "")

	result := callTool(t, TreeTool, treeArgs{Path: dir, Offset: 100})
	if !strings.Contains(result, "error:") {
		t.Errorf("expected error for offset beyond lines, got: %s", result)
	}
}

func TestTreeTool_NonexistentDir(t *testing.T) {
	result := callTool(t, TreeTool, treeArgs{Path: "/nonexistent/dir"})
	if !strings.Contains(result, "error:") {
		t.Errorf("expected error, got: %s", result)
	}
}

func TestTreeTool_TreeFormat(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "dir1"), 0755)
	writeTestFile(t, dir, "file1.txt", "")

	result := callTool(t, TreeTool, treeArgs{Path: dir})
	if !strings.Contains(result, "├── ") && !strings.Contains(result, "└── ") {
		t.Errorf("expected tree connectors in output, got: %s", result)
	}
}

// --- find_replace tool tests ---

func TestFindReplaceTool_Basic(t *testing.T) {
	dir := t.TempDir()
	path := writeTestFile(t, dir, "test.txt", "hello world\nfoo bar\n")

	result := callTool(t, FindReplaceTool, findReplaceArgs{
		Path:    path,
		OldText: "foo bar",
		NewText: "baz qux",
	})
	if !strings.Contains(result, "ok: replaced") {
		t.Fatalf("expected ok, got: %s", result)
	}

	data, _ := os.ReadFile(path)
	if string(data) != "hello world\nbaz qux\n" {
		t.Errorf("unexpected file content: %q", string(data))
	}
}

func TestFindReplaceTool_MultiLine(t *testing.T) {
	dir := t.TempDir()
	path := writeTestFile(t, dir, "test.txt", "aaa\nbbb\nccc\nddd\n")

	result := callTool(t, FindReplaceTool, findReplaceArgs{
		Path:    path,
		OldText: "bbb\nccc",
		NewText: "xxx\nyyy\nzzz",
	})
	if !strings.Contains(result, "ok: replaced") {
		t.Fatalf("expected ok, got: %s", result)
	}

	data, _ := os.ReadFile(path)
	if string(data) != "aaa\nxxx\nyyy\nzzz\nddd\n" {
		t.Errorf("unexpected file content: %q", string(data))
	}
}

func TestFindReplaceTool_Delete(t *testing.T) {
	dir := t.TempDir()
	path := writeTestFile(t, dir, "test.txt", "keep\ndelete_me\nkeep2\n")

	result := callTool(t, FindReplaceTool, findReplaceArgs{
		Path:    path,
		OldText: "delete_me\n",
		NewText: "",
	})
	if !strings.Contains(result, "ok: replaced") {
		t.Fatalf("expected ok, got: %s", result)
	}

	data, _ := os.ReadFile(path)
	if string(data) != "keep\nkeep2\n" {
		t.Errorf("unexpected file content: %q", string(data))
	}
}

func TestFindReplaceTool_NotFound(t *testing.T) {
	dir := t.TempDir()
	path := writeTestFile(t, dir, "test.txt", "hello world\n")

	result := callTool(t, FindReplaceTool, findReplaceArgs{
		Path:    path,
		OldText: "nonexistent",
		NewText: "replacement",
	})
	if !strings.Contains(result, "error: old_text not found") {
		t.Errorf("expected not-found error, got: %s", result)
	}
}

func TestFindReplaceTool_MultipleOccurrences(t *testing.T) {
	dir := t.TempDir()
	path := writeTestFile(t, dir, "test.txt", "aaa\nbbb\naaa\n")

	result := callTool(t, FindReplaceTool, findReplaceArgs{
		Path:    path,
		OldText: "aaa",
		NewText: "xxx",
	})
	if !strings.Contains(result, "found 2 occurrences") {
		t.Errorf("expected multiple-occurrence error, got: %s", result)
	}
	if !strings.Contains(result, "more surrounding context") {
		t.Errorf("expected context hint in error, got: %s", result)
	}

	// File should be unchanged
	data, _ := os.ReadFile(path)
	if string(data) != "aaa\nbbb\naaa\n" {
		t.Errorf("file should be unchanged, got: %q", string(data))
	}
}

func TestFindReplaceTool_MissingPath(t *testing.T) {
	result := callTool(t, FindReplaceTool, findReplaceArgs{
		OldText: "x",
		NewText: "y",
	})
	if !strings.Contains(result, "error:") {
		t.Errorf("expected error for missing path, got: %s", result)
	}
}

func TestFindReplaceTool_MissingOldText(t *testing.T) {
	result := callTool(t, FindReplaceTool, findReplaceArgs{
		Path:    "/tmp/x",
		NewText: "y",
	})
	if !strings.Contains(result, "error:") {
		t.Errorf("expected error for missing old_text, got: %s", result)
	}
}

func TestFindReplaceTool_NonexistentFile(t *testing.T) {
	result := callTool(t, FindReplaceTool, findReplaceArgs{
		Path:    "/nonexistent/file.txt",
		OldText: "x",
		NewText: "y",
	})
	if !strings.Contains(result, "error:") {
		t.Errorf("expected error, got: %s", result)
	}
}

// --- Integration: read then edit ---

func TestIntegration_ReadThenEdit(t *testing.T) {
	dir := t.TempDir()
	path := writeTestFile(t, dir, "test.go", "package main\n\nfunc hello() {\n\treturn \"world\"\n}\n")

	// Read the file
	readResult := callTool(t, ReadFileTool, readFileArgs{Path: path})

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
	editResult := callTool(t, EditFileTool, editFileArgs{
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
