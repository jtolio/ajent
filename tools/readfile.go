package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/modfin/bellman/tools"
)

const maxReadLines = 200

type readFileArgs struct {
	Path      string `json:"path" json-description:"The path to the file to read"`
	StartLine int    `json:"start_line,omitempty" json-description:"First line to return (1-indexed, default 1)"`
	EndLine   int    `json:"end_line,omitempty" json-description:"Last line to return (1-indexed, inclusive, default start_line+199). Max 200 lines per call."`
}

var ReadFileTool = tools.NewTool("read_file",
	tools.WithDescription("Read a file with hashline-prefixed lines. Returns up to 200 lines per call. Each line is prefixed with its line number and a content hash in the format 'line:hash|content'. Use the line:hash references with the edit_file tool. Use start_line and end_line to read specific line ranges."),
	tools.WithArgSchema(readFileArgs{}),
	tools.WithFunction(func(ctx context.Context, call tools.Call) (string, error) {
		var params readFileArgs
		if err := json.Unmarshal(call.Argument, &params); err != nil {
			return fmt.Sprintf("error: invalid arguments: %v", err), nil
		}
		if params.Path == "" {
			return "error: path is required", nil
		}

		lines, _, err := readFileLines(params.Path)
		if err != nil {
			return fmt.Sprintf("error: %v", err), nil
		}

		totalLines := len(lines)
		if totalLines == 0 {
			return "[lines 0-0 of 0]\n(empty file)", nil
		}

		// Defaults
		if params.StartLine < 1 {
			params.StartLine = 1
		}
		if params.EndLine < params.StartLine {
			params.EndLine = params.StartLine + maxReadLines - 1
		}

		// Cap range to maxReadLines
		if params.EndLine-params.StartLine+1 > maxReadLines {
			params.EndLine = params.StartLine + maxReadLines - 1
		}

		// Clamp to file bounds
		if params.StartLine > totalLines {
			return fmt.Sprintf("error: start_line %d is beyond end of file (%d lines)", params.StartLine, totalLines), nil
		}
		if params.EndLine > totalLines {
			params.EndLine = totalLines
		}

		startIdx := params.StartLine - 1
		endIdx := params.EndLine

		pageLines := lines[startIdx:endIdx]
		content := formatHashlines(pageLines, startIdx)

		header := fmt.Sprintf("[lines %d-%d of %d]\n", params.StartLine, params.EndLine, totalLines)
		return header + content, nil
	}),
)
