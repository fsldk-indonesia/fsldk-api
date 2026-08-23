package setting_service

import (
	"context"
	"errors"

	"fsldk-api/base/apperror"
	"fsldk-api/modules/setting/setting_dto"
	"fsldk-api/modules/setting/setting_model"
	"fsldk-api/modules/setting/setting_repository"
)

// ServiceImpl adalah implementasi Service.
type ServiceImpl struct {
	repo setting_repository.Repository
}

// NewService membuat Service setting.
func NewService(repo setting_repository.Repository) Service {
	return &ServiceImpl{repo: repo}
}

func toResponse(s setting_model.Setting) setting_dto.Response {
	value := ""
	if s.SettingValue != nil {
		value = *s.SettingValue
	}
	updatedDate := ""
	if s.UpdatedDate != nil {
		updatedDate = s.UpdatedDate.Format("2006-01-02 15:04:05")
	}
	return setting_dto.Response{
		SettingID:    s.SettingID,
		SettingGroup: s.SettingGroup,
		SettingKey:   s.SettingKey,
		SettingLabel: s.SettingLabel,
		SettingValue: value,
		UpdatedDate:  updatedDate,
	}
}

func (s *ServiceImpl) List(ctx context.Context) ([]setting_dto.Response, error) {
	rows, err := s.repo.List(ctx)
	if err != nil {
		return nil, apperror.Internal("")
	}
	out := make([]setting_dto.Response, 0, len(rows))
	for _, r := range rows {
		out = append(out, toResponse(r))
	}
	return out, nil
}

func (s *ServiceImpl) Update(ctx context.Context, id int64, req setting_dto.UpdateRequest, actorID int64) (setting_dto.Response, error) {
	if _, err := s.repo.FindByID(ctx, id); err != nil {
		return setting_dto.Response{}, apperror.NotFound("Setting tidak ditemukan")
	}
	if err := s.repo.Update(ctx, id, req.SettingValue, actorID); err != nil {
		return setting_dto.Response{}, apperror.Internal("")
	}
	updated, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return setting_dto.Response{}, apperror.Internal("")
	}
	return toResponse(updated), nil
}

func (s *ServiceImpl) GetValue(ctx context.Context, group, key string) (string, error) {
	setting, err := s.repo.FindByGroupKey(ctx, group, key)
	if err != nil {
		if errors.Is(err, setting_repository.ErrNotFound) {
			return "", nil
		}
		return "", err
	}
	if setting.SettingValue == nil {
		return "", nil
	}
	return *setting.SettingValue, nil
}
