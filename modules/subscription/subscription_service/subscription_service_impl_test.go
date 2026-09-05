package subscription_service_test

import (
	"context"
	"testing"
	"time"

	"fsldk-api/base/dto"
	"fsldk-api/modules/subscription/subscription_dto"
	"fsldk-api/modules/subscription/subscription_model"
	"fsldk-api/modules/subscription/subscription_repository"
	"fsldk-api/modules/subscription/subscription_service"
)

// fakeMailer implements mailer.Mailer as no-ops, tracking welcome/unsubscribe-email calls.
type fakeMailer struct {
	welcomeEmailsSent     int
	unsubscribeEmailsSent int
}

func (f *fakeMailer) SendVerificationEmail(toEmail, toName, verifyURL string) error     { return nil }
func (f *fakeMailer) SendPasswordResetEmail(toEmail, toName, resetURL string) error     { return nil }
func (f *fakeMailer) SendShortlinkApprovedEmail(toEmail, toName, shortURL string) error { return nil }
func (f *fakeMailer) SendShortlinkRejectedEmail(toEmail, toName, reason string) error   { return nil }
func (f *fakeMailer) SendDonationReceipt(toEmail, toName, campaignTitle, amount, total, dateStr, publicRef, receiptURL string, pdfBytes []byte, pdfFilename string) error {
	return nil
}
func (f *fakeMailer) SendDonationInvoice(toEmail, toName, campaignTitle, amount, qrURL, expiredDateStr string) error {
	return nil
}
func (f *fakeMailer) SendOtpEmail(toEmail, code, validityText string) error { return nil }
func (f *fakeMailer) SendSubscriptionWelcomeEmail(toEmail, unsubscribeURL string) error {
	f.welcomeEmailsSent++
	return nil
}
func (f *fakeMailer) SendUnsubscribeConfirmationEmail(toEmail string) error {
	f.unsubscribeEmailsSent++
	return nil
}

func newService(repo *mockRepository) (subscription_service.Service, *fakeMailer) {
	mail := &fakeMailer{}
	return subscription_service.NewService(repo, mail, "https://fsldk.test"), mail
}

type mockRepository struct {
	subs   []subscription_model.Subscriber
	nextID int64
}

func newMockRepository() *mockRepository {
	return &mockRepository{subs: make([]subscription_model.Subscriber, 0), nextID: 1}
}

func (m *mockRepository) FindByEmail(ctx context.Context, email string) (*subscription_model.Subscriber, error) {
	for i := range m.subs {
		if m.subs[i].Email == email {
			s := m.subs[i]
			return &s, nil
		}
	}
	return nil, subscription_repository.ErrNotFound
}

func (m *mockRepository) FindByID(ctx context.Context, id int64) (*subscription_model.Subscriber, error) {
	for i := range m.subs {
		if m.subs[i].SubscriberID == id {
			s := m.subs[i]
			return &s, nil
		}
	}
	return nil, subscription_repository.ErrNotFound
}

func (m *mockRepository) FindAll(ctx context.Context, q dto.ListQuery, isActive *bool, from, to string) ([]subscription_model.Subscriber, int, error) {
	return m.subs, len(m.subs), nil
}

func (m *mockRepository) Create(ctx context.Context, sub *subscription_model.Subscriber) error {
	sub.SubscriberID = m.nextID
	m.nextID++
	sub.CreatedDate = time.Now()
	m.subs = append(m.subs, *sub)
	return nil
}

func (m *mockRepository) Update(ctx context.Context, sub *subscription_model.Subscriber) error {
	for i := range m.subs {
		if m.subs[i].SubscriberID == sub.SubscriberID {
			m.subs[i] = *sub
			return nil
		}
	}
	return subscription_repository.ErrNotFound
}

func (m *mockRepository) EmailExistsExcluding(ctx context.Context, email string, excludeID int64) (bool, error) {
	for _, s := range m.subs {
		if s.Email == email && s.SubscriberID != excludeID {
			return true, nil
		}
	}
	return false, nil
}

func (m *mockRepository) Delete(ctx context.Context, id int64) error {
	for i := range m.subs {
		if m.subs[i].SubscriberID == id {
			m.subs = append(m.subs[:i], m.subs[i+1:]...)
			return nil
		}
	}
	return subscription_repository.ErrNotFound
}

func (m *mockRepository) DeleteMany(ctx context.Context, ids []int64) error {
	idSet := make(map[int64]bool, len(ids))
	for _, id := range ids {
		idSet[id] = true
	}
	filtered := m.subs[:0]
	for _, s := range m.subs {
		if !idSet[s.SubscriberID] {
			filtered = append(filtered, s)
		}
	}
	m.subs = filtered
	return nil
}

func TestSubscriptionService_Subscribe_New(t *testing.T) {
	repo := newMockRepository()
	svc, _ := newService(repo)

	isResubscribe, err := svc.Subscribe(context.Background(), "fulan@example.com")
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if isResubscribe {
		t.Errorf("expected isResubscribe false for a brand new email")
	}
	if len(repo.subs) != 1 || !repo.subs[0].IsActive {
		t.Fatalf("expected 1 active subscriber, got %+v", repo.subs)
	}
}

