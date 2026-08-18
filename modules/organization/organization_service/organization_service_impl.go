package organization_service

import (
	"context"
	"database/sql"
	"errors"
	"strings"

	"fsldk-api/base/apperror"
	"fsldk-api/base/dto"
	"fsldk-api/constants"
	"fsldk-api/modules/organization/organization_dto"
	"fsldk-api/modules/organization/organization_model"
	"fsldk-api/modules/organization/organization_repository"
)

var sortColumns = map[string]string{
	"organizationName": "o.organizationName",
	"organizationCode": "o.organizationCode",
	"createdDate":      "o.createdDate",
}

// ServiceImpl adalah implementasi Service.
type ServiceImpl struct {
	repo organization_repository.Repository
}

// NewService membuat Service organization.
func NewService(repo organization_repository.Repository) Service { return &ServiceImpl{repo: repo} }

func containsTier(wildcardTierAccess, tier string) bool {
	for _, t := range strings.Split(wildcardTierAccess, ",") {
		if t == tier {
			return true
		}
	}
	return false
}

func nullIfEmpty(s string) sql.NullString {
	s = strings.TrimSpace(s)
	return sql.NullString{String: s, Valid: s != ""}
}

func toResponse(o organization_model.Organization) organization_dto.Response {
	var parentID *int64
	if o.ParentOrganizationID.Valid {
		v := o.ParentOrganizationID.Int64
		parentID = &v
	}
	return organization_dto.Response{
		OrganizationID:         o.OrganizationID,
		OrganizationTypeCode:   o.OrganizationTypeCode,
		OrganizationName:       o.OrganizationName,
		OrganizationCode:       o.OrganizationCode,
		ParentOrganizationID:   parentID,
		ParentOrganizationName: o.ParentOrganizationName.String,
		ProvinceName:           o.ProvinceName.String,
		CityName:               o.CityName.String,
		ContactEmail:           o.ContactEmail.String,
		ContactPhone:           o.ContactPhone.String,
		IsActive:               o.IsActive,
		CreatedDate:            o.CreatedDate,
	}
}

func toResponses(list []organization_model.Organization) []organization_dto.Response {
	out := make([]organization_dto.Response, 0, len(list))
	for _, o := range list {
		out = append(out, toResponse(o))
	}
	return out
}

// IsAccessible mengevaluasi cascade (organizationID → tipe → parent/child)
// atau wildcardTierAccess pemanggil terhadap organisasi target.
func (s *ServiceImpl) IsAccessible(ctx context.Context, callerOrganizationID *int64, callerOrganizationTypeCode, wildcardTierAccess string, targetOrganizationID int64) (bool, error) {
	if wildcardTierAccess != "" {
		targetType, err := s.repo.TypeCodeByID(ctx, targetOrganizationID)
		if err != nil {
			if errors.Is(err, organization_repository.ErrNotFound) {
				return false, nil
			}
			return false, err
		}
		return containsTier(wildcardTierAccess, targetType), nil
	}
	if callerOrganizationID == nil {
		return false, nil
	}
	if *callerOrganizationID == targetOrganizationID {
		return true, nil
	}
	switch callerOrganizationTypeCode {
	case constants.OrgTypePuskomda:
		return s.repo.IsChildOf(ctx, targetOrganizationID, *callerOrganizationID)
	case constants.OrgTypePuskomnas:
		_, err := s.repo.TypeCodeByID(ctx, targetOrganizationID)
		if err != nil {
			if errors.Is(err, organization_repository.ErrNotFound) {
				return false, nil
			}
			return false, err
		}
		return true, nil
	default:
		return false, nil
	}
}

// TypeCodeByID mengembalikan organizationTypeCode organisasi.
func (s *ServiceImpl) TypeCodeByID(ctx context.Context, id int64) (string, error) {
	t, err := s.repo.TypeCodeByID(ctx, id)
	if err != nil {
		if errors.Is(err, organization_repository.ErrNotFound) {
			return "", apperror.NotFound("Organisasi tidak ditemukan")
		}
		return "", err
	}
	return t, nil
}

