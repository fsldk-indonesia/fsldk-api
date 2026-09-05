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
	"io"
	"log"
	"os"
	"path/filepath"

	"fsldk-api/config"

	gomail "gopkg.in/gomail.v2"
)

const (
	assetsDir                 = "assets"
	logoAsset                 = "logo-fsldk.png"
	logoCID                   = "logo-fsldk.png"
	templateVerification      = "verification"
	templatePasswordReset     = "password_reset"
	templateShortlinkApproved = "shortlink_approved"
	templateShortlinkRejected = "shortlink_rejected"
	templateDonationReceipt   = "donation_receipt"
	templateDonationInvoice   = "donation_invoice"
	templateOtpWithdrawal     = "otp_kantong_amal"
	templateContactReply      = "contact_reply"
)

// Mailer adalah kontrak layanan email.
type Mailer interface {
	SendVerificationEmail(toEmail, toName, verifyURL string) error
	SendPasswordResetEmail(toEmail, toName, resetURL string) error
	SendShortlinkApprovedEmail(toEmail, toName, shortURL string) error
	SendShortlinkRejectedEmail(toEmail, toName, reason string) error
	// SendDonationReceipt mengirim konfirmasi donasi lunas (Kantong Amal) —
	// amount/total/date sudah diformat pemanggil (Rupiah/tanggal Indonesia),
	// bukan angka mentah, karena template menerima map[string]string.
	// pdfBytes/pdfFilename melampirkan PDF "Bukti Donasi" ke email ini
	// (revision-prompt-4.md item tambahan) — pdfBytes nil berarti tidak ada
	// lampiran (mis. pembuatan PDF gagal, email tetap dikirim tanpa lampiran
	// alih-alih menggagalkan seluruh notifikasi).
	SendDonationReceipt(toEmail, toName, campaignTitle, amount, total, dateStr, publicRef, receiptURL string, pdfBytes []byte, pdfFilename string) error
	// SendDonationInvoice dikirim segera setelah donasi dibuat (sebelum
	// dibayar) — email pertama dari dua email donasi (item 2
	// revision-prompt-2.md); SendDonationReceipt di atas adalah email kedua.
	SendDonationInvoice(toEmail, toName, campaignTitle, amount, qrURL, expiredDateStr string) error
	// SendOtpEmail mengirim kode OTP verifikasi keamanan withdrawal Kantong
	// Amal ke email yang dikonfigurasi di ms_setting (item 8
	// revision-prompt-2.md) — menggantikan pengiriman via WhatsApp.
	SendOtpEmail(toEmail, code, validityText string) error
	// SendContactReplyEmail mengirimkan balasan resmi atas pesan kontak masuk.
	SendContactReplyEmail(toEmail, toName, subject, replyBody, originalSubject, originalMessage string) error
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
	return m.send(toEmail, "Verifikasi Email Anda — FSLDK Indonesia", body, verifyURL, nil, "")
}

func (m *smtpMailer) SendPasswordResetEmail(toEmail, toName, resetURL string) error {
	body, err := generateFromAsset(templatePasswordReset, map[string]string{
		"Name": toName, "URL": resetURL, "LogoCID": logoCID,
	})
	if err != nil {
		return err
	}
	return m.send(toEmail, "Atur Ulang Kata Sandi — FSLDK Indonesia", body, resetURL, nil, "")
}

func (m *smtpMailer) SendShortlinkApprovedEmail(toEmail, toName, shortURL string) error {
	body, err := generateFromAsset(templateShortlinkApproved, map[string]string{
		"Name": toName, "URL": shortURL, "LogoCID": logoCID,
	})
	if err != nil {
		return err
	}
	return m.send(toEmail, "Permintaan Shortlink Disetujui — FSLDK Indonesia", body, shortURL, nil, "")
}

func (m *smtpMailer) SendShortlinkRejectedEmail(toEmail, toName, reason string) error {
	body, err := generateFromAsset(templateShortlinkRejected, map[string]string{
		"Name": toName, "Reason": reason, "LogoCID": logoCID,
	})
	if err != nil {
		return err
	}
	return m.send(toEmail, "Permintaan Shortlink Ditolak — FSLDK Indonesia", body, "", nil, "")
}

func (m *smtpMailer) SendDonationReceipt(toEmail, toName, campaignTitle, amount, total, dateStr, publicRef, receiptURL string, pdfBytes []byte, pdfFilename string) error {
	body, err := generateFromAsset(templateDonationReceipt, map[string]string{
		"Name": toName, "CampaignTitle": campaignTitle, "Amount": amount, "Total": total,
		"Date": dateStr, "PublicRef": publicRef, "URL": receiptURL, "LogoCID": logoCID,
	})
	if err != nil {
		return err
	}
	return m.send(toEmail, "Konfirmasi Donasi Diterima — FSLDK Indonesia", body, receiptURL, pdfBytes, pdfFilename)
}

func (m *smtpMailer) SendDonationInvoice(toEmail, toName, campaignTitle, amount, qrURL, expiredDateStr string) error {
	body, err := generateFromAsset(templateDonationInvoice, map[string]string{
		"Name": toName, "CampaignTitle": campaignTitle, "Amount": amount,
		"QrURL": qrURL, "ExpiredDate": expiredDateStr, "LogoCID": logoCID,
	})
	if err != nil {
		return err
	}
	return m.send(toEmail, "Konfirmasi Donasi — FSLDK Indonesia", body, qrURL, nil, "")
}

func (m *smtpMailer) SendContactReplyEmail(toEmail, toName, subject, replyBody, originalSubject, originalMessage string) error {
	body, err := generateFromAsset(templateContactReply, map[string]string{
		"Name":            toName,
		"OriginalSubject": originalSubject,
		"OriginalMessage": originalMessage,
		"ReplyBody":       replyBody,
		"LogoCID":         logoCID,
	})
	if err != nil {
		return err
	}
	return m.send(toEmail, subject, body, "", nil, "")
}

func (m *smtpMailer) SendOtpEmail(toEmail, code, validityText string) error {
	body, err := generateFromAsset(templateOtpWithdrawal, map[string]string{
		"Code": code, "ValidityText": validityText, "LogoCID": logoCID,
	})
	if err != nil {
		return err
	}
	return m.send(toEmail, "Kode OTP Penarikan Saldo Kantong Amal — FSLDK Indonesia", body, "", nil, "")
}

func (m *smtpMailer) send(to, subject, htmlBody, link string, attachData []byte, attachFilename string) error {
	// Mode pengembangan: SMTP belum dikonfigurasi → cetak tautan ke log.
	if m.cfg.SMTPHost == "" || m.cfg.SMTPUsername == "" {
		attachNote := ""
		if len(attachData) > 0 {
			attachNote = fmt.Sprintf(" | Lampiran: %s (%d byte)", attachFilename, len(attachData))
		}
		log.Printf("[MAILER:DEV] Email ke %s | Subjek: %s | Tautan: %s%s", to, subject, link, attachNote)
		return nil
	}

	msg := gomail.NewMessage()
	msg.SetHeader("From", msg.FormatAddress(m.cfg.MailFromAddress, m.cfg.MailFromName))
	msg.SetHeader("To", to)
	msg.SetHeader("Subject", subject)
	msg.SetBody("text/html", htmlBody)
	msg.Embed(filepath.Join(assetsDir, logoAsset))
	if len(attachData) > 0 {
		msg.Attach(attachFilename, gomail.SetCopyFunc(func(w io.Writer) error {
			_, err := w.Write(attachData)
			return err
		}))
	}

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
