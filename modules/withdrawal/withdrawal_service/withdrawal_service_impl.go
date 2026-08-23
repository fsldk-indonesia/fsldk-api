package withdrawal_service

import (
	"context"
	"crypto/hmac"
	"encoding/json"
	"errors"
	"log"
	"strconv"
	"strings"
	"time"

	"gorm.io/gorm"

	"fsldk-api/base/apperror"
	"fsldk-api/base/dbretry"
	"fsldk-api/base/dto"
	"fsldk-api/base/idgen"
	"fsldk-api/config"
	"fsldk-api/constants"
	"fsldk-api/modules/campaign/campaign_repository"
	"fsldk-api/modules/wallet/wallet_service"
	"fsldk-api/modules/withdrawal/withdrawal_dto"
	"fsldk-api/modules/withdrawal/withdrawal_model"
	"fsldk-api/modules/withdrawal/withdrawal_repository"
	"fsldk-api/pkg/bisatopup"
)

// staleProcessingThreshold adalah ambang waktu withdrawal PROCESSING
// dianggap butuh tinjauan manual admin (§11.7 — timeout tidak boleh
// langsung dianggap gagal, hanya "belum pasti").
const staleProcessingThreshold = 10 * time.Minute

var sortColumns = map[string]string{"createdDate": "w.createdDate", "amount": "w.amount"}

// ServiceImpl adalah implementasi Service.
type ServiceImpl struct {
	repo         withdrawal_repository.Repository
	campaignRepo campaign_repository.Repository
	walletSvc    wallet_service.Service
	gateway      bisatopup.Gateway
	db           *gorm.DB
	cfg          config.AppConfig
}

// NewService membuat Service withdrawal.
func NewService(repo withdrawal_repository.Repository, campaignRepo campaign_repository.Repository, walletSvc wallet_service.Service, gateway bisatopup.Gateway, db *gorm.DB, cfg config.AppConfig) Service {
	return &ServiceImpl{repo: repo, campaignRepo: campaignRepo, walletSvc: walletSvc, gateway: gateway, db: db, cfg: cfg}
}

