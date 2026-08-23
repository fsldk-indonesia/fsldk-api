package donation_service

import (
	"context"
	"database/sql"
	"errors"
	"log"
	"math"
	"math/rand"
	"strconv"
	"strings"
	"time"

	"github.com/go-sql-driver/mysql"
	"gorm.io/gorm"

	"fsldk-api/base/apperror"
	"fsldk-api/base/dto"
	"fsldk-api/base/idgen"
	"fsldk-api/config"
	"fsldk-api/constants"
	"fsldk-api/modules/campaign/campaign_repository"
	"fsldk-api/modules/donation/donation_dto"
	"fsldk-api/modules/donation/donation_model"
	"fsldk-api/modules/donation/donation_repository"
	"fsldk-api/modules/wallet/wallet_service"
	"fsldk-api/pkg/bisatopup"
)

var sortColumns = map[string]string{
	"createdDate": "d.createdDate",
	"amount":      "d.amount",
}

// ServiceImpl adalah implementasi Service.
type ServiceImpl struct {
	repo         donation_repository.Repository
	campaignRepo campaign_repository.Repository
	walletSvc    wallet_service.Service
	gateway      bisatopup.Gateway
	db           *gorm.DB // hanya dipakai ProcessCallback, yang membuka transaksinya sendiri
	cfg          config.AppConfig
}

// NewService membuat Service donation.
func NewService(repo donation_repository.Repository, campaignRepo campaign_repository.Repository, walletSvc wallet_service.Service, gateway bisatopup.Gateway, db *gorm.DB, cfg config.AppConfig) Service {
	return &ServiceImpl{repo: repo, campaignRepo: campaignRepo, walletSvc: walletSvc, gateway: gateway, db: db, cfg: cfg}
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
	grossTotal := bisatopup.CalculateGrossTotal(amount, s.cfg.BisatopupQrisMdrPercentCrowdfunding)
	adminFee := bisatopup.CalculateAdminFee(grossTotal, s.cfg.BisatopupQrisMdrPercentCrowdfunding)
	expiredAt := now.Add(time.Duration(s.cfg.BisatopupQrisExpiryHoursCrowdfunding) * time.Hour)

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
		ExpiredDate:     sql.NullTime{Time: expiredAt, Valid: true},
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

	// transactionID adalah identitas donasi ini di sisi Bisabiller (echoed
	// balik pada callback) — sengaja terpisah dari publicRef (referensi
	// publik) meski keduanya sama-sama UUID non-enumerable.
	transactionID := idgen.NewUUIDv4()
	qris, gerr := s.gateway.CreateQRISTransaction(ctx, bisatopup.CreateQRISTransactionParams{
		TransactionID:   transactionID,
		Nominal:         grossTotal,
		ExpiredDate:     expiredAt,
		TransactionName: truncateNoEllipsis("Donasi "+camp.Title, 49),
		TransactionDesc: truncateNoEllipsis(donationDesc(req.Message), 100),
		CustomerName:    req.DonorName,
		CustomerEmail:   req.DonorEmail,
		CustomerNumber:  req.DonorPhone,
	})
	if gerr != nil {
		// Tidak ada retry otomatis di sini (lihat pkg/bisatopup) — retry
		// create-transaction berisiko membuat transaksi duplikat di sisi
		// gateway. User diminta mengulang secara eksplisit (donasi baru).
		_ = s.repo.MarkGatewayFailed(ctx, id)
		if errors.Is(gerr, bisatopup.ErrGatewayRejected) {
			return donation_dto.Response{}, apperror.PaymentFailed("")
		}
		return donation_dto.Response{}, apperror.ProviderError("")
	}

	if err := s.repo.UpdateGatewayResult(ctx, id, donation_model.GatewayResultParams{
		ExternalTransactionID: transactionID,
		QrPayload:             qris.QrCode,
		PaymentCode:           qris.PaymentCode,
		PaymentLink:           qris.PaymentLinks,
	}); err != nil {
		return donation_dto.Response{}, apperror.Internal("")
	}
	return s.getResponse(ctx, id)
}

