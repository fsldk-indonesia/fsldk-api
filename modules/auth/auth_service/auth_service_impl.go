package auth_service

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"fsldk-api/base/apperror"
	"fsldk-api/base/security"
	"fsldk-api/base/token"
	"fsldk-api/config"
	"fsldk-api/constants"
	"fsldk-api/modules/auth/auth_dto"
	"fsldk-api/modules/auth/auth_repository"
	"fsldk-api/modules/permission/permission_service"
	"fsldk-api/modules/role/role_repository"
	"fsldk-api/modules/user/user_model"
	"fsldk-api/modules/user/user_repository"
	"fsldk-api/pkg/googleauth"
	"fsldk-api/pkg/mailer"
)

// emailVerified & hasPassword adalah helper murni fungsi (bukan method pada
// model) yang mengevaluasi status akun berdasarkan field user_model.User.
func emailVerified(u user_model.User) bool { return u.EmailVerifiedDate.Valid }
func hasPassword(u user_model.User) bool   { return u.Password.Valid && u.Password.String != "" }

func organizationID(u user_model.User) *int64 {
	if !u.OrganizationID.Valid {
		return nil
	}
	id := u.OrganizationID.Int64
	return &id
}

// ServiceImpl adalah implementasi Service.
type ServiceImpl struct {
	users  user_repository.Repository
	roles  role_repository.Repository
	perms  permission_service.Service
	tokens *token.Manager
	store  auth_repository.TokenStore
	mail   mailer.Mailer
	google *googleauth.Verifier
	cfg    config.AppConfig
}

// NewService membuat Service auth.
func NewService(
	users user_repository.Repository,
	roles role_repository.Repository,
	perms permission_service.Service,
	tokens *token.Manager,
	store auth_repository.TokenStore,
	mail mailer.Mailer,
	google *googleauth.Verifier,
	cfg config.AppConfig,
) Service {
	return &ServiceImpl{users: users, roles: roles, perms: perms, tokens: tokens, store: store, mail: mail, google: google, cfg: cfg}
}

func (s *ServiceImpl) Register(ctx context.Context, req auth_dto.RegisterRequest) (auth_dto.RegisterResponse, error) {
	email := strings.ToLower(strings.TrimSpace(req.Email))

	exists, err := s.users.ExistsByEmail(ctx, email)
	if err != nil {
		return auth_dto.RegisterResponse{}, apperror.Internal("")
	}
	if exists {
		return auth_dto.RegisterResponse{}, apperror.Conflict("Email sudah terdaftar")
	}

	roleID, err := s.roles.IDByName(ctx, s.cfg.RegisterDefaultRole)
	if err != nil {
		return auth_dto.RegisterResponse{}, apperror.Internal("Role default tidak ditemukan")
	}

	hashed, err := security.HashPassword(req.Password)
	if err != nil {
		return auth_dto.RegisterResponse{}, apperror.Internal("")
	}

	userID, err := s.users.Create(ctx, user_model.CreateParams{
		RoleID:        roleID,
		FullName:      strings.TrimSpace(req.FullName),
		Email:         email,
		Password:      sql.NullString{String: hashed, Valid: true},
		EmailVerified: false,
	})
	if err != nil {
		return auth_dto.RegisterResponse{}, apperror.Internal("Gagal membuat akun")
	}

	if err := s.sendVerification(ctx, userID, req.FullName, email); err != nil {
		return auth_dto.RegisterResponse{}, apperror.Internal("Akun dibuat namun email verifikasi gagal dikirim. Silakan gunakan kirim ulang.")
	}

	return auth_dto.RegisterResponse{
		UserID:        userID,
		Email:         email,
		EmailVerified: false,
		Message:       "Registrasi berhasil. Silakan cek email untuk verifikasi akun.",
	}, nil
}

