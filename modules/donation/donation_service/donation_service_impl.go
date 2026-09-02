package donation_service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"math"
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
	"fsldk-api/modules/donation/donation_dto"
	"fsldk-api/modules/donation/donation_model"
	"fsldk-api/modules/donation/donation_repository"
	"fsldk-api/modules/jobqueue/jobqueue_dto"
	"fsldk-api/modules/jobqueue/jobqueue_model"
	"fsldk-api/modules/wallet/wallet_service"
	"fsldk-api/pkg/auditlog"
	"fsldk-api/pkg/bisatopup"
	"fsldk-api/pkg/kirimdev"
)

// defaultDonationMessage dipakai saat donatur tidak menulis pesan/doa apa pun
// di form donasi publik — persis teks default ldksyahid-app (app/Models/
// Donation.php: pesan_donatur ?? "Bismillah Semoga Berkah yaaa ! tetap
// Semangat Semuanya !!"), supaya daftar donatur & notifikasi tidak pernah
// tampil kosong. Hanya berlaku di alur Create() (donasi publik/self-service)
// — donasi manual yang diinput admin (AdminCreate/AdminUpdate) TIDAK
// mendapat teks ini, karena field itu murni catatan admin apa adanya.
const defaultDonationMessage = "Bismillah Semoga Berkah yaaa ! tetap Semangat Semuanya !!"

// JobEnqueuer adalah irisan sempit jobqueue_service.Service yang dibutuhkan
// modul ini — dipenuhi otomatis oleh jobqueue_service.Service (pola sama
// shortlinkrequest_service.JobEnqueuer).
type JobEnqueuer interface {
	Enqueue(ctx context.Context, in jobqueue_dto.EnqueueInput) (int64, error)
}

// FinanceAuditor adalah irisan sempit auditlog.Logger yang dibutuhkan modul
// ini — pola sama JobEnqueuer, memenuhi kontrak dengan *auditlog.Logger
// sungguhan sekaligus memudahkan fake di unit test tanpa perlu *gorm.DB.
type FinanceAuditor interface {
	LogFinance(ctx context.Context, e auditlog.Entry)
}

// Mailer adalah irisan sempit mailer.Mailer yang dibutuhkan modul ini — pola
// sama JobEnqueuer/FinanceAuditor.
type Mailer interface {
	SendDonationReceipt(toEmail, toName, campaignTitle, amount, total, dateStr, publicRef, receiptURL string) error
	// SendDonationInvoice dikirim SEGERA setelah donasi dibuat (sebelum
	// dibayar) — konfirmasi pertama dari dua email donasi (item 2
	// revision-prompt-2.md), berisi tagihan QRIS. SendDonationReceipt
	// (di atas) adalah email kedua, dikirim setelah pembayaran dikonfirmasi PAID.
	SendDonationInvoice(toEmail, toName, campaignTitle, amount, qrURL, expiredDateStr string) error
}

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
	jobs         JobEnqueuer
	audit        FinanceAuditor
	mail         Mailer
	db           *gorm.DB // hanya dipakai ProcessCallback, yang membuka transaksinya sendiri
	cfg          config.AppConfig
}

