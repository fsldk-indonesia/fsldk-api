package user_service

import (
	"context"
	"database/sql"
	"strings"

	"fsldk-api/base/apperror"
	"fsldk-api/base/dto"
	"fsldk-api/base/security"
	"fsldk-api/constants"
	"fsldk-api/modules/user/user_dto"
	"fsldk-api/modules/user/user_model"
	"fsldk-api/modules/user/user_repository"
	"fsldk-api/pkg/auditlog"
)

// sortColumns memetakan field sort yang diizinkan ke kolom database.
var sortColumns = map[string]string{
	"fullName":    "u.fullName",
	"email":       "u.email",
	"createdDate": "u.createdDate",
}

// ServiceImpl adalah implementasi Service.
type ServiceImpl struct {
	repo     user_repository.Repository
	orgScope OrgScopeChecker
	audit    *auditlog.Logger
}

// NewService membuat Service pengguna.
func NewService(repo user_repository.Repository, orgScope OrgScopeChecker, audit *auditlog.Logger) Service {
	return &ServiceImpl{repo: repo, orgScope: orgScope, audit: audit}
}

// nullIntValue mengekstrak nilai polos dari sql.NullInt64 untuk payload audit
// (nil bila kosong), supaya afterJSON/beforeJSON tidak membocorkan bentuk
// internal sql.NullInt64.
func nullIntValue(v sql.NullInt64) interface{} {
	if !v.Valid {
		return nil
	}
	return v.Int64
}

// nullStringValue adalah padanan nullIntValue untuk sql.NullString.
func nullStringValue(v sql.NullString) interface{} {
	if !v.Valid {
		return nil
	}
	return v.String
}

// resolvedPhotoURL menerapkan prioritas tampil foto profil: foto yang
// diunggah sendiri (customPhotoURL) selalu menang bila ada, baru fallback ke
// photoURL (disinkronkan otomatis dari akun Google) — inisial huruf adalah
// fallback terakhir, ditangani di frontend saat keduanya kosong.
func resolvedPhotoURL(u user_model.User) string {
	if u.CustomPhotoURL.Valid && u.CustomPhotoURL.String != "" {
		return u.CustomPhotoURL.String
	}
	return u.PhotoURL.String
}

// toResponse memetakan model User ke DTO Response (logika pemetaan berada di
// service, bukan pada model/dto, agar keduanya tetap murni struct data).
func toResponse(u user_model.User) user_dto.Response {
	resp := user_dto.Response{
		UserID:               u.UserID,
		FullName:             u.FullName,
		Email:                u.Email,
		RoleID:               u.RoleID,
		Role:                 u.RoleName,
		OrganizationTypeCode: u.OrganizationTypeCode.String,
		EmailVerified:        u.EmailVerifiedDate.Valid,
		IsActive:             u.IsActive,
		PhotoURL:             resolvedPhotoURL(u),
		HasGoogle:            u.GoogleID.Valid,
		HasPassword:          u.Password.Valid && u.Password.String != "",
	}
	if u.OrganizationID.Valid {
		id := u.OrganizationID.Int64
		resp.OrganizationID = &id
	}
	if u.WildcardTierAccess.Valid && u.WildcardTierAccess.String != "" {
		resp.WildcardTierAccess = strings.Split(u.WildcardTierAccess.String, ",")
	}
	return resp
}

