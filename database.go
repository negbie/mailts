package main

import (
	"fmt"

	"github.com/golang/glog"
	"github.com/jmoiron/sqlx"
)

func queryDB(report *Report, writer chan<- WriterData) {
	connstr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		report.Connection.Host,
		report.Connection.Port,
		report.Connection.User,
		report.Connection.Password,
		report.Connection.Database,
	)
	db, err := sqlx.Connect(report.Connection.Type, connstr)
	if err != nil {
		glog.Errorf("can't connect to [%v]", connstr)
		return
	}
	defer db.Close()

	var withHeader bool
	if len(report.Header) > 0 {
		withHeader = true
	}

	rows, err := db.Queryx(report.Query)
	if err != nil {
		glog.Errorf("can't execute query [%v]\n", report.Query)
		return
	}
	defer rows.Close()

	var csvLineCount, xlsLinecount int64

	for rows.Next() {
		r, err := rows.SliceScan()
		if err != nil {
			glog.Error(err)
			continue
		}
		s := make([]string, len(r))
		for i, v := range r {
			s[i] = fmt.Sprint(v)
		}

		if !withHeader {
			header, err := rows.Columns()
			if err != nil {
				glog.Error(err)
			}
			report.Header = header
			withHeader = true
		} else {
			report.Header = nil
		}

		if report.Output != nil {
			for _, screen := range report.Output.Screen {
				writer <- WriterData{
					Header: report.Header,
					Data:   s,
					Flush:  false,
					Writer: screen.Writer,
				}
			}
			for _, csv := range report.Output.Csv {
				writer <- WriterData{
					Header: report.Header,
					Data:   s,
					Flush:  csvLineCount >= 50,
					Writer: csv.Writer,
				}
				if csvLineCount >= 50 {
					csvLineCount = 0
				}
				csvLineCount++
			}
			for _, xls := range report.Output.Xls {
				writer <- WriterData{
					Header: report.Header,
					Data:   s,
					Flush:  xlsLinecount >= 50,
					Writer: xls.Writer,
				}
				if xlsLinecount >= 50 {
					xlsLinecount = 0
				}
				xlsLinecount++
			}
		} else {
			writer <- WriterData{
				Header: report.Header,
				Data:   s,
				Flush:  false,
				Writer: newStdOutWriter(','),
			}
		}
	}
}
