package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/modfin/bellman/tools"
)

type fetchArgs struct {
	URL string `json:"url" json-description:"The URL to fetch"`
}

const (
	fetchTimeout   = 10 * time.Second
	fetchMaxBody   = 128 * 1024
	fetchUserAgent = "ajent/1.0"
)

var WebFetchTool = tools.NewTool("web_fetch",
	tools.WithDescription("Fetch the contents of a URL and return the response body as text."),
	tools.WithArgSchema(fetchArgs{}),
	tools.WithFunction(func(ctx context.Context, call tools.Call) (string, error) {
		var params fetchArgs
		if err := json.Unmarshal(call.Argument, &params); err != nil {
			return fmt.Sprintf("error: invalid arguments: %v", err), nil
		}
		if params.URL == "" {
			return "error: url is required", nil
		}

		client := &http.Client{Timeout: fetchTimeout}
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, params.URL, nil)
		if err != nil {
			return fmt.Sprintf("error: %v", err), nil
		}
		req.Header.Set("User-Agent", fetchUserAgent)

		resp, err := client.Do(req)
		if err != nil {
			return fmt.Sprintf("error: %v", err), nil
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(io.LimitReader(resp.Body, fetchMaxBody))
		if err != nil {
			return fmt.Sprintf("error reading response: %v", err), nil
		}

		if resp.StatusCode >= 400 {
			return fmt.Sprintf("HTTP %d: %s", resp.StatusCode, string(body)), nil
		}
		return string(body), nil
	}),
)