// NewService membuat Service donation.
func NewService(repo donation_repository.Repository, campaignRepo campaign_repository.Repository, walletSvc wallet_service.Service, gateway bisatopup.Gateway, jobs JobEnqueuer, audit FinanceAuditor, mail Mailer, db *gorm.DB, cfg config.AppConfig) Service {
	return &ServiceImpl{repo: repo, campaignRepo: campaignRepo, walletSvc: walletSvc, gateway: gateway, jobs: jobs, audit: audit, mail: mail, db: db, cfg: cfg}
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
		Message:         nullStringFrom(messageOrDefault(req.Message)),
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
		ItemID:          camp.CampaignID,
		ItemName:        truncateNoEllipsis(camp.Title, 49),
	})
	if gerr != nil {
		// Tidak ada retry otomatis di sini (lihat pkg/bisatopup) — retry
		// create-transaction berisiko membuat transaksi duplikat di sisi
		// gateway. User diminta mengulang secara eksplisit (donasi baru).
		// Alasan penolakan asli dari gateway di-log di sini (bukan
		// dikembalikan ke client) supaya bisa didiagnosis dari server tanpa
		// membocorkan detail internal gateway ke publik.
		log.Printf("[DONATION] gateway create transaction gagal, transactionID=%s nominal=%d: %v", transactionID, grossTotal, gerr)
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

	s.notify(ctx, req.DonorPhone, "invoice_donasi_kantong_amal",
		[]string{req.DonorName, formatRupiah(float64(amount)), camp.Title, qris.PaymentLinks},
		id)
	// Email pertama dari dua email donasi (item 2 revision-prompt-2.md) —
	// konfirmasi/tagihan segera setelah donasi dibuat, sebelum dibayar.
	// Email kedua (SendDonationReceipt) menyusul setelah PAID, lihat
	// notifyDonationPaid(). Best-effort, gagal kirim tidak menggagalkan alur donasi.
	if req.DonorEmail != "" {
		if err := s.mail.SendDonationInvoice(req.DonorEmail, req.DonorName, camp.Title, formatRupiah(float64(grossTotal)), qris.QrCode, expiredAt.Format("02 Jan 2006 15:04")); err != nil {
			log.Printf("[DONATION] gagal kirim email invoice donasi %d: %v", id, err)
		}
	}
	var actorID int64
	if donorUserID != nil {
		actorID = *donorUserID
	}
	s.audit.LogFinance(ctx, auditlog.Entry{
		ActorUserID: actorID, Action: "donation.created", Entity: "donation", EntityID: id,
		After: map[string]interface{}{"campaignID": camp.CampaignID, "amount": float64(amount), "isAnonymous": req.IsAnonymous},
	})

	return s.getResponse(ctx, id)
}

// notify mengirim satu notifikasi WhatsApp lewat job queue (async, tidak
// pernah sinkron — §14.4 techspec). Kegagalan enqueue di-log, tidak pernah
// menggagalkan alur utama (donasi/callback tetap sukses meski notifikasi
// gagal terkirim) — konsisten prinsip "notification is best-effort".
func (s *ServiceImpl) notify(ctx context.Context, toPhone, templateName string, params []string, donationID int64) {
	if strings.TrimSpace(toPhone) == "" {
		return
	}
	if _, err := s.jobs.Enqueue(ctx, jobqueue_dto.EnqueueInput{
		Queue: jobqueue_model.QueueWhatsApp, JobType: jobqueue_model.JobTypeWhatsAppTemplate,
		Payload:         kirimdev.TemplateMessage{ToPhone: toPhone, TemplateName: templateName, Params: params},
		CorrelationType: jobqueue_model.CorrelationTypeDonation, CorrelationID: donationID,
	}); err != nil {
		log.Printf("[DONATION] gagal enqueue notifikasi WA (%s) untuk donationID=%d: %v", templateName, donationID, err)
	}
}

// formatRupiah memformat nominal Rupiah dengan pemisah ribuan titik untuk
// parameter template WhatsApp (mis. "20.000"), bukan angka mentah.
func formatRupiah(amount float64) string {
	s := strconv.FormatInt(int64(amount), 10)
	neg := strings.HasPrefix(s, "-")
	if neg {
		s = s[1:]
	}
	var out []byte
	for i, r := range []byte(s) {
		if i > 0 && (len(s)-i)%3 == 0 {
			out = append(out, '.')
		}
		out = append(out, r)
	}
	if neg {
		return "-" + string(out)
	}
	return string(out)
}

