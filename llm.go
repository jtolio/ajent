package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

const (
	defaultMaxTokens = 1024
)

type Config struct {
	APIURL       string
	APIKey       string
	Model        string
	SystemPrompt string
	MaxTokens    *int
	Temperature  *float32
}

type LLM struct {
	HTTP *http.Client
	Cfg  Config
}

func NewLLM(cfg Config) *LLM {
	if cfg.MaxTokens == nil {
		maxTokens := defaultMaxTokens
		cfg.MaxTokens = &maxTokens
	}
	return &LLM{
		HTTP: http.DefaultClient,
		Cfg:  cfg,
	}
}

func (l *LLM) Exchange(ctx context.Context, msgs []Message) (m Message, err error) {
	data := request{
		Model:        l.Cfg.Model,
		MaxTokens:    *l.Cfg.MaxTokens,
		Stream:       false,
		SystemPrompt: l.Cfg.SystemPrompt,
		Messages:     msgs,
		Temperature:  l.Cfg.Temperature,
	}

	var buf bytes.Buffer
	err = json.NewEncoder(&buf).Encode(&data)
	if err != nil {
		return Message{}, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, l.Cfg.APIURL, &buf)
	if err != nil {
		return Message{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("anthropic-version", "2023-06-01")
	req.Header.Set("X-Api-Key", l.Cfg.APIKey)

	resp, err := l.HTTP.Do(req)
	if err != nil {
		return Message{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		if message, err := io.ReadAll(resp.Body); err == nil {
			return Message{}, fmt.Errorf("unknown response %q\n%s", resp.Status, string(message))
		}
		return Message{}, fmt.Errorf("unknown response status %q", resp.Status)
	}

	err = json.NewDecoder(resp.Body).Decode(&m)
	if err != nil {
		return Message{}, err
	}

	return m, nil
}
