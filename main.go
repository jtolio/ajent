package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/modfin/bellman/models/gen"
	"github.com/modfin/bellman/services/anthropic"
	"github.com/modfin/bellman/services/ollama"
	"github.com/modfin/bellman/services/openai"
	"github.com/modfin/bellman/services/vertexai"
	"github.com/modfin/bellman/services/vllm"
	"github.com/modfin/bellman/tools"

	atools "github.com/jtolio/ajent/tools"
)

var (
	flagProvider = flag.String("provider", "anthropic", "the provider/protocol to use. must be either anthropic, ollama, openai, vertexai, or vllm")

	flagModel = flag.String("model", "claude-haiku-4-5", "the model to use")

	flagURL    = flag.String("url", "", "connection url (depends on provider)")
	flagAPIKey = flag.String("api-key", "", "your api key (depends on provider)")

	flagProject    = flag.String("project", "", "project (vertexai specific)")
	flagRegion     = flag.String("region", "", "region (vertexai specific)")
	flagCredential = flag.String("credential", "", "credential (vertexai specific)")

	flagSystemPrompt = flag.String("system-prompt", "", "path to a system prompt file")
	flagMaxTokens    = flag.Int("max-tokens", 0, "max tokens")
	flagBraveAPIKey  = flag.String("brave-api-key", "", "Brave Search API key (enables web_search tool)")
	flagSearchURL    = flag.String("search-url", "", "Custom search endpoint URL (defaults to Brave Search API)")

	flagNoTimestamps = flag.Bool("no-timestamps", false, "if true, disable timestamps")
)

func usage() {
	fmt.Fprintf(os.Stderr, "Usage: %s [flags] <session.hjl>\n", os.Args[0])
	flag.PrintDefaults()
	os.Exit(1)
}

func buildTools(braveAPIKey, searchURL string) []tools.Tool {
	t := []tools.Tool{
		atools.WebFetchTool,
		atools.ReadFileTool,
		atools.ListDirTool,
		atools.EditFileTool,
		atools.BashTool,
		atools.CreateFileTool,
		atools.GrepFileTool,
		atools.TreeTool,
	}
	if braveAPIKey != "" {
		t = append(t, atools.NewWebSearchTool(braveAPIKey, searchURL))
	}
	return t
}

func main() {
	flag.Parse()
	ctx := context.Background()

	if flag.Arg(0) == "" {
		usage()
	}

	sessionPath := flag.Arg(0)

	var client gen.Gen
	switch strings.ToLower(*flagProvider) {
	case "anthropic":
		client = anthropic.New(*flagAPIKey)
	case "ollama":
		client = ollama.New(*flagURL)
	case "openai":
		client = openai.New(*flagAPIKey)
	case "vertexai":
		var err error
		client, err = vertexai.New(vertexai.GoogleConfig{
			Project:    *flagProject,
			Region:     *flagRegion,
			Credential: *flagCredential,
		})
		if err != nil {
			panic(err)
		}
	case "vllm":
		client = vllm.New([]string{*flagURL}, []string{*flagModel})
	default:
		usage()
	}

	cfg := Config{
		MaxTokens:    *flagMaxTokens,
		Tools:        buildTools(*flagBraveAPIKey, *flagSearchURL),
		Serializer:   NewFileSerializer(sessionPath),
		NoTimestamps: *flagNoTimestamps,
	}

	if *flagSystemPrompt != "" {
		data, err := os.ReadFile(*flagSystemPrompt)
		if err != nil {
			panic(err)
		}
		cfg.SystemPrompt = string(data)
	}

	session, err := NewSession(client, *flagModel, os.Stdin, os.Stdout, cfg)
	if err != nil {
		panic(err)
	}
	defer session.Close()
	if err := session.Run(ctx); err != nil {
		panic(err)
	}
}
