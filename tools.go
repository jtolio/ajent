package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/modfin/bellman/tools"
)

type fetchArgs struct {
	URL string `json:"url" json-description:"The URL to fetch"`
}

type searchArgs struct {
	Query string `json:"query" json-description:"The search query"`
}

const (
	fetchTimeout   = 10 * time.Second
	fetchMaxBody   = 128 * 1024
	fetchUserAgent = "ajent/1.0"
)

var webFetchTool = tools.NewTool("web_fetch",
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

func newWebSearchTool(apiKey string) tools.Tool {
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

			reqURL := "https://api.search.brave.com/res/v1/web/search?q=" + url.QueryEscape(params.Query)
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
