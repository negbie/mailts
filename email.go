package main

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"net/smtp"
	"strings"
	"time"

	"github.com/golang/glog"
)

func sendEmails(query *Query) {
	var attachments []string
	if query.Output != nil {
		for _, csv := range query.Output.Csv {
			if csv.Mail {
				attachments = append(attachments, csv.Filename)
			}
		}
		for _, xls := range query.Output.Xls {
			if xls.Mail {
				attachments = append(attachments, xls.Filename)
			}
		}
	}
	if len(attachments) > 0 {
		sendEmail(query, attachments)
	}
}

func sendEmail(query *Query, attachments []string) {
	var (
		server, from  string
		subject, body string
		to, cc        []string
		msg           bytes.Buffer
		boundary      = "__MAIL_TS_"
	)

	if server = query.Email.Server; server == "" {
		server = "localhost:25"
	}
	if from = query.Email.From; from == "" {
		from = "noreply@localhost.localdomain"
	}
	if to = query.Email.To; len(to) == 0 {
		to = []string{"to@localhost.localdomain"}
	}
	cc = query.Email.Cc
	if subject = query.Email.Subject; subject == "" {
		subject = "Mail TS - " + time.Now().Format(`20060102150405`)
	}
	if body = query.Email.Body; body == "" {
		body = "Hi,\r\n\r\nhere is your requested timeseries report!\r\n\r\nBr\r\nMail TS"
	}

	// header
	msg.WriteString(fmt.Sprintf("From: Mail TS <%s>\r\n", from))
	msg.WriteString(fmt.Sprintf("To: %s\r\n", strings.Join(to, ",")))
	if len(cc) > 0 {
		msg.WriteString(fmt.Sprintf("Cc: %s\r\n", strings.Join(cc, ",")))
	}
	// subject
	msg.WriteString(fmt.Sprintf("Subject: %s\r\n", subject))
	msg.WriteString(fmt.Sprintf("MIME-Version: 1.0\r\n"))
	if len(attachments) > 0 {
		msg.WriteString(fmt.Sprintf("Content-Type: multipart/mixed; boundary=%s\r\n--%s\r\n", boundary, boundary))
	}

	// body
	msg.WriteString(fmt.Sprintf("Content-Type: text/plain; charset=utf-8\r\n\r\n%s\r\n--%s", body, boundary))

	// attachment
	for _, attachment := range attachments {
		content, err := ioutil.ReadFile(attachment)
		if err != nil {
			glog.Error(err)
		}
		encoded := base64.StdEncoding.EncodeToString(content)

		msg.WriteString(fmt.Sprintf("\r\nContent-Type: application/octet-stream; name=\"%s\"\r\nContent-Transfer-Encoding:base64\r\n", attachment))
		msg.WriteString(fmt.Sprintf("Content-Disposition: attachment; filename=\"%s\"\r\n\r\n%s\r\n--%s", attachment, encoded, boundary))
	}
	msg.WriteString("--")

	err := smtp.SendMail(server, nil, from, to, msg.Bytes())
	if err != nil {
		glog.Errorf("can't send email to %v %v", to, err)
	}
}
