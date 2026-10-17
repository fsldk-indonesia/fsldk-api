package auth_service

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"fsldk-api/config"
	"fsldk-api/modules/auth/auth_model"
	"fsldk-api/pkg/mailer"
)

// fakeTokenStore adalah implementasi auth_repository.TokenStore in-memory —
// hanya mencatat panggilan Create()/InvalidateAll(), cukup untuk menguji
// business logic SendVerificationEmail tanpa DB sungguhan.
type fakeTokenStore struct {
	createCalls     []tokenCreateCall
	invalidateCalls []string
}

type tokenCreateCall struct {
	userID    int64
	token     string
	tokenType string
	expiresAt time.Time
}

func (f *fakeTokenStore) Create(ctx context.Context, userID int64, token, tokenType string, expiresAt time.Time) error {
	f.createCalls = append(f.createCalls, tokenCreateCall{userID, token, tokenType, expiresAt})
	return nil
}
func (f *fakeTokenStore) FindValid(ctx context.Context, token, tokenType string) (auth_model.EmailToken, error) {
	return auth_model.EmailToken{}, errors.New("not found")
}
func (f *fakeTokenStore) MarkUsed(ctx context.Context, tokenID int64) error { return nil }
func (f *fakeTokenStore) InvalidateAll(ctx context.Context, userID int64, tokenType string) error {
	f.invalidateCalls = append(f.invalidateCalls, tokenType)
	return nil
}

// fakeMailer adalah implementasi mailer.Mailer no-op — hanya
// SendVerificationEmail yang berperilaku bermakna (dicatat), sisanya no-op
// karena tidak relevan untuk skenario uji ini.
type fakeMailer struct {
	verificationCalls []verificationCall
}

type verificationCall struct {
	toEmail, toName, verifyURL string
}

func (f *fakeMailer) SendVerificationEmail(toEmail, toName, verifyURL string) error {
	f.verificationCalls = append(f.verificationCalls, verificationCall{toEmail, toName, verifyURL})
	return nil
}
func (f *fakeMailer) SendPasswordResetEmail(toEmail, toName, resetURL string) error { return nil }
func (f *fakeMailer) SendShortlinkApprovedEmail(toEmail, toName, shortURL string) error {
	return nil
}
func (f *fakeMailer) SendShortlinkRejectedEmail(toEmail, toName, reason string) error { return nil }
func (f *fakeMailer) SendQRCodeApprovedEmail(toEmail, toName, pageURL, imageURL string) error {
	return nil
}
func (f *fakeMailer) SendQRCodeRejectedEmail(toEmail, toName, reason string) error { return nil }
func (f *fakeMailer) SendFormSubmissionConfirmation(toEmail, toName, formTitle string, answers []mailer.AnswerPair, submittedAt string) error {
	return nil
}
func (f *fakeMailer) SendDonationReceipt(toEmail, toName, campaignTitle, amount, total, dateStr, publicRef, receiptURL string, pdfBytes []byte, pdfFilename string) error {
	return nil
}
func (f *fakeMailer) SendDonationInvoice(toEmail, toName, campaignTitle, amount, qrURL, expiredDateStr string) error {
	return nil
}
func (f *fakeMailer) SendOtpEmail(toEmail, code, validityText string) error { return nil }
func (f *fakeMailer) SendContactReplyEmail(toEmail, toName, subject, replyBody, originalSubject, originalMessage string) error {
	return nil
}
func (f *fakeMailer) SendSubscriptionWelcomeEmail(toEmail, unsubscribeURL string) error { return nil }
func (f *fakeMailer) SendUnsubscribeConfirmationEmail(toEmail string) error             { return nil }

func TestServiceImpl_SendVerificationEmail(t *testing.T) {
	store := &fakeTokenStore{}
	mail := &fakeMailer{}
	cfg := config.AppConfig{FrontendURL: "http://localhost:4200", EmailVerificationExpireMinutes: 60}
	svc := NewService(nil, nil, nil, nil, nil, store, mail, nil, cfg)

	if err := svc.SendVerificationEmail(context.Background(), 42, "Budi", "budi@example.com"); err != nil {
		t.Fatalf("SendVerificationEmail() error = %v", err)
	}

	if len(store.createCalls) != 1 {
		t.Fatalf("expected 1 token store Create() call, got %d", len(store.createCalls))
	}
	if store.createCalls[0].userID != 42 {
		t.Errorf("Create() userID = %d, want 42", store.createCalls[0].userID)
	}
	if len(store.invalidateCalls) != 1 {
		t.Fatalf("expected 1 InvalidateAll() call (invalidate stale tokens first), got %d", len(store.invalidateCalls))
	}

	if len(mail.verificationCalls) != 1 {
		t.Fatalf("expected 1 SendVerificationEmail() call, got %d", len(mail.verificationCalls))
	}
	got := mail.verificationCalls[0]
	if got.toEmail != "budi@example.com" || got.toName != "Budi" {
		t.Errorf("SendVerificationEmail() toEmail/toName = %q/%q, want budi@example.com/Budi", got.toEmail, got.toName)
	}
	wantPrefix := "http://localhost:4200/verifikasi-email?token="
	if !strings.HasPrefix(got.verifyURL, wantPrefix) {
		t.Errorf("verifyURL = %q, want prefix %q", got.verifyURL, wantPrefix)
	}
}
