package main

import (
	"errors"
	"fmt"
	"io"
	"net/smtp"
	"strings"

	"github.com/domodwyer/mailyak/v3"
)

type Attachment interface {
	io.Reader
	Name() string
}

type Message struct {
	From        string
	To          []string
	CC          []string // seperated by comma
	BCC         []string // seperated by comma
	Subject     string   // plain text
	Body        string   // HTML body
	Attachments []Attachment
}

func (m Message) PrintDebug() {
	fmt.Println("\n----- Message Debug -----")
	fmt.Printf("From: %s\n", m.From)
	fmt.Printf("To: %s\n", strings.Join(m.To, ", "))
	fmt.Printf("CC: %s\n", strings.Join(m.CC, ", "))
	fmt.Printf("BCC: %s\n", strings.Join(m.BCC, ", "))
	fmt.Printf("Subject: %s\n", m.Subject)
	fmt.Printf("Body:\n%s\n", m.Body)

	var attachmentNames []string
	for _, at := range m.Attachments {
		attachmentNames = append(attachmentNames, at.Name())
	}
	fmt.Printf("Attachments: %s\n", strings.Join(attachmentNames, ", "))
	fmt.Println("-------------------------")
}

func NewMail(m *Message) (*mailyak.MailYak, error) {
	mail := mailyak.New(cfg.Host+":"+cfg.Port, LoginAuth(cfg.User, cfg.Password))

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
	for _, at := range m.Attachments {
		mail.Attach(at.Name(), at)
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
