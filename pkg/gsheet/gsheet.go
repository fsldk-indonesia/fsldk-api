package gsheet

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"fsldk-api/config"

	"github.com/golang-jwt/jwt/v5"
)

const (
	scopes        = "https://www.googleapis.com/auth/spreadsheets https://www.googleapis.com/auth/drive.file"
	sheetsBaseURL = "https://sheets.googleapis.com/v4/spreadsheets"
	driveBaseURL  = "https://www.googleapis.com/drive/v3/files"
	oauthTokenURL = "https://oauth2.googleapis.com/token"
	callTimeout   = 15 * time.Second
)

// Client is the Google Sheets mirror contract. Every method is a no-op (and
// returns nil) when Enabled() is false; a non-nil error only ever signals a
// real API failure.
type Client interface {
	Enabled() bool
	CreateSpreadsheet(ctx context.Context, title, folderID string) (id, sheetURL string, err error)
	SetHeaderRow(ctx context.Context, id, tab string, headers []string) error
	ReorderColumns(ctx context.Context, id, tab string, newHeaders []string) error
	AppendRow(ctx context.Context, id, tab string, row []string) (rowIndex int, err error)
	UpdateRowByIndex(ctx context.Context, id, tab string, rowIndex int, row []string) error
	DeleteRowByIndex(ctx context.Context, id, tab string, rowIndex int) error
	FindRowBySubmissionID(ctx context.Context, id, tab string, submissionID int64) (rowIndex int, err error)
	ClearDataRows(ctx context.Context, id, tab string) error
	Share(ctx context.Context, id string, emails []string) error
}

// New picks the service-account flow (GSHEET_CREDENTIALS_JSON) or the
// OAuth-user flow (client id/secret/refresh token); if neither is configured,
// or the global switch is off, it returns a disabled no-op client.
func New(cfg config.AppConfig) Client {
	if !cfg.GSheetSyncEnabled {
		return disabledClient{}
	}
	if cfg.GSheetCredentialsJSON != "" {
		raw, err := os.ReadFile(cfg.GSheetCredentialsJSON)
		if err != nil {
			return disabledClient{}
		}
		var key serviceAccountKey
		if json.Unmarshal(raw, &key) != nil || key.ClientEmail == "" || key.PrivateKey == "" {
			return disabledClient{}
		}
		if key.TokenURI == "" {
			key.TokenURI = oauthTokenURL
		}
		return &apiClient{http: &http.Client{Timeout: callTimeout}, saKey: &key, impersonate: cfg.GSheetImpersonateEmail}
	}
	if cfg.GSheetOAuthClientID != "" && cfg.GSheetOAuthClientSecret != "" && cfg.GSheetOAuthRefreshToken != "" {
		return &apiClient{http: &http.Client{Timeout: callTimeout}, oauth: &oauthCreds{
			clientID: cfg.GSheetOAuthClientID, clientSecret: cfg.GSheetOAuthClientSecret, refreshToken: cfg.GSheetOAuthRefreshToken,
		}}
	}
	return disabledClient{}
}

// ---------------------------------------------------------------------------
// disabled client
// ---------------------------------------------------------------------------

type disabledClient struct{}

func (disabledClient) Enabled() bool { return false }
func (disabledClient) CreateSpreadsheet(context.Context, string, string) (string, string, error) {
	return "", "", nil
}
func (disabledClient) SetHeaderRow(context.Context, string, string, []string) error   { return nil }
func (disabledClient) ReorderColumns(context.Context, string, string, []string) error { return nil }
func (disabledClient) AppendRow(context.Context, string, string, []string) (int, error) {
	return 0, nil
}
func (disabledClient) UpdateRowByIndex(context.Context, string, string, int, []string) error {
	return nil
}
func (disabledClient) DeleteRowByIndex(context.Context, string, string, int) error { return nil }
func (disabledClient) FindRowBySubmissionID(context.Context, string, string, int64) (int, error) {
	return 0, nil
}
func (disabledClient) ClearDataRows(context.Context, string, string) error { return nil }
func (disabledClient) Share(context.Context, string, []string) error       { return nil }

