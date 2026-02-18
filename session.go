package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/jtolio/ajent/private"
	"github.com/modfin/bellman/models/gen"
	"github.com/modfin/bellman/prompt"
	"github.com/modfin/bellman/tools"
)

const (
	maxUserLineLength = 32768
)

// jsonUnescapeHTML reverses Go's default JSON HTML-safety escaping for
// display purposes. Go's encoding/json escapes &, <, and > as \u0026,
// \u003c, and \u003e respectively. These are the only three characters
// affected.
var jsonUnescapeHTML = strings.NewReplacer(
	`\u0026`, `&`,
	`\u003c`, `<`,
	`\u003e`, `>`,
)

type Config struct {
	MaxTokens    int
	SystemPrompt string
	Serializer   Serializer
	Tools        []tools.Tool
}

type Session struct {
	gen        *gen.Generator
	input      *private.UnbufferedLineReader
	output     io.Writer
	cfg        Config
	history    []prompt.Prompt
	serialized SerializedSession
}

func NewSession(client gen.Gen, model string,
	input io.Reader, output io.Writer, cfg Config) (*Session, error) {

	opts := []gen.Option{
		gen.WithModel(gen.Model{Provider: client.Provider(), Name: model}),
	}
	if cfg.MaxTokens != 0 {
		opts = append(opts, gen.WithMaxTokens(cfg.MaxTokens))
	}
	if len(cfg.Tools) > 0 {
		opts = append(opts, gen.WithTools(cfg.Tools...))
	}

	var history []prompt.Prompt
	var serialized SerializedSession
	if cfg.Serializer != nil {
		s, meta, loaded, err := cfg.Serializer.CreateOrOpen(
			SessionMeta{SystemPrompt: cfg.SystemPrompt})
		if err != nil {
			return nil, err
		}
		serialized = s
		cfg.SystemPrompt = meta.SystemPrompt
		history = loaded
	}

	if cfg.SystemPrompt != "" {
		opts = append(opts, gen.WithSystem(cfg.SystemPrompt))
	}

	return &Session{
		gen:        client.Generator(opts...),
		input:      private.NewUnbufferedLineReader(input, maxUserLineLength),
		output:     output,
		cfg:        cfg,
		history:    history,
		serialized: serialized,
	}, nil
}

func (s *Session) Close() error {
	if s.serialized != nil {
		return s.serialized.Close()
	}
	return nil
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
	return time.Now().Format("[2006-01-02 15:04:05 MST]\n") + result
}

func (s *Session) callTool(ctx context.Context, call tools.Call) (rv string) {
	start := time.Now()
	defer func() {
		rv = fmt.Sprintf("[start: %s, duration: %s]\n",
			time.Now().Format("2006-01-02 15:04:05 MST"),
			time.Since(start)) + rv
	}()

	if call.Ref == nil {
		return fmt.Sprintf("error: unknown tool %q", call.Name)
	}
	result, err := call.Ref.Function(ctx, call)
	if err != nil {
		return fmt.Sprintf("error: %v", err)
	}
	return result
}

func (s *Session) buildContextMessage() string {
	cwd, err := os.Getwd()
	if err != nil {
		cwd = "(unknown)"
	}
	return fmt.Sprintf("Working directory: %s\n", cwd)
}

func (s *Session) Run(ctx context.Context) error {
	addToHistory := func(p ...prompt.Prompt) error {
		s.history = append(s.history, p...)
		if s.serialized != nil {
			return s.serialized.Append(p...)
		}
		return nil
	}

	// Inject execution context at the start of every run
	if err := addToHistory(prompt.AsUser(s.buildContextMessage())); err != nil {
		return err
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
				_, err = fmt.Fprintf(s.output, "[%s %s]\n", call.Name, jsonUnescapeHTML.Replace(string(call.Argument)))
				if err != nil {
					return err
				}
				if err := addToHistory(
					prompt.AsToolCall(call.ID, call.Name, call.Argument),
					prompt.AsToolResponse(call.ID, call.Name, s.callTool(ctx, call)),
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
