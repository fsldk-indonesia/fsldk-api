package googleauth

// Payload berisi klaim relevan dari Google ID Token. Murni struct data
// (bentuk response JSON dari endpoint tokeninfo Google), tanpa method.
type Payload struct {
	Sub           string `json:"sub"`
	Email         string `json:"email"`
	EmailVerified string `json:"email_verified"`
	Name          string `json:"name"`
	Picture       string `json:"picture"`
	Aud           string `json:"aud"`
}
