package main

import (
	"github.com/golang/glog"
)

type WriterData struct {
	Header []string
	Data   []string
	Flush  bool
	Writer Writer
}

type Writer interface {
	Print([]string, []string) error
	Close() error
	Flush() error
}

func WriterRoutine(done chan<- struct{}, data <-chan WriterData) {
	defer func() { done <- struct{}{} }()

	for d := range data {
		if err := d.Writer.Print(d.Header, d.Data); err != nil {
			glog.Error(err)
		}

		if d.Flush {
			if err := d.Writer.Flush(); err != nil {
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
