// Package bisatopup adalah klien HTTP ke BisaTopup/Bisabiller (payment
// gateway QRIS untuk donasi, transfer bank untuk pencairan withdrawal).
// File ini murni struct data (parameter/hasil method Client serta bentuk
// wire JSON internal); logika HTTP ada di bisatopup.go.
package bisatopup

import "time"

// Config menampung kredensial dan pengaturan klien BisaTopup/Bisabiller.
type Config struct {
	Username      string
	PasswordAPI   string
	Env           string // "dev" | "live"
	BaseURLLive   string
	BaseURLDev    string
	QrisPaymentID int
}

// CreateQRISTransactionParams adalah parameter membuat transaksi QRIS baru.
type CreateQRISTransactionParams struct {
	TransactionID   string
	Nominal         int64
	ExpiredDate     time.Time
	TransactionName string
	TransactionDesc string
	CustomerName    string
	CustomerEmail   string
	CustomerNumber  string
}

// Transaction merepresentasikan data transaksi payment gateway Bisabiller —
// dipakai sebagai hasil CreateQRISTransaction/DetailTransaction dan entri
// ListTransactions.
type Transaction struct {
	ID               int64  `json:"id"`
	PaymentID        int    `json:"payment_id"`
	PaymentName      string `json:"payment_name"`
	StatusID         int    `json:"status_id"`
	Status           string `json:"status"`
	TransactionID    string `json:"transaction_id"`
	TransactionTotal string `json:"transaction_total"`
	ExpiredDate      string `json:"expired_date"`
	PaymentLinks     string `json:"payment_links"`
	PaymentCode      string `json:"payment_code"`
	QrCode           string `json:"qr_code"`
}

// InquiryBankResult adalah hasil verifikasi rekening tujuan sebelum disbursement.
type InquiryBankResult struct {
	ID            int64  `json:"id"`
	BankCode      string `json:"bank_code"`
	AccountNumber string `json:"account_number"`
	AccountHolder string `json:"account_holder"`
	Status        string `json:"status"`
	Name          string `json:"name"`
	Fee           string `json:"fee"`
}

// DisburseParams adalah parameter eksekusi transfer dana ke rekening tujuan.
type DisburseParams struct {
	BankCode      string
	AccountNumber string
	Amount        int64
	Remark        string
	RecipientCity string
	ReffID        string
}

// DisburseResult adalah hasil eksekusi transfer.
type DisburseResult struct {
	ID          int64  `json:"id"`
	ReffID      string `json:"reff_id"`
	IDStatus    int    `json:"id_status"`
	Amount      int64  `json:"amount"`
	Fee         int64  `json:"fee"`
	TotalAmount int64  `json:"total_amount"`
}

// BankListItem adalah satu entri bank tujuan transfer beserta fee live-nya
// (bervariasi per bank, bukan flat — ditampilkan apa adanya ke frontend).
type BankListItem struct {
	ID       int64  `json:"id"`
	BankCode string `json:"bank_code"`
	Name     string `json:"name"`
	Fee      int64  `json:"fee"`
	Status   string `json:"status"`
}

// WalletBalanceResult adalah saldo wallet Bisabiller milik akun mitra.
type WalletBalanceResult struct {
	Amount int64
}

// -- bentuk wire JSON internal Bisabiller (envelope response), tidak diekspor --

type loginResponse struct {
	Error   bool   `json:"error"`
	Message string `json:"message"`
	Data    struct {
		AccessToken string `json:"access_token"`
	} `json:"data"`
}

type transactionResponse struct {
	Error   bool        `json:"error"`
	Message string      `json:"message"`
	Data    Transaction `json:"data"`
}

type transactionListResponse struct {
	Error bool          `json:"error"`
	Data  []Transaction `json:"data"`
}

type inquiryBankResponse struct {
	Error   bool              `json:"error"`
	Message string            `json:"message"`
	Data    InquiryBankResult `json:"data"`
}

type disburseResponse struct {
	Error   bool           `json:"error"`
	Message string         `json:"message"`
	Data    DisburseResult `json:"data"`
}

type bankListResponse struct {
	Error  bool           `json:"error"`
	Result []BankListItem `json:"result"`
}

type accountInfoResponse struct {
	Error bool `json:"error"`
	Data  struct {
		Wallet struct {
			Jumlah int64 `json:"jumlah"`
		} `json:"wallet"`
	} `json:"data"`
}
