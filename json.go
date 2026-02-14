package main

type Content struct {
	Type string `json:"type"`

	Text string `json:"text,omitempty"`

	Id    string            `json:"id,omitempty"`
	Name  string            `json:"name,omitempty"`
	Input map[string]string `json:"input,omitempty"`

	ToolUseId string `json:"tool_use_id,omitempty"`
	Content   any    `json:"content,omitempty"`
	IsError   bool   `json:"is_error,omitempty"`
}

type Message struct {
	Role       string    `json:"role"`
	StopReason string    `json:"stop_reason,omitempty"`
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

type ToolDefinition struct {
	Type string `json:"type"`
	Name string `json:"name"`
}

type request struct {
	Model        string           `json:"model"`
	MaxTokens    int              `json:"max_tokens"`
	Stream       bool             `json:"stream"`
	SystemPrompt string           `json:"system"`
	Tools        []ToolDefinition `json:"tools,omitempty"`
	Messages     []Message        `json:"messages"`
	Temperature  *float32         `json:"temperature,omitempty"`
}
