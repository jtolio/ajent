package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/modfin/bellman/tools"
)

type listDirArgs struct {
	Path   string `json:"path,omitempty" json-description:"The directory path to list. Defaults to the current working directory."`
	Offset int    `json:"offset,omitempty" json-description:"1-indexed entry number to start from (default 1)"`
	Limit  int    `json:"limit,omitempty" json-description:"Maximum number of entries to return (default 200, max 500)"`
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

var ListDirTool = tools.NewTool("list_directory",
	tools.WithDescription("List directory contents with details (permissions, size, modification time, name). Use offset and limit to paginate (default 200 entries, max 500)."),
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
		if params.Offset < 1 {
			params.Offset = 1
		}
		if params.Limit <= 0 {
			params.Limit = 200
		}
		if params.Limit > 500 {
			params.Limit = 500
		}

		entries, err := os.ReadDir(path)
		if err != nil {
			return fmt.Sprintf("error: %v", err), nil
		}

		totalEntries := len(entries)
		if totalEntries == 0 {
			return "[entries 0-0 of 0]\n(empty directory)", nil
		}

		if params.Offset > totalEntries {
			return fmt.Sprintf("error: offset %d is beyond the number of entries (%d)", params.Offset, totalEntries), nil
		}

		startIdx := params.Offset - 1
		endIdx := startIdx + params.Limit
		if endIdx > totalEntries {
			endIdx = totalEntries
		}

		var sb strings.Builder
		fmt.Fprintf(&sb, "[entries %d-%d of %d]\n", params.Offset, endIdx, totalEntries)
		for _, entry := range entries[startIdx:endIdx] {
			sb.WriteString(formatDirEntry(path, entry))
			sb.WriteByte('\n')
		}
		return sb.String(), nil
	}),
)
