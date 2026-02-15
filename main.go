package main

import (
	"context"
	"flag"
	"os"

	anyllm "github.com/mozilla-ai/any-llm-go"
	"github.com/mozilla-ai/any-llm-go/providers/anthropic"
)

var (
	flagBaseURL      = flag.String("base-url", "https://api.anthropic.com", "the API base URL")
	flagAPIKey       = flag.String("api-key", "", "your api key")
	flagModel        = flag.String("model", "claude-haiku-4-5", "the model to use")
	flagSystemPrompt = flag.String("system-prompt", "", "path to a system prompt file")
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

	maxTokens := 1024
	err = NewSession(provider, anyllm.CompletionParams{
		Model:     *flagModel,
		MaxTokens: &maxTokens,
	}, systemPrompt, os.Stdin, os.Stdout).Run(ctx)
	if err != nil {
		panic(err)
	}
}
