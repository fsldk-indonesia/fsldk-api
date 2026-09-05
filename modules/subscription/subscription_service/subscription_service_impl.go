package subscription_service

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/mail"
	"net/url"
	"strings"
	"time"

	"fsldk-api/base/apperror"
	"fsldk-api/base/dto"
	"fsldk-api/base/idgen"
	"fsldk-api/modules/subscription/subscription_dto"
	"fsldk-api/modules/subscription/subscription_model"
	"fsldk-api/modules/subscription/subscription_repository"
	"fsldk-api/pkg/mailer"
)

type serviceImpl struct {
	repo        subscription_repository.Repository
	mail        mailer.Mailer
	frontendURL string
}

// NewService creates a new instance of subscription Service.
func NewService(repo subscription_repository.Repository, mail mailer.Mailer, frontendURL string) Service {
	return &serviceImpl{repo: repo, mail: mail, frontendURL: frontendURL}
}

func (s *serviceImpl) unsubscribeURL(email, token string) string {
	return fmt.Sprintf("%s/unsubscribe?email=%s&token=%s", s.frontendURL, url.QueryEscape(email), url.QueryEscape(token))
}

// sendWelcomeEmail is best-effort: a flaky SMTP relay should never fail a
// subscribe request, so errors are logged, not returned (same pattern as
// SendPasswordResetEmail in auth_service).
func (s *serviceImpl) sendWelcomeEmail(email, token string) {
	if err := s.mail.SendSubscriptionWelcomeEmail(email, s.unsubscribeURL(email, token)); err != nil {
		log.Printf("[SUBSCRIPTION] gagal kirim email selamat datang ke %s: %v", email, err)
	}
}

// sendUnsubscribeConfirmation is best-effort, same reasoning as sendWelcomeEmail.
func (s *serviceImpl) sendUnsubscribeConfirmation(email string) {
	if err := s.mail.SendUnsubscribeConfirmationEmail(email); err != nil {
		log.Printf("[SUBSCRIPTION] gagal kirim email konfirmasi berhenti berlangganan ke %s: %v", email, err)
	}
}

func (s *serviceImpl) Subscribe(ctx context.Context, email string) (bool, error) {
	email = strings.TrimSpace(email)

	existing, err := s.repo.FindByEmail(ctx, email)
	if err != nil && !errors.Is(err, subscription_repository.ErrNotFound) {
		return false, err
	}

	if existing != nil {
		if existing.IsActive {
			return false, apperror.Conflict("Email ini sudah terdaftar sebagai pelanggan aktif kami")
		}
		now := time.Now()
		existing.IsActive = true
		existing.SubscribedDate = now
		existing.UnsubscribedDate = nil
		existing.UnsubscribeToken = idgen.NewUUIDv4()
		if err := s.repo.Update(ctx, existing); err != nil {
			return false, err
		}
		s.sendWelcomeEmail(existing.Email, existing.UnsubscribeToken)
		return true, nil
	}

	token := idgen.NewUUIDv4()
	sub := &subscription_model.Subscriber{
		Email:            email,
		IsActive:         true,
		SubscribedDate:   time.Now(),
		UnsubscribeToken: token,
	}
	if err := s.repo.Create(ctx, sub); err != nil {
		return false, err
	}
	s.sendWelcomeEmail(email, token)
	return false, nil
}

func (s *serviceImpl) Unsubscribe(ctx context.Context, email, token string) error {
	sub, err := s.repo.FindByEmail(ctx, strings.TrimSpace(email))
	if err != nil {
		if errors.Is(err, subscription_repository.ErrNotFound) {
			return apperror.NotFound("Email tidak ditemukan di daftar pelanggan kami")
		}
		return err
	}
	if !sub.IsActive {
		return apperror.Conflict("Email ini sudah tidak berlangganan")
	}
	if sub.UnsubscribeToken == "" || sub.UnsubscribeToken != token {
		return apperror.Forbidden("Tautan berhenti berlangganan tidak valid")
	}

	now := time.Now()
	sub.IsActive = false
	sub.UnsubscribedDate = &now
	if err := s.repo.Update(ctx, sub); err != nil {
		return err
	}
	s.sendUnsubscribeConfirmation(sub.Email)
	return nil
}