// ProcessCallback menangani webhook payment callback Bisabiller.
func (s *ServiceImpl) ProcessCallback(ctx context.Context, req donation_dto.CallbackRequest) error {
	if isTestCallback(req) {
		log.Printf("[BISATOPUP:CALLBACK] test event acknowledged, transactionID=%s", req.TransactionID)
		return nil
	}

	if s.cfg.BisatopupEnforceCallbackSignatureCrowdfunding &&
		!bisatopup.VerifySignature(s.cfg.BisatopupUsernameCrowdfunding, req.TransactionID, req.Signature) {
		log.Printf("[BISATOPUP:CALLBACK] signature mismatch, transactionID=%s", req.TransactionID)
		s.audit.LogFinance(ctx, auditlog.Entry{
			Action: "donation.callback.signature_invalid", Entity: "donation",
			Metadata: map[string]string{"transactionID": req.TransactionID},
		})
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
	// dbretry.Do menyerap deadlock InnoDB (error 1213/1205) yang normal
	// terjadi di bawah kontensi tinggi pada baris yang sama, memastikan
	// callback tetap sukses tanpa mengharuskan Bisabiller mengirim ulang
	// webhook-nya sendiri.
	return dbretry.Do(func() error {
		return s.processCallbackTx(ctx, req, newStatus, actualTotal)
	})
}

func (s *ServiceImpl) processCallbackTx(ctx context.Context, req donation_dto.CallbackRequest, newStatus string, actualTotal float64) error {
	var notifyDonation donation_model.Donation
	shouldNotifyPaid := false
	var auditAction string
	var auditDonationID int64
	var auditBefore, auditAfter string

	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		d, ferr := s.repo.FindByExternalTransactionIDForUpdate(tx, req.TransactionID)
		if ferr != nil {
			return apperror.NotFound("Donasi tidak ditemukan")
		}
		auditDonationID, auditBefore = d.DonationID, d.PaymentStatus
		if isFinalDonationStatus(d.PaymentStatus) {
			// Idempotent ack — status final tidak pernah ditimpa ulang
			// (mencegah duplicate/out-of-order callback men-downgrade status).
			auditAction, auditAfter = "donation.callback.duplicate", d.PaymentStatus
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
				auditAction = "donation.callback.amount_mismatch"
			} else {
				total := actualTotal
				fee := float64(bisatopup.CalculateAdminFee(roundRupiah(actualTotal), s.cfg.BisatopupQrisMdrPercentCrowdfunding))
				params.TotalAmount = &total
				params.AdminFee = &fee
				credit = true
				// donation.late_callback_recovered: donasi sudah sempat
				// di-EXPIRE scheduler tapi callback PAID terlambat masuk dan
				// tetap diproses (isFinalDonationStatus() sengaja tidak
				// menganggap EXPIRED final — lihat komentarnya di bawah).
				if d.PaymentStatus == constants.DonationStatusExpired {
					auditAction = "donation.late_callback_recovered"
				} else {
					auditAction = "donation.callback.processed"
				}
			}
		} else {
			auditAction = "donation.callback.processed"
		}
		auditAfter = params.PaymentStatus
		if err := s.repo.UpdateCallbackStatus(tx, d.DonationID, params); err != nil {
			return err
		}
		if credit {
			// d.Amount adalah nominal donasi murni (net setelah MDR) —
			// itulah yang benar-benar masuk wallet Bisabiller, bukan
			// totalAmount (gross yang dibayar donor termasuk fee).
			if err := s.walletSvc.CreditDonation(tx, d.CampaignID, d.DonationID, d.Amount, ""); err != nil {
				return err
			}
			notifyDonation, shouldNotifyPaid = d, true
		}
		return nil
	})
	if err != nil {
		return err
	}

	// Audit ditulis SETELAH transaksi commit sungguhan — pkg/auditlog tidak
	// ikut serta dalam transaksi SQL (LogFinance pakai koneksinya sendiri),
	// jadi pola sama notifikasi di atas: tulis di sini, bukan di dalam
	// closure yang bisa di-retry dbretry.Do, mencegah entri audit phantom
	// untuk transaksi yang di-rollback.
	if auditAction != "" {
		s.audit.LogFinance(ctx, auditlog.Entry{
			Action: auditAction, Entity: "donation", EntityID: auditDonationID,
			Before: map[string]string{"paymentStatus": auditBefore}, After: map[string]string{"paymentStatus": auditAfter},
		})
	}

	// Notifikasi dikirim SETELAH transaksi commit sungguhan — bukan di
	// dalam closure di atas (yang bisa di-retry dbretry.Do saat deadlock),
	// mencegah notifikasi ganda/phantom untuk transaksi yang di-rollback.
	if shouldNotifyPaid {
		s.notifyDonationPaid(ctx, notifyDonation)
	}
	return nil
}

