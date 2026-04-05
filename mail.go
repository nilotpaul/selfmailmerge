package main

import (
	"errors"
	"fmt"
	"net/smtp"
	"os"
	"strings"

	"github.com/domodwyer/mailyak/v3"
)

type Message struct {
	From            string
	To              []string
	CC              []string // seperated by comma
	BCC             []string // seperated by comma
	Subject         string
	Body            string   // HTML body
	AttachmentPaths []string // file path seperated by comma
}

func (m Message) PrintDebug() {
	fmt.Println("\n----- Message Debug -----")
	fmt.Printf("From: %s\n", m.From)
	fmt.Printf("To: %s\n", strings.Join(m.To, ", "))
	fmt.Printf("CC: %s\n", strings.Join(m.CC, ", "))
	fmt.Printf("BCC: %s\n", strings.Join(m.BCC, ", "))
	fmt.Printf("Subject: %s\n", m.Subject)
	fmt.Printf("Body:\n%s\n", m.Body)
	fmt.Printf("Attachments: %s\n", strings.Join(m.AttachmentPaths, ", "))
	fmt.Println("-------------------------")
}

func NewMail(m *Message) (*mailyak.MailYak, error) {
	mail := mailyak.New(SmtpHost+":"+SmtpPort, LoginAuth(m.From, password))

	mail.To(m.To...)
	mail.From(m.From)
	mail.HTML().Set(m.Body)

	if m.Subject != "" {
		mail.Subject(m.Subject)
	}
	if len(m.CC) > 0 {
		mail.Cc(m.CC...)
	}
	if len(m.BCC) > 0 {
		mail.Bcc(m.BCC...)
	}
	if len(m.AttachmentPaths) > 0 {
		for _, path := range m.AttachmentPaths {
			f, err := os.Open(path)
			if err != nil {
				return nil, fmt.Errorf("attachment error: %v\n", err)
			}

			mail.Attach(path, f)
		}
	}

	return mail, nil
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
