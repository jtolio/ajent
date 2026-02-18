package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/modfin/bellman/tools"
)

type searchArgs struct {
	Query string `json:"query" json-description:"The search query"`
}

const defaultSearchEndpoint = "https://api.search.brave.com/res/v1/web/search"

func NewWebSearchTool(apiKey, endpoint string) tools.Tool {
	if endpoint == "" {
		endpoint = defaultSearchEndpoint
	}
	return tools.NewTool("web_search",
		tools.WithDescription("Search the web using Brave Search and return the top results."),
		tools.WithArgSchema(searchArgs{}),
		tools.WithFunction(func(ctx context.Context, call tools.Call) (string, error) {
			var params searchArgs
			if err := json.Unmarshal(call.Argument, &params); err != nil {
				return fmt.Sprintf("error: invalid arguments: %v", err), nil
			}
			if params.Query == "" {
				return "error: query is required", nil
			}

			reqURL := endpoint + "?q=" + url.QueryEscape(params.Query)
			req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
			if err != nil {
				return fmt.Sprintf("error: %v", err), nil
			}
			req.Header.Set("Accept", "application/json")
			req.Header.Set("X-Subscription-Token", apiKey)

			client := &http.Client{Timeout: fetchTimeout}
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

			var result struct {
				Web struct {
					Results []struct {
						Title       string `json:"title"`
						URL         string `json:"url"`
						Description string `json:"description"`
					} `json:"results"`
				} `json:"web"`
			}
			if err := json.Unmarshal(body, &result); err != nil {
				return fmt.Sprintf("error parsing response: %v", err), nil
			}

			var sb strings.Builder
			for i, r := range result.Web.Results {
				if i >= 5 {
					break
				}
				fmt.Fprintf(&sb, "%d. %s\n   %s\n   %s\n\n", i+1, r.Title, r.URL, r.Description)
			}
			if sb.Len() == 0 {
				return "No results found.", nil
			}
			return sb.String(), nil
		}),
	)
}