func (s *ServiceImpl) resolveAccessible(ctx context.Context, caller CallerScope, f organization_dto.ListFilter) ([]organization_model.Organization, int64, error) {
	switch {
	case caller.WildcardTierAccess != "":
		return s.repo.ListByTypes(ctx, strings.Split(caller.WildcardTierAccess, ","), f)
	case caller.OrganizationTypeCode == constants.OrgTypePuskomnas:
		return s.repo.ListAll(ctx, f)
	case caller.OrganizationTypeCode == constants.OrgTypePuskomda:
		if caller.OrganizationID == nil {
			return nil, 0, apperror.Forbidden("Akun Anda tidak terhubung ke organisasi")
		}
		return s.repo.ListSelfAndChildren(ctx, *caller.OrganizationID, f)
	case caller.OrganizationTypeCode == constants.OrgTypeLDK:
		if caller.OrganizationID == nil {
			return nil, 0, apperror.Forbidden("Akun Anda tidak terhubung ke organisasi")
		}
		org, err := s.repo.FindByID(ctx, *caller.OrganizationID)
		if err != nil {
			return nil, 0, apperror.NotFound("Organisasi tidak ditemukan")
		}
		return []organization_model.Organization{org}, 1, nil
	default:
		return nil, 0, apperror.Forbidden("Anda tidak memiliki akses organisasi")
	}
}

func (s *ServiceImpl) AccessibleOrganizationIDs(ctx context.Context, callerOrganizationID *int64, callerOrganizationTypeCode, wildcardTierAccess string) ([]int64, error) {
	orgs, _, err := s.resolveAccessible(ctx, CallerScope{
		OrganizationID:       callerOrganizationID,
		OrganizationTypeCode: callerOrganizationTypeCode,
		WildcardTierAccess:   wildcardTierAccess,
	}, organization_dto.ListFilter{})
	if err != nil {
		return nil, err
	}
	ids := make([]int64, 0, len(orgs))
	for _, o := range orgs {
		ids = append(ids, o.OrganizationID)
	}
	return ids, nil
}

// AccessibleOrganizationIDsForTarget menghitung cascade yang berakar pada
// targetOrganizationID sendiri: LDK → dirinya saja, Puskomda → dirinya +
// seluruh LDK anak, Puskomnas → seluruh organisasi nasional. Dipakai setelah
// targetOrganizationID divalidasi accessible terhadap caller (IsAccessible).
func (s *ServiceImpl) AccessibleOrganizationIDsForTarget(ctx context.Context, targetOrganizationID int64) ([]int64, error) {
	targetType, err := s.repo.TypeCodeByID(ctx, targetOrganizationID)
	if err != nil {
		if errors.Is(err, organization_repository.ErrNotFound) {
			return nil, apperror.NotFound("Organisasi tidak ditemukan")
		}
		return nil, err
	}
	switch targetType {
	case constants.OrgTypeLDK:
		return []int64{targetOrganizationID}, nil
	case constants.OrgTypePuskomda:
		orgs, _, err := s.repo.ListSelfAndChildren(ctx, targetOrganizationID, organization_dto.ListFilter{})
		if err != nil {
			return nil, err
		}
		ids := make([]int64, 0, len(orgs))
		for _, o := range orgs {
			ids = append(ids, o.OrganizationID)
		}
		return ids, nil
	case constants.OrgTypePuskomnas:
		orgs, _, err := s.repo.ListAll(ctx, organization_dto.ListFilter{})
		if err != nil {
			return nil, err
		}
		ids := make([]int64, 0, len(orgs))
		for _, o := range orgs {
			ids = append(ids, o.OrganizationID)
		}
		return ids, nil
	default:
		return []int64{targetOrganizationID}, nil
	}
}

func (s *ServiceImpl) AccessibleList(ctx context.Context, caller CallerScope, organizationTypeCode string, siblingOf *int64, q string) ([]organization_dto.MeOrganization, error) {
	if caller.OrganizationID == nil && caller.WildcardTierAccess == "" {
		return []organization_dto.MeOrganization{}, nil
	}
	orgs, _, err := s.resolveAccessible(ctx, caller, organization_dto.ListFilter{Search: q, OrganizationTypeCode: organizationTypeCode})
	if err != nil {
		return nil, err
	}
	if q == "" && siblingOf != nil {
		sibling, err := s.repo.FindByID(ctx, *siblingOf)
		if err == nil && sibling.ParentOrganizationID.Valid {
			parentID := sibling.ParentOrganizationID.Int64
			filtered := orgs[:0]
			for _, o := range orgs {
				if o.ParentOrganizationID.Valid && o.ParentOrganizationID.Int64 == parentID {
					filtered = append(filtered, o)
				}
			}
			orgs = filtered
		}
	}
	out := make([]organization_dto.MeOrganization, 0, len(orgs))
	for _, o := range orgs {
		var parentID *int64
		if o.ParentOrganizationID.Valid {
			v := o.ParentOrganizationID.Int64
			parentID = &v
		}
		out = append(out, organization_dto.MeOrganization{
			OrganizationID:       o.OrganizationID,
			OrganizationTypeCode: o.OrganizationTypeCode,
			OrganizationName:     o.OrganizationName,
			ParentOrganizationID: parentID,
		})
	}
	return out, nil
}

