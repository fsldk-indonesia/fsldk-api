package withdrawal_service

import (
	"context"
	"crypto/hmac"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"gorm.io/gorm"

	"fsldk-api/base/apperror"
	"fsldk-api/base/dbretry"
	"fsldk-api/base/dto"
	"fsldk-api/base/idgen"
	"fsldk-api/base/security"
	"fsldk-api/config"
	"fsldk-api/constants"
	"fsldk-api/modules/campaign/campaign_repository"
	"fsldk-api/modules/jobqueue/jobqueue_dto"
	"fsldk-api/modules/jobqueue/jobqueue_model"
	"fsldk-api/modules/setting/setting_model"
	"fsldk-api/modules/user/user_repository"
	"fsldk-api/modules/wallet/wallet_service"
	"fsldk-api/modules/withdrawal/withdrawal_dto"
	"fsldk-api/modules/withdrawal/withdrawal_model"
	"fsldk-api/modules/withdrawal/withdrawal_repository"
	"fsldk-api/pkg/auditlog"
	"fsldk-api/pkg/bisatopup"
	"fsldk-api/pkg/kirimdev"
)

// JobEnqueuer adalah irisan sempit jobqueue_service.Service yang dibutuhkan
// modul ini — pola sama donation_service/campaign_service.JobEnqueuer.
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
// sama donation_service.Mailer. SendOtpEmail dipakai RequestSecurityOtp
// (item 8 revision-prompt-2.md — OTP withdrawal via email, bukan WhatsApp).
type Mailer interface {
	SendOtpEmail(toEmail, code, validityText string) error
}

// SettingReader adalah irisan sempit setting_repository.Repository yang
// dibutuhkan modul ini — membaca email penerima OTP withdrawal yang
// dikonfigurasi admin di ms_setting (isHide=true, tidak tampil di App
// Settings UI, lihat setting_model.KeyWithdrawalOtpEmail).
type SettingReader interface {
	FindByGroupKey(ctx context.Context, group, key string) (setting_model.Setting, error)
}

// processingEtaText adalah estimasi waktu pencairan yang ditampilkan di
// notifikasi "withdrawal_diproses" — techspec §14.3 mensyaratkan parameter
// ini tapi tidak memberi angka pasti; nilai reuse dari konvensi umum proses
// disbursement bank di Indonesia (1-3 hari kerja), dicatat sebagai asumsi.
const processingEtaText = "1-3 hari kerja"

// staleProcessingThreshold adalah ambang waktu withdrawal PROCESSING
// dianggap butuh tinjauan manual admin (§11.7 — timeout tidak boleh
// langsung dianggap gagal, hanya "belum pasti").
const staleProcessingThreshold = 10 * time.Minute

// riskAmountThreshold adalah ambang nominal yang memicu step-up OTP
// tambahan (Option D) di atas password (Option B) — keputusan final OQ-01/
// §12.1: >Rp10.000.000.
const riskAmountThreshold = 10_000_000

// otpValidityDuration adalah masa berlaku satu OTP challenge.
const otpValidityDuration = 5 * time.Minute

// maxOtpAttempts adalah batas percobaan verifikasi per challenge — reuse
// pola ldksyahid-app: 5 percobaan (§12.9).
const maxOtpAttempts = 5

// maxOtpRequestsPerWithdrawal adalah batas KUMULATIF permintaan kode OTP
// baru per withdrawal (Phase 13 security hardening) — tanpa ini,
// FindActiveOtpChallenge selalu melayani challenge terbaru sehingga caller
// yang kehabisan maxOtpAttempts di satu challenge bisa memanggil
// RequestSecurityOtp lagi untuk mendapat attemptCount baru yang bersih,
// membuat batas 5-percobaan-per-challenge efektif tidak berlaku (rate limit
// per-IP di router tidak cukup karena dikunci per ClientIP, mudah dihindari
// lewat banyak IP). Setelah batas ini tercapai, withdrawal wajib dibatalkan
// dan diajukan ulang (lewat Cancel) — bukan dikunci permanen.
const maxOtpRequestsPerWithdrawal = 3

var sortColumns = map[string]string{"createdDate": "w.createdDate", "amount": "w.amount"}

