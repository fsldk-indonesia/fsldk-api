package contact_service

import (
	"context"
	"errors"
	"strings"

	"fsldk-api/base/apperror"
	"fsldk-api/modules/contact/contact_dto"
	"fsldk-api/modules/contact/contact_model"
	"fsldk-api/modules/contact/contact_repository"
	"fsldk-api/pkg/mailer"
)

type serviceImpl struct {
	repo contact_repository.Repository
	mail mailer.Mailer
}

// NewService creates a new instance of contact Service.
func NewService(repo contact_repository.Repository, mail mailer.Mailer) Service {
	return &serviceImpl{repo: repo, mail: mail}
}

func (s *serviceImpl) Send(ctx context.Context, req contact_dto.SendContactRequest, ip string) error {
	var ipPtr *string
	if ip = strings.TrimSpace(ip); ip != "" {
		ipPtr = &ip
	}

	msg := &contact_model.ContactMessage{
		SenderName: strings.TrimSpace(req.SenderName),
		Email:      strings.TrimSpace(req.Email),
		Subject:    strings.TrimSpace(req.Subject),
		Message:    strings.TrimSpace(req.Message),
		IPAddress:  ipPtr,
		IsRead:     false,
	}

	return s.repo.Create(ctx, msg)
}

func (s *serviceImpl) GetByID(ctx context.Context, id int64) (*contact_dto.ContactDetail, error) {
	msg, err := s.repo.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, contact_repository.ErrNotFound) {
			return nil, apperror.NotFound("Pesan kontak tidak ditemukan")
		}
		return nil, err
	}

	// Auto-mark as read upon viewing if currently unread.
	if !msg.IsRead {
		_ = s.repo.MarkAsRead(ctx, id)
		msg.IsRead = true
	}

	return &contact_dto.ContactDetail{
		MessageID:   msg.MessageID,
		SenderName:  msg.SenderName,
		Email:       msg.Email,
		Subject:     msg.Subject,
		Message:     msg.Message,
		IPAddress:   msg.IPAddress,
		IsRead:      msg.IsRead,
		CreatedDate: msg.CreatedDate,
	}, nil
}

func (s *serviceImpl) List(ctx context.Context, q contact_dto.ContactListQuery) (*contact_dto.ContactListResponse, error) {
	messages, total, err := s.repo.FindAll(ctx, q)
	if err != nil {
		return nil, err
	}

	items := make([]contact_dto.ContactListItem, 0, len(messages))
	for _, m := range messages {
		items = append(items, contact_dto.ContactListItem{
			MessageID:   m.MessageID,
			SenderName:  m.SenderName,
			Email:       m.Email,
			Subject:     m.Subject,
			IsRead:      m.IsRead,
			CreatedDate: m.CreatedDate,
		})
	}

	page := q.Page
	if page < 1 {
		page = 1
	}
	limit := q.Limit
	if limit < 1 || limit > 100 {
		limit = 15
	}

	return &contact_dto.ContactListResponse{
		Data:  items,
		Page:  page,
		Limit: limit,
		Total: total,
	}, nil
}

func (s *serviceImpl) MarkRead(ctx context.Context, id int64) error {
	err := s.repo.MarkAsRead(ctx, id)
	if err != nil {
		if errors.Is(err, contact_repository.ErrNotFound) {
			return apperror.NotFound("Pesan kontak tidak ditemukan")
		}
		return err
	}
	return nil
}

func (s *serviceImpl) Delete(ctx context.Context, id int64) error {
	err := s.repo.Delete(ctx, id)
	if err != nil {
		if errors.Is(err, contact_repository.ErrNotFound) {
			return apperror.NotFound("Pesan kontak tidak ditemukan")
		}
		return err
	}
	return nil
}

func (s *serviceImpl) CountUnread(ctx context.Context) (int, error) {
	return s.repo.CountUnread(ctx)
}

func (s *serviceImpl) Reply(ctx context.Context, id int64, req contact_dto.ReplyContactRequest) error {
	msg, err := s.repo.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, contact_repository.ErrNotFound) {
			return apperror.NotFound("Pesan kontak tidak ditemukan")
		}
		return err
	}

	subject := strings.TrimSpace(req.Subject)
	if subject == "" {
		subject = "Re: " + msg.Subject
	}
	body := strings.TrimSpace(req.Message)

	if s.mail != nil {
		if err := s.mail.SendContactReplyEmail(msg.Email, msg.SenderName, subject, body, msg.Subject, msg.Message); err != nil {
			return err
		}
	}

	if !msg.IsRead {
		_ = s.repo.MarkAsRead(ctx, id)
	}

	return nil
}

