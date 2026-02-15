package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/modfin/bellman/models/gen"
	"github.com/modfin/bellman/prompt"
	"github.com/modfin/bellman/tools"
)

const (
	maxUserLineLength = 32768
)

type Session struct {
	gen    *gen.Generator
	input  *UnbufferedLineReader
	output io.Writer
}

func NewSession(g *gen.Generator, input io.Reader, output io.Writer) *Session {
	return &Session{
		gen:    g,
		input:  NewUnbufferedLineReader(input, maxUserLineLength),
		output: output,
	}
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
	var prompts []prompt.Prompt

	input, err := s.getUserInput(ctx)
	if err != nil {
		return err
	}
	prompts = append(prompts, prompt.AsUser(input))

	for {
		resp, err := s.gen.WithContext(ctx).Prompt(prompts...)
		if err != nil {
			return err
		}

		for _, text := range resp.Texts {
			if _, err := fmt.Fprintf(s.output, "%s\n\n", text); err != nil {
				return err
			}
			prompts = append(prompts, prompt.AsAssistant(text))
		}

		for _, call := range resp.Tools {
			prompts = append(prompts,
				prompt.AsToolCall(call.ID, call.Name, call.Argument),
				prompt.AsToolResponse(call.ID, call.Name, s.callTool(ctx, call)),
			)
		}

		if len(resp.Tools) == 0 {
			input, err := s.getUserInput(ctx)
			if err != nil {
				if errors.Is(err, io.EOF) {
					return nil
				}
				return err
			}
			prompts = append(prompts, prompt.AsUser(input))
		}
	}
}
