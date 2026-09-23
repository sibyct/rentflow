// Package mail adapts domain.Mailer to SMTP using go-mail.
package mail

import (
	"bytes"
	"context"
	"fmt"

	gomail "github.com/wneessen/go-mail"

	"propertymanagement/internal/domain"
)

type SMTPConfig struct {
	Host     string
	Port     int
	Username string
	Password string
	From     string
	// TLS is "starttls" (default; port 587), "tls" (implicit TLS; 465)
	// or "none" (plain — only for a local catcher such as Mailpit).
	TLS string
}

type SMTP struct {
	cfg SMTPConfig
}

var _ domain.Mailer = (*SMTP)(nil)

func NewSMTP(cfg SMTPConfig) *SMTP { return &SMTP{cfg: cfg} }

func (s *SMTP) client() (*gomail.Client, error) {
	opts := []gomail.Option{gomail.WithPort(s.cfg.Port)}
	switch s.cfg.TLS {
	case "none":
		opts = append(opts, gomail.WithTLSPolicy(gomail.NoTLS))
	case "tls":
		opts = append(opts, gomail.WithSSL())
	default:
		opts = append(opts, gomail.WithTLSPolicy(gomail.TLSMandatory))
	}
	if s.cfg.Username != "" {
		opts = append(opts,
			gomail.WithSMTPAuth(gomail.SMTPAuthAutoDiscover),
			gomail.WithUsername(s.cfg.Username),
			gomail.WithPassword(s.cfg.Password),
		)
	}
	c, err := gomail.NewClient(s.cfg.Host, opts...)
	if err != nil {
		return nil, fmt.Errorf("configure smtp client: %w", err)
	}
	return c, nil
}

func (s *SMTP) Send(ctx context.Context, e domain.Email) error {
	msg := gomail.NewMsg()
	if err := msg.From(s.cfg.From); err != nil {
		return fmt.Errorf("invalid from address %q: %w", s.cfg.From, err)
	}
	if err := msg.To(e.To); err != nil {
		return fmt.Errorf("invalid recipient %q: %w", e.To, err)
	}
	msg.Subject(e.Subject)
	msg.SetBodyString(gomail.TypeTextHTML, e.HTML)
	for _, a := range e.Attachments {
		if err := msg.AttachReader(a.Filename, bytes.NewReader(a.Data), gomail.WithFileContentType(gomail.ContentType(a.ContentType))); err != nil {
			return fmt.Errorf("attach %q: %w", a.Filename, err)
		}
	}

	c, err := s.client()
	if err != nil {
		return err
	}
	if err := c.DialAndSendWithContext(ctx, msg); err != nil {
		return fmt.Errorf("send email to %s: %w", e.To, err)
	}
	return nil
}
