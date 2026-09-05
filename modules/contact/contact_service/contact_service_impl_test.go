package contact_service_test

import (
	"context"
	"testing"
	"time"

	"fsldk-api/modules/contact/contact_dto"
	"fsldk-api/modules/contact/contact_model"
	"fsldk-api/modules/contact/contact_repository"
	"fsldk-api/modules/contact/contact_service"
)

type mockRepository struct {
	messages []contact_model.ContactMessage
	nextID   int64
}

func newMockRepository() *mockRepository {
	return &mockRepository{
		messages: make([]contact_model.ContactMessage, 0),
		nextID:   1,
	}
}

func (m *mockRepository) Create(ctx context.Context, msg *contact_model.ContactMessage) error {
	msg.MessageID = m.nextID
	m.nextID++
	msg.CreatedDate = time.Now()
	m.messages = append(m.messages, *msg)
	return nil
}

func (m *mockRepository) FindByID(ctx context.Context, id int64) (*contact_model.ContactMessage, error) {
	for i := range m.messages {
		if m.messages[i].MessageID == id {
			c := m.messages[i]
			return &c, nil
		}
	}
	return nil, contact_repository.ErrNotFound
}

func (m *mockRepository) FindAll(ctx context.Context, q contact_dto.ContactListQuery) ([]contact_model.ContactMessage, int64, error) {
	var result []contact_model.ContactMessage
	for _, msg := range m.messages {
		if q.IsRead != nil && msg.IsRead != *q.IsRead {
			continue
		}
		result = append(result, msg)
	}
	return result, int64(len(result)), nil
}

func (m *mockRepository) MarkAsRead(ctx context.Context, id int64) error {
	for i := range m.messages {
		if m.messages[i].MessageID == id {
			m.messages[i].IsRead = true
			return nil
		}
	}
	return contact_repository.ErrNotFound
}

func (m *mockRepository) Delete(ctx context.Context, id int64) error {
	for i := range m.messages {
		if m.messages[i].MessageID == id {
			m.messages = append(m.messages[:i], m.messages[i+1:]...)
			return nil
		}
	}
	return contact_repository.ErrNotFound
}

func (m *mockRepository) CountUnread(ctx context.Context) (int, error) {
	count := 0
	for _, msg := range m.messages {
		if !msg.IsRead {
			count++
		}
	}
	return count, nil
}

func TestContactService_Send(t *testing.T) {
	repo := newMockRepository()
	svc := contact_service.NewService(repo, nil)

	req := contact_dto.SendContactRequest{
		SenderName: "Fulan",
		Email:      "fulan@example.com",
		Subject:    "Tanya Muktamar",
		Message:    "Assalamu'alaikum, info muktamar kapan ya?",
	}

	err := svc.Send(context.Background(), req, "127.0.0.1")
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	if len(repo.messages) != 1 {
		t.Fatalf("expected 1 message in repo, got %d", len(repo.messages))
	}
	if repo.messages[0].SenderName != "Fulan" {
		t.Errorf("expected SenderName Fulan, got %s", repo.messages[0].SenderName)
	}
	if repo.messages[0].IsRead {
		t.Errorf("expected isRead false initially")
	}
}

func TestContactService_GetByID_AutoMarkRead(t *testing.T) {
	repo := newMockRepository()
	svc := contact_service.NewService(repo, nil)

	req := contact_dto.SendContactRequest{
		SenderName: "Ahmad",
		Email:      "ahmad@example.com",
		Subject:    "Pertanyaan",
		Message:    "Halo admin, ini pesan tes.",
	}
	_ = svc.Send(context.Background(), req, "192.168.1.1")

	detail, err := svc.GetByID(context.Background(), 1)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if !detail.IsRead {
		t.Errorf("expected detail.IsRead to be true after viewing")
	}

	unread, _ := svc.CountUnread(context.Background())
	if unread != 0 {
		t.Errorf("expected 0 unread messages, got %d", unread)
	}
}

func TestContactService_Delete(t *testing.T) {
	repo := newMockRepository()
	svc := contact_service.NewService(repo, nil)

	req := contact_dto.SendContactRequest{
		SenderName: "Budi",
		Email:      "budi@example.com",
		Subject:    "Spam",
		Message:    "Pesan yang akan dihapus.",
	}
	_ = svc.Send(context.Background(), req, "10.0.0.1")

	err := svc.Delete(context.Background(), 1)
	if err != nil {
		t.Fatalf("expected nil error on delete, got %v", err)
	}

	_, err = svc.GetByID(context.Background(), 1)
	if err == nil {
		t.Fatalf("expected error finding deleted message, got nil")
	}
}

func TestContactService_Reply(t *testing.T) {
	repo := newMockRepository()
	svc := contact_service.NewService(repo, nil)

	req := contact_dto.SendContactRequest{
		SenderName: "Fulan",
		Email:      "fulan@example.com",
		Subject:    "Tanya Program",
		Message:    "Bisa minta proposal kegiatan?",
	}
	_ = svc.Send(context.Background(), req, "127.0.0.1")

	replyReq := contact_dto.ReplyContactRequest{
		Subject: "Re: Tanya Program",
		Message: "Assalamu'alaikum, berikut kami lampirkan informasinya.",
	}
	err := svc.Reply(context.Background(), 1, replyReq)
	if err != nil {
		t.Fatalf("expected nil error on reply, got %v", err)
	}

	detail, _ := svc.GetByID(context.Background(), 1)
	if !detail.IsRead {
		t.Errorf("expected isRead true after reply")
	}
}