// notifyDonationPaid mengirim notifikasi WA ke donor dan (bila tidak
// anonim/PIC tetap perlu tahu) ke pemilik campaign — §14.3/§14.9 techspec.
// Juga mengirim email konfirmasi ke donor (revisi 2026-08-30, pola sama
// Celengan Syahid ldksyahid-app) — best-effort, gagal kirim email tidak
// pernah menggagalkan proses callback pembayaran.
func (s *ServiceImpl) notifyDonationPaid(ctx context.Context, d donation_model.Donation) {
	amountStr := formatRupiah(d.Amount)
	if d.DonorPhone != "" {
		s.notify(ctx, d.DonorPhone, "donasi_berhasil_kantong_amal", []string{d.DonorName, amountStr, d.CampaignTitle}, d.DonationID)
	}
	if d.DonorEmail != "" {
		receiptURL := fmt.Sprintf("%s/kantong-amal/donasi/%s/bukti", strings.TrimRight(s.cfg.FrontendURL, "/"), d.PublicRef)
		if err := s.mail.SendDonationReceipt(d.DonorEmail, d.DonorName, d.CampaignTitle, amountStr, formatRupiah(d.TotalAmount), d.CreatedDate.Format("02 Jan 2006 15:04"), d.PublicRef, receiptURL); err != nil {
			log.Printf("[DONATION] gagal kirim email konfirmasi donasi %d: %v", d.DonationID, err)
		}
	}

	camp, err := s.campaignRepo.FindByID(ctx, d.CampaignID)
	if err != nil || camp.PicPhone == "" {
		if err != nil {
			log.Printf("[DONATION] gagal ambil campaign %d untuk notifikasi PIC: %v", d.CampaignID, err)
		}
		return
	}
	donorLabel := d.DonorName
	if d.IsAnonymous {
		donorLabel = "Donatur (anonim)"
	}
	s.notify(ctx, camp.PicPhone, "notifikasi_pic_donasi_kantong_amal", []string{amountStr, donorLabel, camp.Title}, d.DonationID)
}

// ExpireStale menandai EXPIRED seluruh donasi PENDING yang sudah lewat
// expiredDate. Tidak ada ledger entry yang dibuat (belum pernah PAID).
func (s *ServiceImpl) ExpireStale(ctx context.Context) (int64, error) {
	n, sampleIDs, err := s.repo.ExpireStalePending(ctx)
	if err != nil {
		return 0, apperror.Internal("")
	}
	if n > 0 {
		s.audit.LogFinance(ctx, auditlog.Entry{
			Action: "donation.auto_expired", Entity: "donation",
			Metadata: map[string]interface{}{"count": n, "sampleDonationIDs": sampleIDs},
		})
	}
	return n, nil
}

// donationExpireCheckInterval — job terjadwal internal `donation.expire_check`
// (§9.6/§13.4 techspec, analog `ExpireStaleQrisDonations` ldksyahid-app yang
// jadwalnya `everyTenMinutes()`). Tidak lewat job queue (§13.4: goroutine
// time.Ticker langsung, bukan event-driven).
const donationExpireCheckInterval = 10 * time.Minute

