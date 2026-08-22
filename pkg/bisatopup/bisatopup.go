package bisatopup

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"
)

// loginTokenTTL adalah masa cache token akses sebelum re-login otomatis —
// token asli berlaku ~1 jam, dicache 50 menit (margin 10 menit), reuse
// strategi ldksyahid-app (Cache::put(..., now()->addMinutes(50))).
const loginTokenTTL = 50 * time.Minute

// httpTimeout adalah timeout seragam untuk seluruh panggilan HTTP ke
// Bisabiller — reuse nilai ldksyahid-app (diterapkan eksplisit hanya ke
// login di sana, diterapkan seragam ke seluruh method di sini).
const httpTimeout = 15 * time.Second

// ErrGatewayRejected menandai bahwa Bisabiller merespons sukses secara HTTP
// namun menolak permintaan (body error:true) — beda dari kegagalan
// jaringan/status HTTP 5xx, dipetakan berbeda oleh caller (lihat
// donation_service: PaymentFailed vs ProviderError).
var ErrGatewayRejected = errors.New("bisabiller menolak permintaan")

// Gateway adalah kontrak klien BisaTopup/Bisabiller yang dibutuhkan modul
// lain (donation_service, withdrawal_service) — memisahkan interface dari
// implementasi HTTP sungguhan agar dapat digantikan test double.
type Gateway interface {
	CreateQRISTransaction(ctx context.Context, p CreateQRISTransactionParams) (Transaction, error)
	DetailTransaction(ctx context.Context, bisabillerID int64) (Transaction, error)
	ListTransactions(ctx context.Context) ([]Transaction, error)
	InquiryBank(ctx context.Context, bankCode, accountNumber string) (InquiryBankResult, error)
	Disburse(ctx context.Context, p DisburseParams) (DisburseResult, error)
	WalletBalance(ctx context.Context) (WalletBalanceResult, error)
	BankList(ctx context.Context) ([]BankListItem, error)
}

// client adalah implementasi Gateway sungguhan berbasis http.Client.
type client struct {
	cfg  Config
	http *http.Client

	mu       sync.RWMutex
	token    string
	tokenExp time.Time
}

// NewClient membuat Gateway berbasis HTTP ke BisaTopup/Bisabiller.
func NewClient(cfg Config) Gateway {
	return &client{cfg: cfg, http: &http.Client{Timeout: httpTimeout}}
}

func (c *client) baseURL() string {
	if c.cfg.Env == "live" {
		return strings.TrimRight(c.cfg.BaseURLLive, "/")
	}
	return strings.TrimRight(c.cfg.BaseURLDev, "/")
}

// BuildSignature membuat signature outbound (create transaction) —
// direplikasi persis dari algoritma Bisabiller: sha256(username+transactionID).
func BuildSignature(username, transactionID string) string {
	sum := sha256.Sum256([]byte(username + transactionID))
	return hex.EncodeToString(sum[:])
}

// VerifySignature memverifikasi signature callback secara timing-safe
// (hmac.Equal, padanan hash_equals() PHP).
func VerifySignature(username, transactionID, signature string) bool {
	expected := BuildSignature(username, transactionID)
	return hmac.Equal([]byte(expected), []byte(signature))
}

// ensureToken mengembalikan token akses yang masih berlaku, login ulang
// bila cache kosong/kedaluwarsa.
func (c *client) ensureToken(ctx context.Context) (string, error) {
	c.mu.RLock()
	if c.token != "" && time.Now().Before(c.tokenExp) {
		tok := c.token
		c.mu.RUnlock()
		return tok, nil
	}
	c.mu.RUnlock()

	c.mu.Lock()
	defer c.mu.Unlock()
	// Re-check di dalam write lock: goroutine lain mungkin sudah login
	// duluan saat menunggu lock ini.
	if c.token != "" && time.Now().Before(c.tokenExp) {
		return c.token, nil
	}

	form := url.Values{"username": {c.cfg.Username}, "password": {c.cfg.PasswordAPI}}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL()+"/login", strings.NewReader(form.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	var out loginResponse
	if err := c.do(req, &out); err != nil {
		return "", fmt.Errorf("bisatopup login gagal: %w", err)
	}
	if out.Error || out.Data.AccessToken == "" {
		return "", fmt.Errorf("bisatopup login ditolak: %s", out.Message)
	}

	c.token = out.Data.AccessToken
	c.tokenExp = time.Now().Add(loginTokenTTL)
	return c.token, nil
}

// do mengeksekusi request, memeriksa status HTTP, dan mendekode body JSON
// ke out (bila diberikan).
func (c *client) do(req *http.Request, out interface{}) error {
	req.Header.Set("Accept", "application/json")
	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode >= 400 {
		return fmt.Errorf("bisatopup HTTP %d: %s", resp.StatusCode, string(body))
	}
	if out == nil {
		return nil
	}
	return json.Unmarshal(body, out)
}

func (c *client) authedJSON(ctx context.Context, method, path string, payload interface{}, out interface{}) error {
	token, err := c.ensureToken(ctx)
	if err != nil {
		return err
	}
	b, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, method, c.baseURL()+path, bytes.NewReader(b))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	return c.do(req, out)
}

func (c *client) authedGET(ctx context.Context, path string, out interface{}) error {
	token, err := c.ensureToken(ctx)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL()+path, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	return c.do(req, out)
}

