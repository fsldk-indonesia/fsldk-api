package dashboard_service

import (
	"context"
	"strings"

	"fsldk-api/base/apperror"
	"fsldk-api/constants"
	"fsldk-api/modules/dashboard/dashboard_dto"
	"fsldk-api/modules/dashboard/dashboard_repository"
	"fsldk-api/modules/submission_form/submission_form_repository"
)

// ServiceImpl adalah implementasi Service.
type ServiceImpl struct {
	repo     dashboard_repository.Repository
	formRepo submission_form_repository.Repository
}

// NewService membuat Service dashboard.
func NewService(repo dashboard_repository.Repository, formRepo submission_form_repository.Repository) Service {
	return &ServiceImpl{repo: repo, formRepo: formRepo}
}

func containsTier(wildcardTierAccess, tier string) bool {
	for _, t := range strings.Split(wildcardTierAccess, ",") {
		if t == tier {
			return true
		}
	}
	return false
}

// levelisasiFormID mencari formID form Levelisasi LDK. Belum tentu ada saat
// instalasi baru (admin belum membuat form) — dikembalikan 0 agar query
// agregat yang bergantung padanya secara alami menghasilkan hitungan kosong,
// bukan error.
func (s *ServiceImpl) levelisasiFormID(ctx context.Context) int64 {
	form, err := s.formRepo.FindFormByCode(ctx, constants.FormCodeLevelisasiLDK)
	if err != nil {
		return 0
	}
	return form.FormID
}

func (s *ServiceImpl) Summary(ctx context.Context, caller CallerScope) (dashboard_dto.Summary, error) {
	tier := caller.OrganizationTypeCode
	if tier == "" && containsTier(caller.WildcardTierAccess, constants.OrgTypePuskomnas) {
		tier = constants.OrgTypePuskomnas
	}

	formID := s.levelisasiFormID(ctx)

	switch tier {
	case constants.OrgTypeLDK:
		if caller.OrganizationID == nil {
			return dashboard_dto.Summary{}, apperror.Forbidden("Akun Anda tidak terhubung ke organisasi")
		}
		ldk, err := s.repo.LDKSummary(ctx, *caller.OrganizationID, formID)
		if err != nil {
			return dashboard_dto.Summary{}, apperror.Internal("")
		}
		return dashboard_dto.Summary{OrganizationTypeCode: tier, LDK: &ldk}, nil

	case constants.OrgTypePuskomda:
		if caller.OrganizationID == nil {
			return dashboard_dto.Summary{}, apperror.Forbidden("Akun Anda tidak terhubung ke organisasi")
		}
		counts, err := s.repo.StatusBuckets(ctx, formID, caller.OrganizationID)
		if err != nil {
			return dashboard_dto.Summary{}, apperror.Internal("")
		}
		totalLDK, err := s.repo.CountLDK(ctx, caller.OrganizationID)
		if err != nil {
			return dashboard_dto.Summary{}, apperror.Internal("")
		}
		kaderAktif, err := s.repo.CountActiveKader(ctx, caller.OrganizationID)
		if err != nil {
			return dashboard_dto.Summary{}, apperror.Internal("")
		}
		return dashboard_dto.Summary{OrganizationTypeCode: tier, Puskomda: &dashboard_dto.PuskomdaSummary{
			TotalLDK: totalLDK, StatusCounts: counts, TotalKaderAktif: kaderAktif,
		}}, nil

	case constants.OrgTypePuskomnas:
		counts, err := s.repo.StatusBuckets(ctx, formID, nil)
		if err != nil {
			return dashboard_dto.Summary{}, apperror.Internal("")
		}
		totalLDK, err := s.repo.CountLDK(ctx, nil)
		if err != nil {
			return dashboard_dto.Summary{}, apperror.Internal("")
		}
		totalPuskomda, err := s.repo.CountPuskomda(ctx)
		if err != nil {
			return dashboard_dto.Summary{}, apperror.Internal("")
		}
		levelEstablished, err := s.repo.CountLevelEstablished(ctx, nil)
		if err != nil {
			return dashboard_dto.Summary{}, apperror.Internal("")
		}
		kaderAktif, err := s.repo.CountActiveKader(ctx, nil)
		if err != nil {
			return dashboard_dto.Summary{}, apperror.Internal("")
		}
		levelDist, err := s.repo.LevelDistribution(ctx)
		if err != nil {
			return dashboard_dto.Summary{}, apperror.Internal("")
		}
		perPuskomda, err := s.repo.PerPuskomdaBreakdown(ctx)
		if err != nil {
			return dashboard_dto.Summary{}, apperror.Internal("")
		}
		return dashboard_dto.Summary{OrganizationTypeCode: tier, Puskomnas: &dashboard_dto.PuskomnasSummary{
			TotalLDKNasional: totalLDK, StatusCounts: counts, LevelEstablishedCount: levelEstablished,
			TotalPuskomda: totalPuskomda, TotalKaderAktifNasional: kaderAktif,
			LevelDistribution: levelDist, PerPuskomda: perPuskomda,
		}}, nil

	default:
		return dashboard_dto.Summary{OrganizationTypeCode: tier}, nil
	}
}
