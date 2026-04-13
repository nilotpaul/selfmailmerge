package main

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/xuri/excelize/v2"
)

var (
	cfg *Config

	templ        = template.New("")
	templatePath = "template.html"
)

func main() {
	var err error
	cfg, err = NewConfig()
	if err != nil {
		log.Fatalf("config error: %+v", err)
	}

	_, err = templ.New(templatePath).Parse(cfg.BodyContent)
	if err != nil {
		log.Fatal(err)
	}

	recs, err := readSpreadsheet(cfg.SpreadsheetPath)
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

		err = templ.ExecuteTemplate(buf, templatePath, row)
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

		attachments := make([]Attachment, len(cfg.AttachmentPaths))
		for i, path := range cfg.AttachmentPaths {
			f, err := os.Open(path)
			if err != nil {
				log.Fatalf("attachment error: %+v", err)
			}
			defer f.Close()

			attachments[i] = f
		}
		emailsToBeSent = append(emailsToBeSent, &Message{
			From:        cfg.From,
			To:          to,
			Subject:     subBuf.String(),
			Body:        strings.ReplaceAll(strings.TrimSpace(buf.String()), "\n", "<br/>"),
			CC:          parseStringSlice(os.Getenv("CC")),
			BCC:         parseStringSlice(os.Getenv("BCC")),
			Attachments: attachments,
		})
	}

	errs := map[string]string{}
	for _, msg := range emailsToBeSent {
		mail, err := NewMail(msg)
		if err != nil {
			log.Fatal(err)
		}

		if cfg.IsTest {
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
		if err := writeErrorsToJson(errs); err != nil {
			log.Println("WARN: failed to write errors in emails.json file")
		}
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
		defer f.Close()

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

func writeErrorsToJson(errs map[string]string) error {
	file, err := os.Create("emails.json")
	if err != nil {
		return err
	}
	defer file.Close()

	enc := json.NewEncoder(file)
	enc.SetIndent("", "  ")
	return enc.Encode(map[string]any{
		"count":  len(errs),
		"failed": errs,
	})
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
