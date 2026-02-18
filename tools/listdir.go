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

var ListDirTool = tools.NewTool("list_directory",
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
