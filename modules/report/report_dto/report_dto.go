// Package report_dto memuat DTO modul report.
package report_dto

import (
	"database/sql"
	"time"
)

// SubmissionRow adalah satu baris laporan submission untuk ekspor.
type SubmissionRow struct {
	OrganizationName string         `gorm:"column:organizationName"`
	ProvinceName     sql.NullString `gorm:"column:provinceName"`
	CityName         sql.NullString `gorm:"column:cityName"`
	Status           string         `gorm:"column:status"`
	LevelLabel       sql.NullString `gorm:"column:levelLabel"`
	SubmittedDate    sql.NullTime   `gorm:"column:submittedDate"`
	LastUpdatedDate  sql.NullTime   `gorm:"column:lastUpdatedDate"`
}

// ExportFilter menampung parameter penyaringan ekspor laporan submission.
type ExportFilter struct {
	FormCode string
	Status   string
	Format   string // "xlsx" atau "csv"
}

// ExportResult adalah berkas hasil ekspor siap dikirim sebagai response.
type ExportResult struct {
	FileName    string
	ContentType string
	Data        []byte
}

// ---------- Kantong Amal (Phase 9) ----------

// KantongAmalReportFilter menampung parameter penyaringan bersama untuk
// laporan campaign/donation/withdrawal — dari/sampai opsional (kosong berarti
// tanpa batas), campaignID 0 berarti seluruh campaign.
type KantongAmalReportFilter struct {
	From       time.Time
	To         time.Time
	CampaignID int64
	Status     string
	Page       int
	Limit      int
}

// BalanceReportFilter menampung parameter laporan saldo §15.1 — beda dari
// KantongAmalReportFilter karena From/To wajib (bukan opsional, laporan
// saldo selalu berbasis periode) dan tidak ada status/pagination.
type BalanceReportFilter struct {
	From       time.Time
	To         time.Time
	CampaignID int64
}

// BalanceReportResponse adalah ringkasan saldo satu periode §15.1 — seluruh
// kolom bersumber dari tr_wallet_ledger/tr_withdrawal (bukan agregasi
// lepas), closingBalance divalidasi otomatis terhadap expectedClosing.
type BalanceReportResponse struct {
	From            time.Time `json:"from"`
	To              time.Time `json:"to"`
	CampaignID      int64     `json:"campaignID,omitempty"`
	OpeningBalance  float64   `json:"openingBalance"`
	Incoming        float64   `json:"incoming"`
	Outgoing        float64   `json:"outgoing"`
	Refund          float64   `json:"refund"`
	Adjustment      float64   `json:"adjustment"`
	Fee             float64   `json:"fee"`
	ClosingBalance  float64   `json:"closingBalance"`
	ExpectedClosing float64   `json:"expectedClosing"`
	IsBalanced      bool      `json:"isBalanced"`
}

// CampaignReportRow adalah satu baris laporan campaign §15.2.
type CampaignReportRow struct {
	CampaignID       int64        `gorm:"column:campaignID" json:"campaignID"`
	Title            string       `gorm:"column:title" json:"title"`
	Status           string       `gorm:"column:status" json:"status"`
	TargetAmount     float64      `gorm:"column:targetAmount" json:"targetAmount"`
	CollectedAmount  float64      `gorm:"column:collectedAmount" json:"collectedAmount"`
	DonorCount       int64        `gorm:"column:donorCount" json:"donorCount"`
	TransactionCount int64        `gorm:"column:transactionCount" json:"transactionCount"`
	StartDate        sql.NullTime `gorm:"column:startDate" json:"startDate"`
	EndDate          sql.NullTime `gorm:"column:endDate" json:"endDate"`
	CreatedDate      time.Time    `gorm:"column:createdDate" json:"createdDate"`
}

// DonationReportRow adalah satu baris laporan donasi §15.3 — donorName
// selalu nama asli (admin selalu lihat identitas, beda dari tampilan
// publik yang menghormati isAnonymous).
type DonationReportRow struct {
	DonationID    int64     `gorm:"column:donationID" json:"donationID"`
	CampaignTitle string    `gorm:"column:campaignTitle" json:"campaignTitle"`
	DonorName     string    `gorm:"column:donorName" json:"donorName"`
	IsAnonymous   bool      `gorm:"column:isAnonymous" json:"isAnonymous"`
	Amount        float64   `gorm:"column:amount" json:"amount"`
	AdminFee      float64   `gorm:"column:adminFee" json:"adminFee"`
	TotalAmount   float64   `gorm:"column:totalAmount" json:"totalAmount"`
	PaymentStatus string    `gorm:"column:paymentStatus" json:"paymentStatus"`
	Gateway       string    `gorm:"column:gateway" json:"gateway"`
	CreatedDate   time.Time `gorm:"column:createdDate" json:"createdDate"`
}