// ---------------------------------------------------------------------------
// live client
// ---------------------------------------------------------------------------

type oauthCreds struct{ clientID, clientSecret, refreshToken string }

type apiClient struct {
	http        *http.Client
	saKey       *serviceAccountKey
	oauth       *oauthCreds
	impersonate string

	mu       sync.Mutex
	token    string
	tokenExp time.Time
	sheetIDs map[string]int64 // "spreadsheetID|tab" -> numeric sheetId
}

func (c *apiClient) Enabled() bool { return true }

// accessToken returns a cached bearer token, refreshing it a minute before expiry.
func (c *apiClient) accessToken(ctx context.Context) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.token != "" && time.Now().Before(c.tokenExp.Add(-time.Minute)) {
		return c.token, nil
	}
	var (
		tok tokenResponse
		err error
	)
	if c.saKey != nil {
		tok, err = c.saToken(ctx)
	} else {
		tok, err = c.refreshToken(ctx)
	}
	if err != nil {
		return "", err
	}
	if tok.AccessToken == "" {
		return "", fmt.Errorf("gsheet: empty access token (%s: %s)", tok.Error, tok.ErrorDesc)
	}
	c.token = tok.AccessToken
	c.tokenExp = time.Now().Add(time.Duration(tok.ExpiresIn) * time.Second)
	return c.token, nil
}

func (c *apiClient) saToken(ctx context.Context) (tokenResponse, error) {
	pk, err := jwt.ParseRSAPrivateKeyFromPEM([]byte(c.saKey.PrivateKey))
	if err != nil {
		return tokenResponse{}, fmt.Errorf("gsheet: parse service-account key: %w", err)
	}
	now := time.Now()
	claims := jwt.MapClaims{
		"iss":   c.saKey.ClientEmail,
		"scope": scopes,
		"aud":   c.saKey.TokenURI,
		"iat":   now.Unix(),
		"exp":   now.Add(time.Hour).Unix(),
	}
	if c.impersonate != "" {
		claims["sub"] = c.impersonate
	}
	assertion, err := jwt.NewWithClaims(jwt.SigningMethodRS256, claims).SignedString(pk)
	if err != nil {
		return tokenResponse{}, err
	}
	form := url.Values{
		"grant_type": {"urn:ietf:params:oauth:grant-type:jwt-bearer"},
		"assertion":  {assertion},
	}
	return c.postToken(ctx, c.saKey.TokenURI, form)
}

func (c *apiClient) refreshToken(ctx context.Context) (tokenResponse, error) {
	form := url.Values{
		"client_id":     {c.oauth.clientID},
		"client_secret": {c.oauth.clientSecret},
		"refresh_token": {c.oauth.refreshToken},
		"grant_type":    {"refresh_token"},
	}
	return c.postToken(ctx, oauthTokenURL, form)
}

func (c *apiClient) postToken(ctx context.Context, endpoint string, form url.Values) (tokenResponse, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return tokenResponse{}, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := c.http.Do(req)
	if err != nil {
		return tokenResponse{}, err
	}
	defer resp.Body.Close()
	var out tokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return tokenResponse{}, err
	}
	if resp.StatusCode/100 != 2 {
		return out, fmt.Errorf("gsheet: token endpoint %d: %s %s", resp.StatusCode, out.Error, out.ErrorDesc)
	}
	return out, nil
}

// do performs an authenticated JSON request and decodes the reply into out
// (out may be nil). A non-2xx status becomes an error carrying the response body.
func (c *apiClient) do(ctx context.Context, method, endpoint string, body, out any) error {
	tok, err := c.accessToken(ctx)
	if err != nil {
		return err
	}
	var reader *bytes.Reader
	if body != nil {
		b, mErr := json.Marshal(body)
		if mErr != nil {
			return mErr
		}
		reader = bytes.NewReader(b)
	} else {
		reader = bytes.NewReader(nil)
	}
	req, err := http.NewRequestWithContext(ctx, method, endpoint, reader)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+tok)
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		buf := new(bytes.Buffer)
		_, _ = buf.ReadFrom(resp.Body)
		return fmt.Errorf("gsheet: %s %s -> %d: %s", method, endpoint, resp.StatusCode, strings.TrimSpace(buf.String()))
	}
	if out != nil {
		return json.NewDecoder(resp.Body).Decode(out)
	}
	return nil
}

