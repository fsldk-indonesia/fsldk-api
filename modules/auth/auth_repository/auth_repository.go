// Package auth_repository adalah lapisan akses data modul auth (token email).
package auth_repository

import (
	"context"
	"time"

	"fsldk-api/modules/auth/auth_model"
)

// TokenStore mengelola penyimpanan token email di tabel tr_email_token.
type TokenStore interface {
	Create(ctx context.Context, userID int64, token, tokenType string, expiresAt time.Time) error
	FindValid(ctx context.Context, token, tokenType string) (auth_model.EmailToken, error)
	MarkUsed(ctx context.Context, tokenID int64) error
	InvalidateAll(ctx context.Context, userID int64, tokenType string) error
}