// resolveProvisioning memvalidasi & mengunci organizationID/wildcardTierAccess
// pengguna baru sesuai kewenangan pemanggil:
//   - Super Admin / Puskomnas Verifikator (wildcard atau tipe PUSKOMNAS): bebas
//     memilih organisasi manapun (divalidasi via OrgScopeChecker) atau memberi wildcard.
//   - Puskomda Verifikator: hanya boleh untuk LDK di wilayahnya, tidak boleh memberi wildcard.
//   - LDK Admin: organizationID selalu dikunci ke organisasi sendiri, tidak boleh memberi wildcard.
func (s *ServiceImpl) resolveProvisioning(ctx context.Context, caller CallerScope, reqOrgID *int64, reqWildcard []string) (sql.NullInt64, sql.NullString, error) {
	wildcardStr := strings.Join(reqWildcard, ",")
	callerIsFree := caller.OrganizationTypeCode == constants.OrgTypePuskomnas || containsTier(caller.WildcardTierAccess, constants.OrgTypePuskomnas)

	switch {
	case callerIsFree:
		if reqOrgID != nil {
			ok, err := s.orgScope.IsAccessible(ctx, caller.OrganizationID, caller.OrganizationTypeCode, caller.WildcardTierAccess, *reqOrgID)
			if err != nil {
				return sql.NullInt64{}, sql.NullString{}, apperror.Internal("")
			}
			if !ok {
				return sql.NullInt64{}, sql.NullString{}, apperror.Forbidden("Organisasi tujuan di luar jangkauan akses Anda")
			}
			return sql.NullInt64{Int64: *reqOrgID, Valid: true}, sql.NullString{String: wildcardStr, Valid: wildcardStr != ""}, nil
		}
		return sql.NullInt64{}, sql.NullString{String: wildcardStr, Valid: wildcardStr != ""}, nil

	case caller.OrganizationTypeCode == constants.OrgTypePuskomda:
		if wildcardStr != "" {
			return sql.NullInt64{}, sql.NullString{}, apperror.Forbidden("Anda tidak dapat memberikan akses lintas organisasi")
		}
		if reqOrgID == nil {
			return sql.NullInt64{}, sql.NullString{}, apperror.BadRequest("organizationID wajib diisi")
		}
		ok, err := s.orgScope.IsAccessible(ctx, caller.OrganizationID, caller.OrganizationTypeCode, caller.WildcardTierAccess, *reqOrgID)
		if err != nil {
			return sql.NullInt64{}, sql.NullString{}, apperror.Internal("")
		}
		if !ok {
			return sql.NullInt64{}, sql.NullString{}, apperror.Forbidden("Organisasi tujuan di luar jangkauan akses Anda")
		}
		return sql.NullInt64{Int64: *reqOrgID, Valid: true}, sql.NullString{}, nil

	case caller.OrganizationTypeCode == constants.OrgTypeLDK:
		if wildcardStr != "" {
			return sql.NullInt64{}, sql.NullString{}, apperror.Forbidden("Anda tidak dapat memberikan akses lintas organisasi")
		}
		if caller.OrganizationID == nil {
			return sql.NullInt64{}, sql.NullString{}, apperror.Forbidden("Akun Anda tidak terhubung ke organisasi")
		}
		// organizationID selalu dikunci ke organisasi sendiri — nilai dari
		// client (bila ada) diabaikan untuk mencegah IDOR.
		return sql.NullInt64{Int64: *caller.OrganizationID, Valid: true}, sql.NullString{}, nil

	default:
		return sql.NullInt64{}, sql.NullString{}, apperror.Forbidden("Anda tidak memiliki hak menambahkan pengguna")
	}
}

func containsTier(wildcardTierAccess, tier string) bool {
	for _, t := range strings.Split(wildcardTierAccess, ",") {
		if t == tier {
			return true
		}
	}
	return false
}

func (s *ServiceImpl) List(ctx context.Context, q dto.ListQuery, roleID int64) ([]user_dto.Response, int, error) {
	users, total, err := s.repo.List(ctx, user_dto.ListFilter{
		Search:  q.Search,
		RoleID:  roleID,
		Limit:   q.Limit,
		Offset:  q.Offset(),
		OrderBy: q.OrderBy(sortColumns, "u.createdDate DESC"),
	})
	if err != nil {
		return nil, 0, apperror.Internal("")
	}
	out := make([]user_dto.Response, 0, len(users))
	for _, u := range users {
		out = append(out, toResponse(u))
	}
	return out, int(total), nil
}

func (s *ServiceImpl) SearchMentionable(ctx context.Context, search string, limit int) ([]user_dto.MentionSearchResult, error) {
	if limit <= 0 || limit > 20 {
		limit = 8
	}
	users, err := s.repo.SearchActive(ctx, strings.TrimSpace(search), limit)
	if err != nil {
		return nil, apperror.Internal("")
	}
	out := make([]user_dto.MentionSearchResult, 0, len(users))
	for _, u := range users {
		out = append(out, user_dto.MentionSearchResult{UserID: u.UserID, FullName: u.FullName, PhotoURL: resolvedPhotoURL(u)})
	}
	return out, nil
}

func (s *ServiceImpl) Get(ctx context.Context, id int64) (user_dto.Response, error) {
	u, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return user_dto.Response{}, apperror.NotFound("Pengguna tidak ditemukan")
	}
	return toResponse(u), nil
}

