package mail

import (
	"bytes"
	"fmt"
	html "html/template"
	"io"
	"net/http"
	text "text/template"

	"github.com/39alpha/dorothy/core"
	gomail "gopkg.in/gomail.v2"
)

type Message struct {
	From         string
	To           string
	Subject      string
	HtmlTemplate string
	TextTemplate string
	Data         map[string]any
}

type executable interface {
	Execute(w io.Writer, data any) error
}

type parsable[E executable] interface {
	Parse(body string) (E, error)
}

type template[E executable] interface {
	parsable[E]
	executable
}

func loadTemplate[E executable](t template[E], fsys http.FileSystem, path string, data any) (string, error) {
	var body bytes.Buffer
	file, err := fsys.Open(path)
	if err != nil {
		return "", fmt.Errorf("failed to open template %q: %w", path, err)
	}

	if _, err = io.Copy(&body, file); err != nil {
		return "", fmt.Errorf("failed to read template %q: %w", path, err)
	}

	e, err := t.Parse(body.String())
	if err != nil {
		return "", fmt.Errorf("failed to parse template %q: %w", path, err)
	}

	body.Reset()

	if err = e.Execute(&body, data); err != nil {
		return "", fmt.Errorf("failed to execute template %q: %w", path, err)
	}

	return body.String(), nil
}

type Mailer struct {
	config core.MailConfig
	fsys   http.FileSystem
}

func NewMailer(config core.MailConfig, fsys http.FileSystem) *Mailer {
	return &Mailer{config, fsys}
}

func (mailer *Mailer) Send(message Message) error {
	m := gomail.NewMessage()
	if message.From != "" {
		m.SetHeader("From", message.From)
	} else {
		m.SetHeader("From", mailer.config.NoReplyAddress)
	}
	m.SetHeader("To", message.To)
	m.SetHeader("Subject", message.Subject)

	body, err := loadTemplate(text.New("text"), mailer.fsys, message.TextTemplate, message.Data)
	if err != nil {
		return err
	}
	m.SetBody("text/plain", body)

	body, err = loadTemplate(html.New("html"), mailer.fsys, message.HtmlTemplate, message.Data)
	if err != nil {
		return err
	}
	m.AddAlternative("text/html", body)

	config := mailer.config
	return gomail.NewDialer(config.Host, config.Port, config.Username, config.Password).DialAndSend(m)
}
