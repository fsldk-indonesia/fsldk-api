package donation_service

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"fsldk-api/base/apperror"
	"fsldk-api/base/dto"
	"fsldk-api/base/idgen"
	"fsldk-api/constants"
	"fsldk-api/modules/campaign/campaign_repository"
	"fsldk-api/modules/donation/donation_dto"
	"fsldk-api/modules/donation/donation_model"
	"fsldk-api/modules/donation/donation_repository"
	"fsldk-api/pkg/bisatopup"
)

var sortColumns = map[string]string{
	"createdDate": "d.createdDate",
	"amount":      "d.amount",
}

// ServiceImpl adalah implementasi Service.
type ServiceImpl struct {
	repo            donation_repository.Repository
	campaignRepo    campaign_repository.Repository
	mdrRatePercent  float64
	qrisExpiryHours int
}

// NewService membuat Service donation. mdrRatePercent dan qrisExpiryHours
// dibaca dari config.AppConfig (env BISATOPUP_QRIS_MDR_PERCENT_CROWDFUNDING/
// BISATOPUP_QRIS_EXPIRY_HOURS_CROWDFUNDING).
func NewService(repo donation_repository.Repository, campaignRepo campaign_repository.Repository, mdrRatePercent float64, qrisExpiryHours int) Service {
	return &ServiceImpl{repo: repo, campaignRepo: campaignRepo, mdrRatePercent: mdrRatePercent, qrisExpiryHours: qrisExpiryHours}
}

func (s *ServiceImpl) Create(ctx context.Context, slug string, donorUserID *int64, req donation_dto.CreateRequest) (donation_dto.Response, error) {
	camp, err := s.campaignRepo.FindBySlug(ctx, slug)
	if err != nil || camp.Status != constants.CampaignStatusPublished {
		return donation_dto.Response{}, apperror.NotFound("Campaign tidak ditemukan")
	}
	if camp.EndDate.Valid && time.Now().After(camp.EndDate.Time) {
		return donation_dto.Response{}, apperror.Unprocessable("Campaign sudah melewati batas waktu penggalangan dana")
	}
	if req.IsAnonymous && !camp.IsAnonymousAllowed {
		return donation_dto.Response{}, apperror.BadRequest("Campaign ini tidak mengizinkan donasi anonim")
	}

	now := time.Now()
	idemKey := strings.TrimSpace(req.IdempotencyKey)
	if idemKey == "" {
		idemKey = fallbackIdempotencyKey(req.DonorEmail, camp.CampaignID, req.Amount, now)
	}

	amount := roundRupiah(req.Amount)
	grossTotal := bisatopup.CalculateGrossTotal(amount, s.mdrRatePercent)
	adminFee := bisatopup.CalculateAdminFee(grossTotal, s.mdrRatePercent)

	id, err := s.repo.Create(ctx, donation_model.CreateParams{
		PublicRef:       idgen.NewUUIDv4(),
		CampaignID:      camp.CampaignID,
		DonorUserID:     nullInt64FromPtr(donorUserID),
		DonorName:       req.DonorName,
		DonorEmail:      req.DonorEmail,
		DonorPhone:      req.DonorPhone,
		DonorAge:        nullStringFrom(req.DonorAge),
		DonorDomicile:   nullStringFrom(req.DonorDomicile),
		DonorOccupation: nullStringFrom(req.DonorOccupation),
		IsAnonymous:     req.IsAnonymous,
		Message:         nullStringFrom(req.Message),
		Amount:          float64(amount),
		AdminFee:        float64(adminFee),
		TotalAmount:     float64(grossTotal),
		Gateway:         constants.DonationGatewayBisatopup,
		IdempotencyKey:  idemKey,
		ExpiredDate:     sql.NullTime{Time: now.Add(time.Duration(s.qrisExpiryHours) * time.Hour), Valid: true},
	})
	if errors.Is(err, donation_repository.ErrDuplicateIdempotencyKey) {
		// Idempotent: request dengan key yang sama (double-click/refresh)
		// mendapat balik donasi yang sudah tercipta, bukan error.
		existing, ferr := s.repo.FindByIdempotencyKey(ctx, idemKey)
		if ferr != nil {
			return donation_dto.Response{}, apperror.DuplicateRequest("")
		}
		return toResponse(existing), nil
	}
	if err != nil {
		return donation_dto.Response{}, apperror.Internal("Gagal membuat donasi")
	}
	return s.getResponse(ctx, id)
}