func (s *ServiceImpl) Directory(ctx context.Context, typeCode string) ([]organization_dto.DirectoryEntry, error) {
	orgs, err := s.repo.ListActiveByType(ctx, typeCode)
	if err != nil {
		return nil, apperror.Internal("")
	}
	out := make([]organization_dto.DirectoryEntry, 0, len(orgs))
	for _, o := range orgs {
		out = append(out, organization_dto.DirectoryEntry{
			OrganizationID:   o.OrganizationID,
			OrganizationName: o.OrganizationName,
			ProvinceName:     o.ProvinceName.String,
			CityName:         o.CityName.String,
		})
	}
	return out, nil
}

func (s *ServiceImpl) List(ctx context.Context, caller CallerScope, q dto.ListQuery, typeFilter string) ([]organization_dto.Response, int, error) {
	f := organization_dto.ListFilter{
		OrganizationTypeCode: typeFilter,
		Search:               q.Search,
		Limit:                q.Limit,
		Offset:               q.Offset(),
		OrderBy:              q.OrderBy(sortColumns, "o.organizationName ASC"),
	}

	// Bila caller memilih organisasi lain lewat org-switcher (shell
	// cms-puskomda/cms-puskomnas), daftar LDK/Puskomda mengikuti organisasi
	// yang DIPILIH (anak-anaknya), bukan seluruh cascade accessible caller —
	// sebelumnya "LDK Wilayah" selalu menampilkan seluruh cascade caller
	// (nasional untuk Puskomnas/wildcard) walau org-switcher sudah diganti.
	if caller.RequestedOrganizationID != nil {
		ok, err := s.IsAccessible(ctx, caller.OrganizationID, caller.OrganizationTypeCode, caller.WildcardTierAccess, *caller.RequestedOrganizationID)
		if err != nil {
			return nil, 0, apperror.Internal("")
		}
		if !ok {
			return nil, 0, apperror.Forbidden("Anda tidak memiliki akses ke organisasi ini")
		}
		targetType, err := s.repo.TypeCodeByID(ctx, *caller.RequestedOrganizationID)
		if err != nil {
			if errors.Is(err, organization_repository.ErrNotFound) {
				return nil, 0, apperror.NotFound("Organisasi tidak ditemukan")
			}
			return nil, 0, err
		}
		var orgs []organization_model.Organization
		var total int64
		switch targetType {
		case constants.OrgTypePuskomnas:
			orgs, total, err = s.repo.ListAll(ctx, f)
		case constants.OrgTypePuskomda:
			orgs, total, err = s.repo.ListSelfAndChildren(ctx, *caller.RequestedOrganizationID, f)
		default:
			return []organization_dto.Response{}, 0, nil
		}
		if err != nil {
			return nil, 0, apperror.Internal("")
		}
		return toResponses(orgs), int(total), nil
	}

	orgs, total, err := s.resolveAccessible(ctx, caller, f)
	if err != nil {
		return nil, 0, err
	}
	return toResponses(orgs), int(total), nil
}

func (s *ServiceImpl) Get(ctx context.Context, id int64) (organization_dto.Response, error) {
	o, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return organization_dto.Response{}, apperror.NotFound("Organisasi tidak ditemukan")
	}
	return toResponse(o), nil
}

func (s *ServiceImpl) Children(ctx context.Context, id int64) ([]organization_dto.Response, error) {
	list, err := s.repo.Children(ctx, id)
	if err != nil {
		return nil, apperror.Internal("")
	}
	return toResponses(list), nil
}

