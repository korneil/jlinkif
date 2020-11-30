package internal

import (
	"bufio"
	"context"
	"io"
)

type LineReader struct {
	r *bufio.Reader
}

func NewLineReader(reader io.Reader) *LineReader {
	return &LineReader{bufio.NewReader(reader)}
}

func (x LineReader) ReadLine(ctx context.Context) (line []byte, err error) {
	var l []byte
	var more bool

	for {
		l, more, err = x.r.ReadLine()
		if ctx.Err() != nil || err != nil {
			return
		}
		if line == nil {
			line = l
		}
		if !more {
			return
		}
		line = append(line, l...)
	}
}
