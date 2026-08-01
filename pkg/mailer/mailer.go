// Package mailer menangani pengiriman email (verifikasi & reset password) via SMTP.
// Bila SMTP belum dikonfigurasi (pengembangan), tautan dicetak ke log alih-alih
// dikirim, agar alur tetap dapat diuji tanpa server email.
package mailer

import (
	"bytes"
	"html/template"
	"log"

	"fsldk-api/config"

	gomail "gopkg.in/gomail.v2"
)

// Mailer adalah kontrak layanan email.
type Mailer interface {
	SendVerificationEmail(toEmail, toName, verifyURL string) error
	SendPasswordResetEmail(toEmail, toName, resetURL string) error
}

type smtpMailer struct {
	cfg config.AppConfig
}

// New membuat Mailer berbasis SMTP.
func New(cfg config.AppConfig) Mailer {
	return &smtpMailer{cfg: cfg}
}

func (m *smtpMailer) SendVerificationEmail(toEmail, toName, verifyURL string) error {
	body, err := render(verificationTemplate, map[string]string{"Name": toName, "URL": verifyURL})
	if err != nil {
		return err
	}
	return m.send(toEmail, "Verifikasi Email Anda — FSLDK Indonesia", body, verifyURL)
}

func (m *smtpMailer) SendPasswordResetEmail(toEmail, toName, resetURL string) error {
	body, err := render(passwordResetTemplate, map[string]string{"Name": toName, "URL": resetURL})
	if err != nil {
		return err
	}
	return m.send(toEmail, "Atur Ulang Kata Sandi — FSLDK Indonesia", body, resetURL)
}

func (m *smtpMailer) send(to, subject, htmlBody, link string) error {
	// Mode pengembangan: SMTP belum dikonfigurasi → cetak tautan ke log.
	if m.cfg.SMTPHost == "" {
		log.Printf("[MAILER:DEV] Email ke %s | Subjek: %s | Tautan: %s", to, subject, link)
		return nil
	}

	msg := gomail.NewMessage()
	msg.SetHeader("From", msg.FormatAddress(m.cfg.MailFromAddress, m.cfg.MailFromName))
	msg.SetHeader("To", to)
	msg.SetHeader("Subject", subject)
	msg.SetBody("text/html", htmlBody)

	dialer := gomail.NewDialer(m.cfg.SMTPHost, m.cfg.SMTPPort, m.cfg.SMTPUsername, m.cfg.SMTPPassword)
	return dialer.DialAndSend(msg)
}

func render(tmpl string, data map[string]string) (string, error) {
	t, err := template.New("email").Parse(tmpl)
	if err != nil {
		return "", err
	}
	var buf bytes.Buffer
	if err := t.Execute(&buf, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}
