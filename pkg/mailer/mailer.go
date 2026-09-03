// Package mailer menangani pengiriman email (verifikasi & reset password) via SMTP.
// Bila SMTP belum dikonfigurasi (pengembangan), tautan dicetak ke log alih-alih
// dikirim, agar alur tetap dapat diuji tanpa server email.
//
// Template email berupa berkas .html terpisah pada folder assets/email_template/,
// dimuat saat runtime via os.ReadFile. Logo FSLDK pada assets/logo-fsldk.png
// disematkan sebagai lampiran inline (Content-ID) agar selalu tampil tanpa
// bergantung pada URL eksternal.
package mailer

import (
	"bytes"
	"fmt"
	"html/template"
	"log"
	"os"
	"path/filepath"

	"fsldk-api/config"

	gomail "gopkg.in/gomail.v2"
)

const (
	assetsDir                  = "assets"
	logoAsset                  = "logo-fsldk.png"
	logoCID                    = "logo-fsldk.png"
	templateVerification       = "verification"
	templatePasswordReset      = "password_reset"
	templateShortlinkApproved  = "shortlink_approved"
	templateShortlinkRejected  = "shortlink_rejected"
	templateFormSubmissionConf = "form_submission_confirmation"
)

// AnswerPair is one label/value row shown in the form-submission confirmation email.
type AnswerPair struct {
	Label string
	Value string
}

// Mailer adalah kontrak layanan email.
type Mailer interface {
	SendVerificationEmail(toEmail, toName, verifyURL string) error
	SendPasswordResetEmail(toEmail, toName, resetURL string) error
	SendShortlinkApprovedEmail(toEmail, toName, shortURL string) error
	SendShortlinkRejectedEmail(toEmail, toName, reason string) error
	// SendFormSubmissionConfirmation confirms a dynamicform submission to the
	// respondent. Best-effort: callers log failures and never fail the submit.
	SendFormSubmissionConfirmation(toEmail, toName, formTitle string, answers []AnswerPair, submittedAt string) error
}

type smtpMailer struct {
	cfg config.AppConfig
}

// New membuat Mailer berbasis SMTP.
func New(cfg config.AppConfig) Mailer {
	return &smtpMailer{cfg: cfg}
}

func (m *smtpMailer) SendVerificationEmail(toEmail, toName, verifyURL string) error {
	body, err := generateFromAsset(templateVerification, map[string]string{
		"Name": toName, "URL": verifyURL, "LogoCID": logoCID,
	})
	if err != nil {
		return err
	}
	return m.send(toEmail, "Verifikasi Email Anda — FSLDK Indonesia", body, verifyURL)
}

func (m *smtpMailer) SendPasswordResetEmail(toEmail, toName, resetURL string) error {
	body, err := generateFromAsset(templatePasswordReset, map[string]string{
		"Name": toName, "URL": resetURL, "LogoCID": logoCID,
	})
	if err != nil {
		return err
	}
	return m.send(toEmail, "Atur Ulang Kata Sandi — FSLDK Indonesia", body, resetURL)
}

func (m *smtpMailer) SendShortlinkApprovedEmail(toEmail, toName, shortURL string) error {
	body, err := generateFromAsset(templateShortlinkApproved, map[string]string{
		"Name": toName, "URL": shortURL, "LogoCID": logoCID,
	})
	if err != nil {
		return err
	}
	return m.send(toEmail, "Permintaan Shortlink Disetujui — FSLDK Indonesia", body, shortURL)
}

func (m *smtpMailer) SendShortlinkRejectedEmail(toEmail, toName, reason string) error {
	body, err := generateFromAsset(templateShortlinkRejected, map[string]string{
		"Name": toName, "Reason": reason, "LogoCID": logoCID,
	})
	if err != nil {
		return err
	}
	return m.send(toEmail, "Permintaan Shortlink Ditolak — FSLDK Indonesia", body, "")
}

func (m *smtpMailer) SendFormSubmissionConfirmation(toEmail, toName, formTitle string, answers []AnswerPair, submittedAt string) error {
	body, err := generateFromAssetData(templateFormSubmissionConf, map[string]any{
		"Name": toName, "FormTitle": formTitle, "Answers": answers,
		"SubmittedAt": submittedAt, "LogoCID": logoCID,
	})
	if err != nil {
		return err
	}
	return m.send(toEmail, "Konfirmasi Pengisian Formulir: "+formTitle+" — FSLDK Indonesia", body, "")
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
	msg.Embed(filepath.Join(assetsDir, logoAsset))

	dialer := gomail.NewDialer(m.cfg.SMTPHost, m.cfg.SMTPPort, m.cfg.SMTPUsername, m.cfg.SMTPPassword)
	return dialer.DialAndSend(msg)
}

// generateFromAsset memuat berkas template .html dari assets/email_template/
// dan merender-nya dengan data yang diberikan.
func generateFromAsset(assetName string, data map[string]string) (string, error) {
	path := filepath.Join(assetsDir, "email_template", assetName+".html")
	templateData, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("gagal membaca template email %s: %w", path, err)
	}

	t, err := template.New("emailTemplate").Parse(string(templateData))
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	if err := t.Execute(&buf, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}

// generateFromAssetData is generateFromAsset for templates that need richer
// data than map[string]string (e.g. a range over answer rows).
func generateFromAssetData(assetName string, data any) (string, error) {
	path := filepath.Join(assetsDir, "email_template", assetName+".html")
	templateData, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("gagal membaca template email %s: %w", path, err)
	}
	t, err := template.New("emailTemplate").Parse(string(templateData))
	if err != nil {
		return "", err
	}
	var buf bytes.Buffer
	if err := t.Execute(&buf, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}