// ---------------------------------------------------------------------------
// operations
// ---------------------------------------------------------------------------

func (c *apiClient) CreateSpreadsheet(ctx context.Context, title, folderID string) (string, string, error) {
	// Create through Drive so the file lands in folderID (and a Shared Drive
	// when folderID points at one). Then rename the default tab.
	createBody := map[string]any{
		"name":     title,
		"mimeType": "application/vnd.google-apps.spreadsheet",
	}
	if folderID != "" {
		createBody["parents"] = []string{folderID}
	}
	var created struct {
		ID string `json:"id"`
	}
	if err := c.do(ctx, http.MethodPost, driveBaseURL+"?supportsAllDrives=true&fields=id", createBody, &created); err != nil {
		return "", "", err
	}
	sheetURL := "https://docs.google.com/spreadsheets/d/" + created.ID + "/edit"
	return created.ID, sheetURL, nil
}

// sheetID resolves (and caches) the numeric sheetId of a tab, creating the tab
// if it does not exist yet.
func (c *apiClient) sheetID(ctx context.Context, spreadsheetID, tab string) (int64, error) {
	cacheKey := spreadsheetID + "|" + tab
	c.mu.Lock()
	if c.sheetIDs == nil {
		c.sheetIDs = map[string]int64{}
	}
	if id, ok := c.sheetIDs[cacheKey]; ok {
		c.mu.Unlock()
		return id, nil
	}
	c.mu.Unlock()

	var meta struct {
		Sheets []struct {
			Properties struct {
				SheetID int64  `json:"sheetId"`
				Title   string `json:"title"`
			} `json:"properties"`
		} `json:"sheets"`
	}
	if err := c.do(ctx, http.MethodGet, sheetsBaseURL+"/"+spreadsheetID+"?fields=sheets.properties", nil, &meta); err != nil {
		return 0, err
	}
	for _, s := range meta.Sheets {
		if s.Properties.Title == tab {
			c.cacheSheetID(cacheKey, s.Properties.SheetID)
			return s.Properties.SheetID, nil
		}
	}
	// Tab missing — rename the first sheet to tab (a brand-new spreadsheet has
	// exactly one "Sheet1"), or add a new tab if there is somehow none.
	if len(meta.Sheets) > 0 {
		first := meta.Sheets[0].Properties.SheetID
		req := map[string]any{"requests": []any{map[string]any{
			"updateSheetProperties": map[string]any{
				"properties": map[string]any{"sheetId": first, "title": tab},
				"fields":     "title",
			},
		}}}
		if err := c.do(ctx, http.MethodPost, sheetsBaseURL+"/"+spreadsheetID+":batchUpdate", req, nil); err != nil {
			return 0, err
		}
		c.cacheSheetID(cacheKey, first)
		return first, nil
	}
	var added struct {
		Replies []struct {
			AddSheet struct {
				Properties struct {
					SheetID int64 `json:"sheetId"`
				} `json:"properties"`
			} `json:"addSheet"`
		} `json:"replies"`
	}
	req := map[string]any{"requests": []any{map[string]any{
		"addSheet": map[string]any{"properties": map[string]any{"title": tab}},
	}}}
	if err := c.do(ctx, http.MethodPost, sheetsBaseURL+"/"+spreadsheetID+":batchUpdate", req, &added); err != nil {
		return 0, err
	}
	if len(added.Replies) == 0 {
		return 0, fmt.Errorf("gsheet: addSheet returned no reply")
	}
	id := added.Replies[0].AddSheet.Properties.SheetID
	c.cacheSheetID(cacheKey, id)
	return id, nil
}

