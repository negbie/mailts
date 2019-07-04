package main

import (
	"io/ioutil"
	"strings"
	"time"
	"unicode/utf8"

	"gopkg.in/yaml.v2"
)

type Telegram struct {
	Report []Report `json:"report"`
}

type Report struct {
	Output     *Output     `json:"output"`
	Email      *Email      `json:"email"`
	Database   *Database   `json:"database"`
	Prometheus *Prometheus `json:"prometheus"`
	Header     []string    `json:"header"`
	Delimiter  string      `json:"delimiter"`
	Query      string      `json:"query"`
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

type Database struct {
	Driver   string `json:"driver"`
	Host     string `json:"host"`
	Port     string `json:"port"`
	Name     string `json:"name"`
	User     string `json:"user"`
	Password string `json:"password"`
	SSL      string `json:"ssl"`
}

type Prometheus struct {
	URL      string `json:"url"`
	User     string `json:"user"`
	Password string `json:"password"`
	Start    string `json:"start"`
	End      string `json:"end"`
	Step     string `json:"step"`
}

func loadConfig(filename string) (*Telegram, error) {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	var config Telegram
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		return nil, err
	}

	for c, report := range config.Report {
		if len(report.Delimiter) == 0 {
			report.Delimiter = ","
		}
		delimiter, _ := utf8.DecodeRuneInString(report.Delimiter)

		if len(report.Header) > 0 {
			config.Report[c].Header = strings.Split(sanitizeHeader(report.Header[0]), string(delimiter))
		}

		if report.Output != nil {
			for _, screen := range report.Output.Screen {
				if strings.ToUpper(screen.Filename) == "STDERR" {
					screen.Writer = newStdErrWriter(delimiter)
				} else {
					screen.Writer = newStdOutWriter(delimiter)
				}
			}
			for _, csv := range report.Output.Csv {
				csv.Filename = replaceTemplate(csv.Filename)
				csv.Writer = newCsvWriter(csv.Filename, delimiter)
			}
			for _, xls := range report.Output.Xls {
				xls.Sheetname = replaceTemplate(xls.Sheetname)
				xls.Filename = replaceTemplate(xls.Filename)
				xls.Writer = newXlsWriter(xls.Filename, xls.Sheetname, config.Report[c].Header)
			}
		}

		if report.Email == nil {
			config.Report[c].Email = &Email{}
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