// splitEmailList splits raw textarea input on newlines/commas and trims whitespace.
func splitEmailList(raw string) []string {
	return strings.FieldsFunc(raw, func(r rune) bool {
		return r == '\n' || r == '\r' || r == ','
	})
}

func (s *serviceImpl) BulkAdd(ctx context.Context, rawEmails string) (*subscription_dto.BulkAddResult, error) {
	result := &subscription_dto.BulkAddResult{Skipped: []string{}, Invalid: []string{}}
	seen := make(map[string]bool)

	for _, raw := range splitEmailList(rawEmails) {
		email := strings.TrimSpace(raw)
		if email == "" || seen[email] {
			continue
		}
		seen[email] = true

		if _, err := mail.ParseAddress(email); err != nil {
			result.Invalid = append(result.Invalid, email)
			continue
		}

		existing, err := s.repo.FindByEmail(ctx, email)
		if err != nil && !errors.Is(err, subscription_repository.ErrNotFound) {
			return nil, err
		}

		if existing != nil {
			if existing.IsActive {
				result.Skipped = append(result.Skipped, email)
				continue
			}
			existing.IsActive = true
			existing.SubscribedDate = time.Now()
			existing.UnsubscribedDate = nil
			if err := s.repo.Update(ctx, existing); err != nil {
				return nil, err
			}
			result.Added++
			continue
		}

		sub := &subscription_model.Subscriber{
			Email:          email,
			IsActive:       true,
			SubscribedDate: time.Now(),
		}
		if err := s.repo.Create(ctx, sub); err != nil {
			return nil, err
		}
		result.Added++
	}

	return result, nil
}

func toResponse(sub *subscription_model.Subscriber) subscription_dto.SubscriberResponse {
	return subscription_dto.SubscriberResponse{
		SubscriberID:     sub.SubscriberID,
		Email:            sub.Email,
		IsActive:         sub.IsActive,
		SubscribedDate:   sub.SubscribedDate,
		UnsubscribedDate: sub.UnsubscribedDate,
		CreatedDate:      sub.CreatedDate,
	}
}

func (s *serviceImpl) List(ctx context.Context, q dto.ListQuery, isActive *bool, from, to string) ([]subscription_dto.SubscriberResponse, int, error) {
	subs, total, err := s.repo.FindAll(ctx, q, isActive, from, to)
	if err != nil {
		return nil, 0, err
	}

	items := make([]subscription_dto.SubscriberResponse, 0, len(subs))
	for i := range subs {
		items = append(items, toResponse(&subs[i]))
	}

	return items, total, nil
}

func (s *serviceImpl) GetByID(ctx context.Context, id int64) (*subscription_dto.SubscriberResponse, error) {
	sub, err := s.repo.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, subscription_repository.ErrNotFound) {
			return nil, apperror.NotFound("Subscriber tidak ditemukan")
		}
		return nil, err
	}
	resp := toResponse(sub)
	return &resp, nil
}

func (s *serviceImpl) Update(ctx context.Context, id int64, req subscription_dto.UpdateSubscriberRequest) (*subscription_dto.SubscriberResponse, error) {
	sub, err := s.repo.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, subscription_repository.ErrNotFound) {
			return nil, apperror.NotFound("Subscriber tidak ditemukan")
		}
		return nil, err
	}

	email := strings.TrimSpace(req.Email)
	exists, err := s.repo.EmailExistsExcluding(ctx, email, id)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, apperror.Conflict("Email ini sudah dipakai subscriber lain")
	}

	if req.IsActive && !sub.IsActive {
		sub.SubscribedDate = time.Now()
		sub.UnsubscribedDate = nil
	} else if !req.IsActive && sub.IsActive {
		now := time.Now()
		sub.UnsubscribedDate = &now
	}
	sub.Email = email
	sub.IsActive = req.IsActive

	if err := s.repo.Update(ctx, sub); err != nil {
		return nil, err
	}
	resp := toResponse(sub)
	return &resp, nil
}

func (s *serviceImpl) Delete(ctx context.Context, id int64) error {
	err := s.repo.Delete(ctx, id)
	if errors.Is(err, subscription_repository.ErrNotFound) {
		return apperror.NotFound("Subscriber tidak ditemukan")
	}
	return err
}

func (s *serviceImpl) BulkDelete(ctx context.Context, ids []int64) error {
	return s.repo.DeleteMany(ctx, ids)
}
