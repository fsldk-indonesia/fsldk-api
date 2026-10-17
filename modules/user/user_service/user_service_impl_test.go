package user_service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"testing"
	"time"

	"fsldk-api/constants"
	"fsldk-api/modules/auth/auth_dto"
	"fsldk-api/modules/auth/auth_service"
	"fsldk-api/modules/user/user_dto"
	"fsldk-api/modules/user/user_model"
	"fsldk-api/pkg/auditlog"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

// fakeUserRepository adalah implementasi user_repository.Repository
// in-memory — hanya method yang benar-benar dipakai skenario uji Create/
// Update di file ini yang berperilaku bermakna, sisanya no-op.
type fakeUserRepository struct {
	byID                   map[int64]user_model.User
	nextID                 int64
	createCalls            []user_model.CreateParams
	setEmailVerifiedCalls  []setEmailVerifiedCall
}

type setEmailVerifiedCall struct {
	id       int64
	verified bool
}

func newFakeUserRepo() *fakeUserRepository {
	return &fakeUserRepository{byID: map[int64]user_model.User{}}
}

func (f *fakeUserRepository) FindByID(ctx context.Context, id int64) (user_model.User, error) {
	u, ok := f.byID[id]
	if !ok {
		return user_model.User{}, errors.New("not found")
	}
	return u, nil
}
func (f *fakeUserRepository) FindByEmail(ctx context.Context, email string) (user_model.User, error) {
	return user_model.User{}, errors.New("not found")
}
func (f *fakeUserRepository) FindByGoogleID(ctx context.Context, googleID string) (user_model.User, error) {
	return user_model.User{}, errors.New("not found")
}
func (f *fakeUserRepository) ExistsByEmail(ctx context.Context, email string) (bool, error) {
	return false, nil
}
func (f *fakeUserRepository) ExistsByEmailExcept(ctx context.Context, email string, exceptID int64) (bool, error) {
	return false, nil
}
func (f *fakeUserRepository) Create(ctx context.Context, p user_model.CreateParams) (int64, error) {
	f.nextID++
	id := f.nextID
	f.createCalls = append(f.createCalls, p)
	u := user_model.User{
		UserID: id, RoleID: p.RoleID, FullName: p.FullName, Email: p.Email,
		Password: p.Password, GoogleID: p.GoogleID, PhotoURL: p.PhotoURL,
		OrganizationID: p.OrganizationID, WildcardTierAccess: p.WildcardTierAccess,
		IsActive: true, CreatedBy: p.CreatedBy,
	}
	if p.EmailVerified {
		u.EmailVerifiedDate = sql.NullTime{Time: time.Now(), Valid: true}
	}
	f.byID[id] = u
	return id, nil
}
func (f *fakeUserRepository) List(ctx context.Context, filt user_dto.ListFilter) ([]user_model.User, int64, error) {
	return nil, 0, nil
}
func (f *fakeUserRepository) SearchActive(ctx context.Context, search string, limit int) ([]user_model.User, error) {
	return nil, nil
}
func (f *fakeUserRepository) Update(ctx context.Context, id int64, fullName, email string, roleID int64, isActive bool, organizationID sql.NullInt64, wildcardTierAccess sql.NullString, updatedBy int64) error {
	return nil
}
func (f *fakeUserRepository) UpdateContactInfo(ctx context.Context, id int64, phoneNumber, address sql.NullString) error {
	return nil
}
func (f *fakeUserRepository) SetActive(ctx context.Context, id int64, active bool, updatedBy int64) error {
	return nil
}
func (f *fakeUserRepository) SetPassword(ctx context.Context, id int64, hashed string, mustChange bool) error {
	return nil
}
func (f *fakeUserRepository) LinkGoogle(ctx context.Context, id int64, googleID string, markVerified bool) error {
	return nil
}
func (f *fakeUserRepository) UpdatePhoto(ctx context.Context, id int64, photoURL string) error {
	return nil
}
func (f *fakeUserRepository) UpdateCustomPhoto(ctx context.Context, id int64, photoURL string) error {
	return nil
}
func (f *fakeUserRepository) MarkEmailVerified(ctx context.Context, id int64) error { return nil }
func (f *fakeUserRepository) SetEmailVerified(ctx context.Context, id int64, verified bool) error {
	f.setEmailVerifiedCalls = append(f.setEmailVerifiedCalls, setEmailVerifiedCall{id, verified})
	u := f.byID[id]
	if verified {
		if !u.EmailVerifiedDate.Valid {
			u.EmailVerifiedDate = sql.NullTime{Time: time.Now(), Valid: true}
		}
	} else {
		u.EmailVerifiedDate = sql.NullTime{}
	}
	f.byID[id] = u
	return nil
}
func (f *fakeUserRepository) SoftDelete(ctx context.Context, id int64, updatedBy int64) error {
	return nil
}
func (f *fakeUserRepository) LogLogin(ctx context.Context, userID int64, ip, ua, status string) error {
	return nil
}
func (f *fakeUserRepository) TokenVersion(ctx context.Context, id int64) (int, error) { return 0, nil }
func (f *fakeUserRepository) BumpTokenVersion(ctx context.Context, id int64) error    { return nil }

