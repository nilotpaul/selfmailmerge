package main

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"html/template"
	"log"
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
var password string

func main() {
	godotenv.Load()
	var (
		isTest    = flag.Bool("test", false, "test mode")
		from      = flag.String("from", "", "sender address")
		pass      = flag.String("password", "", "microsoft account password")
		templPath = flag.String("template", "template.html", "email template path")
		csvPath   = flag.String("csv", "", "email template path")

		attachment = os.Getenv("ATTACHMENT")
		body       = os.Getenv("BODY")
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
	password = *pass

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

	data, headers, err := mapDataToAny(recs)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("\navailable: ", strings.Join(headers, ", "))

	_, err = templ.New("subject").Parse(os.Getenv("SUBJECT"))
	if err != nil {
		log.Fatal(err)
	}

	emailsToBeSent := []*Message{}
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
			vStr, ok := value.(string)
			if !ok {
				continue
			}
			if len(to) == 0 && isValidEmail(vStr) {
				to = append(to, vStr)
				break
			}
		}

		cc := parseStringSlice(os.Getenv("CC"))
		bcc := parseStringSlice(os.Getenv("BCC"))
		emailsToBeSent = append(emailsToBeSent, &Message{
			From:            *from,
			To:              to,
			Subject:         subBuf.String(),
			Body:            strings.ReplaceAll(strings.TrimSpace(buf.String()), "\n", "<br/>"),
			CC:              cc,
			BCC:             bcc,
			AttachmentPaths: parseStringSlice(attachment),
		})
	}

	errs := map[string]string{}
	for _, msg := range emailsToBeSent {
		mail, err := NewMail(msg)
		if err != nil {
			log.Fatal(err)
		}

		if *isTest {
			msg.PrintDebug()
			return
		} else {
			if err := mail.Send(); err != nil {
				for _, e := range msg.To {
					errs[e] = err.Error()
				}
			}
		}
	}
	if len(errs) == 0 {
		fmt.Printf("%d emails sent succesfully!", len(emailsToBeSent))
	} else {
		res := map[string]any{"count": len(errs), "failed": errs}
		file, _ := os.Create("emails.json")
		defer file.Close()
		enc := json.NewEncoder(file)
		enc.SetIndent("", "  ")
		enc.Encode(res)
		fmt.Println("failed to send some/all emails, check emails.json file")
	}
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
	default:
		f, err := excelize.OpenFile(filepathStr)
		if err != nil {
			return nil, err
		}

		sheets := f.GetSheetList()
		if len(sheets) == 0 {
			return nil, fmt.Errorf("no sheets found in excel")
		}

		rows, err := f.GetRows(sheets[0])
		if err != nil {
			return nil, err
		}
		return rows, nil
	}
}

func mapDataToAny(data [][]string) ([]map[string]any, []string, error) {
	if len(data) == 0 {
		return nil, nil, fmt.Errorf("empty data")
	}

	headers := data[0]
	records := make([]map[string]any, 0, len(data)-1)
	for i, s := range headers {
		headers[i] = normalize(s)
	}

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

	return records, headers, nil
}

var reEmail = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)

func isValidEmail(email string) bool {
	// RFC 5322 simplified regex for general email validation
	return reEmail.MatchString(email)
}

func parseStringSlice(emailsStr string) []string {
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

var reNormal = regexp.MustCompile(`[^a-zA-Z0-9]+`)

func normalize(s string) string {
	// s = strings.ToLower(s) maybe?
	s = reNormal.ReplaceAllString(s, "_")
	s = strings.Trim(s, "_")
	return s
}