func (s *ServiceImpl) sendVerification(ctx context.Context, userID int64, name, email string) error {
	_ = s.store.InvalidateAll(ctx, userID, constants.EmailTokenVerification)

	tokenStr, err := security.RandomToken(32)
	if err != nil {
		return err
	}
	expires := time.Now().Add(time.Duration(s.cfg.EmailVerificationExpireMinutes) * time.Minute)
	if err := s.store.Create(ctx, userID, tokenStr, constants.EmailTokenVerification, expires); err != nil {
		return err
	}

	verifyURL := fmt.Sprintf("%s/verifikasi-email?token=%s", strings.TrimRight(s.cfg.FrontendURL, "/"), tokenStr)
	return s.mail.SendVerificationEmail(email, name, verifyURL)
}

func (s *ServiceImpl) VerifyEmail(ctx context.Context, tokenStr string) error {
	et, err := s.store.FindValid(ctx, tokenStr, constants.EmailTokenVerification)
	if err != nil {
		return apperror.BadRequest("Tautan verifikasi tidak valid atau telah kedaluwarsa")
	}
	if err := s.users.MarkEmailVerified(ctx, et.UserID); err != nil {
		return apperror.Internal("")
	}
	_ = s.store.MarkUsed(ctx, et.TokenID)
	return nil
}

func (s *ServiceImpl) ResendVerification(ctx context.Context, userID int64) error {
	u, err := s.users.FindByID(ctx, userID)
	if err != nil {
		return apperror.NotFound("Pengguna tidak ditemukan")
	}
	if emailVerified(u) {
		return apperror.Unprocessable("Email Anda sudah terverifikasi")
	}
	if err := s.sendVerification(ctx, u.UserID, u.FullName, u.Email); err != nil {
		return apperror.Internal("Gagal mengirim email verifikasi")
	}
	return nil
}

func (s *ServiceImpl) Login(ctx context.Context, req auth_dto.LoginRequest, ip, ua string) (auth_dto.AuthResponse, error) {
	email := strings.ToLower(strings.TrimSpace(req.Email))
	u, err := s.users.FindByEmail(ctx, email)
	if err != nil {
		return auth_dto.AuthResponse{}, apperror.Unauthorized("Email atau kata sandi salah")
	}
	if !u.IsActive {
		return auth_dto.AuthResponse{}, apperror.Forbidden("Akun Anda nonaktif")
	}
	if !hasPassword(u) || !security.CheckPassword(u.Password.String, req.Password) {
		_ = s.users.LogLogin(ctx, u.UserID, ip, ua, "failed")
		return auth_dto.AuthResponse{}, apperror.Unauthorized("Email atau kata sandi salah")
	}

	_ = s.users.LogLogin(ctx, u.UserID, ip, ua, "success")
	return s.buildAuthResponse(ctx, u)
}

