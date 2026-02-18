package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/modfin/bellman/tools"
)

type editFileArgs struct {
	Path      string `json:"path" json-description:"The path to the file to edit"`
	Operation string `json:"operation" json-description:"The edit operation: replace or insert_after" json-enum:"replace,insert_after"`
	Start     string `json:"start" json-description:"Hashline reference for the target line (format: line_number:hash, e.g. 5:a3)"`
	End       string `json:"end,omitempty" json-description:"Hashline reference for end of a range (format: line_number:hash). Only used with replace for multi-line ranges. If omitted, only the start line is replaced."`
	Content   string `json:"content" json-description:"The new content to insert or replace with. Use newlines for multiple lines. Empty string with replace deletes lines."`
}

// formatContextLines returns a few lines around targetLine (1-indexed) in hashline format for error messages.
func formatContextLines(lines []string, targetLine, contextRadius int) string {
	start := targetLine - contextRadius
	if start < 1 {
		start = 1
	}
	end := targetLine + contextRadius
	if end > len(lines) {
		end = len(lines)
	}
	var sb strings.Builder
	for i := start; i <= end; i++ {
		hash := hashLineContent(lines[i-1])
		fmt.Fprintf(&sb, "  %d:%s|%s\n", i, hash, lines[i-1])
	}
	return sb.String()
}

var EditFileTool = tools.NewTool("edit_file",
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
			// Provide context in the error message
			if startLine > len(lines) {
				// Out of bounds: show the last few lines
				lastStart := len(lines) - 2
				if lastStart < 1 {
					lastStart = 1
				}
				ctx := formatContextLines(lines, len(lines), 2)
				return fmt.Sprintf("error: line %d does not exist (file has %d lines)\nEnd of file:\n%s", startLine, len(lines), ctx), nil
			}
			// Hash mismatch: show context around the target line
			ctx := formatContextLines(lines, startLine, 3)
			return fmt.Sprintf("error: hash mismatch at line %d: expected %s, got %s (file has changed since last read)\nCurrent content around line %d:\n%s",
				startLine, startHash, hashLineContent(lines[startLine-1]), startLine, ctx), nil
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
					if endLine > len(lines) {
						lastStart := len(lines) - 2
						if lastStart < 1 {
							lastStart = 1
						}
						ctx := formatContextLines(lines, len(lines), 2)
						return fmt.Sprintf("error: line %d does not exist (file has %d lines)\nEnd of file:\n%s", endLine, len(lines), ctx), nil
					}
					ctx := formatContextLines(lines, endLine, 3)
					return fmt.Sprintf("error: hash mismatch at line %d: expected %s, got %s (file has changed since last read)\nCurrent content around line %d:\n%s",
						endLine, endHash, hashLineContent(lines[endLine-1]), endLine, ctx), nil
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
