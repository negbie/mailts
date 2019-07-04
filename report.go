package main

import (
	"os"

	_ "github.com/go-sql-driver/mysql"
	"github.com/golang/glog"
	_ "github.com/lib/pq"
)

func doReport(report *Report, done chan<- struct{}, writer chan<- WriterData) {
	defer func() { done <- struct{}{} }()

	if report.Prometheus != nil {
		queryTS(report, writer)
	} else if report.Database != nil {
		queryDB(report, writer)
	} else {
		return
	}

	flushFiles(report)
	sendEmails(report)
	closeFiles(report)

	if report.Output != nil {
		for _, csv := range report.Output.Csv {
			if csv.Temporary {
				if err := os.Remove(csv.Filename); err != nil {
					glog.Error(err)
				}
			}
		}
		for _, xls := range report.Output.Xls {
			if xls.Temporary {
				if err := os.Remove(xls.Filename); err != nil {
					glog.Error(err)
				}
			}
		}
	}
}