// WithdrawalReportRow adalah satu baris laporan withdrawal §15.4 — rekening
// ditampilkan penuh tanpa masking tambahan (akses sudah dibatasi permission
// kantong_amal.report.view/.export, least-privilege di level permission).
// processingDate techspec dipetakan ke executedDate skema aktual (satu-satunya
// kolom timestamp yang diisi saat Process() mengeksekusi disbursement).
type WithdrawalReportRow struct {
	WithdrawalID             int64        `gorm:"column:withdrawalID" json:"withdrawalID"`
	WithdrawalRef            string       `gorm:"column:withdrawalRef" json:"withdrawalRef"`
	CampaignTitle            string       `gorm:"column:campaignTitle" json:"campaignTitle"`
	Amount                   float64      `gorm:"column:amount" json:"amount"`
	Fee                      float64      `gorm:"column:fee" json:"fee"`
	NetAmount                float64      `gorm:"column:netAmount" json:"netAmount"`
	Status                   string       `gorm:"column:status" json:"status"`
	BeneficiaryBankCode      string       `gorm:"column:beneficiaryBankCode" json:"beneficiaryBankCode"`
	BeneficiaryAccountNumber string       `gorm:"column:beneficiaryAccountNumber" json:"beneficiaryAccountNumber"`
	RequestedDate            time.Time    `gorm:"column:requestedDate" json:"requestedDate"`
	ApprovedDate             sql.NullTime `gorm:"column:approvedDate" json:"approvedDate"`
	ProcessingDate           sql.NullTime `gorm:"column:processingDate" json:"processingDate"`
	CompletedDate            sql.NullTime `gorm:"column:completedDate" json:"completedDate"`
}

// WithdrawalStatusFunnel adalah satu baris breakdown jumlah per status —
// bagian dari laporan withdrawal §15.4 untuk melihat funnel/bottleneck approval.
type WithdrawalStatusFunnel struct {
	Status string `gorm:"column:status" json:"status"`
	Count  int64  `gorm:"column:cnt" json:"count"`
}

// ReconciliationSnapshotResponse adalah satu baris histori rekonsiliasi
// harian §15.5 — snapshot tersimpan, bukan dihitung ulang saat halaman dibuka.
type ReconciliationSnapshotResponse struct {
	SnapshotID                 int64          `gorm:"column:snapshotID" json:"snapshotID"`
	SnapshotDate                time.Time      `gorm:"column:snapshotDate" json:"snapshotDate"`
	DonationPaidCount            int64         `gorm:"column:donationPaidCount" json:"donationPaidCount"`
	DonationPaidAmount           float64       `gorm:"column:donationPaidAmount" json:"donationPaidAmount"`
	LedgerDonationCreditAmount   float64       `gorm:"column:ledgerDonationCreditAmount" json:"ledgerDonationCreditAmount"`
	WithdrawalSuccessCount       int64         `gorm:"column:withdrawalSuccessCount" json:"withdrawalSuccessCount"`
	WithdrawalSuccessAmount      float64       `gorm:"column:withdrawalSuccessAmount" json:"withdrawalSuccessAmount"`
	ExpectedBalance              float64       `gorm:"column:expectedBalance" json:"expectedBalance"`
	GatewayWalletBalance         float64       `gorm:"column:gatewayWalletBalance" json:"gatewayWalletBalance"`
	DiscrepancyAmount            float64       `gorm:"column:discrepancyAmount" json:"discrepancyAmount"`
	// SettlementPendingAmount = jumlah donasi PAID dalam SettlementMinutes menit
	// sebelum snapshot ini dijalankan — masih mungkin belum settle penuh di
	// wallet gateway saat itu, jadi selisih sebesar ini ditoleransi (tidak
	// dianggap anomali). Setara "Settlement Pending" di ldksyahid-app.
	SettlementPendingAmount      float64       `gorm:"column:settlementPendingAmount" json:"settlementPendingAmount"`
	SettlementMinutes            int           `gorm:"column:settlementMinutes" json:"settlementMinutes"`
	HasAnomaly                   bool          `gorm:"column:hasAnomaly" json:"hasAnomaly"`
	GatewayError                 *string        `gorm:"column:gatewayError" json:"gatewayError,omitempty"`
	CreatedDate                  time.Time     `gorm:"column:createdDate" json:"createdDate"`
}

