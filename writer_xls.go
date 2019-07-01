package main

import (
	"sync"

	"github.com/golang/glog"
	"github.com/tealeg/xlsx"
)

type XlsWriter struct {
	Out *xlsx.StreamFile
	*sync.RWMutex
}

func (x XlsWriter) Print(header []string, data map[string]interface{}) error {
	x.RLock()
	defer x.RUnlock()
	return x.Out.Write(appendRow(header, data))
}

func (x XlsWriter) Flush() error {
	x.Lock()
	defer x.Unlock()
	x.Out.Flush()
	return x.Out.Error()
}

func (x XlsWriter) Close() error {
	x.Flush()
	return x.Out.Close()
}

func newXlsWriter(filename, sheetname string, header []string) *XlsWriter {
	file, err := xlsx.NewStreamFileBuilderForPath(filename)
	if err != nil {
		glog.Errorf("can't create xlsx file: %v", filename)
	}

	err = file.AddSheet(sheetname, header, nil)
	if err != nil {
		glog.Errorf("can't create xlsx sheet: %v", sheetname)
	}

	streamFile, err := file.Build()
	if err != nil {
		glog.Errorf("can't create xlsx stream file: %v", filename)
	}

	return &XlsWriter{streamFile, new(sync.RWMutex)}
}
