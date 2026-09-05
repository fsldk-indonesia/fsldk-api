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
	orgScope OrgScopeResolver
}

// NewService membuat Service dashboard.
func NewService(repo dashboard_repository.Repository, formRepo submission_form_repository.Repository, orgScope OrgScopeResolver) Service {
	return &ServiceImpl{repo: repo, formRepo: formRepo, orgScope: orgScope}
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

// utamaSummary membangun ringkasan dashboard CMS Utama — metrik administrasi
// sistem, sengaja terpisah total dari metrik Levelisasi/Kader Puskomnas.
func (s *ServiceImpl) utamaSummary(ctx context.Context) (dashboard_dto.Summary, error) {
	totalUsers, err := s.repo.CountUsers(ctx)
	if err != nil {
		return dashboard_dto.Summary{}, apperror.Internal("")
	}
	totalNews, err := s.repo.CountNews(ctx)
	if err != nil {
		return dashboard_dto.Summary{}, apperror.Internal("")
	}
	totalArticles, err := s.repo.CountArticles(ctx)
	if err != nil {
		return dashboard_dto.Summary{}, apperror.Internal("")
	}
	totalShortlinks, err := s.repo.CountShortlinks(ctx)
	if err != nil {
		return dashboard_dto.Summary{}, apperror.Internal("")
	}
	unreadMessages, _ := s.repo.CountUnreadContactMessages(ctx)
	return dashboard_dto.Summary{
		OrganizationTypeCode: "FSLDK",
		Utama: &dashboard_dto.UtamaSummary{
			TotalUsers: totalUsers, TotalNews: totalNews, TotalArticles: totalArticles, TotalShortlinks: totalShortlinks, UnreadContactMessages: unreadMessages,
		},
	}, nil
}

func (s *ServiceImpl) Summary(ctx context.Context, caller CallerScope) (dashboard_dto.Summary, error) {
	// Shell CMS Utama TIDAK punya konsep organisasi/tier sama sekali — jangan
	// jatuhkan ke deteksi berbasis identitas caller (wildcard Super Admin
	// sebelumnya SELALU resolve ke tier Puskomnas, membuat dashboard CMS Utama
	// tak bisa dibedakan dari dashboard Puskomnas, miss-development-prompt-3.md
	// poin 5). `tier` di sini murni konteks shell yang diminta frontend.
	if caller.RequestedTier == "FSLDK" {
		return s.utamaSummary(ctx)
	}

	tier := caller.OrganizationTypeCode
	if tier == "" && containsTier(caller.WildcardTierAccess, constants.OrgTypePuskomnas) {
		tier = constants.OrgTypePuskomnas
	}
	orgID := caller.OrganizationID

	// Bila caller memilih organisasi lain lewat org-switcher (shell
	// cms-ldk/cms-puskomda), ringkasan mengikuti organisasi yang DIPILIH,
	// bukan home org caller — divalidasi accessible lebih dulu (IDOR-
	// prevention, sama seperti submission/report). Ini akar fix untuk
	// dashboard yang datanya tidak berubah saat pindah organisasi.
	if caller.RequestedOrganizationID != nil {
		ok, err := s.orgScope.IsAccessible(ctx, caller.OrganizationID, caller.OrganizationTypeCode, caller.WildcardTierAccess, *caller.RequestedOrganizationID)
		if err != nil {
			return dashboard_dto.Summary{}, apperror.Internal("")
		}
		if !ok {
			return dashboard_dto.Summary{}, apperror.Forbidden("Anda tidak memiliki akses ke organisasi ini")
		}
		targetType, err := s.orgScope.TypeCodeByID(ctx, *caller.RequestedOrganizationID)
		if err != nil {
			return dashboard_dto.Summary{}, err
		}
		tier = targetType
		orgID = caller.RequestedOrganizationID
	}

	formID := s.levelisasiFormID(ctx)

	switch tier {
	case constants.OrgTypeLDK:
		if orgID == nil {
			return dashboard_dto.Summary{}, apperror.Forbidden("Akun Anda tidak terhubung ke organisasi")
		}
		ldk, err := s.repo.LDKSummary(ctx, *orgID, formID)
		if err != nil {
			return dashboard_dto.Summary{}, apperror.Internal("")
		}
		return dashboard_dto.Summary{OrganizationTypeCode: tier, LDK: &ldk}, nil

	case constants.OrgTypePuskomda:
		if orgID == nil {
			return dashboard_dto.Summary{}, apperror.Forbidden("Akun Anda tidak terhubung ke organisasi")
		}
		counts, err := s.repo.StatusBuckets(ctx, formID, orgID)
		if err != nil {
			return dashboard_dto.Summary{}, apperror.Internal("")
		}
		totalLDK, err := s.repo.CountLDK(ctx, orgID)
		if err != nil {
			return dashboard_dto.Summary{}, apperror.Internal("")
		}
		kaderAktif, err := s.repo.CountActiveKader(ctx, orgID)
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
