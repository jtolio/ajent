package main

import (
	"errors"
	"io"
	"os"

	"github.com/jtolio/ajent/hjl"
	"github.com/modfin/bellman/prompt"
)

type SessionMeta struct {
	SystemPrompt string `json:"system_prompt"`
}

type Serializer interface {
	CreateOrOpen(meta SessionMeta) (SerializedSession, SessionMeta, []prompt.Prompt, error)
}

type SerializedSession interface {
	Append(prompts ...prompt.Prompt) error
	Close() error
}

type FileSerializer struct {
	path string
}

func NewFileSerializer(path string) *FileSerializer {
	return &FileSerializer{path: path}
}

func (s *FileSerializer) CreateOrOpen(meta SessionMeta) (SerializedSession, SessionMeta, []prompt.Prompt, error) {
	fh, err := os.Open(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return s.create(meta)
		}
		return nil, SessionMeta{}, nil, err
	}
	defer fh.Close()

	d := hjl.NewDecoder(fh)

	var fileMeta SessionMeta
	if err := d.Decode(&fileMeta); err != nil {
		return nil, SessionMeta{}, nil, err
	}
	var history []prompt.Prompt
	for {
		var p prompt.Prompt
		if err := d.Decode(&p, "tool_call.arguments:base64"); err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return nil, SessionMeta{}, nil, err
		}
		history = append(history, p)
	}

	// Reopen the file in append mode for future writes.
	afh, err := os.OpenFile(s.path, os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, SessionMeta{}, nil, err
	}

	return &fileSession{fh: afh, enc: hjl.NewEncoder(afh)}, fileMeta, history, nil
}

func (s *FileSerializer) create(meta SessionMeta) (SerializedSession, SessionMeta, []prompt.Prompt, error) {
	fh, err := os.Create(s.path)
	if err != nil {
		return nil, SessionMeta{}, nil, err
	}
	enc := hjl.NewEncoder(fh)
	if err := enc.Encode(meta, "system_prompt"); err != nil {
		fh.Close()
		return nil, SessionMeta{}, nil, err
	}
	return &fileSession{fh: fh, enc: enc}, meta, nil, nil
}

type fileSession struct {
	fh  *os.File
	enc *hjl.Encoder
}

func (s *fileSession) Append(prompts ...prompt.Prompt) error {
	for _, p := range prompts {
		if err := s.enc.Encode(p, "text", "tool_response.content", "tool_call.arguments:base64"); err != nil {
			return err
		}
	}
	return s.fh.Sync()
}

func (s *fileSession) Close() error {
	if s.fh != nil {
		return s.fh.Close()
	}
	return nil
}
