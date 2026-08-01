package auth

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
	"fsldk-api/modules/permission"
	"fsldk-api/modules/role"
	"fsldk-api/modules/user"
	"fsldk-api/pkg/googleauth"
	"fsldk-api/pkg/mailer"
)

// Service memuat seluruh logika autentikasi.
type Service struct {
	users  user.Repository
	roles  role.Repository
	perms  *permission.Service
	tokens *token.Manager
	store  TokenStore
	mail   mailer.Mailer
	google *googleauth.Verifier
	cfg    config.AppConfig
}

// NewService membuat Service auth.
func NewService(
	users user.Repository,
	roles role.Repository,
	perms *permission.Service,
	tokens *token.Manager,
	store TokenStore,
	mail mailer.Mailer,
	google *googleauth.Verifier,
	cfg config.AppConfig,
) *Service {
	return &Service{users: users, roles: roles, perms: perms, tokens: tokens, store: store, mail: mail, google: google, cfg: cfg}
}

// Register membuat akun baru dan mengirim email verifikasi.
func (s *Service) Register(ctx context.Context, req RegisterRequest) (RegisterResponse, error) {
	email := strings.ToLower(strings.TrimSpace(req.Email))

	exists, err := s.users.ExistsByEmail(ctx, email)
	if err != nil {
		return RegisterResponse{}, apperror.Internal("")
	}
	if exists {
		return RegisterResponse{}, apperror.Conflict("Email sudah terdaftar")
	}

	roleID, err := s.roles.IDByName(ctx, s.cfg.RegisterDefaultRole)
	if err != nil {
		return RegisterResponse{}, apperror.Internal("Role default tidak ditemukan")
	}

	hashed, err := security.HashPassword(req.Password)
	if err != nil {
		return RegisterResponse{}, apperror.Internal("")
	}

	userID, err := s.users.Create(ctx, user.CreateParams{
		RoleID:        roleID,
		FullName:      strings.TrimSpace(req.FullName),
		Email:         email,
		Password:      sql.NullString{String: hashed, Valid: true},
		EmailVerified: false,
	})
	if err != nil {
		return RegisterResponse{}, apperror.Internal("Gagal membuat akun")
	}

	if err := s.sendVerification(ctx, userID, req.FullName, email); err != nil {
		// Akun sudah dibuat; kegagalan email tidak membatalkan registrasi.
		// Pengguna dapat meminta kirim ulang.
		return RegisterResponse{}, apperror.Internal("Akun dibuat namun email verifikasi gagal dikirim. Silakan gunakan kirim ulang.")
	}

	return RegisterResponse{
		UserID:        userID,
		Email:         email,
		EmailVerified: false,
		Message:       "Registrasi berhasil. Silakan cek email untuk verifikasi akun.",
	}, nil
}

