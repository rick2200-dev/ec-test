package email

import (
	"context"
	"fmt"
	"log/slog"
	"net/smtp"
)

// Sender is the interface for sending emails.
type Sender interface {
	Send(ctx context.Context, to, subject, body string) error
}

// LogSender is a development/MVP implementation that logs emails instead of sending them.
type LogSender struct{}

// NewLogSender creates a new LogSender.
func NewLogSender() *LogSender {
	return &LogSender{}
}

// Send logs the email details instead of actually sending.
func (s *LogSender) Send(ctx context.Context, to, subject, body string) error {
	slog.InfoContext(ctx, "email notification",
		"to", to,
		"subject", subject,
		"body_length", len(body),
	)
	return nil
}

// SMTPSender sends emails via SMTP. Placeholder for future implementation.
type SMTPSender struct {
	host string
	port string
	from string
}

// NewSMTPSender creates a new SMTPSender.
func NewSMTPSender(host, port, from string) *SMTPSender {
	return &SMTPSender{
		host: host,
		port: port,
		from: from,
	}
}

// Send sends an email via SMTP.
func (s *SMTPSender) Send(ctx context.Context, to, subject, body string) error {
	addr := s.host + ":" + s.port

	msg := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\nMIME-Version: 1.0\r\nContent-Type: text/html; charset=UTF-8\r\n\r\n%s",
		s.from, to, subject, body)

	if err := smtp.SendMail(addr, nil, s.from, []string{to}, []byte(msg)); err != nil {
		return fmt.Errorf("smtp send: %w", err)
	}
	return nil
}
