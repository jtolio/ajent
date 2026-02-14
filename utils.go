package main

import (
	"errors"
	"io"
)

type UnbufferedLineReader struct {
	r         io.Reader
	maxLength int
}

func NewUnbufferedLineReader(r io.Reader, maxLength int) *UnbufferedLineReader {
	return &UnbufferedLineReader{
		r:         r,
		maxLength: maxLength,
	}
}

func (r *UnbufferedLineReader) ReadLine() (rv string, err error) {
	if r.r == nil {
		return "", io.EOF
	}

	var outbuf [4096]byte
	out := outbuf[:0]
	for {
		var b [1]byte
		_, err := io.ReadFull(r.r, b[:])
		if err != nil {
			if errors.Is(err, io.EOF) {
				r.r = nil
				if len(out) > 0 {
					return string(out), nil
				}
			}
			return "", err
		}
		out = append(out, b[0])
		if len(out) > r.maxLength {
			return "", errors.New("max line length exceeded")
		}
		if b[0] == '\n' {
			return string(out), nil
		}
	}
}