// fakeOrgScope selalu mengizinkan — skenario uji di file ini tidak mengirim
// OrganizationID pada request (lihat caller.OrganizationTypeCode =
// OrgTypePuskomnas di bawah), jadi IsAccessible tidak pernah benar-benar
// dipanggil, tapi tetap diimplementasikan untuk memenuhi OrgScopeChecker.
type fakeOrgScope struct{}

func (fakeOrgScope) IsAccessible(ctx context.Context, callerOrganizationID *int64, callerOrganizationTypeCode, wildcardTierAccess string, targetOrganizationID int64) (bool, error) {
	return true, nil
}

// fakeAuthService adalah implementasi auth_service.Service no-op — hanya
// SendVerificationEmail yang berperilaku bermakna (dicatat), dipakai
// menguji bahwa user_service.Create memicunya saat verifikasi TIDAK di-bypass.
type fakeAuthService struct {
	sendVerificationCalls []sendVerificationCall
}

type sendVerificationCall struct {
	userID      int64
	name, email string
}

func (f *fakeAuthService) SendVerificationEmail(ctx context.Context, userID int64, name, email string) error {
	f.sendVerificationCalls = append(f.sendVerificationCalls, sendVerificationCall{userID, name, email})
	return nil
}
func (f *fakeAuthService) Register(ctx context.Context, req auth_dto.RegisterRequest) (auth_dto.RegisterResponse, error) {
	return auth_dto.RegisterResponse{}, nil
}
func (f *fakeAuthService) VerifyEmail(ctx context.Context, token string) error         { return nil }
func (f *fakeAuthService) ResendVerification(ctx context.Context, userID int64) error { return nil }
func (f *fakeAuthService) Login(ctx context.Context, req auth_dto.LoginRequest, ip, ua string) (auth_dto.AuthResponse, error) {
	return auth_dto.AuthResponse{}, nil
}
func (f *fakeAuthService) LoginGoogle(ctx context.Context, idToken, ip, ua string) (auth_dto.AuthResponse, error) {
	return auth_dto.AuthResponse{}, nil
}
func (f *fakeAuthService) Refresh(ctx context.Context, refreshToken string) (auth_dto.AuthResponse, error) {
	return auth_dto.AuthResponse{}, nil
}
func (f *fakeAuthService) Me(ctx context.Context, userID int64) (auth_dto.UserProfile, error) {
	return auth_dto.UserProfile{}, nil
}
func (f *fakeAuthService) ChangePassword(ctx context.Context, userID int64, req auth_dto.ChangePasswordRequest) error {
	return nil
}
func (f *fakeAuthService) UpdateContact(ctx context.Context, userID int64, req auth_dto.UpdateContactRequest) (auth_dto.UserProfile, error) {
	return auth_dto.UserProfile{}, nil
}
func (f *fakeAuthService) UpdatePhoto(ctx context.Context, userID int64, req auth_dto.UpdatePhotoRequest) (auth_dto.UserProfile, error) {
	return auth_dto.UserProfile{}, nil
}
func (f *fakeAuthService) ForgotPassword(ctx context.Context, email string) error { return nil }
func (f *fakeAuthService) ResetPassword(ctx context.Context, req auth_dto.ResetPasswordRequest) error {
	return nil
}

