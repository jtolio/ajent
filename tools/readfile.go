package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/modfin/bellman/tools"
)

type readFileArgs struct {
	Path string `json:"path" json-description:"The path to the file to read"`
	Page int    `json:"page,omitempty" json-description:"Page number (1-indexed). If omitted or 0, returns page 1."`
}

var ReadFileTool = tools.NewTool("read_file",
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
