// Package gsheet is a dependency-light Google Sheets + Drive client used by
// modules/dynamicform to mirror form responses into one spreadsheet per form.
//
// It talks to the Sheets v4 and Drive v3 REST APIs directly over net/http and
// authenticates with a service-account JWT bearer grant (or an OAuth-user
// refresh token), so it adds no new module dependencies. When no credentials
// are configured, New returns a disabled client whose every method is a no-op —
// the same graceful-degradation pattern as an empty GIPHY_API_KEY.
package gsheet

// serviceAccountKey is the subset of a Google service-account JSON key file we
// need for the JWT bearer flow. Pure data.
type serviceAccountKey struct {
	Type         string `json:"type"`
	ClientEmail  string `json:"client_email"`
	PrivateKey   string `json:"private_key"`
	PrivateKeyID string `json:"private_key_id"`
	TokenURI     string `json:"token_uri"`
}

// tokenResponse is the OAuth2 token endpoint reply. Pure data.
type tokenResponse struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   int    `json:"expires_in"`
	Error       string `json:"error"`
	ErrorDesc   string `json:"error_description"`
}
