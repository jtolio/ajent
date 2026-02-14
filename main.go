package main

import (
	"context"
	"flag"
	"os"
)

var (
	flagAPIURL = flag.String("api-url", "https://api.anthropic.com/v1/messages", "the API URL")
	flagAPIKey = flag.String("api-key", "", "your api key")
	flagModel  = flag.String("model", "claude-opus-4-6", "the model to use")
)

func main() {
	flag.Parse()
	ctx := context.Background()

	llm := NewLLM(Config{
		APIURL: *flagAPIURL,
		APIKey: *flagAPIKey,
		Model:  *flagModel,
	})
	err := NewSession(llm, os.Stdin, os.Stdout).Run(ctx)
	if err != nil {
		panic(err)
	}
}
