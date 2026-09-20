package welcomepopup_service

import (
	"context"

	"fsldk-api/base/apperror"
	"fsldk-api/modules/welcomepopup/welcomepopup_dto"
	"fsldk-api/modules/welcomepopup/welcomepopup_model"
	"fsldk-api/modules/welcomepopup/welcomepopup_repository"
)

// ServiceImpl adalah implementasi Service.
type ServiceImpl struct {
	repo welcomepopup_repository.Repository
}

// NewService membuat Service welcomepopup.
func NewService(repo welcomepopup_repository.Repository) Service {
	return &ServiceImpl{repo: repo}
}

func toResponse(w welcomepopup_model.WelcomePopup) welcomepopup_dto.Response {
	html, js, css := "", "", ""
	if w.HTMLContent != nil {
		html = *w.HTMLContent
	}
	if w.JSContent != nil {
		js = *w.JSContent
	}
	if w.CSSContent != nil {
		css = *w.CSSContent
	}
	updatedDate := ""
	if w.UpdatedDate != nil {
		updatedDate = w.UpdatedDate.Format("2006-01-02 15:04:05")
	}
	return welcomepopup_dto.Response{
		IsEnabled: w.IsEnabled, HTMLContent: html, JSContent: js, CSSContent: css, UpdatedDate: updatedDate,
	}
}

func toPublicResponse(w welcomepopup_model.WelcomePopup) welcomepopup_dto.PublicResponse {
	r := toResponse(w)
	return welcomepopup_dto.PublicResponse{
		IsEnabled: r.IsEnabled, HTMLContent: r.HTMLContent, JSContent: r.JSContent, CSSContent: r.CSSContent,
	}
}

func (s *ServiceImpl) Get(ctx context.Context) (welcomepopup_dto.Response, error) {
	w, err := s.repo.Get(ctx)
	if err != nil {
		return welcomepopup_dto.Response{}, apperror.Internal("")
	}
	return toResponse(w), nil
}

func (s *ServiceImpl) GetPublic(ctx context.Context) (welcomepopup_dto.PublicResponse, error) {
	w, err := s.repo.Get(ctx)
	if err != nil {
		return welcomepopup_dto.PublicResponse{}, apperror.Internal("")
	}
	return toPublicResponse(w), nil
}

func (s *ServiceImpl) Update(ctx context.Context, req welcomepopup_dto.UpdateRequest, actorID int64) (welcomepopup_dto.Response, error) {
	popup := welcomepopup_model.WelcomePopup{
		ID:          welcomepopup_model.SingletonID,
		IsEnabled:   req.IsEnabled,
		HTMLContent: &req.HTMLContent,
		JSContent:   &req.JSContent,
		CSSContent:  &req.CSSContent,
		UpdatedBy:   &actorID,
	}
	if err := s.repo.Update(ctx, popup); err != nil {
		return welcomepopup_dto.Response{}, apperror.Internal("")
	}
	return s.Get(ctx)
}
