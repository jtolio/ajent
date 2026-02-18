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
	Path         string `json:"path" json-description:"The path to the file to search"`
	Query        string `json:"query" json-description:"The substring to search for (case-sensitive)"`
	ContextLines int    `json:"context_lines,omitempty" json-description:"Number of lines to show before and after each match (default 0)"`
}

var GrepFileTool = tools.NewTool("grep_file",
	tools.WithDescription("Search a file for lines containing a substring. Returns matching lines in hashline format. Use context_lines to show surrounding context. Output is capped at 100 matches."),
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

		// Find all matching line indices (0-indexed)
		var matchIndices []int
		for i, line := range lines {
			if strings.Contains(line, params.Query) {
				matchIndices = append(matchIndices, i)
				if len(matchIndices) >= grepMaxMatches {
					break
				}
			}
		}

		if len(matchIndices) == 0 {
			return fmt.Sprintf("no matches for %q in %s", params.Query, params.Path), nil
		}

		var sb strings.Builder

		if params.ContextLines <= 0 {
			// No context: simple output
			for _, idx := range matchIndices {
				lineNum := idx + 1
				hash := hashLineContent(lines[idx])
				fmt.Fprintf(&sb, "%d:%s|%s\n", lineNum, hash, lines[idx])
			}
		} else {
			// Build groups of contiguous ranges (merging overlapping context windows)
			type lineRange struct {
				start int // 0-indexed inclusive
				end   int // 0-indexed inclusive
			}
			var groups []lineRange
			for _, idx := range matchIndices {
				start := idx - params.ContextLines
				if start < 0 {
					start = 0
				}
				end := idx + params.ContextLines
				if end >= len(lines) {
					end = len(lines) - 1
				}
				if len(groups) > 0 && start <= groups[len(groups)-1].end+1 {
					// Merge with previous group
					groups[len(groups)-1].end = end
				} else {
					groups = append(groups, lineRange{start: start, end: end})
				}
			}

			// Build a set of match indices for quick lookup
			matchSet := make(map[int]bool, len(matchIndices))
			for _, idx := range matchIndices {
				matchSet[idx] = true
			}

			for gi, group := range groups {
				if gi > 0 {
					sb.WriteByte('\n')
				}
				for i := group.start; i <= group.end; i++ {
					lineNum := i + 1
					hash := hashLineContent(lines[i])
					if matchSet[i] {
						fmt.Fprintf(&sb, "> %d:%s|%s\n", lineNum, hash, lines[i])
					} else {
						fmt.Fprintf(&sb, "  %d:%s|%s\n", lineNum, hash, lines[i])
					}
				}
			}
		}

		matches := len(matchIndices)
		result := fmt.Sprintf("%d match", matches)
		if matches != 1 {
			result += "es"
		}
		result += ":\n" + sb.String()
		if matches >= grepMaxMatches {
			result += fmt.Sprintf("... (output truncated at %d matches)\n", grepMaxMatches)
		}
		return result, nil
	}),
)