func (s *ServiceImpl) Create(ctx context.Context, req user_dto.CreateRequest, caller CallerScope) (user_dto.Response, error) {
	email := strings.ToLower(strings.TrimSpace(req.Email))
	exists, err := s.repo.ExistsByEmail(ctx, email)
	if err != nil {
		return user_dto.Response{}, apperror.Internal("")
	}
	if exists {
		return user_dto.Response{}, apperror.Conflict("Email sudah terdaftar")
	}
	orgID, wildcard, err := s.resolveProvisioning(ctx, caller, req.OrganizationID, req.WildcardTierAccess)
	if err != nil {
		return user_dto.Response{}, err
	}
	hashed, err := security.HashPassword(req.Password)
	if err != nil {
		return user_dto.Response{}, apperror.Internal("")
	}
	active := true
	if req.IsActive != nil {
		active = *req.IsActive
	}
	id, err := s.repo.Create(ctx, user_model.CreateParams{
		RoleID:             req.RoleID,
		OrganizationID:     orgID,
		WildcardTierAccess: wildcard,
		FullName:           strings.TrimSpace(req.FullName),
		Email:              email,
		Password:           sql.NullString{String: hashed, Valid: true},
		EmailVerified:      true,
		CreatedBy:          sql.NullInt64{Int64: caller.UserID, Valid: caller.UserID > 0},
	})
	if err != nil {
		return user_dto.Response{}, apperror.Internal("Gagal membuat pengguna")
	}
	if !active {
		_ = s.repo.SetActive(ctx, id, false, caller.UserID)
	}
	s.audit.LogUser(ctx, auditlog.Entry{
		ActorUserID: caller.UserID, ActorOrganizationID: caller.OrganizationID,
		Action: "CREATE", Entity: "ms_user", EntityID: id,
		After: map[string]interface{}{"roleID": req.RoleID, "organizationID": nullIntValue(orgID), "wildcardTierAccess": nullStringValue(wildcard)},
	})
	return s.Get(ctx, id)
}

func (s *ServiceImpl) Update(ctx context.Context, id int64, req user_dto.UpdateRequest, caller CallerScope) (user_dto.Response, error) {
	before, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return user_dto.Response{}, apperror.NotFound("Pengguna tidak ditemukan")
	}
	email := strings.ToLower(strings.TrimSpace(req.Email))
	exists, err := s.repo.ExistsByEmailExcept(ctx, email, id)
	if err != nil {
		return user_dto.Response{}, apperror.Internal("")
	}
	if exists {
		return user_dto.Response{}, apperror.Conflict("Email sudah dipakai pengguna lain")
	}
	orgID, wildcard, err := s.resolveProvisioning(ctx, caller, req.OrganizationID, req.WildcardTierAccess)
	if err != nil {
		return user_dto.Response{}, err
	}
	if err := s.repo.Update(ctx, id, strings.TrimSpace(req.FullName), email, req.RoleID, req.IsActive, orgID, wildcard, caller.UserID); err != nil {
		return user_dto.Response{}, apperror.Internal("")
	}
	s.audit.LogUser(ctx, auditlog.Entry{
		ActorUserID: caller.UserID, ActorOrganizationID: caller.OrganizationID,
		Action: "UPDATE", Entity: "ms_user", EntityID: id,
		Before: map[string]interface{}{"roleID": before.RoleID, "organizationID": nullIntValue(before.OrganizationID), "wildcardTierAccess": nullStringValue(before.WildcardTierAccess)},
		After:  map[string]interface{}{"roleID": req.RoleID, "organizationID": nullIntValue(orgID), "wildcardTierAccess": nullStringValue(wildcard)},
	})
	if strings.TrimSpace(req.Password) != "" {
		hashed, err := security.HashPassword(req.Password)
		if err != nil {
			return user_dto.Response{}, apperror.Internal("")
		}
		if err := s.repo.SetPassword(ctx, id, hashed, false); err != nil {
			return user_dto.Response{}, apperror.Internal("")
		}
	}
	return s.Get(ctx, id)
}

func (s *ServiceImpl) SetStatus(ctx context.Context, id int64, active bool, actorID int64) error {
	if _, err := s.repo.FindByID(ctx, id); err != nil {
		return apperror.NotFound("Pengguna tidak ditemukan")
	}
	if err := s.repo.SetActive(ctx, id, active, actorID); err != nil {
		return apperror.Internal("")
	}
	return nil
}

func (s *ServiceImpl) Delete(ctx context.Context, id, actorID int64) error {
	if id == actorID {
		return apperror.Unprocessable("Anda tidak dapat menghapus akun Anda sendiri")
	}
	if _, err := s.repo.FindByID(ctx, id); err != nil {
		return apperror.NotFound("Pengguna tidak ditemukan")
	}
	if err := s.repo.SoftDelete(ctx, id, actorID); err != nil {
		return apperror.Internal("")
	}
	return nil
}