// ReconciliationSnapshotParams menampung hasil satu kali jalan
// finance.daily_reconciliation, siap disimpan sebagai baris snapshot baru.
type ReconciliationSnapshotParams struct {
	SnapshotDate               time.Time
	DonationPaidCount          int64
	DonationPaidAmount         float64
	LedgerDonationCreditAmount float64
	WithdrawalSuccessCount     int64
	WithdrawalSuccessAmount    float64
	ExpectedBalance            float64
	GatewayWalletBalance       float64
	DiscrepancyAmount          float64
	SettlementPendingAmount    float64
	SettlementMinutes          int
	HasAnomaly                 bool
	GatewayError               string
}

// GlobalLedgerFilter menampung parameter laporan debit/kredit global (item 6
// revision-prompt-2.md) — campaignID 0 berarti seluruh campaign. Sumbernya
// selalu tr_wallet_ledger, yang sejak revisi 2026-09-01 HANYA pernah diisi
// donasi Bisatopup (donasi manual tidak pernah menyentuh ledger, lihat
// constants.DonationGatewayManual) — laporan ini otomatis "khusus Amdigipay-
// Bisatopup" tanpa perlu filter gateway terpisah.
type GlobalLedgerFilter struct {
	CampaignID int64
	Direction  string // "" = semua, CREDIT, atau DEBIT
	Page       int
	Limit      int
}

// GlobalLedgerRow adalah satu baris laporan debit/kredit global.
type GlobalLedgerRow struct {
	LedgerID      int64     `gorm:"column:ledgerID" json:"ledgerID"`
	CampaignID    int64     `gorm:"column:campaignID" json:"campaignID"`
	CampaignTitle string    `gorm:"column:campaignTitle" json:"campaignTitle"`
	EntryType     string    `gorm:"column:entryType" json:"entryType"`
	Direction     string    `gorm:"column:direction" json:"direction"`
	Amount        float64   `gorm:"column:amount" json:"amount"`
	BalanceAfter  float64   `gorm:"column:balanceAfter" json:"balanceAfter"`
	ReferenceType string    `gorm:"column:referenceType" json:"referenceType"`
	ReferenceID   int64     `gorm:"column:referenceID" json:"referenceID"`
	CreatedDate   time.Time `gorm:"column:createdDate" json:"createdDate"`
}

// AmountBandRow/AgeBandRow adalah satu bucket distribusi untuk tab Analitik
// (item 7 revision-prompt-2.md) — dihitung sekali per band lewat UNION di
// repository, bukan di-groupby nilai mentah (nominal/usia donor terlalu
// bervariasi untuk GROUP BY langsung bermakna sebagai chart).
type AmountBandRow struct {
	BandLabel string `json:"bandLabel"`
	Count     int64  `json:"count"`
}

type AgeBandRow struct {
	BandLabel string `json:"bandLabel"`
	Count     int64  `json:"count"`
}

// AnalyticsResponse membungkus seluruh data tab Analitik satu campaign
// (atau seluruh campaign bila campaignID 0) sekali panggil.
type AnalyticsResponse struct {
	DonationAmountBands []AmountBandRow    `json:"donationAmountBands"`
	DonorAgeBands       []AgeBandRow       `json:"donorAgeBands"`
	CampaignProgress    []CampaignReportRow `json:"campaignProgress"`
}

// FinanceAuditLogFilter menampung parameter penyaringan histori audit log finance §16.1.
type FinanceAuditLogFilter struct {
	Entity string
	Action string
	Page   int
	Limit  int
}

// FinanceAuditLogItem adalah satu baris tr_finance_audit_log untuk CMS.
type FinanceAuditLogItem struct {
	LogID       int64          `gorm:"column:logID" json:"logID"`
	ActorUserID int64          `gorm:"column:actorUserID" json:"actorUserID"`
	ActorName   sql.NullString `gorm:"column:actorName" json:"actorName,omitempty"`
	Action      string         `gorm:"column:action" json:"action"`
	Entity      string         `gorm:"column:entity" json:"entity"`
	EntityID    int64          `gorm:"column:entityID" json:"entityID"`
	BeforeJSON  sql.NullString `gorm:"column:beforeJSON" json:"beforeJSON,omitempty"`
	AfterJSON   sql.NullString `gorm:"column:afterJSON" json:"afterJSON,omitempty"`
	Metadata    sql.NullString `gorm:"column:metadata" json:"metadata,omitempty"`
	CreatedDate time.Time      `gorm:"column:createdDate" json:"createdDate"`
}
