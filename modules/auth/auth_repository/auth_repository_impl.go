package auth_repository

import (
	"context"
	"errors"
	"time"

	"fsldk-api/modules/auth/auth_model"

	"gorm.io/gorm"
)

// TokenStoreImpl adalah implementasi TokenStore berbasis GORM.
type TokenStoreImpl struct{ db *gorm.DB }

// NewTokenStore membuat implementasi TokenStore.
func NewTokenStore(db *gorm.DB) TokenStore { return &TokenStoreImpl{db: db} }

func (s *TokenStoreImpl) Create(ctx context.Context, userID int64, token, tokenType string, expiresAt time.Time) error {
	return s.db.WithContext(ctx).Table("tr_email_token").Create(map[string]interface{}{
		"userID":    userID,
		"token":     token,
		"tokenType": tokenType,
		"expiresAt": expiresAt,
	}).Error
}

func (s *TokenStoreImpl) FindValid(ctx context.Context, token, tokenType string) (auth_model.EmailToken, error) {
	var t auth_model.EmailToken
	err := s.db.WithContext(ctx).Table("tr_email_token").
		Where("token = ? AND tokenType = ? AND usedAt IS NULL AND expiresAt > ?", token, tokenType, time.Now()).
		Take(&t).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return auth_model.EmailToken{}, errors.New("token tidak valid atau kedaluwarsa")
	}
	return t, err
}

func (s *TokenStoreImpl) MarkUsed(ctx context.Context, tokenID int64) error {
	return s.db.WithContext(ctx).Table("tr_email_token").Where("tokenID = ?", tokenID).
		Update("usedAt", time.Now()).Error
}

func (s *TokenStoreImpl) InvalidateAll(ctx context.Context, userID int64, tokenType string) error {
	return s.db.WithContext(ctx).Table("tr_email_token").
		Where("userID = ? AND tokenType = ? AND usedAt IS NULL", userID, tokenType).
		Update("usedAt", time.Now()).Error
}
