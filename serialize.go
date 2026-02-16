package main

import (
	"errors"
	"io"
	"os"

	"github.com/modfin/bellman/prompt"
)

type SessionMeta struct {
	SystemPrompt string `json:"system_prompt"`
}

type Serializer interface {
	Serialize(meta SessionMeta, history []prompt.Prompt) error
	Load() (meta SessionMeta, history []prompt.Prompt, found bool, err error)
}

type FileSerializer struct {
	path string
}

func NewFileSerializer(path string) *FileSerializer {
	return &FileSerializer{path: path}
}

func (s *FileSerializer) Serialize(meta SessionMeta, history []prompt.Prompt) error {
	fh, err := os.Create(s.path)
	if err != nil {
		return err
	}
	defer fh.Close()

	e := NewEncoder(fh)

	if err := e.Encode(meta); err != nil {
		return err
	}

	for _, p := range history {
		if err := e.Encode(p); err != nil {
			return err
		}
	}

	return fh.Close()
}

func (s *FileSerializer) Load() (meta SessionMeta, history []prompt.Prompt, found bool, err error) {
	fh, err := os.Open(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return SessionMeta{}, nil, false, nil
		}
		return SessionMeta{}, nil, false, err
	}
	defer fh.Close()

	d := NewDecoder(fh)

	if err := d.Decode(&meta); err != nil {
		return SessionMeta{}, nil, true, err
	}
	for {
		var p prompt.Prompt
		if err := d.Decode(&p); err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return SessionMeta{}, nil, true, err
		}
		history = append(history, p)
	}
	return meta, history, true, nil
}