// ProcessCallback menangani webhook payment callback Bisabiller.
func (s *ServiceImpl) ProcessCallback(ctx context.Context, req donation_dto.CallbackRequest) error {
	if isTestCallback(req, s.cfg.BisatopupEnvCrowdfunding) {
		log.Printf("[BISATOPUP:CALLBACK] test event acknowledged, transactionID=%s", req.TransactionID)
		return nil
	}

	if s.cfg.BisatopupEnforceCallbackSignatureCrowdfunding &&
		!bisatopup.VerifySignature(s.cfg.BisatopupUsernameCrowdfunding, req.TransactionID, req.Signature) {
		log.Printf("[BISATOPUP:CALLBACK] signature mismatch, transactionID=%s", req.TransactionID)
		return apperror.Unauthorized("Signature callback tidak valid")
	}

	actualTotal, err := strconv.ParseFloat(req.TransactionTotal, 64)
	if err != nil {
		return apperror.BadRequest("transaction_total tidak valid")
	}
	newStatus := mapGatewayStatus(req.StatusID)

	// Beberapa campaign populer bisa menerima banyak donasi PAID nyaris
	// bersamaan — baris ms_campaign yang sama dikunci FOR UPDATE oleh
	// wallet_service untuk tiap donasi (lihat 10-balance-ledger.md §10.9).
	// InnoDB bisa memilih salah satu transaksi sebagai "deadlock victim"
	// (error 1213) di bawah kontensi tinggi meski tidak ada bug logika —
	// ini normal untuk row lock yang sama diperebutkan banyak transaksi
	// pendek, bukan indikasi lost update (transaksi yang jadi victim
	// otomatis di-rollback utuh oleh InnoDB). Retry singkat di sini
	// memastikan callback tetap sukses tanpa mengharuskan Bisabiller
	// mengirim ulang webhook-nya sendiri.
	var err2 error
	for attempt := 0; attempt < maxDeadlockRetries; attempt++ {
		err2 = s.processCallbackTx(ctx, req, newStatus, actualTotal)
		if err2 == nil || !isRetryableDBError(err2) {
			return err2
		}
		// Exponential backoff + jitter: dasar 20ms digandakan tiap attempt,
		// dibatasi 300ms, ditambah jitter acak agar transaksi yang tadi
		// bertabrakan tidak langsung bertabrakan lagi secara serempak.
		backoff := 20 * time.Millisecond * time.Duration(1<<attempt)
		if backoff > 300*time.Millisecond {
			backoff = 300 * time.Millisecond
		}
		time.Sleep(backoff + time.Duration(rand.Intn(20))*time.Millisecond)
	}
	return err2
}

const maxDeadlockRetries = 8

// isRetryableDBError mengenali error MySQL yang aman untuk di-retry utuh
// (seluruh transaksi di-rollback InnoDB, tidak ada efek samping parsial):
// 1213 deadlock, 1205 lock wait timeout.
func isRetryableDBError(err error) bool {
	var mysqlErr *mysql.MySQLError
	if errors.As(err, &mysqlErr) {
		return mysqlErr.Number == 1213 || mysqlErr.Number == 1205
	}
	return false
}

