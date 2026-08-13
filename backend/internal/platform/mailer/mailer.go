package mailer

import (
	"fmt"
	"log/slog"
)

type Mailer interface {
	Send(to, subject, body string) error
}

type LogMailer struct {
	From string
}

func (m LogMailer) Send(to, subject, body string) error {
	slog.Info("mailer.log", "from", m.From, "to", to, "subject", subject, "body_len", len(body))
	fmt.Printf("\n===== EMAIL (dev) =====\nTo: %s\nSubject: %s\n%s\n=======================\n\n", to, subject, body)
	return nil
}