func (c *apiClient) cacheSheetID(key string, id int64) {
	c.mu.Lock()
	if c.sheetIDs == nil {
		c.sheetIDs = map[string]int64{}
	}
	c.sheetIDs[key] = id
	c.mu.Unlock()
}

func (c *apiClient) SetHeaderRow(ctx context.Context, id, tab string, headers []string) error {
	rng := tab + "!A1:" + colLetter(len(headers)) + "1"
	endpoint := fmt.Sprintf("%s/%s/values/%s?valueInputOption=RAW", sheetsBaseURL, id, url.PathEscape(rng))
	body := map[string]any{"values": [][]string{headers}}
	if err := c.do(ctx, http.MethodPut, endpoint, body, nil); err != nil {
		return err
	}
	// Best-effort: freeze + bold the header row. Failure here is not fatal.
	if sid, sErr := c.sheetID(ctx, id, tab); sErr == nil {
		fmtReq := map[string]any{"requests": []any{
			map[string]any{"updateSheetProperties": map[string]any{
				"properties": map[string]any{"sheetId": sid, "gridProperties": map[string]any{"frozenRowCount": 1}},
				"fields":     "gridProperties.frozenRowCount",
			}},
			map[string]any{"repeatCell": map[string]any{
				"range":  map[string]any{"sheetId": sid, "startRowIndex": 0, "endRowIndex": 1},
				"cell":   map[string]any{"userEnteredFormat": map[string]any{"textFormat": map[string]any{"bold": true}}},
				"fields": "userEnteredFormat.textFormat.bold",
			}},
		}}
		_ = c.do(ctx, http.MethodPost, sheetsBaseURL+"/"+id+":batchUpdate", fmtReq, nil)
	}
	return nil
}

func (c *apiClient) ReorderColumns(ctx context.Context, id, tab string, newHeaders []string) error {
	values, err := c.getValues(ctx, id, tab+"!A1:ZZ")
	if err != nil {
		return err
	}
	if len(values) == 0 {
		return c.SetHeaderRow(ctx, id, tab, newHeaders)
	}
	oldHeaders := values[0]
	oldIndex := make(map[string]int, len(oldHeaders))
	for i, h := range oldHeaders {
		oldIndex[h] = i
	}
	remapped := make([][]string, len(values))
	remapped[0] = newHeaders
	for r := 1; r < len(values); r++ {
		src := values[r]
		row := make([]string, len(newHeaders))
		for j, h := range newHeaders {
			if oi, ok := oldIndex[h]; ok && oi < len(src) {
				row[j] = src[oi]
			}
		}
		remapped[r] = row
	}
	if err := c.clearRange(ctx, id, tab+"!A1:ZZ"); err != nil {
		return err
	}
	rng := tab + "!A1"
	endpoint := fmt.Sprintf("%s/%s/values/%s?valueInputOption=RAW", sheetsBaseURL, id, url.PathEscape(rng))
	return c.do(ctx, http.MethodPut, endpoint, map[string]any{"values": remapped}, nil)
}

func (c *apiClient) AppendRow(ctx context.Context, id, tab string, row []string) (int, error) {
	rng := tab + "!A1"
	endpoint := fmt.Sprintf("%s/%s/values/%s:append?valueInputOption=RAW&insertDataOption=INSERT_ROWS",
		sheetsBaseURL, id, url.PathEscape(rng))
	var out struct {
		Updates struct {
			UpdatedRange string `json:"updatedRange"`
		} `json:"updates"`
	}
	if err := c.do(ctx, http.MethodPost, endpoint, map[string]any{"values": [][]string{row}}, &out); err != nil {
		return 0, err
	}
	return rowNumberFromRange(out.Updates.UpdatedRange), nil
}

func (c *apiClient) UpdateRowByIndex(ctx context.Context, id, tab string, rowIndex int, row []string) error {
	if rowIndex < 1 {
		return fmt.Errorf("gsheet: invalid row index %d", rowIndex)
	}
	rng := fmt.Sprintf("%s!A%d:%s%d", tab, rowIndex, colLetter(len(row)), rowIndex)
	endpoint := fmt.Sprintf("%s/%s/values/%s?valueInputOption=RAW", sheetsBaseURL, id, url.PathEscape(rng))
	return c.do(ctx, http.MethodPut, endpoint, map[string]any{"values": [][]string{row}}, nil)
}