// ServiceImpl adalah implementasi Service.
type ServiceImpl struct {
	repo         withdrawal_repository.Repository
	campaignRepo campaign_repository.Repository
	userRepo     user_repository.Repository
	walletSvc    wallet_service.Service
	gateway      bisatopup.Gateway
	jobs         JobEnqueuer
	audit        FinanceAuditor
	mail         Mailer
	settings     SettingReader
	db           *gorm.DB
	cfg          config.AppConfig
}

// NewService membuat Service withdrawal.
func NewService(repo withdrawal_repository.Repository, campaignRepo campaign_repository.Repository, userRepo user_repository.Repository, walletSvc wallet_service.Service, gateway bisatopup.Gateway, jobs JobEnqueuer, audit FinanceAuditor, mail Mailer, settings SettingReader, db *gorm.DB, cfg config.AppConfig) Service {
	return &ServiceImpl{repo: repo, campaignRepo: campaignRepo, userRepo: userRepo, walletSvc: walletSvc, gateway: gateway, jobs: jobs, audit: audit, mail: mail, settings: settings, db: db, cfg: cfg}
}

// notify mengirim satu notifikasi WhatsApp lewat job queue (async, tidak
// pernah sinkron — §14.4 techspec). Kegagalan enqueue di-log, tidak pernah
// menggagalkan alur utama — konsisten prinsip "notification is best-effort"
// (pola sama donation_service.notify/campaign_service.notify).
func (s *ServiceImpl) notify(ctx context.Context, toPhone, templateName string, params []string, withdrawalID int64) {
	if strings.TrimSpace(toPhone) == "" {
		return
	}
	if _, err := s.jobs.Enqueue(ctx, jobqueue_dto.EnqueueInput{
		Queue: jobqueue_model.QueueWhatsApp, JobType: jobqueue_model.JobTypeWhatsAppTemplate,
		Payload:         kirimdev.TemplateMessage{ToPhone: toPhone, TemplateName: templateName, Params: params},
		CorrelationType: jobqueue_model.CorrelationTypeWithdrawal, CorrelationID: withdrawalID,
	}); err != nil {
		log.Printf("[WITHDRAWAL] gagal enqueue notifikasi WA (%s) untuk withdrawalID=%d: %v", templateName, withdrawalID, err)
	}
}

// notifyPic mengirim notifikasi progres withdrawal ke PIC campaign (bukan
// lagi "owner" — withdrawal murni CRUD/permission-gated, siapapun dengan
// akses boleh mengajukan/memproses withdrawal campaign manapun, revisi
// 2026-09-01, lihat campaign_model.PicPhone).
func (s *ServiceImpl) notifyPic(ctx context.Context, campaignID, withdrawalID int64, templateName string, params []string) {
	camp, err := s.campaignRepo.FindByID(ctx, campaignID)
	if err != nil || camp.PicPhone == "" {
		return
	}
	s.notify(ctx, camp.PicPhone, templateName, params, withdrawalID)
}

// formatRupiah memformat nominal Rupiah dengan pemisah ribuan titik untuk
// parameter template WhatsApp (mis. "20.000") — duplikat kecil dari
// donation_service.formatRupiah (tidak diekspor lintas modul by design,
// _service tidak saling impor).
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

