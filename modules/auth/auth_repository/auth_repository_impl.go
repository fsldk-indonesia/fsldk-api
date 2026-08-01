package auth_repository

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"fsldk-api/modules/auth/auth_model"

	"github.com/jmoiron/sqlx"
)

// TokenStoreImpl adalah implementasi TokenStore berbasis sqlx.
type TokenStoreImpl struct{ db *sqlx.DB }

// NewTokenStore membuat implementasi TokenStore.
func NewTokenStore(db *sqlx.DB) TokenStore { return &TokenStoreImpl{db: db} }

func (s *TokenStoreImpl) Create(ctx context.Context, userID int64, token, tokenType string, expiresAt time.Time) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO tr_email_token (userID, token, tokenType, expiresAt) VALUES (?, ?, ?, ?)`,
		userID, token, tokenType, expiresAt)
	return err
}

func (s *TokenStoreImpl) FindValid(ctx context.Context, token, tokenType string) (auth_model.EmailToken, error) {
	var t auth_model.EmailToken
	err := s.db.GetContext(ctx, &t,
		`SELECT tokenID, userID, token, tokenType, expiresAt, usedAt
		 FROM tr_email_token
		 WHERE token = ? AND tokenType = ? AND usedAt IS NULL AND expiresAt > NOW()
		 LIMIT 1`, token, tokenType)
	if errors.Is(err, sql.ErrNoRows) {
		return auth_model.EmailToken{}, errors.New("token tidak valid atau kedaluwarsa")
	}
	return t, err
}

func (s *TokenStoreImpl) MarkUsed(ctx context.Context, tokenID int64) error {
	_, err := s.db.ExecContext(ctx, `UPDATE tr_email_token SET usedAt = NOW() WHERE tokenID = ?`, tokenID)
	return err
}

func (s *TokenStoreImpl) InvalidateAll(ctx context.Context, userID int64, tokenType string) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE tr_email_token SET usedAt = NOW() WHERE userID = ? AND tokenType = ? AND usedAt IS NULL`,
		userID, tokenType)
	return err
}