func (s *ServiceImpl) Request(ctx context.Context, campaignID, requesterUserID int64, req withdrawal_dto.CreateRequest) (withdrawal_dto.Response, error) {
	camp, err := s.campaignRepo.FindByID(ctx, campaignID)
	if err != nil || camp.OwnerUserID != requesterUserID {
		return withdrawal_dto.Response{}, apperror.NotFound("Campaign tidak ditemukan")
	}

	// Fail-fast pre-check di luar transaksi — menghindari panggilan inquiry
	// gateway yang sia-sia untuk request yang jelas akan ditolak. Bukan
	// penjamin concurrency-safe (TOCTOU); penjamin sesungguhnya ada di
	// CountNonFinalByCampaignForUpdate di dalam transaksi di bawah.
	active, err := s.repo.CountNonFinalByCampaign(ctx, campaignID)
	if err != nil {
		return withdrawal_dto.Response{}, apperror.Internal("")
	}
	if active > 0 {
		return withdrawal_dto.Response{}, apperror.Conflict("Sudah ada permintaan penarikan yang masih berjalan untuk campaign ini")
	}

	// Beneficiary selalu direct dari campaign (bukan diinput ulang) —
	// divalidasi ulang via inquiry live karena rekening bisa saja sudah
	// tidak valid lagi sejak campaign disubmit (reuse ldksyahid-app).
	inq, ierr := s.gateway.InquiryBank(ctx, camp.BeneficiaryBankCode, camp.BeneficiaryAccountNumber)
	if ierr != nil || !strings.EqualFold(inq.Status, "SUCCESS") {
		return withdrawal_dto.Response{}, apperror.Unprocessable("Rekening penerima tidak valid, silakan hubungi admin untuk memperbarui data campaign")
	}
	fee, _ := strconv.ParseFloat(inq.Fee, 64)

	amount := roundRupiah(req.Amount)
	netAmount := float64(amount) - fee
	if netAmount <= 0 {
		return withdrawal_dto.Response{}, apperror.BadRequest("Nominal penarikan terlalu kecil setelah dikurangi biaya transfer")
	}

	now := time.Now()
	idemKey := strings.TrimSpace(req.IdempotencyKey)
	if idemKey == "" {
		idemKey = fallbackIdempotencyKey(campaignID, req.Amount, now)
	}

	var id int64
	// dbretry.Do menyerap deadlock InnoDB (error 1213/1205) yang bisa
	// muncul saat baris ms_campaign yang sama sedang dikunci flow lain
	// (donation callback, withdrawal reject/cancel) — lihat donation_service
	// untuk kasus serupa yang terbukti perlu retry ini di bawah beban nyata.
	err = dbretry.Do(func() error {
		return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			// Penjamin sesungguhnya "hanya satu withdrawal aktif per
			// campaign" — locking read ini membuat request konkuren untuk
			// campaign yang sama antre (gap lock), bukan sama-sama lolos
			// pre-check di atas lalu sama-sama insert.
			activeTx, cerr := s.repo.CountNonFinalByCampaignForUpdate(tx, campaignID)
			if cerr != nil {
				return cerr
			}
			if activeTx > 0 {
				return apperror.Conflict("Sudah ada permintaan penarikan yang masih berjalan untuk campaign ini")
			}

			id, cerr = s.repo.Create(tx, withdrawal_model.CreateParams{
				WithdrawalRef:            "WD-" + idgen.NewUUIDv4(),
				CampaignID:               campaignID,
				RequestedByUserID:        requesterUserID,
				Amount:                   float64(amount),
				Fee:                      fee,
				NetAmount:                netAmount,
				BeneficiaryBankCode:      camp.BeneficiaryBankCode,
				BeneficiaryAccountNumber: camp.BeneficiaryAccountNumber,
				BeneficiaryAccountHolder: camp.BeneficiaryAccountHolder,
				IdempotencyKey:           idemKey,
			})
			if cerr != nil {
				return cerr
			}
			return s.walletSvc.ReserveWithdrawal(tx, campaignID, id, float64(amount), requesterUserID, "")
		})
	})
	if errors.Is(err, withdrawal_repository.ErrDuplicateIdempotencyKey) {
		// Idempotent: request dengan key yang sama (double-click/refresh)
		// mendapat balik withdrawal yang sudah tercipta, bukan error.
		existing, ferr := s.repo.FindByIdempotencyKey(ctx, idemKey)
		if ferr != nil {
			return withdrawal_dto.Response{}, apperror.DuplicateRequest("")
		}
		return toResponse(existing), nil
	}
	if appErr, ok := err.(*apperror.AppError); ok {
		return withdrawal_dto.Response{}, appErr
	}
	if err != nil {
		return withdrawal_dto.Response{}, apperror.Internal("Gagal membuat permintaan penarikan")
	}
	return s.getResponse(ctx, id)
}

func (s *ServiceImpl) Cancel(ctx context.Context, withdrawalID, requesterUserID int64) error {
	w, err := s.repo.FindByID(ctx, withdrawalID)
	if err != nil || w.RequestedByUserID != requesterUserID {
		return apperror.NotFound("Penarikan tidak ditemukan")
	}
	if !isCancellableStatus(w.Status) {
		return apperror.InvalidStatusTransition("Penarikan tidak dapat dibatalkan pada status saat ini")
	}
	return dbretry.Do(func() error {
		return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			if err := s.repo.UpdateStatus(tx, withdrawalID, constants.WithdrawalStatusCancelled, withdrawal_model.StatusUpdateParams{}); err != nil {
				return err
			}
			return s.walletSvc.ReleaseWithdrawal(tx, w.CampaignID, withdrawalID, w.Amount, "")
		})
	})
}

