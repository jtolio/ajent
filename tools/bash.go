package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"time"

	"github.com/modfin/bellman/tools"
)

const (
	bashTimeout   = 60 * time.Second
	bashMaxOutput = 64 * 1024
)

type bashArgs struct {
	Command string `json:"command" json-description:"The bash command to execute"`
}

var BashTool = tools.NewTool("bash",
	tools.WithDescription("Execute a bash command and return its combined stdout and stderr output."),
	tools.WithArgSchema(bashArgs{}),
	tools.WithFunction(func(ctx context.Context, call tools.Call) (string, error) {
		var params bashArgs
		if err := json.Unmarshal(call.Argument, &params); err != nil {
			return fmt.Sprintf("error: invalid arguments: %v", err), nil
		}
		if params.Command == "" {
			return "error: command is required", nil
		}

		ctx, cancel := context.WithTimeout(ctx, bashTimeout)
		defer cancel()

		cmd := exec.CommandContext(ctx, "bash", "-c", params.Command)
		output, err := cmd.CombinedOutput()
		if len(output) > bashMaxOutput {
			output = append([]byte("... (output truncated)\n"), output[len(output)-bashMaxOutput:]...)
		}
		if ctx.Err() == context.DeadlineExceeded {
			return fmt.Sprintf("error: command timed out after %v\n%s", bashTimeout, string(output)), nil
		}
		if err != nil {
			return fmt.Sprintf("exit status: %v\n%s", err, string(output)), nil
		}
		return string(output), nil
	}),
)