func (s *ServiceImpl) Request(ctx context.Context, campaignID, requesterUserID int64, req withdrawal_dto.CreateRequest) (withdrawal_dto.Response, error) {
	camp, err := s.campaignRepo.FindByID(ctx, campaignID)
	if err != nil {
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

	// Rekening tujuan diinput ulang setiap pengajuan (revisi 2026-09-01 —
	// campaign tidak lagi menyimpan beneficiary, lihat campaign_dto) dan
	// selalu divalidasi live via inquiry gateway. AccountHolder diambil dari
	// hasil inquiry yang terverifikasi gateway, bukan input client, supaya
	// tidak bisa dipalsukan.
	inq, ierr := s.gateway.InquiryBank(ctx, req.BeneficiaryBankCode, req.BeneficiaryAccountNumber)
	if ierr != nil || !strings.EqualFold(inq.Status, "SUCCESS") {
		return withdrawal_dto.Response{}, apperror.Unprocessable("Rekening penerima tidak valid, silakan periksa kembali nomor rekening")
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
				BeneficiaryBankCode:      req.BeneficiaryBankCode,
				BeneficiaryAccountNumber: req.BeneficiaryAccountNumber,
				BeneficiaryAccountHolder: inq.AccountHolder,
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
	s.notifyPic(ctx, campaignID, id, "withdrawal_diajukan", []string{formatRupiah(float64(amount)), camp.Title})
	s.audit.LogFinance(ctx, auditlog.Entry{
		ActorUserID: requesterUserID, Action: "withdrawal.requested", Entity: "withdrawal", EntityID: id,
		After: map[string]interface{}{"campaignID": campaignID, "amount": float64(amount)},
	})
	return s.getResponse(ctx, id)
}

func (s *ServiceImpl) Cancel(ctx context.Context, withdrawalID, requesterUserID int64) error {
	w, err := s.repo.FindByID(ctx, withdrawalID)
	if err != nil {
		return apperror.NotFound("Penarikan tidak ditemukan")
	}
	if !isCancellableStatus(w.Status) {
		return apperror.InvalidStatusTransition("Penarikan tidak dapat dibatalkan pada status saat ini")
	}
	err = dbretry.Do(func() error {
		return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			if err := s.repo.UpdateStatus(tx, withdrawalID, constants.WithdrawalStatusCancelled, withdrawal_model.StatusUpdateParams{}); err != nil {
				return err
			}
			return s.walletSvc.ReleaseWithdrawal(tx, w.CampaignID, withdrawalID, w.Amount, "")
		})
	})
	if err != nil {
		return err
	}
	s.audit.LogFinance(ctx, auditlog.Entry{
		ActorUserID: requesterUserID, Action: "withdrawal.cancelled", Entity: "withdrawal", EntityID: withdrawalID,
		Before: map[string]string{"status": w.Status}, After: map[string]string{"status": constants.WithdrawalStatusCancelled},
	})
	return nil
}

// isRiskyWithdrawal menentukan apakah withdrawal ini memicu step-up OTP
// tambahan (Option D) di atas password (Option B) — keputusan final OQ-01/
// §12.1. Diimplementasikan: nominal >Rp10 juta ATAU withdrawal pertama yang
// pernah SUCCESS untuk campaign ini. Sejak revisi 2026-09-01, rekening
// tujuan selalu diinput ulang & divalidasi live tiap pengajuan (tidak ada
// lagi cooling period campaign-level), jadi trigger "rekening baru" tidak
// relevan lagi di sini. "Banyak percobaan gagal sebelumnya" tidak
// diimplementasikan di fase ini karena techspec tidak memberi angka ambang
// konkret (beda dari Rp10 juta yang eksplisit) — dicatat sebagai gap di
// phase-07-summary.md, bukan diputuskan diam-diam.
func (s *ServiceImpl) isRiskyWithdrawal(ctx context.Context, w withdrawal_model.Withdrawal) (bool, error) {
	if w.Amount > riskAmountThreshold {
		return true, nil
	}
	successCount, err := s.repo.CountSuccessByCampaign(ctx, w.CampaignID)
	if err != nil {
		return false, err
	}
	return successCount == 0, nil
}

func (s *ServiceImpl) RequestSecurityOtp(ctx context.Context, withdrawalID, requesterUserID int64) error {
	w, err := s.repo.FindByID(ctx, withdrawalID)
	if err != nil {
		return apperror.NotFound("Penarikan tidak ditemukan")
	}
	if w.Status != constants.WithdrawalStatusSecurityCheck {
		return apperror.InvalidStatusTransition("Penarikan tidak dalam status menunggu verifikasi keamanan")
	}

	// Email penerima OTP dikonfigurasi admin di ms_setting (isHide=true,
	// tidak tampil di App Settings UI) — item 8 revision-prompt-2.md, BUKAN
	// email requester (OTP ini adalah kontrol keamanan operasional tim
	// keuangan, bukan notifikasi personal).
	setting, serr := s.settings.FindByGroupKey(ctx, setting_model.GroupKantongAmal, setting_model.KeyWithdrawalOtpEmail)
	if serr != nil || setting.SettingValue == nil || strings.TrimSpace(*setting.SettingValue) == "" {
		return apperror.Internal("Email penerima OTP withdrawal belum dikonfigurasi, hubungi Super Admin")
	}
	otpEmail := strings.TrimSpace(*setting.SettingValue)

	requestCount, err := s.repo.CountOtpChallengesByWithdrawal(ctx, withdrawalID)
	if err != nil {
		return apperror.Internal("")
	}
	if requestCount >= maxOtpRequestsPerWithdrawal {
		return apperror.TooManyRequests("Batas permintaan kode OTP untuk penarikan ini sudah tercapai, silakan batalkan dan ajukan penarikan baru")
	}

	code, err := generateOtpCode()
	if err != nil {
		return apperror.Internal("")
	}
	if _, err := s.repo.CreateOtpChallenge(ctx, withdrawal_model.OtpChallengeParams{
		WithdrawalID: withdrawalID,
		UserID:       requesterUserID,
		CodeHash:     hashOtpCode(code),
		Channel:      constants.OtpChannelEmail,
		ExpiredDate:  time.Now().Add(otpValidityDuration),
	}); err != nil {
		return apperror.Internal("")
	}

	// Item 8 revision-prompt-2.md (2026-09-01): OTP dikirim via email
	// (sinkron, bukan lewat job queue WhatsApp — kirimdev.TemplateMessage
	// "kode_otp_kantong_amal" kini dead template, lihat docs/DEPLOYMENT.md).
	// Gagal kirim TETAP mengembalikan error ke caller (beda dari notifikasi
	// best-effort lain di modul ini) — tanpa email terkirim, requester tidak
	// akan pernah punya kode untuk diverifikasi sama sekali.
	if err := s.mail.SendOtpEmail(otpEmail, code, fmt.Sprintf("%.0f menit", otpValidityDuration.Minutes())); err != nil {
		log.Printf("[WITHDRAWAL] gagal kirim email OTP untuk withdrawalID=%d: %v", withdrawalID, err)
		return apperror.Internal("Gagal mengirim kode OTP, silakan coba lagi")
	}
	return nil
}

