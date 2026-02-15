package main

import (
	"context"
	"flag"
	"os"

	anyllm "github.com/mozilla-ai/any-llm-go"
	"github.com/mozilla-ai/any-llm-go/providers/anthropic"
)

func buildTools(braveAPIKey string) ([]anyllm.Tool, map[string]func(ctx context.Context, args string) string) {
	tools := []anyllm.Tool{webFetchTool}
	registry := map[string]func(ctx context.Context, args string) string{
		"web_fetch": webFetch,
	}
	if braveAPIKey != "" {
		tools = append(tools, webSearchTool)
		registry["web_search"] = newWebSearch(braveAPIKey)
	}
	return tools, registry
}

var (
	flagBaseURL      = flag.String("base-url", "https://api.anthropic.com", "the API base URL")
	flagAPIKey       = flag.String("api-key", "", "your api key")
	flagModel        = flag.String("model", "claude-haiku-4-5", "the model to use")
	flagSystemPrompt = flag.String("system-prompt", "", "path to a system prompt file")
	flagBraveAPIKey  = flag.String("brave-api-key", "", "Brave Search API key (enables web_search tool)")
)

func main() {
	flag.Parse()
	ctx := context.Background()

	provider, err := anthropic.New(
		anyllm.WithAPIKey(*flagAPIKey),
		anyllm.WithBaseURL(*flagBaseURL),
	)
	if err != nil {
		panic(err)
	}

	var systemPrompt string
	if *flagSystemPrompt != "" {
		data, err := os.ReadFile(*flagSystemPrompt)
		if err != nil {
			panic(err)
		}
		systemPrompt = string(data)
	}

	toolDefs, toolRegistry := buildTools(*flagBraveAPIKey)

	maxTokens := 1024
	err = NewSession(provider, anyllm.CompletionParams{
		Model:     *flagModel,
		MaxTokens: &maxTokens,
		Tools:     toolDefs,
	}, systemPrompt, toolRegistry, os.Stdin, os.Stdout).Run(ctx)
	if err != nil {
		panic(err)
	}
}
