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

type Content struct {
	Type string `json:"type"`

	Text string `json:"text"`

	Id    string            `json:"id"`
	Name  string            `json:"name"`
	Input map[string]string `json:"input"`

	ToolUseId string `json:"tool_use_id"`
	Content   string `json:"content"`
	IsError   bool   `json:"is_error"`
}

type Message struct {
	Role       string    `json:"role"`
	StopReason string    `json:"stop_reason"`
	Content    []Content `json:"content"`
}

func (m *Message) Filter(typ string) (rv []Content) {
	for _, c := range m.Content {
		if c.Type == typ {
			rv = append(rv, c)
		}
	}
	return rv
}

type Session struct {
	input  *UnbufferedLineReader
	output io.Writer
}

func NewSession(input io.Reader, output io.Writer) *Session {
	return &Session{
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

func (s *Session) getLLM(ctx context.Context, msgs []Message) (Message, error) {
	return Message{
		Content: []Content{
			{
				Type: "text",
				Text: "unimplemented",
			},
		},
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
		resp, err := s.getLLM(ctx, messages)
		if err != nil {
			return err
		}
		messages = append(messages, resp)

		next := Message{
			Role: "user",
		}

		for _, c := range resp.Filter("text") {
			err := s.messageOutput(ctx, c)
			if err != nil {
				return err
			}
		}

		for _, toolCall := range resp.Filter("tool_use") {
			result, err := s.callTool(ctx, toolCall)
			if err != nil {
				return err
			}
			next.Content = append(next.Content, result)
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
