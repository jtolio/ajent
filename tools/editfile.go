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
