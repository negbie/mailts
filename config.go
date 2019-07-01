package main

import (
	"io/ioutil"
	"strings"
	"time"
	"unicode/utf8"

	"gopkg.in/yaml.v2"
)

type Report struct {
	Query []Query `json:"query"`
}

type Query struct {
	Output     *Output    `json:"output"`
	Email      *Email     `json:"email"`
	Connection Connection `json:"connection"`
	Range      *Range     `json:"range"`
	Header     []string   `json:"header"`
	Delimiter  string     `json:"delimiter"`
	Statement  string     `json:"statement"`
}

type Output struct {
	Csv    []*Csv    `json:"csv"`
	Xls    []*Xls    `json:"xls"`
	Screen []*Screen `json:"screen"`
}

type Csv struct {
	Filename  string `json:"filename"`
	Mail      bool   `json:"mail"`
	Temporary bool   `json:"temporary"`
	Writer    Writer
}

type Xls struct {
	Filename  string `json:"filename"`
	Sheetname string `json:"sheetname"`
	Mail      bool   `json:"mail"`
	Temporary bool   `json:"temporary"`
	Writer    Writer
}

type Screen struct {
	Filename string `json:"filename"`
	Mail     bool   `json:"mail"`
	Writer   Writer
}

type Email struct {
	Server  string   `json:"server"`
	From    string   `json:"from"`
	Subject string   `json:"subject"`
	Body    string   `json:"body"`
	To      []string `json:"to"`
	Cc      []string `json:"cc"`
}

type Connection struct {
	Driver   string `json:"driver"`
	Host     string `json:"host"`
	Port     string `json:"port"`
	Database string `json:"database"`
	User     string `json:"user"`
	Password string `json:"password"`
}

type Range struct {
	Start        int64  `json:"start"`
	Stepsize     int64  `json:"stepsize"`
	Steps        int64  `json:"steps"`
	Parallel     int    `json:"parallel"`
	BindvarStart string `json:"bindvar_start"`
	BindvarEnd   string `json:"bindvar_end"`
}

func loadConfig(filename string) (*Report, error) {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	var config Report
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		return nil, err
	}

	for c, query := range config.Query {
		if len(query.Delimiter) == 0 {
			query.Delimiter = ","
		}
		delimiter, _ := utf8.DecodeRuneInString(query.Delimiter)

		if len(query.Header) > 0 {
			config.Query[c].Header = strings.Split(sanitizeHeader(query.Header[0]), string(delimiter))
		}

		if query.Output != nil {
			for _, screen := range query.Output.Screen {
				if strings.ToUpper(screen.Filename) == "STDERR" {
					screen.Writer = newStdErrWriter(delimiter)
				} else {
					screen.Writer = newStdOutWriter(delimiter)
				}
			}
			for _, csv := range query.Output.Csv {
				csv.Filename = replaceTemplate(csv.Filename)
				csv.Writer = newCsvWriter(csv.Filename, delimiter)
			}
			for _, xls := range query.Output.Xls {
				xls.Sheetname = replaceTemplate(xls.Sheetname)
				xls.Filename = replaceTemplate(xls.Filename)
				xls.Writer = newXlsWriter(xls.Filename, xls.Sheetname, config.Query[c].Header)
			}
		}

		if query.Email == nil {
			config.Query[c].Email = &Email{}
		}
	}

	return &config, nil
}

func sanitizeHeader(header string) string {
	return strings.Trim(header, "\t\n\r ")
}

func replaceTemplate(filename string) string {
	ts := time.Now().Format(`20060102150405`)
	filename = strings.Replace(filename, `{DATE}`, ts[:8], -1)
	filename = strings.Replace(filename, `{TIME}`, ts[8:], -1)
	filename = strings.Replace(filename, `{DATETIME}`, ts, -1)
	filename = strings.Replace(filename, `{TIMESTAMP}`, ts, -1)
	return filename
}