// RunExpireScheduler menjalankan ExpireStale tiap donationExpireCheckInterval
// sampai proses berhenti — dipanggil sebagai goroutine terpisah dari
// router.go, pola sama jobqueue_service.RunStuckSweeper.
func (s *ServiceImpl) RunExpireScheduler() {
	ticker := time.NewTicker(donationExpireCheckInterval)
	defer ticker.Stop()
	for range ticker.C {
		n, err := s.ExpireStale(context.Background())
		if err != nil {
			log.Printf("[DONATION] expire_check: gagal expire donasi stale: %v", err)
			continue
		}
		if n > 0 {
			log.Printf("[DONATION] expire_check: %d donasi PENDING di-expire otomatis", n)
		}
	}
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

func (s *ServiceImpl) PublicRecentDonations(ctx context.Context, slug string, limit int) ([]donation_dto.PublicDonationItem, error) {
	camp, err := s.campaignRepo.FindBySlug(ctx, slug)
	if err != nil || camp.Status != constants.CampaignStatusPublished {
		return nil, apperror.NotFound("Campaign tidak ditemukan")
	}
	if limit <= 0 || limit > 50 {
		limit = 10
	}
	rows, _, err := s.repo.List(ctx, donation_dto.ListFilter{
		CampaignID: camp.CampaignID, Status: constants.DonationStatusPaid,
		Limit: limit, OrderBy: "d.createdDate DESC",
	})
	if err != nil {
		return nil, apperror.Internal("")
	}
	out := make([]donation_dto.PublicDonationItem, 0, len(rows))
	for _, d := range rows {
		name := d.DonorName
		if d.IsAnonymous {
			name = "Hamba Allah"
		}
		out = append(out, donation_dto.PublicDonationItem{
			DonorName: name, IsAnonymous: d.IsAnonymous, Amount: d.Amount, Message: d.Message.String, CreatedDate: d.CreatedDate,
		})
	}
	return out, nil
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

func (s *ServiceImpl) CMSGet(ctx context.Context, id int64) (donation_dto.AdminDetailResponse, error) {
	d, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return donation_dto.AdminDetailResponse{}, apperror.NotFound("Donasi tidak ditemukan")
	}
	return donation_dto.AdminDetailResponse{
		Response:        toResponse(d),
		DonorEmail:      d.DonorEmail,
		DonorPhone:      d.DonorPhone,
		DonorAge:        d.DonorAge.String,
		DonorDomicile:   d.DonorDomicile.String,
		DonorOccupation: d.DonorOccupation.String,
	}, nil
}

func (s *ServiceImpl) AdminCreate(ctx context.Context, req donation_dto.AdminCreateRequest) (donation_dto.Response, error) {
	if _, err := s.campaignRepo.FindByID(ctx, req.CampaignID); err != nil {
		return donation_dto.Response{}, apperror.NotFound("Campaign tidak ditemukan")
	}
	id, err := s.repo.AdminCreate(ctx, donation_model.AdminCreateParams{
		PublicRef:       idgen.NewUUIDv4(),
		CampaignID:      req.CampaignID,
		DonorName:       req.DonorName,
		DonorEmail:      nullStringFrom(req.DonorEmail),
		DonorPhone:      nullStringFrom(req.DonorPhone),
		DonorAge:        nullStringFrom(req.DonorAge),
		DonorDomicile:   nullStringFrom(req.DonorDomicile),
		DonorOccupation: nullStringFrom(req.DonorOccupation),
		IsAnonymous:     req.IsAnonymous,
		Message:         nullStringFrom(req.Message),
		Amount:          float64(roundRupiah(req.Amount)),
		PaymentMethod:   nullStringFrom(req.PaymentMethod),
		PaymentStatus:   req.PaymentStatus,
		IdempotencyKey:  idgen.NewUUIDv4(),
	})
	if err != nil {
		return donation_dto.Response{}, apperror.Internal("Gagal mencatat donasi manual")
	}
	s.audit.LogFinance(ctx, auditlog.Entry{
		Action: "donation.manual_created", Entity: "donation", EntityID: id,
		After: map[string]interface{}{"campaignID": req.CampaignID, "amount": req.Amount, "paymentStatus": req.PaymentStatus},
	})
	return s.getResponse(ctx, id)
}

// AdminUpdate/AdminDelete hanya berlaku untuk donasi gateway="manual" —
// donasi Bisatopup adalah catatan finansial yang tidak boleh diubah/dihapus
// dari sini (pola sama celengan syahid destroyAdminDonation).
func (s *ServiceImpl) AdminUpdate(ctx context.Context, id int64, req donation_dto.AdminUpdateRequest) (donation_dto.Response, error) {
	d, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return donation_dto.Response{}, apperror.NotFound("Donasi tidak ditemukan")
	}
	if d.Gateway != constants.DonationGatewayManual {
		return donation_dto.Response{}, apperror.Forbidden("Donasi via Bisatopup tidak dapat diubah dari sini")
	}
	if err := s.repo.AdminUpdate(ctx, id, donation_model.AdminUpdateParams{
		DonorName:       req.DonorName,
		DonorEmail:      nullStringFrom(req.DonorEmail),
		DonorPhone:      nullStringFrom(req.DonorPhone),
		DonorAge:        nullStringFrom(req.DonorAge),
		DonorDomicile:   nullStringFrom(req.DonorDomicile),
		DonorOccupation: nullStringFrom(req.DonorOccupation),
		IsAnonymous:     req.IsAnonymous,
		Message:         nullStringFrom(req.Message),
		Amount:          float64(roundRupiah(req.Amount)),
		PaymentMethod:   nullStringFrom(req.PaymentMethod),
		PaymentStatus:   req.PaymentStatus,
	}); err != nil {
		return donation_dto.Response{}, apperror.Internal("Gagal memperbarui donasi")
	}
	s.audit.LogFinance(ctx, auditlog.Entry{
		Action: "donation.manual_updated", Entity: "donation", EntityID: id,
		Before: map[string]interface{}{"amount": d.Amount, "paymentStatus": d.PaymentStatus},
		After:  map[string]interface{}{"amount": req.Amount, "paymentStatus": req.PaymentStatus},
	})
	return s.getResponse(ctx, id)
}

