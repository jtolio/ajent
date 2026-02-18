package tools

import (
	"fmt"
	"hash/fnv"
	"os"
	"strconv"
	"strings"
)

const (
	linesPerPage   = 100
	entriesPerPage = 100
)

// hashLineContent returns a 2-character hex hash of the line content.
func hashLineContent(line string) string {
	h := fnv.New32a()
	h.Write([]byte(line))
	return fmt.Sprintf("%02x", byte(h.Sum32()))
}

// readFileLines reads a file and returns its lines (without trailing newlines)
// and whether the original file ended with a newline.
func readFileLines(path string) ([]string, bool, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, false, err
	}
	content := string(data)
	if content == "" {
		return nil, false, nil
	}
	endsWithNewline := strings.HasSuffix(content, "\n")
	lines := strings.Split(content, "\n")
	if endsWithNewline && len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	return lines, endsWithNewline, nil
}

// formatHashlines formats lines with hashline prefixes.
// lineOffset is 0-indexed (the index of the first line in the full file).
func formatHashlines(lines []string, lineOffset int) string {
	var sb strings.Builder
	for i, line := range lines {
		lineNum := lineOffset + i + 1 // 1-indexed
		hash := hashLineContent(line)
		fmt.Fprintf(&sb, "%d:%s|%s\n", lineNum, hash, line)
	}
	return sb.String()
}

// parseHashlineRef parses a hashline reference like "5:a3" into line number and hash.
func parseHashlineRef(ref string) (lineNum int, hash string, err error) {
	parts := strings.SplitN(ref, ":", 2)
	if len(parts) != 2 {
		return 0, "", fmt.Errorf("invalid hashline reference %q: expected format 'line:hash'", ref)
	}
	lineNum, err = strconv.Atoi(parts[0])
	if err != nil {
		return 0, "", fmt.Errorf("invalid line number in %q: %v", ref, err)
	}
	if lineNum < 1 {
		return 0, "", fmt.Errorf("line number must be >= 1, got %d", lineNum)
	}
	return lineNum, parts[1], nil
}

// validateHashlineRef checks that the line at lineNum (1-indexed) has the expected hash.
func validateHashlineRef(lines []string, lineNum int, expectedHash string) error {
	if lineNum > len(lines) {
		return fmt.Errorf("line %d does not exist (file has %d lines)", lineNum, len(lines))
	}
	actualHash := hashLineContent(lines[lineNum-1])
	if actualHash != expectedHash {
		return fmt.Errorf("hash mismatch at line %d: expected %s, got %s (file has changed since last read)", lineNum, expectedHash, actualHash)
	}
	return nil
}