func (s *ServiceImpl) Approve(ctx context.Context, withdrawalID, approverUserID int64) (withdrawal_dto.Response, error) {
	w, err := s.repo.FindByID(ctx, withdrawalID)
	if err != nil {
		return withdrawal_dto.Response{}, apperror.NotFound("Penarikan tidak ditemukan")
	}
	if w.Status != constants.WithdrawalStatusPendingApproval {
		return withdrawal_dto.Response{}, apperror.InvalidStatusTransition("Penarikan tidak dalam status menunggu persetujuan")
	}
	if w.RequestedByUserID == approverUserID {
		return withdrawal_dto.Response{}, apperror.Forbidden("Approver tidak boleh sama dengan pengaju penarikan (maker-checker)")
	}

	approver := approverUserID
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return s.repo.UpdateStatus(tx, withdrawalID, constants.WithdrawalStatusApproved, withdrawal_model.StatusUpdateParams{ApprovedByUserID: &approver})
	})
	if err != nil {
		return withdrawal_dto.Response{}, apperror.Internal("")
	}
	return s.getResponse(ctx, withdrawalID)
}

func (s *ServiceImpl) Reject(ctx context.Context, withdrawalID, approverUserID int64, reason string) error {
	w, err := s.repo.FindByID(ctx, withdrawalID)
	if err != nil {
		return apperror.NotFound("Penarikan tidak ditemukan")
	}
	if w.Status != constants.WithdrawalStatusPendingApproval {
		return apperror.InvalidStatusTransition("Penarikan tidak dalam status menunggu persetujuan")
	}
	reason = strings.TrimSpace(reason)
	if reason == "" {
		return apperror.BadRequest("Alasan penolakan wajib diisi")
	}

	return dbretry.Do(func() error {
		return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			if err := s.repo.UpdateStatus(tx, withdrawalID, constants.WithdrawalStatusRejected, withdrawal_model.StatusUpdateParams{RejectionReason: &reason}); err != nil {
				return err
			}
			return s.walletSvc.ReleaseWithdrawal(tx, w.CampaignID, withdrawalID, w.Amount, "")
		})
	})
}

func (s *ServiceImpl) Process(ctx context.Context, withdrawalID, actorUserID int64) (withdrawal_dto.Response, error) {
	w, err := s.repo.FindByID(ctx, withdrawalID)
	if err != nil {
		return withdrawal_dto.Response{}, apperror.NotFound("Penarikan tidak ditemukan")
	}
	if w.Status != constants.WithdrawalStatusApproved {
		return withdrawal_dto.Response{}, apperror.InvalidStatusTransition("Penarikan belum disetujui")
	}

	result, gerr := s.gateway.Disburse(ctx, bisatopup.DisburseParams{
		BankCode:      w.BeneficiaryBankCode,
		AccountNumber: w.BeneficiaryAccountNumber,
		Amount:        roundRupiah(w.NetAmount),
		Remark:        truncateNoEllipsis("Penarikan saldo "+w.CampaignTitle, 100),
		ReffID:        w.WithdrawalRef,
	})

	if gerr != nil && errors.Is(gerr, bisatopup.ErrGatewayRejected) {
		// Penolakan eksplisit gateway — beda dari timeout/tidak pasti, aman
		// untuk langsung FAILED + release reservasi segera (§11.7).
		_ = dbretry.Do(func() error {
			return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
				if err := s.repo.UpdateStatus(tx, withdrawalID, constants.WithdrawalStatusFailed, withdrawal_model.StatusUpdateParams{}); err != nil {
					return err
				}
				return s.walletSvc.ReleaseWithdrawal(tx, w.CampaignID, withdrawalID, w.Amount, "")
			})
		})
		return withdrawal_dto.Response{}, apperror.WithdrawalFailed("")
	}

	// Sukses ATAU kegagalan jaringan/timeout (bukan penolakan eksplisit) —
	// keduanya berujung PROCESSING, reservasi TIDAK dilepas: ketidakpastian
	// apakah gateway sudah memproses request sebelum timeout mencegah
	// pelepasan dini yang berisiko double-disbursement bila ternyata sudah
	// diproses gateway (§11.7). Status final ditentukan callback async atau
	// rekonsiliasi manual (ReconcileStaleProcessing).
	params := withdrawal_model.StatusUpdateParams{SetExecutedNow: true}
	if gerr == nil {
		statusID := result.IDStatus
		b, _ := json.Marshal(result)
		rj := string(b)
		params.GatewayStatusID = &statusID
		params.GatewayResponseJSON = &rj
	}
	if err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return s.repo.UpdateStatus(tx, withdrawalID, constants.WithdrawalStatusProcessing, params)
	}); err != nil {
		return withdrawal_dto.Response{}, apperror.Internal("")
	}
	if gerr != nil {
		return withdrawal_dto.Response{}, apperror.ProviderError("Gateway tidak merespons, status penarikan menunggu rekonsiliasi")
	}
	return s.getResponse(ctx, withdrawalID)
}

