package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/modfin/bellman/models/gen"
	"github.com/modfin/bellman/prompt"
	"github.com/modfin/bellman/tools"
)

const (
	maxUserLineLength = 32768
)

type Config struct {
	MaxTokens    int
	SystemPrompt string
	Serializer   Serializer
	Tools        []tools.Tool
	NoTimestamps bool
}

type Session struct {
	gen     *gen.Generator
	input   *UnbufferedLineReader
	output  io.Writer
	cfg     Config
	history []prompt.Prompt
	meta    SessionMeta
}

func NewSession(client gen.Gen, model string,
	input io.Reader, output io.Writer, cfg Config) (*Session, error) {

	opts := []gen.Option{
		gen.WithModel(gen.Model{Provider: client.Provider(), Name: model}),
	}
	if cfg.MaxTokens != 0 {
		opts = append(opts, gen.WithMaxTokens(1024))
	}
	if len(cfg.Tools) > 0 {
		opts = append(opts, gen.WithTools(cfg.Tools...))
	}

	var initialHistory []prompt.Prompt
	if cfg.Serializer != nil {
		meta, history, found, err := cfg.Serializer.Load()
		if err != nil {
			return nil, err
		}
		if found {
			cfg.SystemPrompt = meta.SystemPrompt
			initialHistory = history
		}
	}

	if cfg.SystemPrompt != "" {
		opts = append(opts, gen.WithSystem(cfg.SystemPrompt))
	}

	return &Session{
		gen:     client.Generator(opts...),
		input:   NewUnbufferedLineReader(input, maxUserLineLength),
		output:  output,
		cfg:     cfg,
		history: initialHistory,
		meta:    SessionMeta{SystemPrompt: cfg.SystemPrompt},
	}, nil
}

func (s *Session) getUserInput(ctx context.Context) (string, error) {
	_, err := fmt.Fprintf(s.output, "> ")
	if err != nil {
		return "", err
	}
	input, err := s.input.ReadLine()
	if err != nil {
		if errors.Is(err, io.EOF) {
			_, _ = fmt.Fprint(s.output, "\r")
		}
		return "", err
	}
	if !strings.HasSuffix(input, "\n") {
		_, _ = fmt.Fprint(s.output, "\n\n")
	} else {
		_, _ = fmt.Fprint(s.output, "\n")
	}

	return input, nil
}

func (s *Session) addTimestamp(result string) string {
	if s.cfg.NoTimestamps {
		return result
	}
	return time.Now().Format("[2006-01-02 15:04:05 MST]\n") + result
}

func (s *Session) callTool(ctx context.Context, call tools.Call) string {
	if call.Ref == nil {
		return fmt.Sprintf("error: unknown tool %q", call.Name)
	}
	result, err := call.Ref.Function(ctx, call)
	if err != nil {
		return fmt.Sprintf("error: %v", err)
	}
	return result
}

func (s *Session) Run(ctx context.Context) error {
	addToHistory := func(p ...prompt.Prompt) error {
		s.history = append(s.history, p...)
		if s.cfg.Serializer != nil {
			return s.cfg.Serializer.Serialize(s.meta, s.history)
		}
		return nil
	}

	for firstLoop := true; true; firstLoop = false {
		if !firstLoop {
			resp, err := s.gen.WithContext(ctx).Prompt(s.history...)
			if err != nil {
				return err
			}

			for _, text := range resp.Texts {
				if _, err := fmt.Fprintf(s.output, "%s\n\n", text); err != nil {
					return err
				}
				if err := addToHistory(prompt.AsAssistant(text)); err != nil {
					return err
				}
			}

			for _, call := range resp.Tools {
				if err := addToHistory(
					prompt.AsToolCall(call.ID, call.Name, call.Argument),
					prompt.AsToolResponse(call.ID, call.Name, s.addTimestamp(s.callTool(ctx, call))),
				); err != nil {
					return err
				}
			}

			if len(resp.Tools) > 0 {
				continue
			}
		}

		input, err := s.getUserInput(ctx)
		if err != nil {
			if errors.Is(err, io.EOF) {
				return nil
			}
			return err
		}
		if err := addToHistory(prompt.AsUser(s.addTimestamp(input))); err != nil {
			return err
		}
	}
	return fmt.Errorf("unreachable")
}
