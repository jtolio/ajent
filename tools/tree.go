package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/modfin/bellman/tools"
)

const treeLinesPerPage = 100

type treeArgs struct {
	Path       string `json:"path,omitempty" json-description:"The directory path. Defaults to current directory."`
	Depth      int    `json:"depth,omitempty" json-description:"Maximum depth to traverse. Defaults to 3."`
	ShowHidden bool   `json:"show_hidden,omitempty" json-description:"If true, include hidden files/directories (names starting with dot). Defaults to false."`
	Page       int    `json:"page,omitempty" json-description:"Page number (1-indexed). Defaults to 1."`
}

type treeEntry struct {
	name  string
	isDir bool
}

var TreeTool = tools.NewTool("tree",
	tools.WithDescription("Display a directory tree structure. Paginated at 100 lines per page. Directories are listed before files at each level."),
	tools.WithArgSchema(treeArgs{}),
	tools.WithFunction(func(ctx context.Context, call tools.Call) (string, error) {
		var params treeArgs
		if err := json.Unmarshal(call.Argument, &params); err != nil {
			return fmt.Sprintf("error: invalid arguments: %v", err), nil
		}
		if params.Path == "" {
			params.Path = "."
		}
		if params.Depth <= 0 {
			params.Depth = 3
		}
		if params.Page < 1 {
			params.Page = 1
		}

		info, err := os.Stat(params.Path)
		if err != nil {
			return fmt.Sprintf("error: %v", err), nil
		}
		if !info.IsDir() {
			return fmt.Sprintf("error: %s is not a directory", params.Path), nil
		}

		var lines []string
		lines = append(lines, ".")
		buildTree(&lines, params.Path, "", params.Depth, params.ShowHidden)

		totalLines := len(lines)
		totalPages := (totalLines + treeLinesPerPage - 1) / treeLinesPerPage

		if params.Page > totalPages {
			return fmt.Sprintf("error: page %d does not exist (tree has %d pages)", params.Page, totalPages), nil
		}

		startIdx := (params.Page - 1) * treeLinesPerPage
		endIdx := startIdx + treeLinesPerPage
		if endIdx > totalLines {
			endIdx = totalLines
		}

		var sb strings.Builder
		fmt.Fprintf(&sb, "[page %d/%d, lines %d-%d of %d]\n", params.Page, totalPages, startIdx+1, endIdx, totalLines)
		for _, line := range lines[startIdx:endIdx] {
			sb.WriteString(line)
			sb.WriteByte('\n')
		}
		return sb.String(), nil
	}),
)

func buildTree(lines *[]string, dir, prefix string, depth int, showHidden bool) {
	if depth <= 0 {
		return
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}

	// Filter and sort: directories first, then files, alphabetical within each group
	var filtered []treeEntry
	for _, e := range entries {
		name := e.Name()
		if !showHidden && strings.HasPrefix(name, ".") {
			continue
		}
		filtered = append(filtered, treeEntry{name: name, isDir: e.IsDir()})
	}
	sort.Slice(filtered, func(i, j int) bool {
		if filtered[i].isDir != filtered[j].isDir {
			return filtered[i].isDir
		}
		return filtered[i].name < filtered[j].name
	})

	for i, entry := range filtered {
		isLast := i == len(filtered)-1
		connector := "├── "
		childPrefix := "│   "
		if isLast {
			connector = "└── "
			childPrefix = "    "
		}

		*lines = append(*lines, prefix+connector+entry.name)

		if entry.isDir {
			buildTree(lines, filepath.Join(dir, entry.name), prefix+childPrefix, depth-1, showHidden)
		}
	}
}