func (s *ServiceImpl) ProcessCallback(ctx context.Context, req withdrawal_dto.DisbursementCallbackRequest, urlSecret string) error {
	expected := s.cfg.BisatopupCallbackDisbursementSecretCrowdfunding
	if expected == "" || !hmac.Equal([]byte(expected), []byte(urlSecret)) {
		log.Printf("[BISATOPUP:WITHDRAWAL_CALLBACK] invalid URL secret, reffID=%s", req.ReffID)
		return apperror.Unauthorized("URL callback tidak valid")
	}

	newStatus := mapDisbursementStatus(req.StatusID)
	if newStatus == "" {
		// status_id tak dikenal/masih in-progress di sisi gateway — ack
		// tanpa proses, bukan error.
		return nil
	}

	return dbretry.Do(func() error {
		return s.processCallbackTx(ctx, req.ReffID, req.StatusID, newStatus)
	})
}

func (s *ServiceImpl) processCallbackTx(ctx context.Context, reffID string, statusID int, newStatus string) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		w, err := s.repo.FindByRefForUpdate(tx, reffID)
		if err != nil {
			return apperror.NotFound("Penarikan tidak ditemukan")
		}
		if isFinalWithdrawalStatus(w.Status) {
			// Idempotent ack — status final tidak pernah ditimpa ulang.
			return nil
		}

		sID := statusID
		params := withdrawal_model.StatusUpdateParams{GatewayStatusID: &sID, SetCompletedNow: newStatus == constants.WithdrawalStatusSuccess}
		if err := s.repo.UpdateStatus(tx, w.WithdrawalID, newStatus, params); err != nil {
			return err
		}
		if newStatus == constants.WithdrawalStatusFailed {
			return s.walletSvc.ReleaseWithdrawal(tx, w.CampaignID, w.WithdrawalID, w.Amount, "")
		}
		// SUCCESS tidak butuh ledger entry baru — saldo sudah didebit sejak
		// WITHDRAWAL_RESERVE di awal (§10.4).
		return nil
	})
}

func (s *ServiceImpl) Inquiry(ctx context.Context, req withdrawal_dto.InquiryRequest) (withdrawal_dto.InquiryResponse, error) {
	res, err := s.gateway.InquiryBank(ctx, req.BankCode, req.AccountNumber)
	if err != nil {
		if errors.Is(err, bisatopup.ErrGatewayRejected) {
			return withdrawal_dto.InquiryResponse{}, apperror.Unprocessable("Rekening tidak ditemukan atau tidak valid")
		}
		return withdrawal_dto.InquiryResponse{}, apperror.ProviderError("")
	}
	fee, _ := strconv.ParseFloat(res.Fee, 64)
	return withdrawal_dto.InquiryResponse{AccountHolder: res.AccountHolder, Fee: fee}, nil
}

func (s *ServiceImpl) ListBanks(ctx context.Context) ([]withdrawal_dto.BankListItem, error) {
	banks, err := s.gateway.BankList(ctx)
	if err != nil {
		return nil, apperror.ProviderError("")
	}
	out := make([]withdrawal_dto.BankListItem, 0, len(banks))
	for _, b := range banks {
		out = append(out, withdrawal_dto.BankListItem{BankCode: b.BankCode, Name: b.Name, Fee: float64(b.Fee), Status: b.Status})
	}
	return out, nil
}

func (s *ServiceImpl) ReconcileStaleProcessing(ctx context.Context) ([]withdrawal_dto.Response, error) {
	rows, err := s.repo.FindStaleProcessing(ctx, time.Now().Add(-staleProcessingThreshold))
	if err != nil {
		return nil, apperror.Internal("")
	}
	out := make([]withdrawal_dto.Response, 0, len(rows))
	for _, w := range rows {
		out = append(out, toResponse(w))
	}
	return out, nil
}

