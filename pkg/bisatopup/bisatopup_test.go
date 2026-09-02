package bisatopup

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestBuildAndVerifySignature(t *testing.T) {
	sig := BuildSignature("mitra-x", "TXN-1")
	if !VerifySignature("mitra-x", "TXN-1", sig) {
		t.Fatal("expected signature built by BuildSignature to verify successfully")
	}
	if VerifySignature("mitra-x", "TXN-1", "signature-palsu") {
		t.Fatal("expected forged signature to fail verification")
	}
	if VerifySignature("mitra-lain", "TXN-1", sig) {
		t.Fatal("expected signature to be scoped to the username that produced it")
	}
}

// newTestServer menyimulasikan endpoint /login + endpoint tambahan yang
// diberikan, menghitung jumlah pemanggilan /login untuk menguji cache token.
func newTestServer(t *testing.T, loginCalls *int32, extra map[string]http.HandlerFunc) *httptest.Server {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/login", func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(loginCalls, 1)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"error":   false,
			"message": "Login success",
			"data":    map[string]interface{}{"access_token": "test-token", "token_type": "bearer", "expires_in": 3600},
		})
	})
	for path, h := range extra {
		mux.HandleFunc(path, h)
	}
	return httptest.NewServer(mux)
}

func TestCreateQRISTransaction_Success(t *testing.T) {
	var loginCalls int32
	srv := newTestServer(t, &loginCalls, map[string]http.HandlerFunc{
		"/api/payment/transaction": func(w http.ResponseWriter, r *http.Request) {
			if auth := r.Header.Get("Authorization"); auth != "Bearer test-token" {
				t.Errorf("expected Bearer test-token, got %q", auth)
			}
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"error":   false,
				"message": "success",
				"data": map[string]interface{}{
					"id": 6, "payment_id": 33, "status_id": 2, "transaction_id": "TXN-1",
					"transaction_total": "20203", "qr_code": "0002...", "payment_code": nil, "payment_links": nil,
				},
			})
		},
	})
	defer srv.Close()

	c := NewClient(Config{Username: "mitra-x", PasswordAPI: "secret", Env: "dev", BaseURLDev: srv.URL, QrisPaymentID: 33})
	txn, err := c.CreateQRISTransaction(context.Background(), CreateQRISTransactionParams{
		TransactionID: "TXN-1", Nominal: 20203, ExpiredDate: time.Now().Add(24 * time.Hour),
		TransactionName: "Donasi", TransactionDesc: "Donasi Kantong Amal",
	})
	if err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}
	if txn.TransactionTotal != "20203" || txn.QrCode == "" {
		t.Fatalf("unexpected transaction result: %+v", txn)
	}
	if loginCalls != 1 {
		t.Fatalf("expected exactly 1 login call, got %d", loginCalls)
	}

	// Panggilan kedua harus reuse token yang sudah dicache, bukan login ulang.
	if _, err := c.CreateQRISTransaction(context.Background(), CreateQRISTransactionParams{TransactionID: "TXN-2", Nominal: 1000}); err != nil {
		t.Fatalf("expected second call to succeed, got error: %v", err)
	}
	if loginCalls != 1 {
		t.Fatalf("expected token to be cached (still 1 login call), got %d", loginCalls)
	}
}

func TestCreateQRISTransaction_GatewayRejected(t *testing.T) {
	var loginCalls int32
	srv := newTestServer(t, &loginCalls, map[string]http.HandlerFunc{
		"/api/payment/transaction": func(w http.ResponseWriter, r *http.Request) {
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"error": true, "message": "nominal invalid"})
		},
	})
	defer srv.Close()

	c := NewClient(Config{Username: "mitra-x", PasswordAPI: "secret", Env: "dev", BaseURLDev: srv.URL, QrisPaymentID: 33})
	_, err := c.CreateQRISTransaction(context.Background(), CreateQRISTransactionParams{TransactionID: "TXN-1", Nominal: 20203})
	if !errors.Is(err, ErrGatewayRejected) {
		t.Fatalf("expected ErrGatewayRejected, got %v", err)
	}
}

func TestCreateQRISTransaction_NetworkFailureIsNotGatewayRejected(t *testing.T) {
	// Server yang selalu 500 mensimulasikan kegagalan jaringan/provider,
	// bukan penolakan bisnis eksplisit — harus dibedakan (ErrGatewayRejected
	// tidak match) agar caller memetakan ke ProviderError, bukan PaymentFailed.
	var loginCalls int32
	srv := newTestServer(t, &loginCalls, map[string]http.HandlerFunc{
		"/api/payment/transaction": func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte("internal error"))
		},
	})
	defer srv.Close()

	c := NewClient(Config{Username: "mitra-x", PasswordAPI: "secret", Env: "dev", BaseURLDev: srv.URL, QrisPaymentID: 33})
	_, err := c.CreateQRISTransaction(context.Background(), CreateQRISTransactionParams{TransactionID: "TXN-1", Nominal: 20203})
	if err == nil {
		t.Fatal("expected an error for HTTP 500 response")
	}
	if errors.Is(err, ErrGatewayRejected) {
		t.Fatal("HTTP-level failure must not be classified as ErrGatewayRejected")
	}
}

func TestWalletBalance_RetriesOnFailureThenSucceeds(t *testing.T) {
	var loginCalls int32
	var attempts int32
	srv := newTestServer(t, &loginCalls, map[string]http.HandlerFunc{
		"/api/account-info": func(w http.ResponseWriter, r *http.Request) {
			if atomic.AddInt32(&attempts, 1) < 2 {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"error": false, "data": map[string]interface{}{"wallet": map[string]interface{}{"jumlah": 84254653}},
			})
		},
	})
	defer srv.Close()

	c := NewClient(Config{Username: "mitra-x", PasswordAPI: "secret", Env: "dev", BaseURLDev: srv.URL})
	start := time.Now()
	res, err := c.WalletBalance(context.Background())
	if err != nil {
		t.Fatalf("expected retry to eventually succeed, got error: %v", err)
	}
	if res.Amount != 84254653 {
		t.Fatalf("unexpected wallet amount: %d", res.Amount)
	}
	if time.Since(start) < 2*time.Second {
		t.Fatal("expected at least one 2s backoff between retries")
	}
}

func TestLogin_FailureSurfacesError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/login") {
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"error": true, "message": "invalid credentials"})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	c := NewClient(Config{Username: "mitra-x", PasswordAPI: "wrong", Env: "dev", BaseURLDev: srv.URL})
	_, err := c.WalletBalance(context.Background())
	if err == nil {
		t.Fatal("expected error when login is rejected")
	}
}