func (s *ServiceImpl) LoginGoogle(ctx context.Context, idToken, ip, ua string) (auth_dto.AuthResponse, error) {
	payload, err := s.google.Verify(idToken)
	if err != nil {
		return auth_dto.AuthResponse{}, apperror.Unauthorized("Token Google tidak valid")
	}

	email := strings.ToLower(strings.TrimSpace(payload.Email))
	if !s.isDomainAllowed(email) {
		return auth_dto.AuthResponse{}, apperror.Forbidden("Domain email tidak diizinkan")
	}

	// 1) googleID sudah cocok → login langsung, segarkan foto profil dari Google.
	if u, err := s.users.FindByGoogleID(ctx, payload.Sub); err == nil {
		if payload.Picture != "" && payload.Picture != u.PhotoURL.String {
			_ = s.users.UpdatePhoto(ctx, u.UserID, payload.Picture)
			u.PhotoURL = sql.NullString{String: payload.Picture, Valid: true}
		}
		_ = s.users.LogLogin(ctx, u.UserID, ip, ua, "success")
		return s.buildAuthResponse(ctx, u)
	}

	// 2) email cocok dengan akun lokal → auto-link + tandai terverifikasi.
	if u, err := s.users.FindByEmail(ctx, email); err == nil {
		if err := s.users.LinkGoogle(ctx, u.UserID, payload.Sub, true); err != nil {
			return auth_dto.AuthResponse{}, apperror.Internal("")
		}
		if payload.Picture != "" {
			_ = s.users.UpdatePhoto(ctx, u.UserID, payload.Picture)
		}
		u, _ = s.users.FindByID(ctx, u.UserID)
		_ = s.users.LogLogin(ctx, u.UserID, ip, ua, "success")
		return s.buildAuthResponse(ctx, u)
	}

	// 3) akun baru → auto-provision (password NULL, langsung terverifikasi).
	roleID, err := s.roles.IDByName(ctx, s.cfg.GoogleDefaultRole)
	if err != nil {
		return auth_dto.AuthResponse{}, apperror.Internal("Role default tidak ditemukan")
	}
	name := payload.Name
	if name == "" {
		name = email
	}
	userID, err := s.users.Create(ctx, user_model.CreateParams{
		RoleID:        roleID,
		FullName:      name,
		Email:         email,
		GoogleID:      sql.NullString{String: payload.Sub, Valid: true},
		PhotoURL:      sql.NullString{String: payload.Picture, Valid: payload.Picture != ""},
		EmailVerified: true,
	})
	if err != nil {
		return auth_dto.AuthResponse{}, apperror.Internal("Gagal membuat akun")
	}
	u, err := s.users.FindByID(ctx, userID)
	if err != nil {
		return auth_dto.AuthResponse{}, apperror.Internal("")
	}
	_ = s.users.LogLogin(ctx, u.UserID, ip, ua, "success")
	return s.buildAuthResponse(ctx, u)
}

func (s *ServiceImpl) Refresh(ctx context.Context, refreshToken string) (auth_dto.AuthResponse, error) {
	claims, err := s.tokens.ParseRefresh(refreshToken)
	if err != nil {
		return auth_dto.AuthResponse{}, apperror.Unauthorized("Refresh token tidak valid")
	}
	u, err := s.users.FindByID(ctx, claims.UserID)
	if err != nil || !u.IsActive {
		return auth_dto.AuthResponse{}, apperror.Unauthorized("Sesi tidak valid")
	}
	return s.buildAuthResponse(ctx, u)
}

func (s *ServiceImpl) Me(ctx context.Context, userID int64) (auth_dto.UserProfile, error) {
	u, err := s.users.FindByID(ctx, userID)
	if err != nil {
		return auth_dto.UserProfile{}, apperror.NotFound("Pengguna tidak ditemukan")
	}
	return s.profileFor(ctx, u)
}

func (s *ServiceImpl) ChangePassword(ctx context.Context, userID int64, req auth_dto.ChangePasswordRequest) error {
	u, err := s.users.FindByID(ctx, userID)
	if err != nil {
		return apperror.NotFound("Pengguna tidak ditemukan")
	}
	if !hasPassword(u) {
		return apperror.Unprocessable("Akun ini belum memiliki kata sandi lokal. Silakan gunakan lupa kata sandi untuk mengaturnya.")
	}
	if !security.CheckPassword(u.Password.String, req.OldPassword) {
		return apperror.BadRequest("Kata sandi lama salah")
	}
	hashed, err := security.HashPassword(req.NewPassword)
	if err != nil {
		return apperror.Internal("")
	}
	if err := s.users.SetPassword(ctx, userID, hashed, false); err != nil {
		return apperror.Internal("")
	}
	return nil
}

