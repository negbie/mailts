package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/golang/glog"
	"github.com/ymotongpoo/datemaki"
)

type QueryRangeResponse struct {
	Status string                  `json:"status"`
	Data   *QueryRangeResponseData `json:"data"`
}

type QueryRangeResponseData struct {
	Result []*QueryRangeResponseResult `json:"result"`
}

type QueryRangeResponseResult struct {
	Metric map[string]string          `json:"metric"`
	Values []*QueryRangeResponseValue `json:"values"`
}

type QueryRangeResponseValue []interface{}

func (v *QueryRangeResponseValue) Time() time.Time {
	t := (*v)[0].(float64)
	return time.Unix(int64(t), 0)
}

func (v *QueryRangeResponseValue) Value() (float64, error) {
	s := (*v)[1].(string)
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0, err
	}

	return f, nil
}

func queryTS(report *Report, writer chan<- WriterData) {
	start, err := datemaki.Parse(report.Connection.Start)
	if err != nil {
		glog.Error(err)
		return
	}

	end, err := datemaki.Parse(report.Connection.End)
	if err != nil {
		glog.Error(err)
		return
	}

	step, err := time.ParseDuration(report.Connection.Step)
	if err != nil {
		glog.Error(err)
		return
	}

	u, err := url.Parse(report.Connection.URL)
	if err != nil {
		glog.Error(err)
		return
	}
	u.Path = "/api/v1/query_range"
	q := u.Query()
	q.Set("query", report.Query)
	q.Set("start", fmt.Sprintf("%d", start.Unix()))
	q.Set("end", fmt.Sprintf("%d", end.Unix()))
	q.Set("step", fmt.Sprintf("%ds", int(step.Seconds())))
	u.RawQuery = q.Encode()

	req, err := http.NewRequest("GET", u.String(), nil)

	if report.Connection.User != "" || report.Connection.Password != "" {
		req.Header.Add("Authorization", "Basic "+
			base64.StdEncoding.EncodeToString([]byte(report.Connection.User+":"+report.Connection.Password)))
	}

	if err != nil {
		glog.Error(err)
		return
	}

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		glog.Error(err)
		return
	}

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		if err == io.EOF {
			//return &QueryRangeResponse{}, nil
			glog.Error(err)
			return
		}
		glog.Error(err)
		return
	}

	if 400 <= res.StatusCode {
		glog.Errorf("error response: %s", string(body))
		return
	}

	resp := &QueryRangeResponse{}
	err = json.Unmarshal(body, resp)
	if err != nil {
		glog.Error(err)
		return
	}

	type valueByMetric map[string]float64

	valuesByTime := map[time.Time]valueByMetric{}
	metrics := []string{}

	for _, r := range resp.Data.Result {
		metric := stringMapToString(r.Metric, "|")
		for _, v := range r.Values {
			t := v.Time()
			d, ok := valuesByTime[t]
			if !ok {
				d = valueByMetric{}
				valuesByTime[t] = d
			}
			var err error
			d[metric], err = v.Value()
			if err != nil {
				glog.Error(err)
				return
			}
		}

		found := false
		for _, m := range metrics {
			if m == metric {
				found = true
			}
		}
		if !found {
			metrics = append(metrics, metric)
		}
	}

	type st struct {
		time time.Time
		v    valueByMetric
	}
	slice := make([]st, len(valuesByTime))
	i := 0
	for t, v := range valuesByTime {
		slice[i] = st{t, v}
		i++
	}
	sort.Slice(slice, func(i, j int) bool {
		return slice[i].time.Before(slice[j].time)
	})

	header := make([]string, len(metrics)+1)
	header[0] = "time"
	copy(header[1:], metrics)
	report.Header = header
	var csvLineCount, xlsLinecount int64

	for k, s := range slice {
		if k > 0 {
			report.Header = nil
		}

		row := make([]string, len(metrics)+1)
		row[0] = s.time.String()
		for i, m := range metrics {
			if v, ok := s.v[m]; ok {
				row[i+1] = fmt.Sprintf("%f", v)
			} else {
				row[i+1] = ""
			}
		}

		if report.Output != nil {
			for _, screen := range report.Output.Screen {
				writer <- WriterData{
					Header: report.Header,
					Data:   row,
					Flush:  false,
					Writer: screen.Writer,
				}
			}
			for _, csv := range report.Output.Csv {
				writer <- WriterData{
					Header: report.Header,
					Data:   row,
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
					Data:   row,
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
				Data:   row,
				Flush:  false,
				Writer: newStdOutWriter(','),
			}
		}
	}
}

func stringMapToString(m map[string]string, delimiter string) string {
	s := make([][]string, len(m))
	i := 0
	for k, v := range m {
		s[i] = []string{k, v}
		i++
	}
	sort.Slice(s, func(i, j int) bool {
		return s[i][0] < s[j][0]
	})

	ss := make([]string, len(s))
	for i, v := range s {
		ss[i] = fmt.Sprintf("%s:%s", v[0], v[1])
	}

	return strings.Join(ss, delimiter)
}