func (s *ServiceImpl) AdminDelete(ctx context.Context, id int64) error {
	d, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return apperror.NotFound("Donasi tidak ditemukan")
	}
	if d.Gateway != constants.DonationGatewayManual {
		return apperror.Forbidden("Donasi via Bisatopup tidak dapat dihapus — merupakan catatan finansial")
	}
	if err := s.repo.AdminDelete(ctx, id); err != nil {
		return apperror.Internal("Gagal menghapus donasi")
	}
	s.audit.LogFinance(ctx, auditlog.Entry{
		Action: "donation.manual_deleted", Entity: "donation", EntityID: id,
		Before: map[string]interface{}{"amount": d.Amount, "campaignID": d.CampaignID},
	})
	return nil
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
		PaymentMethod: d.PaymentMethod.String,
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

// isTestCallback mendeteksi ping "Test" dari tombol Url Callback di
// dashboard Bisatopup — dikirim di environment APAPUN (dev maupun live,
// dashboard mereka tidak tahu env kita), jadi TIDAK boleh digerbang oleh
// s.cfg.BisatopupEnvCrowdfunding seperti sebelumnya (itu bug: di live,
// isTestCallback selalu false, jadi ping selalu gagal di signature
// verification/lookup transaksi, bukan di-ack). Kriteria persis pola
// ldksyahid-app (PublicController::callbackDonation) — transaction_id
// kosong, berawalan "TEST-", atau signature "testing".
func isTestCallback(req donation_dto.CallbackRequest) bool {
	return req.TransactionID == "" ||
		strings.HasPrefix(req.TransactionID, "TEST-") ||
		req.Signature == "testing"
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

// messageOrDefault mengembalikan defaultDonationMessage bila donatur tidak
// menulis pesan sama sekali — lihat komentar defaultDonationMessage.
func messageOrDefault(s string) string {
	if strings.TrimSpace(s) == "" {
		return defaultDonationMessage
	}
	return s
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