func (s *ServiceImpl) ForgotPassword(ctx context.Context, email string) error {
	email = strings.ToLower(strings.TrimSpace(email))
	u, err := s.users.FindByEmail(ctx, email)
	if err != nil {
		// Jangan bocorkan keberadaan email.
		return nil
	}

	_ = s.store.InvalidateAll(ctx, u.UserID, constants.EmailTokenPasswordReset)
	tokenStr, err := security.RandomToken(32)
	if err != nil {
		return apperror.Internal("")
	}
	expires := time.Now().Add(time.Duration(s.cfg.PasswordResetExpireMinutes) * time.Minute)
	if err := s.store.Create(ctx, u.UserID, tokenStr, constants.EmailTokenPasswordReset, expires); err != nil {
		return apperror.Internal("")
	}
	resetURL := fmt.Sprintf("%s/reset-password?token=%s", strings.TrimRight(s.cfg.FrontendURL, "/"), tokenStr)
	_ = s.mail.SendPasswordResetEmail(u.Email, u.FullName, resetURL)
	return nil
}

func (s *ServiceImpl) ResetPassword(ctx context.Context, req auth_dto.ResetPasswordRequest) error {
	et, err := s.store.FindValid(ctx, req.Token, constants.EmailTokenPasswordReset)
	if err != nil {
		return apperror.BadRequest("Tautan reset tidak valid atau telah kedaluwarsa")
	}
	hashed, err := security.HashPassword(req.Password)
	if err != nil {
		return apperror.Internal("")
	}
	if err := s.users.SetPassword(ctx, et.UserID, hashed, false); err != nil {
		return apperror.Internal("")
	}
	_ = s.store.MarkUsed(ctx, et.TokenID)
	return nil
}

// ---------- helper ----------

func (s *ServiceImpl) buildAuthResponse(ctx context.Context, u user_model.User) (auth_dto.AuthResponse, error) {
	profile, err := s.profileFor(ctx, u)
	if err != nil {
		return auth_dto.AuthResponse{}, err
	}
	access, err := s.tokens.GenerateAccess(token.AccessParams{
		UserID:               u.UserID,
		RoleID:               u.RoleID,
		Email:                u.Email,
		RoleName:             u.RoleName,
		EmailVerified:        emailVerified(u),
		OrganizationID:       organizationID(u),
		OrganizationTypeCode: u.OrganizationTypeCode.String,
		WildcardTierAccess:   u.WildcardTierAccess.String,
	})
	if err != nil {
		return auth_dto.AuthResponse{}, apperror.Internal("")
	}
	refresh, err := s.tokens.GenerateRefresh(u.UserID)
	if err != nil {
		return auth_dto.AuthResponse{}, apperror.Internal("")
	}
	return auth_dto.AuthResponse{
		AccessToken:  access,
		RefreshToken: refresh,
		ExpiresIn:    s.tokens.AccessExpireSeconds(),
		User:         profile,
	}, nil
}

func (s *ServiceImpl) profileFor(ctx context.Context, u user_model.User) (auth_dto.UserProfile, error) {
	perms, err := s.perms.RolePermissions(ctx, u.RoleID)
	if err != nil {
		return auth_dto.UserProfile{}, apperror.Internal("")
	}
	if perms == nil {
		perms = []string{}
	}
	profile := auth_dto.UserProfile{
		UserID:               u.UserID,
		FullName:             u.FullName,
		Email:                u.Email,
		EmailVerified:        emailVerified(u),
		Role:                 u.RoleName,
		Permissions:          perms,
		PhotoURL:             u.PhotoURL.String,
		OrganizationID:       organizationID(u),
		OrganizationTypeCode: u.OrganizationTypeCode.String,
	}
	if u.WildcardTierAccess.Valid && u.WildcardTierAccess.String != "" {
		profile.WildcardTierAccess = strings.Split(u.WildcardTierAccess.String, ",")
	}
	return profile, nil
}

func (s *ServiceImpl) isDomainAllowed(email string) bool {
	allowed := s.cfg.AllowedGoogleDomains()
	if len(allowed) == 0 {
		return true
	}
	at := strings.LastIndex(email, "@")
	if at < 0 {
		return false
	}
	domain := strings.ToLower(email[at+1:])
	for _, d := range allowed {
		if d == domain {
			return true
		}
	}
	return false
}