func TestSubscriptionService_Subscribe_AlreadyActive(t *testing.T) {
	repo := newMockRepository()
	svc, _ := newService(repo)

	_, _ = svc.Subscribe(context.Background(), "fulan@example.com")
	_, err := svc.Subscribe(context.Background(), "fulan@example.com")
	if err == nil {
		t.Fatalf("expected conflict error for already-active subscriber, got nil")
	}
}

func TestSubscriptionService_Subscribe_Reactivate(t *testing.T) {
	repo := newMockRepository()
	svc, _ := newService(repo)

	_, _ = svc.Subscribe(context.Background(), "fulan@example.com")
	_ = svc.Delete(context.Background(), 1) // simulate removal, re-add manually as inactive
	repo.subs = append(repo.subs, subscription_model.Subscriber{SubscriberID: 2, Email: "fulan@example.com", IsActive: false})

	isResubscribe, err := svc.Subscribe(context.Background(), "fulan@example.com")
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if !isResubscribe {
		t.Errorf("expected isResubscribe true when reactivating an inactive subscriber")
	}
	if !repo.subs[0].IsActive {
		t.Errorf("expected subscriber to be reactivated")
	}
}

func TestSubscriptionService_BulkAdd(t *testing.T) {
	repo := newMockRepository()
	svc, _ := newService(repo)

	_, _ = svc.Subscribe(context.Background(), "existing@example.com")

	raw := "new1@example.com\nnew2@example.com, not-an-email\nexisting@example.com\nnew1@example.com"
	result, err := svc.BulkAdd(context.Background(), raw)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if result.Added != 2 {
		t.Errorf("expected 2 added (new1, new2; duplicate new1 ignored), got %d", result.Added)
	}
	if len(result.Skipped) != 1 || result.Skipped[0] != "existing@example.com" {
		t.Errorf("expected existing@example.com to be skipped, got %+v", result.Skipped)
	}
	if len(result.Invalid) != 1 || result.Invalid[0] != "not-an-email" {
		t.Errorf("expected not-an-email to be invalid, got %+v", result.Invalid)
	}
}

func TestSubscriptionService_Update_EmailConflict(t *testing.T) {
	repo := newMockRepository()
	svc, _ := newService(repo)

	_, _ = svc.Subscribe(context.Background(), "one@example.com")
	_, _ = svc.Subscribe(context.Background(), "two@example.com")

	_, err := svc.Update(context.Background(), 2, subscription_dto.UpdateSubscriberRequest{Email: "one@example.com", IsActive: true})
	if err == nil {
		t.Fatalf("expected conflict error when updating to an email used by another subscriber, got nil")
	}
}

func TestSubscriptionService_Update_Deactivate(t *testing.T) {
	repo := newMockRepository()
	svc, _ := newService(repo)

	_, _ = svc.Subscribe(context.Background(), "one@example.com")

	resp, err := svc.Update(context.Background(), 1, subscription_dto.UpdateSubscriberRequest{Email: "one@example.com", IsActive: false})
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if resp.IsActive {
		t.Errorf("expected subscriber to be inactive after update")
	}
	if resp.UnsubscribedDate == nil {
		t.Errorf("expected unsubscribedDate to be set after deactivation")
	}
}

func TestSubscriptionService_Subscribe_SendsWelcomeEmail(t *testing.T) {
	repo := newMockRepository()
	svc, mail := newService(repo)

	_, _ = svc.Subscribe(context.Background(), "fulan@example.com")
	if mail.welcomeEmailsSent != 1 {
		t.Fatalf("expected 1 welcome email sent, got %d", mail.welcomeEmailsSent)
	}
	if repo.subs[0].UnsubscribeToken == "" {
		t.Errorf("expected an unsubscribeToken to be generated")
	}
}

func TestSubscriptionService_Unsubscribe_Success(t *testing.T) {
	repo := newMockRepository()
	svc, mail := newService(repo)

	_, _ = svc.Subscribe(context.Background(), "fulan@example.com")
	token := repo.subs[0].UnsubscribeToken

	err := svc.Unsubscribe(context.Background(), "fulan@example.com", token)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if repo.subs[0].IsActive {
		t.Errorf("expected subscriber to be inactive after unsubscribe")
	}
	if repo.subs[0].UnsubscribedDate == nil {
		t.Errorf("expected unsubscribedDate to be set")
	}
	if mail.unsubscribeEmailsSent != 1 {
		t.Errorf("expected 1 unsubscribe confirmation email sent, got %d", mail.unsubscribeEmailsSent)
	}
}

func TestSubscriptionService_Unsubscribe_WrongToken(t *testing.T) {
	repo := newMockRepository()
	svc, _ := newService(repo)

	_, _ = svc.Subscribe(context.Background(), "fulan@example.com")

	err := svc.Unsubscribe(context.Background(), "fulan@example.com", "wrong-token")
	if err == nil {
		t.Fatalf("expected error for mismatched token, got nil")
	}
	if !repo.subs[0].IsActive {
		t.Errorf("subscriber should remain active when token is invalid")
	}
}

func TestSubscriptionService_Unsubscribe_UnknownEmail(t *testing.T) {
	repo := newMockRepository()
	svc, _ := newService(repo)

	err := svc.Unsubscribe(context.Background(), "unknown@example.com", "any-token")
	if err == nil {
		t.Fatalf("expected not-found error for unknown email, got nil")
	}
}
