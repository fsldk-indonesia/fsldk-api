// Package googleauth memverifikasi Google ID Token melalui endpoint tokeninfo
// resmi Google. Pendekatan ini memvalidasi tanda tangan & masa berlaku token
// di sisi Google tanpa memerlukan pustaka JWKS tambahan.
package googleauth

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

// Verifier memverifikasi ID token terhadap client ID aplikasi.
type Verifier struct {
	clientID     string
	tokenInfoURL string
	client       *http.Client
}

// NewVerifier membuat Verifier baru. tokenInfoURL diambil dari konfigurasi
// (GOOGLE_TOKENINFO_URL) agar endpoint tidak ditulis tetap (hardcode) di kode.
func NewVerifier(clientID, tokenInfoURL string) *Verifier {
	return &Verifier{clientID: clientID, tokenInfoURL: tokenInfoURL, client: &http.Client{Timeout: 10 * time.Second}}
}

// Verify memvalidasi idToken dan mengembalikan payload-nya.
func (v *Verifier) Verify(idToken string) (*Payload, error) {
	endpoint := v.tokenInfoURL + "?id_token=" + url.QueryEscape(idToken)
	resp, err := v.client.Get(endpoint)
	if err != nil {
		return nil, fmt.Errorf("gagal menghubungi Google: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, errors.New("ID token Google tidak valid")
	}

	var p Payload
	if err := json.NewDecoder(resp.Body).Decode(&p); err != nil {
		return nil, errors.New("gagal membaca payload Google")
	}

	// Verifikasi audience harus sama dengan Client ID aplikasi.
	if v.clientID != "" && p.Aud != v.clientID {
		return nil, errors.New("audience token Google tidak sesuai")
	}
	if p.Email == "" {
		return nil, errors.New("email tidak tersedia pada token Google")
	}
	return &p, nil
}
