package main

import (
	"context"
	"encoding/json"
	"fmt"
	"hash/fnv"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/modfin/bellman/tools"
)

const (
	linesPerPage    = 100
	entriesPerPage  = 100
	bashTimeout     = 60 * time.Second
	bashMaxOutput   = 64 * 1024
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

// --- Read File Tool ---

type readFileArgs struct {
	Path string `json:"path" json-description:"The path to the file to read"`
	Page int    `json:"page,omitempty" json-description:"Page number (1-indexed). If omitted or 0, returns page 1."`
}

var readFileTool = tools.NewTool("read_file",
	tools.WithDescription("Read a file with hashline-prefixed lines. Returns one page at a time (100 lines per page). Each line is prefixed with its line number and a content hash in the format 'line:hash|content'. Use the line:hash references with the edit_file tool."),
	tools.WithArgSchema(readFileArgs{}),
	tools.WithFunction(func(ctx context.Context, call tools.Call) (string, error) {
		var params readFileArgs
		if err := json.Unmarshal(call.Argument, &params); err != nil {
			return fmt.Sprintf("error: invalid arguments: %v", err), nil
		}
		if params.Path == "" {
			return "error: path is required", nil
		}
		if params.Page < 1 {
			params.Page = 1
		}

		lines, _, err := readFileLines(params.Path)
		if err != nil {
			return fmt.Sprintf("error: %v", err), nil
		}

		totalLines := len(lines)
		if totalLines == 0 {
			return "[page 1/1, 0 lines]\n(empty file)", nil
		}

		totalPages := (totalLines + linesPerPage - 1) / linesPerPage

		if params.Page > totalPages {
			return fmt.Sprintf("error: page %d does not exist (file has %d pages)", params.Page, totalPages), nil
		}

		startIdx := (params.Page - 1) * linesPerPage
		endIdx := startIdx + linesPerPage
		if endIdx > totalLines {
			endIdx = totalLines
		}

		pageLines := lines[startIdx:endIdx]
		content := formatHashlines(pageLines, startIdx)

		header := fmt.Sprintf("[page %d/%d, lines %d-%d of %d]\n", params.Page, totalPages, startIdx+1, endIdx, totalLines)
		return header + content, nil
	}),
)

// --- List Directory Tool ---

type listDirArgs struct {
	Path string `json:"path,omitempty" json-description:"The directory path to list. Defaults to the current working directory."`
	Page int    `json:"page,omitempty" json-description:"Page number (1-indexed). If omitted or 0, returns page 1."`
}

// formatDirEntry formats a single directory entry with permissions, size, mod time, and name.
func formatDirEntry(path string, entry os.DirEntry) string {
	info, err := entry.Info()
	if err != nil {
		return fmt.Sprintf("  ?  %s", entry.Name())
	}
	return fmt.Sprintf("%s  %10d  %s  %s",
		info.Mode().String(),
		info.Size(),
		info.ModTime().Format("2006-01-02 15:04"),
		entry.Name(),
	)
}

var listDirTool = tools.NewTool("list_directory",
	tools.WithDescription("List directory contents with details (permissions, size, modification time, name). Returns one page at a time (100 entries per page)."),
	tools.WithArgSchema(listDirArgs{}),
	tools.WithFunction(func(ctx context.Context, call tools.Call) (string, error) {
		var params listDirArgs
		if err := json.Unmarshal(call.Argument, &params); err != nil {
			return fmt.Sprintf("error: invalid arguments: %v", err), nil
		}
		path := params.Path
		if path == "" {
			path = "."
		}
		if params.Page < 1 {
			params.Page = 1
		}

		entries, err := os.ReadDir(path)
		if err != nil {
			return fmt.Sprintf("error: %v", err), nil
		}

		totalEntries := len(entries)
		if totalEntries == 0 {
			return "[page 1/1, 0 entries]\n(empty directory)", nil
		}

		totalPages := (totalEntries + entriesPerPage - 1) / entriesPerPage

		if params.Page > totalPages {
			return fmt.Sprintf("error: page %d does not exist (directory has %d pages)", params.Page, totalPages), nil
		}

		startIdx := (params.Page - 1) * entriesPerPage
		endIdx := startIdx + entriesPerPage
		if endIdx > totalEntries {
			endIdx = totalEntries
		}

		var sb strings.Builder
		fmt.Fprintf(&sb, "[page %d/%d, entries %d-%d of %d]\n", params.Page, totalPages, startIdx+1, endIdx, totalEntries)
		for _, entry := range entries[startIdx:endIdx] {
			sb.WriteString(formatDirEntry(path, entry))
			sb.WriteByte('\n')
		}
		return sb.String(), nil
	}),
)

// --- Edit File Tool ---

type editFileArgs struct {
	Path      string `json:"path" json-description:"The path to the file to edit"`
	Operation string `json:"operation" json-description:"The edit operation: replace or insert_after" json-enum:"replace,insert_after"`
	Start     string `json:"start" json-description:"Hashline reference for the target line (format: line_number:hash, e.g. 5:a3)"`
	End       string `json:"end,omitempty" json-description:"Hashline reference for end of a range (format: line_number:hash). Only used with replace for multi-line ranges. If omitted, only the start line is replaced."`
	Content   string `json:"content" json-description:"The new content to insert or replace with. Use newlines for multiple lines. Empty string with replace deletes lines."`
}

var editFileTool = tools.NewTool("edit_file",
	tools.WithDescription("Edit a file using hashline references from read_file. Supports replacing lines (single or range) and inserting after a line. The hash in each reference is validated to ensure the file hasn't changed since it was last read."),
	tools.WithArgSchema(editFileArgs{}),
	tools.WithFunction(func(ctx context.Context, call tools.Call) (string, error) {
		var params editFileArgs
		if err := json.Unmarshal(call.Argument, &params); err != nil {
			return fmt.Sprintf("error: invalid arguments: %v", err), nil
		}
		if params.Path == "" {
			return "error: path is required", nil
		}
		if params.Operation == "" {
			return "error: operation is required", nil
		}
		if params.Start == "" {
			return "error: start is required", nil
		}

		lines, endsWithNewline, err := readFileLines(params.Path)
		if err != nil {
			return fmt.Sprintf("error: %v", err), nil
		}

		startLine, startHash, err := parseHashlineRef(params.Start)
		if err != nil {
			return fmt.Sprintf("error: %v", err), nil
		}
		if err := validateHashlineRef(lines, startLine, startHash); err != nil {
			return fmt.Sprintf("error: %v", err), nil
		}

		var newLines []string
		if params.Content != "" {
			newLines = strings.Split(params.Content, "\n")
		}

		switch params.Operation {
		case "replace":
			endLine := startLine
			if params.End != "" {
				var endHash string
				endLine, endHash, err = parseHashlineRef(params.End)
				if err != nil {
					return fmt.Sprintf("error: %v", err), nil
				}
				if err := validateHashlineRef(lines, endLine, endHash); err != nil {
					return fmt.Sprintf("error: %v", err), nil
				}
				if endLine < startLine {
					return fmt.Sprintf("error: end line %d is before start line %d", endLine, startLine), nil
				}
			}

			result := make([]string, 0, len(lines)-(endLine-startLine+1)+len(newLines))
			result = append(result, lines[:startLine-1]...)
			result = append(result, newLines...)
			result = append(result, lines[endLine:]...)
			lines = result

		case "insert_after":
			result := make([]string, 0, len(lines)+len(newLines))
			result = append(result, lines[:startLine]...)
			result = append(result, newLines...)
			result = append(result, lines[startLine:]...)
			lines = result

		default:
			return fmt.Sprintf("error: unknown operation %q", params.Operation), nil
		}

		content := strings.Join(lines, "\n")
		if endsWithNewline || len(lines) > 0 {
			content += "\n"
		}
		if err := os.WriteFile(params.Path, []byte(content), 0644); err != nil {
			return fmt.Sprintf("error writing file: %v", err), nil
		}

		return fmt.Sprintf("ok: %s applied to %s (%d lines)", params.Operation, params.Path, len(lines)), nil
	}),
)

// --- Bash Tool ---

type bashArgs struct {
	Command string `json:"command" json-description:"The bash command to execute"`
}

var bashTool = tools.NewTool("bash",
	tools.WithDescription("Execute a bash command and return its combined stdout and stderr output."),
	tools.WithArgSchema(bashArgs{}),
	tools.WithFunction(func(ctx context.Context, call tools.Call) (string, error) {
		var params bashArgs
		if err := json.Unmarshal(call.Argument, &params); err != nil {
			return fmt.Sprintf("error: invalid arguments: %v", err), nil
		}
		if params.Command == "" {
			return "error: command is required", nil
		}

		ctx, cancel := context.WithTimeout(ctx, bashTimeout)
		defer cancel()

		cmd := exec.CommandContext(ctx, "bash", "-c", params.Command)
		output, err := cmd.CombinedOutput()
		if len(output) > bashMaxOutput {
			output = append([]byte("... (output truncated)\n"), output[len(output)-bashMaxOutput:]...)
		}
		if ctx.Err() == context.DeadlineExceeded {
			return fmt.Sprintf("error: command timed out after %v\n%s", bashTimeout, string(output)), nil
		}
		if err != nil {
			return fmt.Sprintf("exit status: %v\n%s", err, string(output)), nil
		}
		return string(output), nil
	}),
)
