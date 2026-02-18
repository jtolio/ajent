package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/modfin/bellman/tools"
)

type createFileArgs struct {
	Path    string `json:"path" json-description:"The path to the file to create"`
	Content string `json:"content" json-description:"The content to write to the file"`
}

var CreateFileTool = tools.NewTool("create_file",
	tools.WithDescription("Create a new file with the given content. Refuses to overwrite existing files. Returns hashline-formatted content of the created file. Use bash/mkdir to create parent directories if needed."),
	tools.WithArgSchema(createFileArgs{}),
	tools.WithFunction(func(ctx context.Context, call tools.Call) (string, error) {
		var params createFileArgs
		if err := json.Unmarshal(call.Argument, &params); err != nil {
			return fmt.Sprintf("error: invalid arguments: %v", err), nil
		}
		if params.Path == "" {
			return "error: path is required", nil
		}

		if _, err := os.Stat(params.Path); err == nil {
			return fmt.Sprintf("error: file already exists: %s", params.Path), nil
		}

		if err := os.WriteFile(params.Path, []byte(params.Content), 0644); err != nil {
			return fmt.Sprintf("error: %v", err), nil
		}

		lines, _, err := readFileLines(params.Path)
		if err != nil {
			return fmt.Sprintf("ok: created %s but error reading back: %v", params.Path, err), nil
		}
		if len(lines) == 0 {
			return fmt.Sprintf("ok: created %s (empty file)", params.Path), nil
		}

		return fmt.Sprintf("ok: created %s (%d lines)\n%s", params.Path, len(lines), formatHashlines(lines, 0)), nil
	}),
)
