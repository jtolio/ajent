package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/modfin/bellman/tools"
)

const grepMaxMatches = 100

type grepFileArgs struct {
	Path  string `json:"path" json-description:"The path to the file to search"`
	Query string `json:"query" json-description:"The substring to search for (case-sensitive)"`
}

var GrepFileTool = tools.NewTool("grep_file",
	tools.WithDescription("Search a file for lines containing a substring. Returns matching lines in hashline format annotated with page numbers. Output is capped at 100 matches."),
	tools.WithArgSchema(grepFileArgs{}),
	tools.WithFunction(func(ctx context.Context, call tools.Call) (string, error) {
		var params grepFileArgs
		if err := json.Unmarshal(call.Argument, &params); err != nil {
			return fmt.Sprintf("error: invalid arguments: %v", err), nil
		}
		if params.Path == "" {
			return "error: path is required", nil
		}
		if params.Query == "" {
			return "error: query is required", nil
		}

		lines, _, err := readFileLines(params.Path)
		if err != nil {
			return fmt.Sprintf("error: %v", err), nil
		}

		var sb strings.Builder
		matches := 0
		for i, line := range lines {
			if strings.Contains(line, params.Query) {
				lineNum := i + 1
				page := (i / linesPerPage) + 1
				hash := hashLineContent(line)
				fmt.Fprintf(&sb, "[page %d] %d:%s|%s\n", page, lineNum, hash, line)
				matches++
				if matches >= grepMaxMatches {
					fmt.Fprintf(&sb, "... (output truncated at %d matches)\n", grepMaxMatches)
					break
				}
			}
		}

		if matches == 0 {
			return fmt.Sprintf("no matches for %q in %s", params.Query, params.Path), nil
		}
		return fmt.Sprintf("%d matches:\n%s", matches, sb.String()), nil
	}),
)
