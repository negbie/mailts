package main

import (
	"fmt"
	"os"

	_ "github.com/go-sql-driver/mysql"
	"github.com/golang/glog"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

func createReport(query *Query, done chan<- interface{}, writer chan<- WriterData) {
	defer func() { done <- nil }()

	connstr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		query.Connection.Host,
		query.Connection.Port,
		query.Connection.User,
		query.Connection.Password,
		query.Connection.Database,
	)
	db, err := sqlx.Connect(query.Connection.Driver, connstr)
	if err != nil {
		glog.Errorf("can't connect to [%v]", connstr)
		return
	}
	defer db.Close()

	if query.Range != nil {
		// parallel queries
		var bindvars map[string]interface{} = make(map[string]interface{})
		rangeStart := query.Range.Start
		rangeEnd := rangeStart + query.Range.Stepsize - 1
		rangeMax := rangeStart + (query.Range.Stepsize * query.Range.Steps)

		procdone := make(chan interface{}, query.Range.Parallel)
		procinput := make(chan InputData, query.Range.Parallel)

		// start parallel processes
		for p := 0; p < query.Range.Parallel; p++ {
			go process(query, db, procdone, writer, procinput)
		}

		for {
			if rangeEnd > rangeMax {
				rangeEnd = rangeMax
			}

			bindvars[query.Range.BindvarStart] = rangeStart
			bindvars[query.Range.BindvarEnd] = rangeEnd

			procinput <- InputData{bindvars}

			if rangeEnd >= rangeMax {
				break
			}
			rangeStart = rangeEnd + 1
			rangeEnd = rangeStart + query.Range.Stepsize - 1
		}
		close(procinput)

		for p := 0; p < query.Range.Parallel; p++ {
			<-procdone
		}

	} else {
		procdone := make(chan interface{}, 1)
		procinput := make(chan InputData, 1)
		go process(query, db, procdone, writer, procinput)
		procinput <- InputData{}
		close(procinput)
		<-procdone
	}

	flushFiles(query)
	sendEmails(query)
	closeFiles(query)

	if query.Output != nil {
		for _, csv := range query.Output.Csv {
			if csv.Temporary {
				if err := os.Remove(csv.Filename); err != nil {
					glog.Error(err)
				}
			}
		}
		for _, xls := range query.Output.Xls {
			if xls.Temporary {
				if err := os.Remove(xls.Filename); err != nil {
					glog.Error(err)
				}
			}
		}
	}
}

type InputData struct {
	Bindvars map[string]interface{}
}

func process(query *Query, db *sqlx.DB, done chan<- interface{}, writer chan<- WriterData, input <-chan InputData) {
	defer func() { done <- nil }()

	var withHeader bool
	if len(query.Header) > 0 {
		withHeader = true
	}

	for in := range input {
		stmt, err := db.PrepareNamed(query.Statement)
		if err != nil {
			glog.Errorf("can't prepare statement [%v]", query.Statement)
			continue
		}
		defer stmt.Close()

		rows, err := stmt.Queryx(in.Bindvars)
		if err != nil {
			glog.Errorf("can't execute statement [%v], with bindvars: %v\n", query.Statement, in.Bindvars)
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
				query.Header = header
				withHeader = true
			}

			if query.Output != nil {
				for _, screen := range query.Output.Screen {
					writer <- WriterData{
						Header: query.Header,
						Data:   data,
						Flush:  false,
						Writer: screen.Writer,
					}
				}
				for _, csv := range query.Output.Csv {
					writer <- WriterData{
						Header: query.Header,
						Data:   data,
						Flush:  csvLineCount >= 20,
						Writer: csv.Writer,
					}
					if csvLineCount >= 20 {
						csvLineCount = 0
					}
					csvLineCount++
				}
				for _, xls := range query.Output.Xls {
					writer <- WriterData{
						Header: query.Header,
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
					Header: query.Header,
					Data:   data,
					Flush:  false,
					Writer: newStdOutWriter(','),
				}
			}
		}
	}
}