var _ auth_service.Service = (*fakeAuthService)(nil)

// newTestAuditLogger membuat *auditlog.Logger sungguhan (bukan nil) ber-DB
// SQLite in-memory — Create()/Update() memanggil s.audit.LogUser(...) tanpa
// syarat, dan *auditlog.Logger adalah struct konkret (bukan interface),
// jadi tidak bisa di-nil-kan begitu saja seperti skenario service lain yang
// tidak pernah mencapai baris logging itu (lihat report_service test).
func newTestAuditLogger(t *testing.T) *auditlog.Logger {
	t.Helper()
	dsn := fmt.Sprintf("file:%s_audit?mode=memory&cache=shared", t.Name())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("gorm.Open() error = %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("db.DB() error = %v", err)
	}
	sqlDB.SetMaxOpenConns(1)
	if err := db.Exec(`CREATE TABLE tr_user_audit_log (
		actorUserID INTEGER, actorOrganizationID INTEGER, action TEXT, entity TEXT, entityID INTEGER,
		beforeJSON TEXT, afterJSON TEXT, ipAddress TEXT, metadata TEXT, createdDate DATETIME
	)`).Error; err != nil {
		t.Fatalf("create audit table error = %v", err)
	}
	return auditlog.New(db)
}

func newTestService(t *testing.T, repo *fakeUserRepository, auth auth_service.Service) Service {
	return NewService(repo, fakeOrgScope{}, newTestAuditLogger(t), auth)
}

func freeCaller() CallerScope {
	return CallerScope{UserID: 1, OrganizationTypeCode: constants.OrgTypePuskomnas}
}

func TestServiceImpl_Create_BypassVerification_SkipsVerificationEmail(t *testing.T) {
	repo := newFakeUserRepo()
	auth := &fakeAuthService{}
	svc := newTestService(t, repo, auth)

	req := user_dto.CreateRequest{FullName: "Budi", Email: "budi@example.com", RoleID: 1, Password: "password123", EmailVerified: true}
	if _, err := svc.Create(context.Background(), req, freeCaller()); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if len(repo.createCalls) != 1 {
		t.Fatalf("expected 1 repo.Create() call, got %d", len(repo.createCalls))
	}
	if !repo.createCalls[0].EmailVerified {
		t.Error("CreateParams.EmailVerified = false, want true (bypass requested)")
	}
	if len(auth.sendVerificationCalls) != 0 {
		t.Errorf("expected SendVerificationEmail NOT to be called when bypassing, got %d call(s)", len(auth.sendVerificationCalls))
	}
}

func TestServiceImpl_Create_RequiresVerification_SendsVerificationEmail(t *testing.T) {
	repo := newFakeUserRepo()
	auth := &fakeAuthService{}
	svc := newTestService(t, repo, auth)

	req := user_dto.CreateRequest{FullName: "Siti", Email: "siti@example.com", RoleID: 1, Password: "password123", EmailVerified: false}
	if _, err := svc.Create(context.Background(), req, freeCaller()); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if len(repo.createCalls) != 1 {
		t.Fatalf("expected 1 repo.Create() call, got %d", len(repo.createCalls))
	}
	if repo.createCalls[0].EmailVerified {
		t.Error("CreateParams.EmailVerified = true, want false (verification requested, not bypassed)")
	}
	if len(auth.sendVerificationCalls) != 1 {
		t.Fatalf("expected 1 SendVerificationEmail() call, got %d", len(auth.sendVerificationCalls))
	}
	got := auth.sendVerificationCalls[0]
	if got.name != "Siti" || got.email != "siti@example.com" {
		t.Errorf("SendVerificationEmail() name/email = %q/%q, want Siti/siti@example.com", got.name, got.email)
	}
}

