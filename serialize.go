package main

import (
	"encoding/json"
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

type fileFormat struct {
	Meta    SessionMeta     `json:"meta"`
	History []prompt.Prompt `json:"history"`
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
	return SessionMeta{}, nil, false, nil

	fh, err := os.Open(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return SessionMeta{}, nil, false, nil
		}
		return SessionMeta{}, nil, false, err
	}
	defer fh.Close()
	var full fileFormat
	err = json.NewDecoder(fh).Decode(&full)
	if err != nil {
		return SessionMeta{}, nil, true, err
	}
	return full.Meta, full.History, true, nil
}
