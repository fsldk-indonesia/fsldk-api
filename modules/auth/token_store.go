package auth

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/jmoiron/sqlx"
)

// EmailToken merepresentasikan token verifikasi email / reset password.
type EmailToken struct {
	TokenID   int64        `db:"tokenID"`
	UserID    int64        `db:"userID"`
	Token     string       `db:"token"`
	TokenType string       `db:"tokenType"`
	ExpiresAt time.Time    `db:"expiresAt"`
	UsedAt    sql.NullTime `db:"usedAt"`
}

// TokenStore mengelola penyimpanan token email di tabel tr_email_token.
type TokenStore interface {
	Create(ctx context.Context, userID int64, token, tokenType string, expiresAt time.Time) error
	FindValid(ctx context.Context, token, tokenType string) (EmailToken, error)
	MarkUsed(ctx context.Context, tokenID int64) error
	InvalidateAll(ctx context.Context, userID int64, tokenType string) error
}

type tokenStore struct{ db *sqlx.DB }

// NewTokenStore membuat implementasi TokenStore.
func NewTokenStore(db *sqlx.DB) TokenStore { return &tokenStore{db: db} }

func (s *tokenStore) Create(ctx context.Context, userID int64, token, tokenType string, expiresAt time.Time) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO tr_email_token (userID, token, tokenType, expiresAt) VALUES (?, ?, ?, ?)`,
		userID, token, tokenType, expiresAt)
	return err
}

func (s *tokenStore) FindValid(ctx context.Context, token, tokenType string) (EmailToken, error) {
	var t EmailToken
	err := s.db.GetContext(ctx, &t,
		`SELECT tokenID, userID, token, tokenType, expiresAt, usedAt
		 FROM tr_email_token
		 WHERE token = ? AND tokenType = ? AND usedAt IS NULL AND expiresAt > NOW()
		 LIMIT 1`, token, tokenType)
	if errors.Is(err, sql.ErrNoRows) {
		return EmailToken{}, errors.New("token tidak valid atau kedaluwarsa")
	}
	return t, err
}

func (s *tokenStore) MarkUsed(ctx context.Context, tokenID int64) error {
	_, err := s.db.ExecContext(ctx, `UPDATE tr_email_token SET usedAt = NOW() WHERE tokenID = ?`, tokenID)
	return err
}

func (s *tokenStore) InvalidateAll(ctx context.Context, userID int64, tokenType string) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE tr_email_token SET usedAt = NOW() WHERE userID = ? AND tokenType = ? AND usedAt IS NULL`,
		userID, tokenType)
	return err
}