// VerifySecurity memverifikasi password/OTP milik CALLER saat ini (aktor
// yang menekan tombol proses) — bukan lagi harus pengaju asli withdrawal
// (revisi 2026-09-01, withdrawal murni permission-gated), sehingga admin
// manapun dengan akses boleh menindaklanjuti withdrawal siapapun.
func (s *ServiceImpl) VerifySecurity(ctx context.Context, withdrawalID, requesterUserID int64, req withdrawal_dto.SecurityVerifyRequest) (withdrawal_dto.Response, error) {
	w, err := s.repo.FindByID(ctx, withdrawalID)
	if err != nil {
		return withdrawal_dto.Response{}, apperror.NotFound("Penarikan tidak ditemukan")
	}
	if w.Status != constants.WithdrawalStatusSecurityCheck {
		return withdrawal_dto.Response{}, apperror.InvalidStatusTransition("Penarikan tidak dalam status menunggu verifikasi keamanan")
	}

	user, err := s.userRepo.FindByID(ctx, requesterUserID)
	if err != nil {
		return withdrawal_dto.Response{}, apperror.Internal("")
	}
	// Akun tanpa password (login Google murni) tidak punya faktor Option B
	// untuk diverifikasi — diperlakukan selalu berisiko (wajib OTP) alih-alih
	// meloloskan tanpa verifikasi apa pun.
	riskySkipPassword := !user.Password.Valid
	if !riskySkipPassword && !security.CheckPassword(user.Password.String, req.Password) {
		s.auditSecurityFailed(ctx, withdrawalID, requesterUserID, "password_mismatch")
		return withdrawal_dto.Response{}, apperror.Unauthorized("Password tidak sesuai")
	}

	risky, err := s.isRiskyWithdrawal(ctx, w)
	if err != nil {
		return withdrawal_dto.Response{}, apperror.Internal("")
	}
	risky = risky || riskySkipPassword

	params := withdrawal_model.StatusUpdateParams{SetSecurityVerifiedNow: true}
	if risky {
		if strings.TrimSpace(req.OtpCode) == "" {
			s.auditSecurityFailed(ctx, withdrawalID, requesterUserID, "otp_required")
			return withdrawal_dto.Response{}, apperror.SecurityVerificationRequired("Kode OTP wajib diisi untuk penarikan ini")
		}
		challenge, cerr := s.repo.FindActiveOtpChallenge(ctx, withdrawalID)
		if cerr != nil {
			s.auditSecurityFailed(ctx, withdrawalID, requesterUserID, "otp_not_found_or_expired")
			return withdrawal_dto.Response{}, apperror.BadRequest("OTP belum diminta atau sudah kedaluwarsa, silakan minta ulang")
		}
		if challenge.AttemptCount >= maxOtpAttempts {
			s.auditSecurityFailed(ctx, withdrawalID, requesterUserID, "otp_too_many_attempts")
			return withdrawal_dto.Response{}, apperror.TooManyRequests("Terlalu banyak percobaan OTP salah, silakan minta kode baru")
		}
		if err := s.repo.IncrementOtpAttempt(ctx, challenge.ChallengeID); err != nil {
			return withdrawal_dto.Response{}, apperror.Internal("")
		}
		if hashOtpCode(req.OtpCode) != challenge.CodeHash {
			s.auditSecurityFailed(ctx, withdrawalID, requesterUserID, "otp_mismatch")
			return withdrawal_dto.Response{}, apperror.BadRequest("Kode OTP tidak sesuai")
		}
		if err := s.repo.MarkOtpVerified(ctx, challenge.ChallengeID); err != nil {
			return withdrawal_dto.Response{}, apperror.Internal("")
		}
		method := constants.WithdrawalSecurityMethodOtpEmail
		params.SecurityVerifiedMethod = &method
	}

	// Revisi (2026-08-30): maker-checker approval dihapus — begitu verifikasi
	// keamanan lolos, withdrawal langsung APPROVED (siap diproses lewat
	// Process(), permission-gated, tanpa perlu orang lain mengklik approve
	// terpisah). ApprovedByUserID di-set ke requester sendiri — kolom lama
	// direuse untuk mencatat "siapa yang membuat status ini tercapai",
	// bukan lagi "siapa yang menyetujui" (tidak ada aksi approve terpisah
	// lagi, jadi tidak perlu migrasi skema baru).
	//
	// Tidak perlu dbretry di sini — transisi ini hanya mengunci baris
	// withdrawal sendiri, tidak menyentuh ms_campaign (beda dari Request/
	// Cancel yang lewat wallet_service dan bisa kontensi lintas-flow).
	params.ApprovedByUserID = &requesterUserID
	if err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return s.repo.UpdateStatus(tx, withdrawalID, constants.WithdrawalStatusApproved, params)
	}); err != nil {
		return withdrawal_dto.Response{}, apperror.Internal("")
	}
	s.audit.LogFinance(ctx, auditlog.Entry{
		ActorUserID: requesterUserID, Action: "withdrawal.security_verified", Entity: "withdrawal", EntityID: withdrawalID,
		Metadata: map[string]bool{"risky": risky},
	})
	return s.getResponse(ctx, withdrawalID)
}