func (s *ServiceImpl) MyList(ctx context.Context, requesterUserID int64, q dto.ListQuery) ([]withdrawal_dto.Response, int, error) {
	uid := requesterUserID
	return s.list(ctx, withdrawal_dto.ListFilter{
		RequestedByUserID: &uid,
		Limit:             q.Limit,
		Offset:            q.Offset(),
		OrderBy:           q.OrderBy(sortColumns, "w.createdDate DESC"),
	})
}

func (s *ServiceImpl) CMSList(ctx context.Context, q dto.ListQuery, status string) ([]withdrawal_dto.Response, int, error) {
	return s.list(ctx, withdrawal_dto.ListFilter{
		Status:  status,
		Limit:   q.Limit,
		Offset:  q.Offset(),
		OrderBy: q.OrderBy(sortColumns, "w.createdDate DESC"),
	})
}

func (s *ServiceImpl) list(ctx context.Context, f withdrawal_dto.ListFilter) ([]withdrawal_dto.Response, int, error) {
	rows, total, err := s.repo.List(ctx, f)
	if err != nil {
		return nil, 0, apperror.Internal("")
	}
	out := make([]withdrawal_dto.Response, 0, len(rows))
	for _, w := range rows {
		out = append(out, toResponse(w))
	}
	return out, int(total), nil
}

func (s *ServiceImpl) getResponse(ctx context.Context, id int64) (withdrawal_dto.Response, error) {
	w, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return withdrawal_dto.Response{}, apperror.NotFound("Penarikan tidak ditemukan")
	}
	return toResponse(w), nil
}

func toResponse(w withdrawal_model.Withdrawal) withdrawal_dto.Response {
	resp := withdrawal_dto.Response{
		WithdrawalID:             w.WithdrawalID,
		WithdrawalRef:            w.WithdrawalRef,
		CampaignID:               w.CampaignID,
		CampaignTitle:            w.CampaignTitle,
		Amount:                   w.Amount,
		Fee:                      w.Fee,
		NetAmount:                w.NetAmount,
		BeneficiaryBankCode:      w.BeneficiaryBankCode,
		BeneficiaryAccountNumber: w.BeneficiaryAccountNumber,
		BeneficiaryAccountHolder: w.BeneficiaryAccountHolder,
		Status:                   w.Status,
		CreatedDate:              w.CreatedDate,
	}
	if w.RejectionReason.Valid {
		resp.RejectionReason = w.RejectionReason.String
	}
	if w.ApprovedDate.Valid {
		t := w.ApprovedDate.Time
		resp.ApprovedDate = &t
	}
	if w.CompletedDate.Valid {
		t := w.CompletedDate.Time
		resp.CompletedDate = &t
	}
	return resp
}

// isCancellableStatus menentukan status withdrawal yang masih boleh
// dibatalkan pengaju sendiri (§7.5 API design).
func isCancellableStatus(status string) bool {
	switch status {
	case constants.WithdrawalStatusRequested, constants.WithdrawalStatusSecurityCheck, constants.WithdrawalStatusPendingApproval:
		return true
	}
	return false
}

// isFinalWithdrawalStatus menentukan status yang tidak boleh ditimpa ulang
// oleh callback berikutnya (idempotency, cegah downgrade out-of-order).
func isFinalWithdrawalStatus(status string) bool {
	switch status {
	case constants.WithdrawalStatusSuccess, constants.WithdrawalStatusFailed, constants.WithdrawalStatusRejected,
		constants.WithdrawalStatusCancelled, constants.WithdrawalStatusReversed:
		return true
	}
	return false
}

// mapDisbursementStatus memetakan status_id Bisabiller ke status internal
// — direplikasi persis dari WithdrawalController::callback ldksyahid-app:
// {3,4}=SUCCESS, {5,14}=FAILED, lainnya dianggap belum final (diabaikan).
func mapDisbursementStatus(statusID int) string {
	switch statusID {
	case 3, 4:
		return constants.WithdrawalStatusSuccess
	case 5, 14:
		return constants.WithdrawalStatusFailed
	default:
		return ""
	}
}
