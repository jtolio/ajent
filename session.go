package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"

	anyllm "github.com/mozilla-ai/any-llm-go"
)

const maxUserLineLength = 32768

type Session struct {
	provider     anyllm.Provider
	params       anyllm.CompletionParams
	systemPrompt string
	input        *UnbufferedLineReader
	output       io.Writer
}

func NewSession(provider anyllm.Provider, params anyllm.CompletionParams, systemPrompt string, input io.Reader, output io.Writer) *Session {
	return &Session{
		provider:     provider,
		params:       params,
		systemPrompt: systemPrompt,
		input:        NewUnbufferedLineReader(input, maxUserLineLength),
		output:       output,
	}
}

func (s *Session) exchange(ctx context.Context, msgs []anyllm.Message) (*anyllm.ChatCompletion, error) {
	params := s.params
	params.Messages = msgs
	return s.provider.Completion(ctx, params)
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

func (s *Session) callTool(ctx context.Context, tc anyllm.ToolCall) (anyllm.Message, error) {
	return anyllm.Message{
		Role:       anyllm.RoleTool,
		ToolCallID: tc.ID,
		Content:    "tool use unimplemented",
	}, nil
}

func (s *Session) messageOutput(ctx context.Context, text string) error {
	_, err := fmt.Fprintf(s.output, "%s\n\n", text)
	return err
}

func (s *Session) Run(ctx context.Context) error {
	var messages []anyllm.Message

	if s.systemPrompt != "" {
		messages = append(messages, anyllm.Message{
			Role:    anyllm.RoleSystem,
			Content: s.systemPrompt,
		})
	}

	input, err := s.getUserInput(ctx)
	if err != nil {
		if errors.Is(err, io.EOF) {
			return nil
		}
		return err
	}
	messages = append(messages, anyllm.Message{
		Role:    anyllm.RoleUser,
		Content: input,
	})

	for {
		resp, err := s.exchange(ctx, messages)
		if err != nil {
			return err
		}
		if len(resp.Choices) == 0 {
			return fmt.Errorf("no choices in response")
		}

		choice := resp.Choices[0]
		messages = append(messages, choice.Message)

		text := choice.Message.ContentString()
		if text != "" {
			if err := s.messageOutput(ctx, text); err != nil {
				return err
			}
		}

		if choice.FinishReason == anyllm.FinishReasonToolCalls {
			for _, tc := range choice.Message.ToolCalls {
				result, err := s.callTool(ctx, tc)
				if err != nil {
					return err
				}
				messages = append(messages, result)
			}
			continue
		}

		input, err := s.getUserInput(ctx)
		if err != nil {
			if errors.Is(err, io.EOF) {
				return nil
			}
			return err
		}
		messages = append(messages, anyllm.Message{
			Role:    anyllm.RoleUser,
			Content: input,
		})
	}
}
