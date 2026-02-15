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

	anyllm "github.com/mozilla-ai/any-llm-go"
)

var webFetchTool = anyllm.Tool{
	Type: "function",
	Function: anyllm.Function{
		Name:        "web_fetch",
		Description: "Fetch the contents of a URL and return the response body as text.",
		Parameters: map[string]any{
			"type":     "object",
			"required": []string{"url"},
			"properties": map[string]any{
				"url": map[string]any{
					"type":        "string",
					"description": "The URL to fetch",
				},
			},
		},
	},
}

var webSearchTool = anyllm.Tool{
	Type: "function",
	Function: anyllm.Function{
		Name:        "web_search",
		Description: "Search the web using Brave Search and return the top results.",
		Parameters: map[string]any{
			"type":     "object",
			"required": []string{"query"},
			"properties": map[string]any{
				"query": map[string]any{
					"type":        "string",
					"description": "The search query",
				},
			},
		},
	},
}

const (
	fetchTimeout   = 10 * time.Second
	fetchMaxBody   = 128 * 1024
	fetchUserAgent = "ajent/1.0"
)

func webFetch(ctx context.Context, args string) string {
	var params struct {
		URL string `json:"url"`
	}
	if err := json.Unmarshal([]byte(args), &params); err != nil {
		return fmt.Sprintf("error: invalid arguments: %v", err)
	}
	if params.URL == "" {
		return "error: url is required"
	}

	client := &http.Client{Timeout: fetchTimeout}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, params.URL, nil)
	if err != nil {
		return fmt.Sprintf("error: %v", err)
	}
	req.Header.Set("User-Agent", fetchUserAgent)

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Sprintf("error: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, fetchMaxBody))
	if err != nil {
		return fmt.Sprintf("error reading response: %v", err)
	}

	if resp.StatusCode >= 400 {
		return fmt.Sprintf("HTTP %d: %s", resp.StatusCode, string(body))
	}
	return string(body)
}

func newWebSearch(apiKey string) func(ctx context.Context, args string) string {
	return func(ctx context.Context, args string) string {
		var params struct {
			Query string `json:"query"`
		}
		if err := json.Unmarshal([]byte(args), &params); err != nil {
			return fmt.Sprintf("error: invalid arguments: %v", err)
		}
		if params.Query == "" {
			return "error: query is required"
		}

		reqURL := "https://api.search.brave.com/res/v1/web/search?q=" + url.QueryEscape(params.Query)
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
		if err != nil {
			return fmt.Sprintf("error: %v", err)
		}
		req.Header.Set("Accept", "application/json")
		req.Header.Set("X-Subscription-Token", apiKey)

		client := &http.Client{Timeout: fetchTimeout}
		resp, err := client.Do(req)
		if err != nil {
			return fmt.Sprintf("error: %v", err)
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(io.LimitReader(resp.Body, fetchMaxBody))
		if err != nil {
			return fmt.Sprintf("error reading response: %v", err)
		}

		if resp.StatusCode >= 400 {
			return fmt.Sprintf("HTTP %d: %s", resp.StatusCode, string(body))
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
			return fmt.Sprintf("error parsing response: %v", err)
		}

		var sb strings.Builder
		for i, r := range result.Web.Results {
			if i >= 5 {
				break
			}
			fmt.Fprintf(&sb, "%d. %s\n   %s\n   %s\n\n", i+1, r.Title, r.URL, r.Description)
		}
		if sb.Len() == 0 {
			return "No results found."
		}
		return sb.String()
	}
}