func (s *ServiceImpl) processCallbackTx(ctx context.Context, req donation_dto.CallbackRequest, newStatus string, actualTotal float64) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		d, ferr := s.repo.FindByExternalTransactionIDForUpdate(tx, req.TransactionID)
		if ferr != nil {
			return apperror.NotFound("Donasi tidak ditemukan")
		}
		if isFinalDonationStatus(d.PaymentStatus) {
			// Idempotent ack — status final tidak pernah ditimpa ulang
			// (mencegah duplicate/out-of-order callback men-downgrade status).
			return nil
		}

		params := donation_model.CallbackUpdateParams{PaymentStatus: newStatus, GatewayStatusID: req.StatusID}
		credit := false
		if newStatus == constants.DonationStatusPaid {
			// Selisih pembulatan CEIL ≤ Rp1 direkonsiliasi otomatis memakai
			// transaction_total dari gateway; selisih lebih besar dianggap
			// mencurigakan dan tidak dipercaya begitu saja ("amount
			// mismatch") — perlu resolusi manual, bukan auto-PAID, dan tidak
			// dikreditkan ke saldo campaign.
			if diff := math.Abs(actualTotal - d.TotalAmount); diff > 1 {
				params.PaymentStatus = constants.DonationStatusAmountMismatch
			} else {
				total := actualTotal
				fee := float64(bisatopup.CalculateAdminFee(roundRupiah(actualTotal), s.cfg.BisatopupQrisMdrPercentCrowdfunding))
				params.TotalAmount = &total
				params.AdminFee = &fee
				credit = true
			}
		}
		if err := s.repo.UpdateCallbackStatus(tx, d.DonationID, params); err != nil {
			return err
		}
		if credit {
			// d.Amount adalah nominal donasi murni (net setelah MDR) —
			// itulah yang benar-benar masuk wallet Bisabiller, bukan
			// totalAmount (gross yang dibayar donor termasuk fee).
			return s.walletSvc.CreditDonation(tx, d.CampaignID, d.DonationID, d.Amount, "")
		}
		return nil
	})
}

// ExpireStale menandai EXPIRED seluruh donasi PENDING yang sudah lewat
// expiredDate. Tidak ada ledger entry yang dibuat (belum pernah PAID).
func (s *ServiceImpl) ExpireStale(ctx context.Context) (int64, error) {
	n, err := s.repo.ExpireStalePending(ctx)
	if err != nil {
		return 0, apperror.Internal("")
	}
	return n, nil
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

// mapGatewayStatus memetakan status_id Bisabiller ke paymentStatus internal
// — direplikasi persis dari BisaTopup::mapStatus ldksyahid-app: {3,4}=PAID,
// 5=CANCELLED, 6=REFUND, 14=FAILED, {1,2,13,default}=PENDING.
func mapGatewayStatus(statusID int) string {
	switch statusID {
	case 3, 4:
		return constants.DonationStatusPaid
	case 5:
		return constants.DonationStatusCancelled
	case 6:
		return constants.DonationStatusRefunded
	case 14:
		return constants.DonationStatusFailed
	default:
		return constants.DonationStatusPending
	}
}

// isFinalDonationStatus menentukan status yang tidak boleh ditimpa ulang
// oleh callback berikutnya (idempotency, cegah downgrade out-of-order).
// EXPIRED sengaja tidak dianggap final di sini — donasi yang di-expire
// scheduler tapi kemudian mendapat callback PAID (late callback) tetap
// wajib diproses, uang yang sudah dibayar tidak boleh hilang.
func isFinalDonationStatus(status string) bool {
	switch status {
	case constants.DonationStatusPaid, constants.DonationStatusFailed, constants.DonationStatusCancelled,
		constants.DonationStatusRefunded, constants.DonationStatusAmountMismatch:
		return true
	}
	return false
}

// isTestCallback mendeteksi ping tombol "Test" dashboard Bisabiller —
// pengecualian sempit (env=dev + signature literal "testing") dari
// signature check normal, bukan bypass umum berdasarkan payload kosong.
func isTestCallback(req donation_dto.CallbackRequest, env string) bool {
	return env == "dev" && req.Signature == "testing"
}

// donationDesc mengembalikan pesan donatur sebagai transaction_desc, atau
// deskripsi default bila donatur tidak mengisi pesan.
func donationDesc(message string) string {
	if strings.TrimSpace(message) == "" {
		return "Donasi Kantong Amal"
	}
	return message
}

// truncateNoEllipsis memotong s ke maksimal max karakter tanpa suffix "..."
// — dipakai untuk field transaction_name/transaction_desc Bisabiller yang
// membatasi panjang string secara ketat.
func truncateNoEllipsis(s string, max int) string {
	r := []rune(s)
	if len(r) <= max {
		return s
	}
	return string(r[:max])
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
