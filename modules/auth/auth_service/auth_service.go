// Package auth_service memuat logika bisnis autentikasi.
package auth_service

import (
	"context"

	"fsldk-api/modules/auth/auth_dto"
)

// Service adalah kontrak logika autentikasi.
type Service interface {
	Register(ctx context.Context, req auth_dto.RegisterRequest) (auth_dto.RegisterResponse, error)
	VerifyEmail(ctx context.Context, token string) error
	ResendVerification(ctx context.Context, userID int64) error
	Login(ctx context.Context, req auth_dto.LoginRequest, ip, ua string) (auth_dto.AuthResponse, error)
	LoginGoogle(ctx context.Context, idToken, ip, ua string) (auth_dto.AuthResponse, error)
	Refresh(ctx context.Context, refreshToken string) (auth_dto.AuthResponse, error)
	Me(ctx context.Context, userID int64) (auth_dto.UserProfile, error)
	ChangePassword(ctx context.Context, userID int64, req auth_dto.ChangePasswordRequest) error
	UpdateContact(ctx context.Context, userID int64, req auth_dto.UpdateContactRequest) (auth_dto.UserProfile, error)
	UpdatePhoto(ctx context.Context, userID int64, req auth_dto.UpdatePhotoRequest) (auth_dto.UserProfile, error)
	ForgotPassword(ctx context.Context, email string) error
	ResetPassword(ctx context.Context, req auth_dto.ResetPasswordRequest) error
}
