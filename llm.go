package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
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
		Stream:       true,
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

	return parseSSE(resp.Body)
}

const maxSSELineLength = 1 << 20

func parseSSE(body io.Reader) (Message, error) {
	var msg Message
	var blocks []Content
	partialJSON := map[int]*strings.Builder{}
	blockTypes := map[int]string{}

	reader := NewUnbufferedLineReader(body, maxSSELineLength)

	var eventType string
	for {
		raw, err := reader.ReadLine()
		if err != nil {
			if err == io.EOF {
				break
			}
			return Message{}, fmt.Errorf("reading SSE stream: %w", err)
		}
		line := strings.TrimRight(raw, "\n")
		if line == "" {
			eventType = ""
			continue
		}
		if strings.HasPrefix(line, "event: ") {
			eventType = strings.TrimPrefix(line, "event: ")
			continue
		}
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		data := strings.TrimPrefix(line, "data: ")

		switch eventType {
		case "message_start":
			var ev struct {
				Message Message `json:"message"`
			}
			if err := json.Unmarshal([]byte(data), &ev); err != nil {
				return Message{}, fmt.Errorf("parsing message_start: %w", err)
			}
			msg = ev.Message

		case "content_block_start":
			var ev struct {
				Index        int     `json:"index"`
				ContentBlock Content `json:"content_block"`
			}
			if err := json.Unmarshal([]byte(data), &ev); err != nil {
				return Message{}, fmt.Errorf("parsing content_block_start: %w", err)
			}
			for len(blocks) <= ev.Index {
				blocks = append(blocks, Content{})
			}
			blocks[ev.Index] = ev.ContentBlock
			blockTypes[ev.Index] = ev.ContentBlock.Type

		case "content_block_delta":
			var ev struct {
				Index int `json:"index"`
				Delta struct {
					Type        string `json:"type"`
					Text        string `json:"text,omitempty"`
					PartialJSON string `json:"partial_json,omitempty"`
				} `json:"delta"`
			}
			if err := json.Unmarshal([]byte(data), &ev); err != nil {
				return Message{}, fmt.Errorf("parsing content_block_delta: %w", err)
			}
			switch ev.Delta.Type {
			case "text_delta":
				if ev.Index < len(blocks) {
					blocks[ev.Index].Text += ev.Delta.Text
				}
			case "input_json_delta":
				b, ok := partialJSON[ev.Index]
				if !ok {
					b = &strings.Builder{}
					partialJSON[ev.Index] = b
				}
				b.WriteString(ev.Delta.PartialJSON)
			}

		case "content_block_stop":
			var ev struct {
				Index int `json:"index"`
			}
			if err := json.Unmarshal([]byte(data), &ev); err != nil {
				return Message{}, fmt.Errorf("parsing content_block_stop: %w", err)
			}
			if blockTypes[ev.Index] == "tool_use" {
				if b, ok := partialJSON[ev.Index]; ok && ev.Index < len(blocks) {
					var input map[string]string
					if json.Unmarshal([]byte(b.String()), &input) == nil {
						blocks[ev.Index].Input = input
					}
				}
			}

		case "message_delta":
			var ev struct {
				Delta struct {
					StopReason string `json:"stop_reason"`
				} `json:"delta"`
			}
			if err := json.Unmarshal([]byte(data), &ev); err != nil {
				return Message{}, fmt.Errorf("parsing message_delta: %w", err)
			}
			msg.StopReason = ev.Delta.StopReason

		case "message_stop", "ping":
			// nothing to do

		case "error":
			var ev struct {
				Error struct {
					Type    string `json:"type"`
					Message string `json:"message"`
				} `json:"error"`
			}
			if err := json.Unmarshal([]byte(data), &ev); err != nil {
				return Message{}, fmt.Errorf("parsing error event: %w", err)
			}
			return Message{}, fmt.Errorf("API error: %s: %s", ev.Error.Type, ev.Error.Message)
		}
	}

	msg.Content = blocks
	return msg, nil
}
