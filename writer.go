package main

import (
	"fmt"

	"github.com/golang/glog"
)

type WriterData struct {
	Header []string
	Data   map[string]interface{}
	Flush  bool
	Writer Writer
}

type Writer interface {
	Print([]string, map[string]interface{}) error
	Close() error
	Flush() error
}

type Row []string

func appendRow(header []string, data map[string]interface{}) (r Row) {
	for _, key := range header {
		r = append(r, fmt.Sprint(data[key]))
	}
	return
}

func WriterRoutine(done chan<- struct{}, input <-chan WriterData) {
	defer func() { done <- struct{}{} }()

	for in := range input {
		if err := in.Writer.Print(in.Header, in.Data); err != nil {
			glog.Error(err)
		}

		if in.Flush {
			if err := in.Writer.Flush(); err != nil {
				glog.Error(err)
			}
		}
	}
}

func flushFiles(report *Report) {
	if report.Output != nil {
		for _, csv := range report.Output.Csv {
			csv.Writer.Flush()
		}
		for _, xls := range report.Output.Xls {
			xls.Writer.Flush()
		}
	}
}

func closeFiles(report *Report) {
	if report.Output != nil {
		for _, csv := range report.Output.Csv {
			csv.Writer.Close()
		}
		for _, xls := range report.Output.Xls {
			xls.Writer.Close()
		}
	}
}
