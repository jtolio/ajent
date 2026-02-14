package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
)

const (
	maxUserLineLength = 32768
)

type Session struct {
	llm    *LLM
	input  *UnbufferedLineReader
	output io.Writer
}

func NewSession(llm *LLM, input io.Reader, output io.Writer) *Session {
	return &Session{
		llm:    llm,
		input:  NewUnbufferedLineReader(input, maxUserLineLength),
		output: output,
	}
}

func (s *Session) getUserInput(ctx context.Context) (Content, error) {
	_, err := fmt.Fprintf(s.output, "> ")
	if err != nil {
		return Content{}, err
	}
	input, err := s.input.ReadLine()
	if err != nil {
		if errors.Is(err, io.EOF) {
			_, _ = fmt.Fprint(s.output, "\r")
		}
		return Content{}, err
	}
	if !strings.HasSuffix(input, "\n") {
		_, _ = fmt.Fprint(s.output, "\n")
	}
	return Content{
		Type: "text",
		Text: input,
	}, nil
}

func (s *Session) callTool(ctx context.Context, call Content) (Content, error) {
	return Content{
		Type:      "tool_result",
		ToolUseId: call.Id,
		IsError:   true,
		Content:   "tool use unimplemented",
	}, nil
}

func (s *Session) messageOutput(ctx context.Context, msg Content) error {
	_, err := fmt.Fprintln(s.output, msg.Text)
	return err
}

func (s *Session) Run(ctx context.Context) error {
	var messages []Message
	input, err := s.getUserInput(ctx)
	if err != nil {
		return err
	}
	messages = append(messages, Message{
		Role:    "user",
		Content: []Content{input}})

	for {
		resp, err := s.llm.Exchange(ctx, messages)
		if err != nil {
			return err
		}
		messages = append(messages, resp)
		messages[len(messages)-1].StopReason = ""

		next := Message{
			Role: "user",
		}

		for _, c := range resp.Content {
			switch c.Type {
			case "text":
				err := s.messageOutput(ctx, c)
				if err != nil {
					return err
				}
			case "tool_use":
				result, err := s.callTool(ctx, c)
				if err != nil {
					return err
				}
				next.Content = append(next.Content, result)
			default:
				return fmt.Errorf("unexpected type %q", c.Type)
			}
		}

		if resp.StopReason == "end_turn" || len(next.Content) == 0 {
			input, err := s.getUserInput(ctx)
			if err != nil {
				if errors.Is(err, io.EOF) {
					return nil
				}
				return err
			}
			next.Content = append(next.Content, input)
		}

		messages = append(messages, next)
	}
}
