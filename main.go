package main

import (
	"bytes"
	"encoding/csv"
	"errors"
	"flag"
	"fmt"
	"html/template"
	"log"
	"net/smtp"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/joho/godotenv"
	"github.com/xuri/excelize/v2"
)

const (
	SmtpHost = "smtp.office365.com"
	SmtpPort = "587"
)

var templ = template.New("")

func main() {
	godotenv.Load()
	var (
		isTest    = flag.Bool("test", false, "test mode")
		from      = flag.String("from", "", "sender address")
		pass      = flag.String("password", "", "microsoft account password")
		templPath = flag.String("template", "template.html", "email template path")
		csvPath   = flag.String("csv", "", "email template path")
		body      = os.Getenv("BODY")
	)
	flag.Parse()

	if len(*from) == 0 {
		*from = os.Getenv("FROM")
	}
	if len(*templPath) == 0 {
		*templPath = os.Getenv("EMAIL_TEMPLATE_PATH")
	}
	if len(*csvPath) == 0 {
		*csvPath = os.Getenv("CSV_FILE_PATH")
	}
	// validations
	if len(*from) == 0 {
		log.Fatal("no sender address provided")
	}
	if len(*pass) == 0 {
		log.Fatal("no password provided")
	}
	if len(*templPath) == 0 && len(strings.TrimSpace(body)) == 0 {
		log.Fatal("no email template provided")
	}
	if len(*csvPath) == 0 {
		log.Fatal("no csv file path provided")
	}

	if len(*templPath) > 0 {
		_, err := templ.ParseFiles(*templPath)
		if err != nil {
			if !errors.Is(err, os.ErrNotExist) {
				log.Fatal(err)
			}

			_, err := templ.New(*templPath).Parse(body)
			if err != nil {
				log.Fatal(err)
			}
		}
	}

	recs, err := readSpreadsheet(*csvPath)
	if err != nil {
		log.Fatal(err)
	}

	data, err := mapDataToAny(recs)
	if err != nil {
		log.Fatal(err)
	}

	_, err = templ.New("subject").Parse(os.Getenv("SUBJECT"))
	if err != nil {
		log.Fatal(err)
	}

	emailsToBeSent := []*Email{}
	for _, row := range data {
		buf := new(bytes.Buffer)
		subBuf := new(bytes.Buffer)

		err = templ.ExecuteTemplate(buf, *templPath, row)
		if err != nil {
			log.Fatal(err)
		}

		err = templ.ExecuteTemplate(subBuf, "subject", row)
		if err != nil {
			log.Fatal(err)
		}

		to := []string{}
		for _, value := range row {
			vStr := value.(string)
			if len(to) == 0 && IsValidEmail(vStr) {
				to = append(to, vStr)
				break
			}
		}

		cc := parseEmails(os.Getenv("CC"))
		bcc := parseEmails(os.Getenv("BCC"))
		emailsToBeSent = append(emailsToBeSent, &Email{
			From:    *from,
			To:      to,
			Subject: subBuf.String(),
			Body:    strings.ReplaceAll(strings.TrimSpace(buf.String()), "\n", "<br/>"),
			CC:      cc,
			BCC:     bcc,
		})
	}

	auth := LoginAuth(*from, *pass)
	for _, email := range emailsToBeSent {
		if *isTest {
			fmt.Println("\n" + string(buildEmail(*email)))
			return
		} else {
			allRecepients := []string{}
			allRecepients = append(allRecepients, email.To...)
			allRecepients = append(allRecepients, email.BCC...)
			allRecepients = append(allRecepients, email.CC...)

			err = smtp.SendMail(SmtpHost+":"+SmtpPort, auth, *from, allRecepients, buildEmail(*email))
			if err != nil {
				log.Fatal(err)
			}
		}
	}
}

type loginAuth struct {
	username, password string
}

// LoginAuth is used for smtp login auth
func LoginAuth(username, password string) smtp.Auth {
	return &loginAuth{
		username: username,
		password: password,
	}
}

func (a *loginAuth) Start(server *smtp.ServerInfo) (string, []byte, error) {
	return "LOGIN", []byte(a.username), nil
}

func (a *loginAuth) Next(fromServer []byte, more bool) ([]byte, error) {
	if more {
		switch string(fromServer) {
		case "Username:":
			return []byte(a.username), nil
		case "Password:":
			return []byte(a.password), nil
		default:
			return nil, errors.New("Unknown from server")
		}
	}
	return nil, nil
}

type Email struct {
	From    string
	To      []string
	CC      []string
	BCC     []string
	Subject string
	Body    string // HTML body
}

func buildEmail(e Email) []byte {
	var msg bytes.Buffer

	// Required headers
	msg.WriteString(fmt.Sprintf("From: %s\r\n", e.From))
	msg.WriteString(fmt.Sprintf("To: %s\r\n", strings.Join(e.To, ",")))

	// Optional CC
	if len(e.CC) > 0 {
		msg.WriteString(fmt.Sprintf("CC: %s\r\n", strings.Join(e.CC, ",")))
	}

	// Optional Subject
	if e.Subject != "" {
		msg.WriteString(fmt.Sprintf("Subject: %s\r\n", e.Subject))
	}

	// MIME headers (important for HTML)
	msg.WriteString("MIME-Version: 1.0\r\n")
	msg.WriteString("Content-Type: text/html; charset=\"UTF-8\"\r\n")

	// End headers
	msg.WriteString("\r\n")

	// Body
	msg.WriteString(e.Body)

	return msg.Bytes()
}

func readSpreadsheet(filepathStr string) ([][]string, error) {
	ext := filepath.Ext(filepathStr)

	switch ext {
	case ".csv":
		f, err := os.Open(filepathStr)
		if err != nil {
			log.Fatal(err)
		}
		defer f.Close()

		r := csv.NewReader(f)
		records, err := r.ReadAll()
		if err != nil {
			return nil, err
		}
		return records, nil
	case ".xlsx":
		f, err := excelize.OpenFile(filepathStr)
		if err != nil {
			return nil, err
		}

		sheets := f.GetSheetList()
		if len(sheets) == 0 {
			return nil, errors.New("no sheets found in XLSX")
		}

		rows, err := f.GetRows(sheets[0])
		if err != nil {
			return nil, err
		}
		return rows, nil
	default:
		return nil, errors.New("unsupported file type")
	}
}

func mapDataToAny(data [][]string) ([]map[string]any, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("empty data")
	}

	headers := data[0]
	records := make([]map[string]any, 0, len(data)-1)

	for _, row := range data[1:] {
		record := make(map[string]any)
		for j, header := range headers {
			if j < len(row) {
				record[header] = row[j]
			} else {
				record[header] = "" // missing value
			}
		}
		records = append(records, record)
	}

	return records, nil
}

func IsValidEmail(email string) bool {
	// RFC 5322 simplified regex for general email validation
	const emailRegex = `^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`
	re := regexp.MustCompile(emailRegex)
	return re.MatchString(email)
}

func parseEmails(emailsStr string) []string {
	val := strings.TrimSpace(emailsStr)
	if val == "" {
		return nil // no CC/BCC
	}

	// Split by comma and trim spaces
	parts := strings.Split(val, ",")
	res := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			res = append(res, p)
		}
	}
	return res
}
