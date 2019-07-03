package main

import (
	"encoding/csv"
	"os"
	"sync"

	"github.com/golang/glog"
)

type CsvWriter struct {
	Out *csv.Writer
	*sync.RWMutex
}

func (w CsvWriter) Print(header []string, data []string) error {
	w.RLock()
	defer w.RUnlock()
	return w.Out.Write(data)
}

func (w CsvWriter) Flush() error {
	w.Lock()
	defer w.Unlock()
	w.Out.Flush()
	return w.Out.Error()
}

func (w CsvWriter) Close() error {
	return w.Flush()
}

func newCsvWriter(filename string, delimiter rune) *CsvWriter {
	file, err := os.Create(filename)
	if err != nil {
		glog.Errorf("can't create CSV file: %v", filename)
	}

	w := csv.NewWriter(file)
	w.Comma = delimiter
	return &CsvWriter{w, new(sync.RWMutex)}
}
