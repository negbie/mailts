package main

import (
	"encoding/csv"
	"io"
	"os"
	"sync"
)

type ScreenWriter struct {
	Out *csv.Writer
	*sync.RWMutex
}

func (w ScreenWriter) Print(header []string, data []string) error {
	w.RLock()
	defer w.RUnlock()
	if header != nil {
		w.Out.Write(header)
	}
	return w.Out.Write(data)
}

func (w ScreenWriter) Flush() error {
	w.Lock()
	defer w.Unlock()
	w.Out.Flush()
	return w.Out.Error()
}

func (w ScreenWriter) Close() error {
	return w.Flush()
}

func newScreenWriter(out io.Writer, delimiter rune) *ScreenWriter {
	w := csv.NewWriter(out)
	w.Comma = delimiter
	return &ScreenWriter{w, new(sync.RWMutex)}
}

func newStdOutWriter(delimiter rune) *ScreenWriter {
	return newScreenWriter(os.Stdout, delimiter)
}

func newStdErrWriter(delimiter rune) *ScreenWriter {
	return newScreenWriter(os.Stderr, delimiter)
}