// auditSecurityFailed mencatat satu upaya verifikasi keamanan withdrawal
// yang gagal (§16.1 techspec: `withdrawal.security_failed`) — reason
// membedakan penyebab tanpa memecah jadi banyak action terpisah.
func (s *ServiceImpl) auditSecurityFailed(ctx context.Context, withdrawalID, requesterUserID int64, reason string) {
	s.audit.LogFinance(ctx, auditlog.Entry{
		ActorUserID: requesterUserID, Action: "withdrawal.security_failed", Entity: "withdrawal", EntityID: withdrawalID,
		Metadata: map[string]string{"reason": reason},
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
		s.notifyPic(ctx, w.CampaignID, withdrawalID, "withdrawal_gagal", []string{formatRupiah(w.Amount), "Ditolak oleh gateway pencairan"})
		// "withdrawal.gateway_rejected" — bukan salah satu action literal di
		// §16.1 techspec (yang hanya menyebut .rejected untuk penolakan
		// admin/maker-checker), ditambahkan supaya audit trail tidak
		// menyamarkan penolakan gateway pembayaran sebagai penolakan admin.
		s.audit.LogFinance(ctx, auditlog.Entry{
			ActorUserID: actorUserID, Action: "withdrawal.gateway_rejected", Entity: "withdrawal", EntityID: withdrawalID,
			Before: map[string]string{"status": w.Status}, After: map[string]string{"status": constants.WithdrawalStatusFailed},
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
	s.notifyPic(ctx, w.CampaignID, withdrawalID, "withdrawal_diproses", []string{formatRupiah(w.Amount), processingEtaText})
	s.audit.LogFinance(ctx, auditlog.Entry{
		ActorUserID: actorUserID, Action: "withdrawal.processing", Entity: "withdrawal", EntityID: withdrawalID,
		Before: map[string]string{"status": w.Status}, After: map[string]string{"status": constants.WithdrawalStatusProcessing},
	})
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
	var notifyWithdrawal withdrawal_model.Withdrawal
	shouldNotify := false

	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		w, err := s.repo.FindByRefForUpdate(tx, reffID)
		if err != nil {
			// Ack (200), bukan error — reffID tak dikenal paling sering
			// terjadi karena ping "Test" dari dashboard Bisatopup (mereka
			// kirim reff_id palsu/placeholder, bukan reff_id withdrawal
			// sungguhan), persis pola ldksyahid-app WithdrawalController::
			// callback() yang membalas {"status":"ignored"} 200 untuk kasus
			// ini, bukan 404. Tetap di-log supaya kasus reff_id genuinely
			// salah (bukan test) masih terlihat di server log.
			log.Printf("[BISATOPUP:WITHDRAWAL_CALLBACK] reffID tidak ditemukan, diabaikan (kemungkinan ping test), reffID=%s", reffID)
			return nil
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
			if err := s.walletSvc.ReleaseWithdrawal(tx, w.CampaignID, w.WithdrawalID, w.Amount, ""); err != nil {
				return err
			}
		}
		notifyWithdrawal, shouldNotify = w, true
		// SUCCESS tidak butuh ledger entry baru — saldo sudah didebit sejak
		// WITHDRAWAL_RESERVE di awal (§10.4).
		return nil
	})
	if err != nil {
		return err
	}

	// Notifikasi dikirim SETELAH transaksi commit sungguhan — bukan di dalam
	// closure di atas (yang bisa di-retry dbretry.Do saat deadlock), mencegah
	// notifikasi ganda/phantom untuk transaksi yang di-rollback (pola sama
	// donation_service.processCallbackTx).
	if shouldNotify {
		w := notifyWithdrawal
		if newStatus == constants.WithdrawalStatusSuccess {
			s.notifyPic(ctx, w.CampaignID, w.WithdrawalID, "withdrawal_berhasil",
				[]string{formatRupiah(w.Amount), w.BeneficiaryBankCode + " " + w.BeneficiaryAccountNumber})
			s.audit.LogFinance(ctx, auditlog.Entry{
				Action: "withdrawal.callback.completed", Entity: "withdrawal", EntityID: w.WithdrawalID,
				Before: map[string]string{"status": w.Status}, After: map[string]string{"status": constants.WithdrawalStatusSuccess},
			})
		} else if newStatus == constants.WithdrawalStatusFailed {
			s.notifyPic(ctx, w.CampaignID, w.WithdrawalID, "withdrawal_gagal",
				[]string{formatRupiah(w.Amount), "Gagal diproses oleh bank/gateway pencairan"})
			s.audit.LogFinance(ctx, auditlog.Entry{
				Action: "withdrawal.callback.failed", Entity: "withdrawal", EntityID: w.WithdrawalID,
				Before: map[string]string{"status": w.Status}, After: map[string]string{"status": constants.WithdrawalStatusFailed},
			})
		}
	}
	return nil
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

// reconcileCheckInterval — job terjadwal internal `withdrawal.reconcile_check`
// (§13.4 techspec: goroutine time.Ticker langsung, bukan event-driven).
// Interval disamakan dengan staleProcessingThreshold: tidak ada gunanya
// mengecek lebih sering dari ambang waktu withdrawal baru dianggap stale.
const reconcileCheckInterval = staleProcessingThreshold

// RunReconcileScheduler menjalankan ReconcileStaleProcessing tiap
// reconcileCheckInterval sampai proses berhenti — hanya mencatat kandidat
// ke log (tidak ada auto-fix, lihat §11.7), pola sama
// jobqueue_service.RunStuckSweeper.
func (s *ServiceImpl) RunReconcileScheduler() {
	ticker := time.NewTicker(reconcileCheckInterval)
	defer ticker.Stop()
	for range ticker.C {
		rows, err := s.ReconcileStaleProcessing(context.Background())
		if err != nil {
			log.Printf("[WITHDRAWAL] reconcile_check: gagal ambil withdrawal stale: %v", err)
			continue
		}
		if len(rows) > 0 {
			ids := make([]int64, 0, len(rows))
			for _, w := range rows {
				ids = append(ids, w.WithdrawalID)
			}
			log.Printf("[WITHDRAWAL] reconcile_check: %d withdrawal PROCESSING melewati threshold, butuh tinjauan manual admin: %v", len(rows), ids)
		}
	}
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
	case constants.WithdrawalStatusRequested, constants.WithdrawalStatusSecurityCheck, constants.WithdrawalStatusApproved:
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
