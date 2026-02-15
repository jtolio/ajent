package main

import (
	"context"
	"flag"
	"os"

	"github.com/modfin/bellman/models/gen"
	"github.com/modfin/bellman/services/anthropic"
	"github.com/modfin/bellman/tools"
)

var (
	flagAPIKey      = flag.String("api-key", "", "your api key")
	flagModel       = flag.String("model", "claude-haiku-4-5", "the model to use")
	flagBraveAPIKey = flag.String("brave-api-key", "", "Brave Search API key (enables web_search tool)")
)

func buildTools(braveAPIKey string) []tools.Tool {
	t := []tools.Tool{webFetchTool}
	if braveAPIKey != "" {
		t = append(t, newWebSearchTool(braveAPIKey))
	}
	return t
}

func main() {
	flag.Parse()
	ctx := context.Background()

	client := anthropic.New(*flagAPIKey)
	g := client.Generator(
		gen.WithModel(gen.Model{Provider: "Anthropic", Name: *flagModel}),
		gen.WithMaxTokens(1024),
		gen.WithTools(buildTools(*flagBraveAPIKey)...),
	)

	err := NewSession(g, os.Stdin, os.Stdout).Run(ctx)
	if err != nil {
		panic(err)
	}
}