func TestServiceImpl_Update_VerifiesEmail_WhenRequestedTrue(t *testing.T) {
	repo := newFakeUserRepo()
	repo.byID[7] = user_model.User{UserID: 7, RoleID: 1, FullName: "Budi", Email: "budi@example.com", IsActive: true}
	auth := &fakeAuthService{}
	svc := newTestService(t, repo, auth)

	req := user_dto.UpdateRequest{FullName: "Budi", Email: "budi@example.com", RoleID: 1, IsActive: true, EmailVerified: true}
	if _, err := svc.Update(context.Background(), 7, req, freeCaller()); err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	if len(repo.setEmailVerifiedCalls) != 1 {
		t.Fatalf("expected 1 SetEmailVerified() call, got %d", len(repo.setEmailVerifiedCalls))
	}
	got := repo.setEmailVerifiedCalls[0]
	if got.id != 7 || !got.verified {
		t.Errorf("SetEmailVerified() = (id=%d, verified=%v), want (id=7, verified=true)", got.id, got.verified)
	}
	if len(auth.sendVerificationCalls) != 0 {
		t.Errorf("verifying directly must not send an email, got %d call(s)", len(auth.sendVerificationCalls))
	}
}

func TestServiceImpl_Update_RevokesVerification_AndResendsEmail_WhenPreviouslyVerified(t *testing.T) {
	repo := newFakeUserRepo()
	repo.byID[7] = user_model.User{
		UserID: 7, RoleID: 1, FullName: "Budi", Email: "budi@example.com", IsActive: true,
		EmailVerifiedDate: sql.NullTime{Time: time.Now(), Valid: true},
	}
	auth := &fakeAuthService{}
	svc := newTestService(t, repo, auth)

	req := user_dto.UpdateRequest{FullName: "Budi", Email: "budi@example.com", RoleID: 1, IsActive: true, EmailVerified: false}
	if _, err := svc.Update(context.Background(), 7, req, freeCaller()); err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	if len(repo.setEmailVerifiedCalls) != 1 {
		t.Fatalf("expected 1 SetEmailVerified() call, got %d", len(repo.setEmailVerifiedCalls))
	}
	got := repo.setEmailVerifiedCalls[0]
	if got.id != 7 || got.verified {
		t.Errorf("SetEmailVerified() = (id=%d, verified=%v), want (id=7, verified=false)", got.id, got.verified)
	}
	// Admin sengaja mencabut verifikasi akun yang sebelumnya sudah
	// terverifikasi — pengguna butuh jalan untuk verifikasi lagi, jadi email
	// verifikasi WAJIB dikirim ulang (beda dari kasus "sudah tidak
	// terverifikasi, tetap tidak dicentang" di bawah, yang TIDAK resend).
	if len(auth.sendVerificationCalls) != 1 {
		t.Fatalf("expected 1 SendVerificationEmail() call (revoking a previously-verified account must resend), got %d", len(auth.sendVerificationCalls))
	}
	gotMail := auth.sendVerificationCalls[0]
	if gotMail.userID != 7 || gotMail.name != "Budi" || gotMail.email != "budi@example.com" {
		t.Errorf("SendVerificationEmail() = %+v, want userID=7 name=Budi email=budi@example.com", gotMail)
	}
}

func TestServiceImpl_Update_NoResend_WhenAlreadyUnverifiedAndStaysUnverified(t *testing.T) {
	repo := newFakeUserRepo()
	repo.byID[7] = user_model.User{UserID: 7, RoleID: 1, FullName: "Budi", Email: "budi@example.com", IsActive: true}
	auth := &fakeAuthService{}
	svc := newTestService(t, repo, auth)

	req := user_dto.UpdateRequest{FullName: "Budi", Email: "budi@example.com", RoleID: 1, IsActive: true, EmailVerified: false}
	if _, err := svc.Update(context.Background(), 7, req, freeCaller()); err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	// Checkbox sudah tidak tercentang sebelumnya dan tetap tidak tercentang —
	// bukan aksi "cabut verifikasi" yang disengaja, jadi TIDAK boleh resend
	// email di setiap kali form ini disimpan (mis. admin cuma mengubah nama).
	if len(auth.sendVerificationCalls) != 0 {
		t.Errorf("expected no SendVerificationEmail() call when status is unchanged, got %d", len(auth.sendVerificationCalls))
	}
}
