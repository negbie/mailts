package main

import (
	"fmt"
	"os"

	_ "github.com/go-sql-driver/mysql"
	"github.com/golang/glog"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

type InputData struct {
	values map[string]interface{}
}

func createReport(report *Report, done chan<- struct{}, writer chan<- WriterData) {
	defer func() { done <- struct{}{} }()

	procdone := make(chan struct{}, 1)
	procinput := make(chan InputData, 1)
	if report.Connection.Driver == "prometheus" {
		//go queryTS(report, procdone, writer, procinput)
	} else {
		go queryDB(report, procdone, writer, procinput)
	}

	procinput <- InputData{}
	close(procinput)
	<-procdone

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

func queryDB(report *Report, done chan<- struct{}, writer chan<- WriterData, input <-chan InputData) {
	defer func() { done <- struct{}{} }()

	connstr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		report.Connection.Host,
		report.Connection.Port,
		report.Connection.User,
		report.Connection.Password,
		report.Connection.Database,
	)
	db, err := sqlx.Connect(report.Connection.Driver, connstr)
	if err != nil {
		glog.Errorf("can't connect to [%v]", connstr)
		return
	}
	defer db.Close()

	var withHeader bool
	if len(report.Header) > 0 {
		withHeader = true
	}

	for in := range input {
		stmt, err := db.PrepareNamed(report.Query)
		if err != nil {
			glog.Errorf("can't prepare query [%v]", report.Query)
			continue
		}
		defer stmt.Close()

		rows, err := stmt.Queryx(in.values)
		if err != nil {
			glog.Errorf("can't execute query [%v], with values: %v\n", report.Query, in.values)
			continue
		}
		defer rows.Close()

		var csvLineCount, xlsLinecount int64
		for rows.Next() {
			data := make(map[string]interface{})
			err = rows.MapScan(data)
			if err != nil {
				glog.Error(err)
				continue
			}

			if !withHeader {
				header, err := rows.Columns()
				if err != nil {
					glog.Error(err)
				}
				report.Header = header
				withHeader = true
			}

			if report.Output != nil {
				for _, screen := range report.Output.Screen {
					writer <- WriterData{
						Header: report.Header,
						Data:   data,
						Flush:  false,
						Writer: screen.Writer,
					}
				}
				for _, csv := range report.Output.Csv {
					writer <- WriterData{
						Header: report.Header,
						Data:   data,
						Flush:  csvLineCount >= 20,
						Writer: csv.Writer,
					}
					if csvLineCount >= 20 {
						csvLineCount = 0
					}
					csvLineCount++
				}
				for _, xls := range report.Output.Xls {
					writer <- WriterData{
						Header: report.Header,
						Data:   data,
						Flush:  xlsLinecount >= 20,
						Writer: xls.Writer,
					}
					if xlsLinecount >= 20 {
						xlsLinecount = 0
					}
					xlsLinecount++
				}
			} else {
				writer <- WriterData{
					Header: report.Header,
					Data:   data,
					Flush:  false,
					Writer: newStdOutWriter(','),
				}
			}
		}
	}
}