func (s *Service) sendVerification(ctx context.Context, userID int64, name, email string) error {
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

// VerifyEmail memverifikasi email berdasarkan token.
func (s *Service) VerifyEmail(ctx context.Context, tokenStr string) error {
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

// ResendVerification mengirim ulang email verifikasi untuk pengguna terautentikasi.
func (s *Service) ResendVerification(ctx context.Context, userID int64) error {
	u, err := s.users.FindByID(ctx, userID)
	if err != nil {
		return apperror.NotFound("Pengguna tidak ditemukan")
	}
	if u.EmailVerified() {
		return apperror.Unprocessable("Email Anda sudah terverifikasi")
	}
	if err := s.sendVerification(ctx, u.UserID, u.FullName, u.Email); err != nil {
		return apperror.Internal("Gagal mengirim email verifikasi")
	}
	return nil
}

// Login memproses login email + password.
func (s *Service) Login(ctx context.Context, req LoginRequest, ip, ua string) (AuthResponse, error) {
	email := strings.ToLower(strings.TrimSpace(req.Email))
	u, err := s.users.FindByEmail(ctx, email)
	if err != nil {
		return AuthResponse{}, apperror.Unauthorized("Email atau kata sandi salah")
	}
	if !u.IsActive {
		return AuthResponse{}, apperror.Forbidden("Akun Anda nonaktif")
	}
	if !u.HasPassword() || !security.CheckPassword(u.Password.String, req.Password) {
		_ = s.users.LogLogin(ctx, u.UserID, ip, ua, "failed")
		return AuthResponse{}, apperror.Unauthorized("Email atau kata sandi salah")
	}

	_ = s.users.LogLogin(ctx, u.UserID, ip, ua, "success")
	return s.buildAuthResponse(ctx, u)
}

// LoginGoogle memproses login/registrasi via Google (auto-link / auto-provision).
func (s *Service) LoginGoogle(ctx context.Context, idToken, ip, ua string) (AuthResponse, error) {
	payload, err := s.google.Verify(idToken)
	if err != nil {
		return AuthResponse{}, apperror.Unauthorized("Token Google tidak valid")
	}

	email := strings.ToLower(strings.TrimSpace(payload.Email))
	if !s.isDomainAllowed(email) {
		return AuthResponse{}, apperror.Forbidden("Domain email tidak diizinkan")
	}

	// 1) googleID sudah cocok → login langsung.
	if u, err := s.users.FindByGoogleID(ctx, payload.Sub); err == nil {
		_ = s.users.LogLogin(ctx, u.UserID, ip, ua, "success")
		return s.buildAuthResponse(ctx, u)
	}

	// 2) email cocok dengan akun lokal → auto-link + tandai terverifikasi.
	if u, err := s.users.FindByEmail(ctx, email); err == nil {
		if err := s.users.LinkGoogle(ctx, u.UserID, payload.Sub, true); err != nil {
			return AuthResponse{}, apperror.Internal("")
		}
		u, _ = s.users.FindByID(ctx, u.UserID)
		_ = s.users.LogLogin(ctx, u.UserID, ip, ua, "success")
		return s.buildAuthResponse(ctx, u)
	}

	// 3) akun baru → auto-provision (password NULL, langsung terverifikasi).
	roleID, err := s.roles.IDByName(ctx, s.cfg.GoogleDefaultRole)
	if err != nil {
		return AuthResponse{}, apperror.Internal("Role default tidak ditemukan")
	}
	name := payload.Name
	if name == "" {
		name = email
	}
	userID, err := s.users.Create(ctx, user.CreateParams{
		RoleID:        roleID,
		FullName:      name,
		Email:         email,
		GoogleID:      sql.NullString{String: payload.Sub, Valid: true},
		EmailVerified: true,
	})
	if err != nil {
		return AuthResponse{}, apperror.Internal("Gagal membuat akun")
	}
	u, err := s.users.FindByID(ctx, userID)
	if err != nil {
		return AuthResponse{}, apperror.Internal("")
	}
	_ = s.users.LogLogin(ctx, u.UserID, ip, ua, "success")
	return s.buildAuthResponse(ctx, u)
}

// Refresh menerbitkan access token baru dari refresh token.
func (s *Service) Refresh(ctx context.Context, refreshToken string) (AuthResponse, error) {
	claims, err := s.tokens.ParseRefresh(refreshToken)
	if err != nil {
		return AuthResponse{}, apperror.Unauthorized("Refresh token tidak valid")
	}
	u, err := s.users.FindByID(ctx, claims.UserID)
	if err != nil || !u.IsActive {
		return AuthResponse{}, apperror.Unauthorized("Sesi tidak valid")
	}
	return s.buildAuthResponse(ctx, u)
}

// Me mengembalikan profil pengguna yang sedang login.
func (s *Service) Me(ctx context.Context, userID int64) (UserProfile, error) {
	u, err := s.users.FindByID(ctx, userID)
	if err != nil {
		return UserProfile{}, apperror.NotFound("Pengguna tidak ditemukan")
	}
	return s.profileFor(ctx, u)
}

// ChangePassword mengubah kata sandi pengguna berpassword lokal.
func (s *Service) ChangePassword(ctx context.Context, userID int64, req ChangePasswordRequest) error {
	u, err := s.users.FindByID(ctx, userID)
	if err != nil {
		return apperror.NotFound("Pengguna tidak ditemukan")
	}
	if !u.HasPassword() {
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

// ForgotPassword mengirim tautan reset password (bila email terdaftar).
func (s *Service) ForgotPassword(ctx context.Context, email string) error {
	email = strings.ToLower(strings.TrimSpace(email))
	u, err := s.users.FindByEmail(ctx, email)
	if err != nil {
		// Jangan bocorkan keberadaan email; kembalikan sukses tanpa mengirim.
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

// ResetPassword menetapkan kata sandi baru berdasarkan token reset.
func (s *Service) ResetPassword(ctx context.Context, req ResetPasswordRequest) error {
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

func (s *Service) buildAuthResponse(ctx context.Context, u user.User) (AuthResponse, error) {
	profile, err := s.profileFor(ctx, u)
	if err != nil {
		return AuthResponse{}, err
	}
	access, err := s.tokens.GenerateAccess(u.UserID, u.RoleID, u.Email, u.RoleName, u.EmailVerified())
	if err != nil {
		return AuthResponse{}, apperror.Internal("")
	}
	refresh, err := s.tokens.GenerateRefresh(u.UserID)
	if err != nil {
		return AuthResponse{}, apperror.Internal("")
	}
	return AuthResponse{
		AccessToken:  access,
		RefreshToken: refresh,
		ExpiresIn:    s.tokens.AccessExpireSeconds(),
		User:         profile,
	}, nil
}

func (s *Service) profileFor(ctx context.Context, u user.User) (UserProfile, error) {
	perms, err := s.perms.RolePermissions(ctx, u.RoleID)
	if err != nil {
		return UserProfile{}, apperror.Internal("")
	}
	if perms == nil {
		perms = []string{}
	}
	return UserProfile{
		UserID:        u.UserID,
		FullName:      u.FullName,
		Email:         u.Email,
		EmailVerified: u.EmailVerified(),
		Role:          u.RoleName,
		Permissions:   perms,
		PhotoURL:      u.PhotoURL.String,
	}, nil
}

func (s *Service) isDomainAllowed(email string) bool {
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