func (s *ServiceImpl) Create(ctx context.Context, caller CallerScope, req organization_dto.CreateRequest) (organization_dto.Response, error) {
	isPuskomnasTier := caller.OrganizationTypeCode == constants.OrgTypePuskomnas || containsTier(caller.WildcardTierAccess, constants.OrgTypePuskomnas)

	var parentID sql.NullInt64
	switch {
	case isPuskomnasTier:
		switch req.OrganizationTypeCode {
		case constants.OrgTypePuskomda:
			rootID, err := s.repo.RootPuskomnasID(ctx)
			if err != nil {
				return organization_dto.Response{}, apperror.Internal("")
			}
			parentID = sql.NullInt64{Int64: rootID, Valid: true}
		case constants.OrgTypeLDK:
			if req.ParentOrganizationID == nil {
				return organization_dto.Response{}, apperror.BadRequest("Puskomda induk wajib dipilih")
			}
			parentType, err := s.repo.TypeCodeByID(ctx, *req.ParentOrganizationID)
			if err != nil {
				return organization_dto.Response{}, apperror.BadRequest("Puskomda induk tidak ditemukan")
			}
			if parentType != constants.OrgTypePuskomda {
				return organization_dto.Response{}, apperror.BadRequest("Organisasi induk harus berupa Puskomda")
			}
			parentID = sql.NullInt64{Int64: *req.ParentOrganizationID, Valid: true}
		default:
			return organization_dto.Response{}, apperror.BadRequest("Tipe organisasi tidak valid")
		}
	case caller.OrganizationTypeCode == constants.OrgTypePuskomda:
		if req.OrganizationTypeCode != constants.OrgTypeLDK {
			return organization_dto.Response{}, apperror.Forbidden("Puskomda hanya dapat membuat organisasi LDK")
		}
		if caller.OrganizationID == nil {
			return organization_dto.Response{}, apperror.Forbidden("Akun Anda tidak terhubung ke organisasi")
		}
		// parentOrganizationID selalu dikunci ke organisasi pembuat sendiri,
		// nilai dari client (bila ada) diabaikan untuk mencegah IDOR.
		parentID = sql.NullInt64{Int64: *caller.OrganizationID, Valid: true}
	default:
		return organization_dto.Response{}, apperror.Forbidden("Anda tidak memiliki hak membuat organisasi")
	}

	exists, err := s.repo.ExistsByCode(ctx, strings.TrimSpace(req.OrganizationCode))
	if err != nil {
		return organization_dto.Response{}, apperror.Internal("")
	}
	if exists {
		return organization_dto.Response{}, apperror.Conflict("Kode organisasi sudah digunakan")
	}

	id, err := s.repo.Create(ctx, organization_model.CreateParams{
		OrganizationTypeCode: req.OrganizationTypeCode,
		ParentOrganizationID: parentID,
		OrganizationName:     strings.TrimSpace(req.OrganizationName),
		OrganizationCode:     strings.TrimSpace(req.OrganizationCode),
		ProvinceName:         nullIfEmpty(req.ProvinceName),
		CityName:             nullIfEmpty(req.CityName),
		ContactEmail:         nullIfEmpty(req.ContactEmail),
		ContactPhone:         nullIfEmpty(req.ContactPhone),
		CreatedBy:            sql.NullInt64{Int64: caller.UserID, Valid: caller.UserID > 0},
	})
	if err != nil {
		return organization_dto.Response{}, apperror.Internal("Gagal membuat organisasi")
	}
	return s.Get(ctx, id)
}

func (s *ServiceImpl) Update(ctx context.Context, id int64, req organization_dto.UpdateRequest, actorID int64) (organization_dto.Response, error) {
	if _, err := s.repo.FindByID(ctx, id); err != nil {
		return organization_dto.Response{}, apperror.NotFound("Organisasi tidak ditemukan")
	}
	if err := s.repo.Update(ctx, id, strings.TrimSpace(req.OrganizationName), req.ProvinceName, req.CityName, req.ContactEmail, req.ContactPhone, actorID); err != nil {
		return organization_dto.Response{}, apperror.Internal("")
	}
	return s.Get(ctx, id)
}

func (s *ServiceImpl) SetActive(ctx context.Context, caller CallerScope, id int64, active bool) error {
	org, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return apperror.NotFound("Organisasi tidak ditemukan")
	}

	isPuskomnasTier := caller.OrganizationTypeCode == constants.OrgTypePuskomnas || containsTier(caller.WildcardTierAccess, constants.OrgTypePuskomnas)
	switch {
	case isPuskomnasTier:
		// boleh menonaktifkan Puskomda maupun LDK manapun.
	case caller.OrganizationTypeCode == constants.OrgTypePuskomda:
		if caller.OrganizationID == nil || !org.ParentOrganizationID.Valid || org.ParentOrganizationID.Int64 != *caller.OrganizationID {
			return apperror.Forbidden("Anda hanya dapat menonaktifkan LDK di wilayah Anda")
		}
	default:
		return apperror.Forbidden("Anda tidak memiliki hak menonaktifkan organisasi")
	}

	if err := s.repo.SetActive(ctx, id, active, caller.UserID); err != nil {
		return apperror.Internal("")
	}
	return nil
}