// withRetry mengulang fn maksimal 2x (backoff 2 detik) — hanya dipakai
// method read-only idempoten (DetailTransaction/ListTransactions/
// InquiryBank/WalletBalance/BankList). CreateQRISTransaction/Disburse
// sengaja TIDAK memakai ini: retry pada operasi yang menciptakan
// transaksi/memindahkan uang berisiko duplikasi di sisi gateway.
func withRetry(fn func() error) error {
	var err error
	for attempt := 0; attempt < 3; attempt++ {
		if err = fn(); err == nil {
			return nil
		}
		if attempt < 2 {
			time.Sleep(2 * time.Second)
		}
	}
	return err
}

// CreateQRISTransaction membuat transaksi QRIS baru. Tidak ada retry
// otomatis (lihat withRetry).
func (c *client) CreateQRISTransaction(ctx context.Context, p CreateQRISTransactionParams) (Transaction, error) {
	payload := map[string]interface{}{
		"payment_id":        c.cfg.QrisPaymentID,
		"username":          c.cfg.Username,
		"signature":         BuildSignature(c.cfg.Username, p.TransactionID),
		"expired_date":      p.ExpiredDate.Format("2006-01-02 15:04:05"),
		"nominal":           p.Nominal,
		"admin_fee":         0, // fee Bisabiller sendiri untuk QRIS selalu 0, sudah termasuk di nominal
		"transaction_id":    p.TransactionID,
		"transaction_total": p.Nominal,
		"transaction_name":  p.TransactionName,
		"transaction_desc":  p.TransactionDesc,
		"customer_number":   p.CustomerNumber,
		"customer_name":     p.CustomerName,
		"customer_email":    p.CustomerEmail,
	}
	var out transactionResponse
	if err := c.authedJSON(ctx, http.MethodPost, "/payment/transaction", payload, &out); err != nil {
		return Transaction{}, err
	}
	if out.Error {
		return Transaction{}, fmt.Errorf("%w: %s", ErrGatewayRejected, out.Message)
	}
	return out.Data, nil
}

// DetailTransaction mengambil detail transaksi berdasarkan ID internal Bisabiller.
func (c *client) DetailTransaction(ctx context.Context, bisabillerID int64) (Transaction, error) {
	var out transactionResponse
	err := withRetry(func() error {
		return c.authedGET(ctx, "/payment/detail-transaction/"+strconv.FormatInt(bisabillerID, 10), &out)
	})
	if err != nil {
		return Transaction{}, err
	}
	if out.Error {
		return Transaction{}, fmt.Errorf("%w: %s", ErrGatewayRejected, out.Message)
	}
	return out.Data, nil
}

// ListTransactions mengambil seluruh transaksi payment gateway milik akun mitra.
func (c *client) ListTransactions(ctx context.Context) ([]Transaction, error) {
	var out transactionListResponse
	err := withRetry(func() error { return c.authedGET(ctx, "/payment/list-transaction", &out) })
	if err != nil {
		return nil, err
	}
	return out.Data, nil
}

// InquiryBank memverifikasi rekening tujuan sebelum disbursement.
func (c *client) InquiryBank(ctx context.Context, bankCode, accountNumber string) (InquiryBankResult, error) {
	payload := map[string]interface{}{"bank_code": bankCode, "account_number": accountNumber}
	var out inquiryBankResponse
	err := withRetry(func() error { return c.authedJSON(ctx, http.MethodPost, "/transfer/inquiry", payload, &out) })
	if err != nil {
		return InquiryBankResult{}, err
	}
	if out.Error {
		return InquiryBankResult{}, fmt.Errorf("%w: %s", ErrGatewayRejected, out.Message)
	}
	return out.Data, nil
}

// Disburse mengeksekusi transfer dana ke rekening tujuan. Tidak ada retry
// otomatis (lihat withRetry).
func (c *client) Disburse(ctx context.Context, p DisburseParams) (DisburseResult, error) {
	payload := map[string]interface{}{
		"bank_code":      p.BankCode,
		"account_number": p.AccountNumber,
		"amount":         p.Amount,
		"remark":         p.Remark,
		"recipient_city": p.RecipientCity,
		"reff_id":        p.ReffID,
	}
	var out disburseResponse
	if err := c.authedJSON(ctx, http.MethodPost, "/transfer/disburstment", payload, &out); err != nil {
		return DisburseResult{}, err
	}
	if out.Error {
		return DisburseResult{}, fmt.Errorf("%w: %s", ErrGatewayRejected, out.Message)
	}
	return out.Data, nil
}

// WalletBalance mengambil saldo wallet Bisabiller milik akun mitra.
func (c *client) WalletBalance(ctx context.Context) (WalletBalanceResult, error) {
	var out accountInfoResponse
	err := withRetry(func() error { return c.authedGET(ctx, "/account-info", &out) })
	if err != nil {
		return WalletBalanceResult{}, err
	}
	return WalletBalanceResult{Amount: out.Data.Wallet.Jumlah}, nil
}

// BankList mengambil daftar bank tujuan transfer beserta fee live-nya.
func (c *client) BankList(ctx context.Context) ([]BankListItem, error) {
	var out bankListResponse
	err := withRetry(func() error { return c.authedGET(ctx, "/transfer/bank-lists", &out) })
	if err != nil {
		return nil, err
	}
	return out.Result, nil
}