func (s *ServiceImpl) GetByPublicRef(ctx context.Context, publicRef string) (donation_dto.Response, error) {
	d, err := s.repo.FindByPublicRef(ctx, publicRef)
	if err != nil {
		return donation_dto.Response{}, apperror.NotFound("Donasi tidak ditemukan")
	}
	return toResponse(d), nil
}

func (s *ServiceImpl) Status(ctx context.Context, publicRef string) (donation_dto.StatusResponse, error) {
	d, err := s.repo.FindByPublicRef(ctx, publicRef)
	if err != nil {
		return donation_dto.StatusResponse{}, apperror.NotFound("Donasi tidak ditemukan")
	}
	return donation_dto.StatusResponse{PaymentStatus: d.PaymentStatus}, nil
}

func (s *ServiceImpl) MyList(ctx context.Context, donorUserID int64, q dto.ListQuery) ([]donation_dto.Response, int, error) {
	uid := donorUserID
	return s.list(ctx, donation_dto.ListFilter{
		DonorUserID: &uid,
		Limit:       q.Limit,
		Offset:      q.Offset(),
		OrderBy:     q.OrderBy(sortColumns, "d.createdDate DESC"),
	})
}

func (s *ServiceImpl) CMSList(ctx context.Context, q dto.ListQuery, campaignID int64, status string) ([]donation_dto.Response, int, error) {
	return s.list(ctx, donation_dto.ListFilter{
		CampaignID: campaignID,
		Status:     status,
		Limit:      q.Limit,
		Offset:     q.Offset(),
		OrderBy:    q.OrderBy(sortColumns, "d.createdDate DESC"),
	})
}

func (s *ServiceImpl) list(ctx context.Context, f donation_dto.ListFilter) ([]donation_dto.Response, int, error) {
	rows, total, err := s.repo.List(ctx, f)
	if err != nil {
		return nil, 0, apperror.Internal("")
	}
	out := make([]donation_dto.Response, 0, len(rows))
	for _, d := range rows {
		out = append(out, toResponse(d))
	}
	return out, int(total), nil
}

func (s *ServiceImpl) getResponse(ctx context.Context, id int64) (donation_dto.Response, error) {
	d, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return donation_dto.Response{}, apperror.NotFound("Donasi tidak ditemukan")
	}
	return toResponse(d), nil
}

func toResponse(d donation_model.Donation) donation_dto.Response {
	resp := donation_dto.Response{
		DonationID:    d.DonationID,
		PublicRef:     d.PublicRef,
		CampaignID:    d.CampaignID,
		CampaignTitle: d.CampaignTitle,
		CampaignSlug:  d.CampaignSlug,
		DonorName:     d.DonorName,
		IsAnonymous:   d.IsAnonymous,
		Amount:        d.Amount,
		AdminFee:      d.AdminFee,
		TotalAmount:   d.TotalAmount,
		PaymentStatus: d.PaymentStatus,
		Gateway:       d.Gateway,
		CreatedDate:   d.CreatedDate,
	}
	if d.Message.Valid {
		resp.Message = d.Message.String
	}
	if d.QrPayload.Valid {
		resp.QrPayload = d.QrPayload.String
	}
	if d.PaymentCode.Valid {
		resp.PaymentCode = d.PaymentCode.String
	}
	if d.PaymentLink.Valid {
		resp.PaymentLink = d.PaymentLink.String
	}
	if d.ExpiredDate.Valid {
		t := d.ExpiredDate.Time
		resp.ExpiredDate = &t
	}
	return resp
}

func nullStringFrom(s string) sql.NullString {
	s = strings.TrimSpace(s)
	return sql.NullString{String: s, Valid: s != ""}
}

func nullInt64FromPtr(id *int64) sql.NullInt64 {
	if id == nil {
		return sql.NullInt64{}
	}
	return sql.NullInt64{Int64: *id, Valid: true}
}