func (c *apiClient) DeleteRowByIndex(ctx context.Context, id, tab string, rowIndex int) error {
	if rowIndex < 1 {
		return fmt.Errorf("gsheet: invalid row index %d", rowIndex)
	}
	sid, err := c.sheetID(ctx, id, tab)
	if err != nil {
		return err
	}
	req := map[string]any{"requests": []any{map[string]any{
		"deleteDimension": map[string]any{
			"range": map[string]any{
				"sheetId":    sid,
				"dimension":  "ROWS",
				"startIndex": rowIndex - 1,
				"endIndex":   rowIndex,
			},
		},
	}}}
	return c.do(ctx, http.MethodPost, sheetsBaseURL+"/"+id+":batchUpdate", req, nil)
}

func (c *apiClient) FindRowBySubmissionID(ctx context.Context, id, tab string, submissionID int64) (int, error) {
	values, err := c.getValues(ctx, id, tab+"!B2:B")
	if err != nil {
		return 0, err
	}
	want := strconv.FormatInt(submissionID, 10)
	for i, r := range values {
		if len(r) > 0 && strings.TrimSpace(r[0]) == want {
			return i + 2, nil // B2 is row 2
		}
	}
	return 0, nil // not found — caller decides (append fallback)
}

func (c *apiClient) ClearDataRows(ctx context.Context, id, tab string) error {
	return c.clearRange(ctx, id, tab+"!A2:ZZ")
}

func (c *apiClient) Share(ctx context.Context, id string, emails []string) error {
	var firstErr error
	for _, email := range emails {
		email = strings.TrimSpace(email)
		if email == "" {
			continue
		}
		endpoint := driveBaseURL + "/" + id + "/permissions?sendNotificationEmail=false&supportsAllDrives=true"
		body := map[string]any{"type": "user", "role": "writer", "emailAddress": email}
		if err := c.do(ctx, http.MethodPost, endpoint, body, nil); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

func (c *apiClient) getValues(ctx context.Context, id, a1Range string) ([][]string, error) {
	endpoint := fmt.Sprintf("%s/%s/values/%s?majorDimension=ROWS", sheetsBaseURL, id, url.PathEscape(a1Range))
	var out struct {
		Values [][]string `json:"values"`
	}
	if err := c.do(ctx, http.MethodGet, endpoint, nil, &out); err != nil {
		return nil, err
	}
	return out.Values, nil
}

func (c *apiClient) clearRange(ctx context.Context, id, a1Range string) error {
	endpoint := fmt.Sprintf("%s/%s/values/%s:clear", sheetsBaseURL, id, url.PathEscape(a1Range))
	return c.do(ctx, http.MethodPost, endpoint, map[string]any{}, nil)
}

// colLetter converts a 1-based column number to its A1 letter (1->A, 27->AA).
func colLetter(n int) string {
	if n < 1 {
		n = 1
	}
	var sb strings.Builder
	for n > 0 {
		n--
		sb.WriteByte(byte('A' + n%26))
		n /= 26
	}
	// reverse
	b := []byte(sb.String())
	for i, j := 0, len(b)-1; i < j; i, j = i+1, j-1 {
		b[i], b[j] = b[j], b[i]
	}
	return string(b)
}

// rowNumberFromRange extracts the row number from an A1 range like "Responses!A5:F5".
func rowNumberFromRange(a1 string) int {
	if idx := strings.LastIndex(a1, "!"); idx >= 0 {
		a1 = a1[idx+1:]
	}
	if idx := strings.Index(a1, ":"); idx >= 0 {
		a1 = a1[:idx]
	}
	digits := strings.TrimLeftFunc(a1, func(r rune) bool { return r < '0' || r > '9' })
	n, _ := strconv.Atoi(digits)
	return n
}
