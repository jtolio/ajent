package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/modfin/bellman/tools"
)

type findReplaceArgs struct {
	Path    string `json:"path" json-description:"The path to the file"`
	OldText string `json:"old_text" json-description:"Exact text to find (can be multi-line)"`
	NewText string `json:"new_text" json-description:"Replacement text (can be multi-line, empty string to delete)"`
}

var FindReplaceTool = tools.NewTool("find_replace",
	tools.WithDescription("Content-based find and replace in a file. The old_text must appear exactly once in the file. No hashline references needed."),
	tools.WithArgSchema(findReplaceArgs{}),
	tools.WithFunction(func(ctx context.Context, call tools.Call) (string, error) {
		var params findReplaceArgs
		if err := json.Unmarshal(call.Argument, &params); err != nil {
			return fmt.Sprintf("error: invalid arguments: %v", err), nil
		}
		if params.Path == "" {
			return "error: path is required", nil
		}
		if params.OldText == "" {
			return "error: old_text is required", nil
		}

		data, err := os.ReadFile(params.Path)
		if err != nil {
			return fmt.Sprintf("error: %v", err), nil
		}
		content := string(data)

		count := strings.Count(content, params.OldText)
		if count == 0 {
			return fmt.Sprintf("error: old_text not found in %s", params.Path), nil
		}
		if count > 1 {
			return fmt.Sprintf("error: found %d occurrences of old_text in %s, provide more surrounding context to uniquely identify the target", count, params.Path), nil
		}

		newContent := strings.Replace(content, params.OldText, params.NewText, 1)

		info, err := os.Stat(params.Path)
		if err != nil {
			return fmt.Sprintf("error: %v", err), nil
		}
		if err := os.WriteFile(params.Path, []byte(newContent), info.Mode().Perm()); err != nil {
			return fmt.Sprintf("error writing file: %v", err), nil
		}

		newLines := strings.Count(newContent, "\n")
		return fmt.Sprintf("ok: replaced text in %s (%d lines)", params.Path, newLines+1), nil
	}),
)
