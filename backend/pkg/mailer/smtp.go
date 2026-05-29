package mailer

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"mime/multipart"
	"mime/quotedprintable"
	"net/smtp"
	"net/textproto"

	"project/pkg/config"
)

// Mailer sends transactional emails.
type Mailer interface {
	SendOTP(ctx context.Context, to, subject, code string) error
	SendPasswordReset(ctx context.Context, to, link string) error
	SendInvite(ctx context.Context, to, inviteLink, vendorName string) error
}

type smtpMailer struct {
	cfg config.SMTPConfig
}

// NewSMTPMailer builds the mailer. Does NOT ping SMTP at init — just validates config presence.
func NewSMTPMailer(cfg config.SMTPConfig) Mailer {
	if cfg.Host == "" {
		slog.Warn("SMTP host not configured — emails will fail at send time")
	}
	return &smtpMailer{cfg: cfg}
}

func (m *smtpMailer) SendOTP(ctx context.Context, to, subject, code string) error {
	html := fmt.Sprintf(`<p>Your verification code is: <strong>%s</strong></p><p>Valid for 5 minutes.</p>`, code)
	text := fmt.Sprintf("Your verification code is: %s\nValid for 5 minutes.", code)
	return m.send(ctx, to, subject, html, text)
}

func (m *smtpMailer) SendPasswordReset(ctx context.Context, to, link string) error {
	html := fmt.Sprintf(`<p>Click the link below to reset your password:</p><p><a href="%s">Reset Password</a></p><p>Link expires in 30 minutes.</p>`, link)
	text := fmt.Sprintf("Reset your password: %s\nLink expires in 30 minutes.", link)
	return m.send(ctx, to, "Password Reset Request", html, text)
}

func (m *smtpMailer) SendInvite(ctx context.Context, to, inviteLink, vendorName string) error {
	html := fmt.Sprintf(`<p>You have been invited to join <strong>%s</strong>.</p><p><a href="%s">Accept Invitation</a></p>`, vendorName, inviteLink)
	text := fmt.Sprintf("You are invited to join %s.\nAccept: %s", vendorName, inviteLink)
	return m.send(ctx, to, fmt.Sprintf("Invitation to join %s", vendorName), html, text)
}

func (m *smtpMailer) send(ctx context.Context, to, subject, htmlBody, textBody string) error {
	addr := fmt.Sprintf("%s:%d", m.cfg.Host, m.cfg.Port)
	auth := smtp.PlainAuth("", m.cfg.Username, m.cfg.Password, m.cfg.Host)

	msg, err := buildMIME(m.cfg.From, to, subject, htmlBody, textBody)
	if err != nil {
		return fmt.Errorf("build MIME: %w", err)
	}

	if err := smtp.SendMail(addr, auth, m.cfg.From, []string{to}, msg); err != nil {
		slog.WarnContext(ctx, "smtp send failed", "to", to, "err", err)
		return fmt.Errorf("smtp send: %w", err)
	}
	slog.InfoContext(ctx, "email sent", "to", to, "subject", subject)
	return nil
}

// buildMIME builds a multipart/alternative message (text + HTML).
func buildMIME(from, to, subject, htmlBody, textBody string) ([]byte, error) {
	var buf bytes.Buffer
	// headers
	buf.WriteString("From: " + from + "\r\n")
	buf.WriteString("To: " + to + "\r\n")
	buf.WriteString("Subject: " + subject + "\r\n")
	buf.WriteString("MIME-Version: 1.0\r\n")

	mw := multipart.NewWriter(&buf)
	buf.WriteString("Content-Type: multipart/alternative; boundary=" + mw.Boundary() + "\r\n\r\n")

	// plain text part
	th := textproto.MIMEHeader{}
	th.Set("Content-Type", "text/plain; charset=UTF-8")
	th.Set("Content-Transfer-Encoding", "quoted-printable")
	tw, err := mw.CreatePart(th)
	if err != nil {
		return nil, err
	}
	qpw := quotedprintable.NewWriter(tw)
	if _, err := qpw.Write([]byte(textBody)); err != nil {
		return nil, err
	}
	_ = qpw.Close()

	// HTML part
	hh := textproto.MIMEHeader{}
	hh.Set("Content-Type", "text/html; charset=UTF-8")
	hh.Set("Content-Transfer-Encoding", "quoted-printable")
	hw, err := mw.CreatePart(hh)
	if err != nil {
		return nil, err
	}
	qph := quotedprintable.NewWriter(hw)
	if _, err := qph.Write([]byte(htmlBody)); err != nil {
		return nil, err
	}
	_ = qph.Close()

	_ = mw.Close()
	return buf.Bytes(), nil
}
