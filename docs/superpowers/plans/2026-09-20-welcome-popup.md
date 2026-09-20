# Welcome Popup (dynamic, CMS-managed) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Let Super Admin edit three raw-content fields (HTML/JS/CSS) in a CMS form, and have the public homepage render them live as a popup — no code deploy needed to change campaigns.

**Architecture:** New minimal singleton module `welcomepopup` in both repos (one DB row, no list/pagination — same "trimmed shape" precedent this codebase already uses for `zakat`/`upload`). Backend exposes a permission-gated CMS `GET`/`PUT /welcome-popup` and an unauthenticated `GET /public/welcome-popup` that only ever returns the 4 content fields (never audit metadata). Frontend CMS is one edit-only page (toggle + 3 textareas). Frontend public side is one non-visual component on the homepage that fetches the public endpoint and injects the HTML/CSS/JS straight into the DOM via native browser APIs — Angular's sanitizer is deliberately bypassed by never touching it (no `[innerHTML]` binding at all; everything is `document.createElement`/`appendChild`).

**Tech Stack:** Go 1.25 + Gin + GORM + MySQL (`fsldk-api`); Angular 19 standalone components + signals (`fsldk-web`).

**Spec:** `fsldk-api/docs/superpowers/specs/2026-09-20-welcome-popup-design.md`

## Global Constraints

- New module is a **singleton** — exactly one DB row, id fixed to `1`. No list endpoint, no pagination, no create/delete.
- CMS access is **Super Admin only** — permissions `welcomepopup.view`/`welcomepopup.update` seeded to the `Super Admin` role exclusively in the migration (copy the exact `INSERT IGNORE ... WHERE r.roleName = 'Super Admin'` pattern from `fsldk-api/migrations/0008_setting.up.sql`).
- The public endpoint (`GET /public/welcome-popup`) must **never** return `updatedDate`/`updatedBy` — it uses a separate `PublicResponse` DTO with only 4 fields, not the CMS `Response` DTO.
- No sanitization of `htmlContent`/`jsContent`/`cssContent` anywhere — this is intentional per the spec's Security section. Do not add `bluemonday`/DOMPurify/etc.
- A `<script>` tag embedded inside `htmlContent` will **not** execute (browser platform behavior when parsed via `innerHTML`) — only the dedicated `jsContent` field (appended as a real `<script>` DOM node) executes. This must be stated as a hint in the CMS form's HTML field, not silently left as a surprise.
- Follow this codebase's module shape exactly: `_model`/`_dto` are pure data (zero methods); `_repository`/`_service`/`_handler` are pure logic (zero data fields except the `XxxImpl` DI receiver struct).
- No new third-party dependencies (no new Go modules, no new npm packages).

---

### Task 1: Backend data foundation — migration, permission constants, model, dto, repository

**Files:**
- Create: `fsldk-api/migrations/0040_welcomepopup.up.sql`
- Modify: `fsldk-api/constants/constants.go` (add 2 permission constants)
- Create: `fsldk-api/modules/welcomepopup/welcomepopup_model/welcomepopup_model.go`
- Create: `fsldk-api/modules/welcomepopup/welcomepopup_dto/welcomepopup_dto.go`
- Create: `fsldk-api/modules/welcomepopup/welcomepopup_repository/welcomepopup_repository.go`
- Create: `fsldk-api/modules/welcomepopup/welcomepopup_repository/welcomepopup_repository_impl.go`

**Interfaces:**
- Produces: `welcomepopup_model.WelcomePopup` struct, `welcomepopup_model.SingletonID` constant (`int64 = 1`); `welcomepopup_dto.Response`, `welcomepopup_dto.PublicResponse`, `welcomepopup_dto.UpdateRequest` structs; `welcomepopup_repository.Repository` interface with `Get(ctx) (welcomepopup_model.WelcomePopup, error)` and `Update(ctx, welcomepopup_model.WelcomePopup) error`; `welcomepopup_repository.NewRepository(db *gorm.DB) Repository`.

This task has no independent unit test — it's schema + pure data + a thin GORM pass-through, mirroring `setting_repository` which also has none. It's verified by compilation in Task 2 and by the migration actually running in Task 3.

- [ ] **Step 1: Write the migration**

Create `fsldk-api/migrations/0040_welcomepopup.up.sql`:

```sql
-- ============================================================
-- FSLDK API — Welcome Popup (ms_welcome_popup)
-- Modul singleton: satu baris (id=1) berisi HTML/JS/CSS mentah yang
-- disuntikkan apa adanya ke halaman Beranda publik. Super Admin only —
-- tidak ada sanitasi di mana pun (lihat design spec, bagian Security).
-- Idempoten: aman dijalankan ulang (CREATE TABLE IF NOT EXISTS / INSERT IGNORE).
-- ============================================================

CREATE TABLE IF NOT EXISTS ms_welcome_popup (
    id          TINYINT UNSIGNED PRIMARY KEY,
    isEnabled   BOOLEAN NOT NULL DEFAULT FALSE,
    htmlContent LONGTEXT NULL,
    jsContent   LONGTEXT NULL,
    cssContent  LONGTEXT NULL,
    updatedDate DATETIME NULL,
    updatedBy   BIGINT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

INSERT IGNORE INTO ms_welcome_popup (id, isEnabled) VALUES (1, FALSE);

INSERT IGNORE INTO lk_permission (permissionCode, permissionName, moduleName, menuLabel, menuIcon, menuRoute, sortOrder)
VALUES ('welcomepopup.view',   'Lihat Welcome Popup', 'welcomepopup', 'Welcome Popup', 'megaphone', '/cms/welcome-popup', 98),
       ('welcomepopup.update', 'Ubah Welcome Popup',  'welcomepopup', NULL, NULL, NULL, 98);

INSERT IGNORE INTO map_role_permission (roleID, permissionID)
SELECT r.roleID, p.permissionID FROM ms_role r, lk_permission p
WHERE r.roleName = 'Super Admin' AND p.permissionCode IN ('welcomepopup.view', 'welcomepopup.update');
-- HANYA Super Admin — raw HTML/JS/CSS ini dieksekusi apa adanya di public
-- site tanpa sanitasi, sama seperti setting.* (0008_setting.up.sql).
```

- [ ] **Step 2: Add permission constants**

In `fsldk-api/constants/constants.go`, find this exact block:

```go
	PermSettingView   = "setting.view"
	PermSettingUpdate = "setting.update"
```

Replace it with:

```go
	PermSettingView   = "setting.view"
	PermSettingUpdate = "setting.update"

	PermWelcomePopupView   = "welcomepopup.view"
	PermWelcomePopupUpdate = "welcomepopup.update"
```

- [ ] **Step 3: Write the model**

Create `fsldk-api/modules/welcomepopup/welcomepopup_model/welcomepopup_model.go`:

```go
// Package welcomepopup_model memuat entitas modul welcomepopup (popup
// selamat datang di halaman Beranda publik — satu baris singleton, HTML/JS/
// CSS mentah dikelola Super Admin lewat CMS). Seluruhnya murni struct data
// (tanpa function/method), kecuali konstanta SingletonID.
package welcomepopup_model

import "time"

// SingletonID adalah satu-satunya id baris yang pernah ada di ms_welcome_popup —
// tabel ini sengaja tidak pernah punya baris kedua.
const SingletonID int64 = 1

// WelcomePopup merepresentasikan satu baris ms_welcome_popup.
type WelcomePopup struct {
	ID          int64      `gorm:"column:id;primaryKey"`
	IsEnabled   bool       `gorm:"column:isEnabled"`
	HTMLContent *string    `gorm:"column:htmlContent"`
	JSContent   *string    `gorm:"column:jsContent"`
	CSSContent  *string    `gorm:"column:cssContent"`
	UpdatedDate *time.Time `gorm:"column:updatedDate"`
	UpdatedBy   *int64     `gorm:"column:updatedBy"`
}
```

- [ ] **Step 4: Write the DTOs**

Create `fsldk-api/modules/welcomepopup/welcomepopup_dto/welcomepopup_dto.go`:

```go
// Package welcomepopup_dto memuat DTO request/response modul welcomepopup.
// Seluruhnya murni struct data (tanpa function/method).
package welcomepopup_dto

// Response adalah representasi lengkap untuk CMS (Super Admin) — termasuk
// metadata audit.
type Response struct {
	IsEnabled   bool   `json:"isEnabled"`
	HTMLContent string `json:"htmlContent"`
	JSContent   string `json:"jsContent"`
	CSSContent  string `json:"cssContent"`
	UpdatedDate string `json:"updatedDate"`
}

// PublicResponse adalah representasi untuk endpoint publik (tanpa auth) —
// SENGAJA cuma 4 field ini, tidak pernah membawa UpdatedDate/UpdatedBy.
type PublicResponse struct {
	IsEnabled   bool   `json:"isEnabled"`
	HTMLContent string `json:"htmlContent"`
	JSContent   string `json:"jsContent"`
	CSSContent  string `json:"cssContent"`
}

// UpdateRequest adalah body memperbarui Welcome Popup. Tidak ada
// validate:"required" pada field kontennya dengan sengaja — mengosongkan
// field adalah state yang sah (mis. mematikan tanpa menghapus histori).
type UpdateRequest struct {
	IsEnabled   bool   `json:"isEnabled"`
	HTMLContent string `json:"htmlContent"`
	JSContent   string `json:"jsContent"`
	CSSContent  string `json:"cssContent"`
}
```

- [ ] **Step 5: Write the repository interface**

Create `fsldk-api/modules/welcomepopup/welcomepopup_repository/welcomepopup_repository.go`:

```go
// Package welcomepopup_repository adalah lapisan akses data modul
// welcomepopup (GORM).
package welcomepopup_repository

import (
	"context"

	"fsldk-api/modules/welcomepopup/welcomepopup_model"
)

// Repository adalah kontrak akses data Welcome Popup — selalu beroperasi
// pada satu baris singleton (welcomepopup_model.SingletonID).
type Repository interface {
	Get(ctx context.Context) (welcomepopup_model.WelcomePopup, error)
	Update(ctx context.Context, popup welcomepopup_model.WelcomePopup) error
}
```

- [ ] **Step 6: Write the repository implementation**

Create `fsldk-api/modules/welcomepopup/welcomepopup_repository/welcomepopup_repository_impl.go`:

```go
package welcomepopup_repository

import (
	"context"
	"time"

	"fsldk-api/modules/welcomepopup/welcomepopup_model"

	"gorm.io/gorm"
)

// RepositoryImpl adalah implementasi Repository berbasis GORM.
type RepositoryImpl struct{ db *gorm.DB }

// NewRepository membuat implementasi Repository.
func NewRepository(db *gorm.DB) Repository { return &RepositoryImpl{db: db} }

func (r *RepositoryImpl) Get(ctx context.Context) (welcomepopup_model.WelcomePopup, error) {
	var w welcomepopup_model.WelcomePopup
	err := r.db.WithContext(ctx).Table("ms_welcome_popup").
		Where("id = ?", welcomepopup_model.SingletonID).Take(&w).Error
	return w, err
}

func (r *RepositoryImpl) Update(ctx context.Context, popup welcomepopup_model.WelcomePopup) error {
	return r.db.WithContext(ctx).Table("ms_welcome_popup").Where("id = ?", welcomepopup_model.SingletonID).
		Updates(map[string]interface{}{
			"isEnabled":   popup.IsEnabled,
			"htmlContent": popup.HTMLContent,
			"jsContent":   popup.JSContent,
			"cssContent":  popup.CSSContent,
			"updatedDate": time.Now(),
			"updatedBy":   popup.UpdatedBy,
		}).Error
}
```

- [ ] **Step 7: Commit**

```bash
cd fsldk-api
git add migrations/0040_welcomepopup.up.sql constants/constants.go modules/welcomepopup/welcomepopup_model modules/welcomepopup/welcomepopup_dto modules/welcomepopup/welcomepopup_repository
git commit -m "feat(welcomepopup): add migration, model, dto, repository"
```

---

### Task 2: Backend service (TDD)

**Files:**
- Create: `fsldk-api/modules/welcomepopup/welcomepopup_service/welcomepopup_service.go`
- Create: `fsldk-api/modules/welcomepopup/welcomepopup_service/welcomepopup_service_impl.go`
- Test: `fsldk-api/modules/welcomepopup/welcomepopup_service/welcomepopup_service_test.go`

**Interfaces:**
- Consumes: `welcomepopup_repository.Repository` (`Get(ctx) (welcomepopup_model.WelcomePopup, error)`, `Update(ctx, welcomepopup_model.WelcomePopup) error`), `welcomepopup_model.WelcomePopup`/`SingletonID`, `welcomepopup_dto.Response`/`PublicResponse`/`UpdateRequest`, `fsldk-api/base/apperror` (`apperror.Internal(msg string) *AppError`).
- Produces: `welcomepopup_service.Service` interface — `Get(ctx) (welcomepopup_dto.Response, error)`, `GetPublic(ctx) (welcomepopup_dto.PublicResponse, error)`, `Update(ctx, welcomepopup_dto.UpdateRequest, actorID int64) (welcomepopup_dto.Response, error)`; `welcomepopup_service.NewService(repo welcomepopup_repository.Repository) Service`.

This project has no existing Go test files anywhere — there is no mocking library convention to follow. Use only the standard `testing` package and a hand-written fake implementing `welcomepopup_repository.Repository`.

- [ ] **Step 1: Write the failing tests**

Create `fsldk-api/modules/welcomepopup/welcomepopup_service/welcomepopup_service_test.go`:

```go
package welcomepopup_service

import (
	"context"
	"encoding/json"
	"testing"

	"fsldk-api/modules/welcomepopup/welcomepopup_dto"
	"fsldk-api/modules/welcomepopup/welcomepopup_model"
)

// fakeRepo is a minimal in-memory stand-in for welcomepopup_repository.Repository.
type fakeRepo struct {
	data welcomepopup_model.WelcomePopup
}

func (f *fakeRepo) Get(ctx context.Context) (welcomepopup_model.WelcomePopup, error) {
	return f.data, nil
}

func (f *fakeRepo) Update(ctx context.Context, popup welcomepopup_model.WelcomePopup) error {
	f.data = popup
	return nil
}

func strPtr(s string) *string { return &s }

func TestGetPublic_NeverExposesAuditFields(t *testing.T) {
	repo := &fakeRepo{data: welcomepopup_model.WelcomePopup{
		ID: welcomepopup_model.SingletonID, IsEnabled: true,
		HTMLContent: strPtr("<div>hi</div>"), JSContent: strPtr("alert(1)"), CSSContent: strPtr("div{color:red}"),
	}}
	svc := NewService(repo)

	res, err := svc.GetPublic(context.Background())
	if err != nil {
		t.Fatalf("GetPublic() error = %v", err)
	}

	raw, err := json.Marshal(res)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}
	var keys map[string]interface{}
	if err := json.Unmarshal(raw, &keys); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	want := map[string]bool{"isEnabled": true, "htmlContent": true, "jsContent": true, "cssContent": true}
	if len(keys) != len(want) {
		t.Fatalf("PublicResponse JSON has %d keys %v, want exactly %v", len(keys), keys, want)
	}
	for k := range keys {
		if !want[k] {
			t.Errorf("PublicResponse leaked unexpected field %q", k)
		}
	}
}

func TestUpdate_PersistsNewContentAndIsReflectedByGet(t *testing.T) {
	repo := &fakeRepo{data: welcomepopup_model.WelcomePopup{ID: welcomepopup_model.SingletonID, IsEnabled: false}}
	svc := NewService(repo)

	if _, err := svc.Update(context.Background(), welcomepopup_dto.UpdateRequest{
		IsEnabled: true, HTMLContent: "<div>new</div>", JSContent: "run();", CSSContent: "body{}",
	}, 42); err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	got, err := svc.Get(context.Background())
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if !got.IsEnabled || got.HTMLContent != "<div>new</div>" || got.JSContent != "run();" || got.CSSContent != "body{}" {
		t.Errorf("Get() after Update() = %+v, want the updated content", got)
	}
}

func TestGetPublic_DisabledStateIsNotFilteredByBackend(t *testing.T) {
	repo := &fakeRepo{data: welcomepopup_model.WelcomePopup{
		ID: welcomepopup_model.SingletonID, IsEnabled: false, HTMLContent: strPtr("<div>hi</div>"),
	}}
	svc := NewService(repo)

	res, err := svc.GetPublic(context.Background())
	if err != nil {
		t.Fatalf("GetPublic() error = %v", err)
	}
	if res.IsEnabled {
		t.Errorf("GetPublic().IsEnabled = true, want false — the backend must return the true state; filtering happens on the frontend")
	}
	if res.HTMLContent == "" {
		t.Errorf("GetPublic().HTMLContent should still be populated even when disabled")
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd fsldk-api && go test ./modules/welcomepopup/welcomepopup_service/...`
Expected: FAIL — compile error, `NewService`/`Service` undefined (they don't exist yet).

- [ ] **Step 3: Write the service interface**

Create `fsldk-api/modules/welcomepopup/welcomepopup_service/welcomepopup_service.go`:

```go
// Package welcomepopup_service memuat logika bisnis modul welcomepopup.
package welcomepopup_service

import (
	"context"

	"fsldk-api/modules/welcomepopup/welcomepopup_dto"
)

// Service adalah kontrak logika bisnis Welcome Popup.
type Service interface {
	Get(ctx context.Context) (welcomepopup_dto.Response, error)
	// GetPublic mengembalikan HANYA 4 field konten (welcomepopup_dto.PublicResponse) —
	// dipakai endpoint tanpa-auth, tidak pernah membawa metadata audit.
	// Mengembalikan isEnabled=false apa adanya (tidak difilter di sini) —
	// pemanggil publik (frontend) yang memutuskan render atau tidak.
	GetPublic(ctx context.Context) (welcomepopup_dto.PublicResponse, error)
	Update(ctx context.Context, req welcomepopup_dto.UpdateRequest, actorID int64) (welcomepopup_dto.Response, error)
}
```

- [ ] **Step 4: Write the minimal implementation**

Create `fsldk-api/modules/welcomepopup/welcomepopup_service/welcomepopup_service_impl.go`:

```go
package welcomepopup_service

import (
	"context"

	"fsldk-api/base/apperror"
	"fsldk-api/modules/welcomepopup/welcomepopup_dto"
	"fsldk-api/modules/welcomepopup/welcomepopup_model"
	"fsldk-api/modules/welcomepopup/welcomepopup_repository"
)

// ServiceImpl adalah implementasi Service.
type ServiceImpl struct {
	repo welcomepopup_repository.Repository
}

// NewService membuat Service welcomepopup.
func NewService(repo welcomepopup_repository.Repository) Service {
	return &ServiceImpl{repo: repo}
}

func toResponse(w welcomepopup_model.WelcomePopup) welcomepopup_dto.Response {
	html, js, css := "", "", ""
	if w.HTMLContent != nil {
		html = *w.HTMLContent
	}
	if w.JSContent != nil {
		js = *w.JSContent
	}
	if w.CSSContent != nil {
		css = *w.CSSContent
	}
	updatedDate := ""
	if w.UpdatedDate != nil {
		updatedDate = w.UpdatedDate.Format("2006-01-02 15:04:05")
	}
	return welcomepopup_dto.Response{
		IsEnabled: w.IsEnabled, HTMLContent: html, JSContent: js, CSSContent: css, UpdatedDate: updatedDate,
	}
}

func toPublicResponse(w welcomepopup_model.WelcomePopup) welcomepopup_dto.PublicResponse {
	r := toResponse(w)
	return welcomepopup_dto.PublicResponse{
		IsEnabled: r.IsEnabled, HTMLContent: r.HTMLContent, JSContent: r.JSContent, CSSContent: r.CSSContent,
	}
}

func (s *ServiceImpl) Get(ctx context.Context) (welcomepopup_dto.Response, error) {
	w, err := s.repo.Get(ctx)
	if err != nil {
		return welcomepopup_dto.Response{}, apperror.Internal("")
	}
	return toResponse(w), nil
}

func (s *ServiceImpl) GetPublic(ctx context.Context) (welcomepopup_dto.PublicResponse, error) {
	w, err := s.repo.Get(ctx)
	if err != nil {
		return welcomepopup_dto.PublicResponse{}, apperror.Internal("")
	}
	return toPublicResponse(w), nil
}

func (s *ServiceImpl) Update(ctx context.Context, req welcomepopup_dto.UpdateRequest, actorID int64) (welcomepopup_dto.Response, error) {
	popup := welcomepopup_model.WelcomePopup{
		ID:          welcomepopup_model.SingletonID,
		IsEnabled:   req.IsEnabled,
		HTMLContent: &req.HTMLContent,
		JSContent:   &req.JSContent,
		CSSContent:  &req.CSSContent,
		UpdatedBy:   &actorID,
	}
	if err := s.repo.Update(ctx, popup); err != nil {
		return welcomepopup_dto.Response{}, apperror.Internal("")
	}
	return s.Get(ctx)
}
```

- [ ] **Step 5: Run tests to verify they pass**

Run: `cd fsldk-api && go test ./modules/welcomepopup/welcomepopup_service/... -v`
Expected: PASS — all 3 tests green.

- [ ] **Step 6: Commit**

```bash
cd fsldk-api
git add modules/welcomepopup/welcomepopup_service
git commit -m "feat(welcomepopup): add service layer with tests"
```

---

### Task 3: Backend handler, router, and wiring into the app

**Files:**
- Create: `fsldk-api/modules/welcomepopup/welcomepopup_handler/welcomepopup_handler.go`
- Create: `fsldk-api/modules/welcomepopup/welcomepopup_handler/welcomepopup_handler_impl.go`
- Create: `fsldk-api/modules/welcomepopup/router.go`
- Modify: `fsldk-api/router.go` (imports, DI construction, route registration)

**Interfaces:**
- Consumes: `welcomepopup_service.Service` (from Task 2); `fsldk-api/base/httphelper` (`httphelper.Success(c, message string, result interface{})`, `httphelper.Error(c, err error)`); `fsldk-api/base/apperror` (`apperror.BadRequest(msg string) *AppError`); `fsldk-api/base/appctx` (`appctx.UserID(c *gin.Context) int64`); `fsldk-api/base/validation` (`validation.Struct(s interface{}) error`); `middlewares.Middleware` (`Auth() gin.HandlerFunc`, `RequireVerified() gin.HandlerFunc`, `RequirePermission(codes ...string) gin.HandlerFunc`); `middlewares.RateLimit(requests, burst int) gin.HandlerFunc`; `constants.PermWelcomePopupView`/`PermWelcomePopupUpdate` (from Task 1).
- Produces: `welcomepopup_handler.Handler` interface — `Get(c *gin.Context)`, `Update(c *gin.Context)`, `GetPublic(c *gin.Context)`; `welcomepopup_handler.NewHandler(svc welcomepopup_service.Service) Handler`; `welcomepopup.RegisterCMSRoutes(rg *gin.RouterGroup, h welcomepopup_handler.Handler, mw *middlewares.Middleware)`; `welcomepopup.RegisterPublicRoutes(pub *gin.RouterGroup, h welcomepopup_handler.Handler)`.

No new unit test in this task — handlers here are thin pass-throughs (identical shape to `setting_handler`, which also has no test), verified by `go build`/`go vet` plus the manual end-to-end check in Task 6.

- [ ] **Step 1: Write the handler interface**

Create `fsldk-api/modules/welcomepopup/welcomepopup_handler/welcomepopup_handler.go`:

```go
// Package welcomepopup_handler adalah lapisan presentasi HTTP modul
// welcomepopup.
package welcomepopup_handler

import "github.com/gin-gonic/gin"

// Handler adalah kontrak handler HTTP modul welcomepopup.
type Handler interface {
	Get(c *gin.Context)
	Update(c *gin.Context)
	GetPublic(c *gin.Context)
}
```

- [ ] **Step 2: Write the handler implementation**

Create `fsldk-api/modules/welcomepopup/welcomepopup_handler/welcomepopup_handler_impl.go`:

```go
package welcomepopup_handler

import (
	"fsldk-api/base/appctx"
	"fsldk-api/base/apperror"
	"fsldk-api/base/httphelper"
	"fsldk-api/base/validation"
	"fsldk-api/modules/welcomepopup/welcomepopup_dto"
	"fsldk-api/modules/welcomepopup/welcomepopup_service"

	"github.com/gin-gonic/gin"
)

// HandlerImpl adalah implementasi Handler.
type HandlerImpl struct{ svc welcomepopup_service.Service }

// NewHandler membuat Handler welcomepopup.
func NewHandler(svc welcomepopup_service.Service) Handler { return &HandlerImpl{svc: svc} }

// Get melayani GET /welcome-popup (CMS, permission welcomepopup.view).
func (h *HandlerImpl) Get(c *gin.Context) {
	res, err := h.svc.Get(c.Request.Context())
	if err != nil {
		httphelper.Error(c, err)
		return
	}
	httphelper.Success(c, "", res)
}

// Update melayani PUT /welcome-popup (CMS, permission welcomepopup.update).
func (h *HandlerImpl) Update(c *gin.Context) {
	var req welcomepopup_dto.UpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httphelper.Error(c, apperror.BadRequest("Format permintaan tidak valid"))
		return
	}
	if err := validation.Struct(req); err != nil {
		httphelper.Error(c, err)
		return
	}
	res, err := h.svc.Update(c.Request.Context(), req, appctx.UserID(c))
	if err != nil {
		httphelper.Error(c, err)
		return
	}
	httphelper.Success(c, "Welcome Popup berhasil diperbarui", res)
}

// GetPublic melayani GET /public/welcome-popup — tanpa auth, tanpa
// permission check, dipanggil langsung dari halaman Beranda publik.
func (h *HandlerImpl) GetPublic(c *gin.Context) {
	res, err := h.svc.GetPublic(c.Request.Context())
	if err != nil {
		httphelper.Error(c, err)
		return
	}
	httphelper.Success(c, "", res)
}
```

- [ ] **Step 3: Write the module router**

Create `fsldk-api/modules/welcomepopup/router.go`:

```go
// Package welcomepopup merangkai routing modul welcomepopup (popup Beranda
// dinamis — CMS Super-Admin-only + satu endpoint publik tanpa auth).
package welcomepopup

import (
	"fsldk-api/constants"
	"fsldk-api/middlewares"
	"fsldk-api/modules/welcomepopup/welcomepopup_handler"

	"github.com/gin-gonic/gin"
)

// RegisterCMSRoutes mendaftarkan endpoint manajemen Welcome Popup.
func RegisterCMSRoutes(rg *gin.RouterGroup, h welcomepopup_handler.Handler, mw *middlewares.Middleware) {
	g := rg.Group("/welcome-popup")
	g.Use(mw.Auth(), mw.RequireVerified())
	{
		g.GET("", mw.RequirePermission(constants.PermWelcomePopupView), h.Get)
		g.PUT("", mw.RequirePermission(constants.PermWelcomePopupUpdate), h.Update)
	}
}

// RegisterPublicRoutes mendaftarkan GET /public/welcome-popup — dipanggil
// halaman Beranda publik, rate limit ringan sekadar jaga dari flooding
// trivial (pola sama dengan zakat.RegisterPublicRoutes).
func RegisterPublicRoutes(pub *gin.RouterGroup, h welcomepopup_handler.Handler) {
	pub.GET("/welcome-popup", middlewares.RateLimit(30, 10), h.GetPublic)
}
```

- [ ] **Step 4: Wire the module into the root router**

In `fsldk-api/router.go`, find this exact import block:

```go
	"fsldk-api/modules/setting"
	"fsldk-api/modules/setting/setting_handler"
	"fsldk-api/modules/setting/setting_repository"
	"fsldk-api/modules/setting/setting_service"
```

Replace it with:

```go
	"fsldk-api/modules/setting"
	"fsldk-api/modules/setting/setting_handler"
	"fsldk-api/modules/setting/setting_repository"
	"fsldk-api/modules/setting/setting_service"

	"fsldk-api/modules/welcomepopup"
	"fsldk-api/modules/welcomepopup/welcomepopup_handler"
	"fsldk-api/modules/welcomepopup/welcomepopup_repository"
	"fsldk-api/modules/welcomepopup/welcomepopup_service"
```

Find this exact line (repository construction):

```go
	settingRepo := setting_repository.NewRepository(db)
```

Replace it with:

```go
	settingRepo := setting_repository.NewRepository(db)
	welcomePopupRepo := welcomepopup_repository.NewRepository(db)
```

Find this exact line (service construction):

```go
	settingSvc := setting_service.NewService(settingRepo)
```

Replace it with:

```go
	settingSvc := setting_service.NewService(settingRepo)
	welcomePopupSvc := welcomepopup_service.NewService(welcomePopupRepo)
```

Find this exact line (handler construction):

```go
	settingH := setting_handler.NewHandler(settingSvc)
```

Replace it with:

```go
	settingH := setting_handler.NewHandler(settingSvc)
	welcomePopupH := welcomepopup_handler.NewHandler(welcomePopupSvc)
```

Find this exact line (route registration):

```go
	setting.RegisterCMSRoutes(api, settingH, mw)
```

Replace it with:

```go
	setting.RegisterCMSRoutes(api, settingH, mw)
	welcomepopup.RegisterCMSRoutes(api, welcomePopupH, mw)
	welcomepopup.RegisterPublicRoutes(pub, welcomePopupH)
```

- [ ] **Step 5: Verify it builds and vets clean**

Run: `cd fsldk-api && go build ./... && go vet ./...`
Expected: no output, exit code 0. If `go build` fails on an import cycle or typo, re-check the exact strings above were matched (not a similarly-named line elsewhere).

- [ ] **Step 6: Run the full test suite**

Run: `cd fsldk-api && go test ./...`
Expected: PASS (includes the Task 2 service tests; every other package has no tests, which `go test` reports as `ok  	fsldk-api/modules/x	[no test files]` — that is expected, not a failure).

- [ ] **Step 7: Commit**

```bash
cd fsldk-api
git add modules/welcomepopup/welcomepopup_handler modules/welcomepopup/router.go router.go
git commit -m "feat(welcomepopup): add handler, router, and wire into app"
```

---

### Task 4: Frontend data layer — entity, API service, repository

**Files:**
- Create: `fsldk-web/src/app/modules/welcomepopup/entities/welcome-popup.ts`
- Create: `fsldk-web/src/app/modules/welcomepopup/services/welcomepopup-api.service.ts`
- Create: `fsldk-web/src/app/modules/welcomepopup/repositories/welcomepopup.repository.ts`

**Interfaces:**
- Consumes: `fsldk-web/src/app/core/services/api.service.ts` — `ApiService.get<T>(path: string): Observable<T>`, `ApiService.put<T>(path: string, body?: unknown): Observable<T>` (already unwrap the `{result}` envelope).
- Produces: `WelcomePopup` interface (`isEnabled: boolean; htmlContent: string; jsContent: string; cssContent: string; updatedDate?: string;`); `WelcomepopupApiService` with `get()`, `update(payload)`, `getPublic()`; `WelcomepopupRepository` with the same three methods, all returning `Observable<WelcomePopup>`.

No test for this task — it's a direct pass-through with zero branching logic, matching `setting-api.service.ts`/`setting.repository.ts` in this codebase, neither of which has a spec file.

- [ ] **Step 1: Write the entity**

Create `fsldk-web/src/app/modules/welcomepopup/entities/welcome-popup.ts`:

```ts
export interface WelcomePopup {
  isEnabled: boolean;
  htmlContent: string;
  jsContent: string;
  cssContent: string;
  /** Absen pada respons publik (GET /public/welcome-popup) — dengan sengaja,
   *  lihat welcomepopup_dto.PublicResponse di backend. */
  updatedDate?: string;
}

export interface WelcomePopupUpdatePayload {
  isEnabled: boolean;
  htmlContent: string;
  jsContent: string;
  cssContent: string;
}
```

- [ ] **Step 2: Write the API service**

Create `fsldk-web/src/app/modules/welcomepopup/services/welcomepopup-api.service.ts`:

```ts
import { Injectable, inject } from '@angular/core';
import { Observable } from 'rxjs';
import { ApiService } from '../../../core/services/api.service';
import { WelcomePopup, WelcomePopupUpdatePayload } from '../entities/welcome-popup';

/** Panggilan HTTP mentah untuk Welcome Popup (/welcome-popup CMS,
 *  /public/welcome-popup publik). */
@Injectable({ providedIn: 'root' })
export class WelcomepopupApiService {
  private api = inject(ApiService);

  get(): Observable<WelcomePopup> { return this.api.get('/welcome-popup'); }
  update(payload: WelcomePopupUpdatePayload): Observable<WelcomePopup> { return this.api.put('/welcome-popup', payload); }
  /** Endpoint publik (tanpa auth) — dipakai halaman Beranda, BUKAN CMS. */
  getPublic(): Observable<WelcomePopup> { return this.api.get('/public/welcome-popup'); }
}
```

- [ ] **Step 3: Write the repository**

Create `fsldk-web/src/app/modules/welcomepopup/repositories/welcomepopup.repository.ts`:

```ts
import { Injectable, inject } from '@angular/core';
import { Observable } from 'rxjs';
import { WelcomepopupApiService } from '../services/welcomepopup-api.service';
import { WelcomePopup, WelcomePopupUpdatePayload } from '../entities/welcome-popup';

@Injectable({ providedIn: 'root' })
export class WelcomepopupRepository {
  private api = inject(WelcomepopupApiService);

  get(): Observable<WelcomePopup> { return this.api.get(); }
  update(payload: WelcomePopupUpdatePayload): Observable<WelcomePopup> { return this.api.update(payload); }
  getPublic(): Observable<WelcomePopup> { return this.api.getPublic(); }
}
```

- [ ] **Step 4: Verify it compiles**

Run: `cd fsldk-web && npx tsc -p tsconfig.app.json --noEmit`
Expected: no errors (these 3 files have no consumers yet, so this only catches syntax/type errors within them).

- [ ] **Step 5: Commit**

```bash
cd fsldk-web
git add src/app/modules/welcomepopup/entities src/app/modules/welcomepopup/services src/app/modules/welcomepopup/repositories
git commit -m "feat(welcomepopup): add frontend entity, API service, repository"
```

---

### Task 5: Frontend CMS form (TDD on the presenter) + routing

**Files:**
- Create: `fsldk-web/src/app/modules/welcomepopup/pages/form/welcomepopup.form.view.ts`
- Create: `fsldk-web/src/app/modules/welcomepopup/pages/form/welcomepopup.form.presenter.ts`
- Test: `fsldk-web/src/app/modules/welcomepopup/pages/form/welcomepopup.form.presenter.spec.ts`
- Create: `fsldk-web/src/app/modules/welcomepopup/pages/form/welcomepopup.form.page.ts`
- Create: `fsldk-web/src/app/modules/welcomepopup/pages/form/welcomepopup.form.page.html`
- Create: `fsldk-web/src/app/modules/welcomepopup/welcomepopup.path.ts`
- Create: `fsldk-web/src/app/modules/welcomepopup/welcomepopup.routes.ts`
- Modify: `fsldk-web/src/app/app.routes.ts` (register the route under the FSLDK `/cms` shell)

**Interfaces:**
- Consumes: `WelcomepopupRepository` (Task 4) — `get()`, `update(payload)` both `Observable<WelcomePopup>`; `fsldk-web/src/app/core/mvp/base.presenter.ts` — `BasePresenter<TView>` with `attachView(view: TView): void` and `protected view!: TView`; `fsldk-web/src/app/core/services/toast.service.ts` — `ToastService.success(msg: string)`/`.error(msg: string)`; `fsldk-web/src/app/core/guards/guards.ts` — `verifiedGuard`, `permissionGuard` (both `CanActivateFn`); `fsldk-web/src/app/shared/icon.component.ts` — `IconComponent`, selector `app-icon`, `[name]`/`[size]` inputs (icon `'check'` and `'megaphone'` both already registered).
- Produces: `WelcomepopupFormView` interface (`setForm(data: WelcomePopup): void; setLoading(loading: boolean): void; setSaving(saving: boolean): void;`); `WelcomepopupFormPresenter extends BasePresenter<WelcomepopupFormView>` with `load(): void` and `save(payload: WelcomePopupUpdatePayload): void`; `WelcomepopupFormPage` component (standalone, selector `app-welcomepopup-form-page`); `welcomepopupRoutes: () => Routes`; `welcomepopupPath.index === '/cms/welcome-popup'`.

- [ ] **Step 1: Write the view interface**

Create `fsldk-web/src/app/modules/welcomepopup/pages/form/welcomepopup.form.view.ts`:

```ts
import { WelcomePopup } from '../../entities/welcome-popup';

export interface WelcomepopupFormView {
  setForm(data: WelcomePopup): void;
  setLoading(loading: boolean): void;
  setSaving(saving: boolean): void;
}
```

- [ ] **Step 2: Write the failing presenter test**

Create `fsldk-web/src/app/modules/welcomepopup/pages/form/welcomepopup.form.presenter.spec.ts`:

```ts
import { TestBed } from '@angular/core/testing';
import { of, throwError } from 'rxjs';
import { WelcomepopupFormPresenter } from './welcomepopup.form.presenter';
import { WelcomepopupRepository } from '../../repositories/welcomepopup.repository';
import { ToastService } from '../../../../core/services/toast.service';
import { WelcomepopupFormView } from './welcomepopup.form.view';
import { WelcomePopup } from '../../entities/welcome-popup';

describe('WelcomepopupFormPresenter', () => {
  let presenter: WelcomepopupFormPresenter;
  let repo: jasmine.SpyObj<WelcomepopupRepository>;
  let toast: jasmine.SpyObj<ToastService>;
  let view: jasmine.SpyObj<WelcomepopupFormView>;

  const sample: WelcomePopup = { isEnabled: true, htmlContent: '<div>hi</div>', jsContent: 'run();', cssContent: 'body{}' };

  beforeEach(() => {
    repo = jasmine.createSpyObj('WelcomepopupRepository', ['get', 'update']);
    toast = jasmine.createSpyObj('ToastService', ['success', 'error']);

    TestBed.configureTestingModule({
      providers: [
        WelcomepopupFormPresenter,
        { provide: WelcomepopupRepository, useValue: repo },
        { provide: ToastService, useValue: toast },
      ],
    });
    presenter = TestBed.inject(WelcomepopupFormPresenter);
    view = jasmine.createSpyObj('WelcomepopupFormView', ['setForm', 'setLoading', 'setSaving']);
    presenter.attachView(view);
  });

  it('load() populates the view and clears loading on success', () => {
    repo.get.and.returnValue(of(sample));

    presenter.load();

    expect(view.setLoading).toHaveBeenCalledWith(true);
    expect(view.setForm).toHaveBeenCalledWith(sample);
    expect(view.setLoading).toHaveBeenCalledWith(false);
  });

  it('load() clears loading even when the request fails', () => {
    repo.get.and.returnValue(throwError(() => new Error('network')));

    presenter.load();

    expect(view.setForm).not.toHaveBeenCalled();
    expect(view.setLoading).toHaveBeenCalledWith(false);
  });

  it('save() shows a success toast and refreshes the form on success', () => {
    repo.update.and.returnValue(of(sample));

    presenter.save({ isEnabled: true, htmlContent: sample.htmlContent, jsContent: sample.jsContent, cssContent: sample.cssContent });

    expect(view.setSaving).toHaveBeenCalledWith(true);
    expect(toast.success).toHaveBeenCalled();
    expect(view.setForm).toHaveBeenCalledWith(sample);
    expect(view.setSaving).toHaveBeenCalledWith(false);
  });

  it('save() clears saving without a success toast when the request fails', () => {
    repo.update.and.returnValue(throwError(() => new Error('network')));

    presenter.save({ isEnabled: true, htmlContent: '', jsContent: '', cssContent: '' });

    expect(toast.success).not.toHaveBeenCalled();
    expect(view.setSaving).toHaveBeenCalledWith(false);
  });
});
```

- [ ] **Step 3: Run the test to verify it fails**

Run: `cd fsldk-web && npx ng test --watch=false --include='**/welcomepopup.form.presenter.spec.ts'`
Expected: FAIL — `welcomepopup.form.presenter` module not found (the presenter file doesn't exist yet).

If Karma cannot launch a browser in this environment (no Chrome installed), note that explicitly rather than assuming the test passed — this is still the correct test file and will run in any environment with `ng test` configured, but launcher availability is an environment concern separate from the code's correctness.

- [ ] **Step 4: Write the presenter**

Create `fsldk-web/src/app/modules/welcomepopup/pages/form/welcomepopup.form.presenter.ts`:

```ts
import { Injectable, inject } from '@angular/core';
import { BasePresenter } from '../../../../core/mvp/base.presenter';
import { ToastService } from '../../../../core/services/toast.service';
import { WelcomepopupRepository } from '../../repositories/welcomepopup.repository';
import { WelcomePopupUpdatePayload } from '../../entities/welcome-popup';
import { WelcomepopupFormView } from './welcomepopup.form.view';

@Injectable()
export class WelcomepopupFormPresenter extends BasePresenter<WelcomepopupFormView> {
  private repo = inject(WelcomepopupRepository);
  private toast = inject(ToastService);

  load(): void {
    this.view.setLoading(true);
    this.repo.get().subscribe({
      next: (data) => { this.view.setForm(data); this.view.setLoading(false); },
      error: () => this.view.setLoading(false),
    });
  }

  save(payload: WelcomePopupUpdatePayload): void {
    this.view.setSaving(true);
    this.repo.update(payload).subscribe({
      next: (data) => {
        this.toast.success('Welcome Popup berhasil disimpan.');
        this.view.setForm(data);
        this.view.setSaving(false);
      },
      error: () => this.view.setSaving(false),
    });
  }
}
```

- [ ] **Step 5: Run the test to verify it passes**

Run: `cd fsldk-web && npx ng test --watch=false --include='**/welcomepopup.form.presenter.spec.ts'`
Expected: PASS — all 4 specs green (same environment caveat as Step 3 applies if no browser launcher is available).

- [ ] **Step 6: Write the page component**

Create `fsldk-web/src/app/modules/welcomepopup/pages/form/welcomepopup.form.page.ts`:

```ts
import { Component, OnInit, inject, signal } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { IconComponent } from '../../../../shared/icon.component';
import { WelcomePopup } from '../../entities/welcome-popup';
import { WelcomepopupFormPresenter } from './welcomepopup.form.presenter';
import { WelcomepopupFormView } from './welcomepopup.form.view';

@Component({
  selector: 'app-welcomepopup-form-page',
  standalone: true,
  templateUrl: './welcomepopup.form.page.html',
  imports: [FormsModule, IconComponent],
  providers: [WelcomepopupFormPresenter],
  styles: [`
    .page-head { margin-bottom: 24px; } .page-head h1 { margin-bottom: 2px; }
    .form-card { display: flex; flex-direction: column; gap: 20px; }
    .popup-code-field { font-family: 'Courier New', monospace; font-size: .85rem; resize: vertical; }
    .form-actions { display: flex; justify-content: flex-end; gap: 10px; padding-top: 22px; margin-top: 4px; border-top: 1px solid var(--color-border); }
  `],
})
export class WelcomepopupFormPage implements OnInit, WelcomepopupFormView {
  private presenter = inject(WelcomepopupFormPresenter);

  loading = signal(true);
  saving = signal(false);

  isEnabled = false;
  htmlContent = '';
  jsContent = '';
  cssContent = '';

  ngOnInit(): void {
    this.presenter.attachView(this);
    this.presenter.load();
  }

  save(): void {
    this.presenter.save({
      isEnabled: this.isEnabled, htmlContent: this.htmlContent, jsContent: this.jsContent, cssContent: this.cssContent,
    });
  }

  setForm(data: WelcomePopup): void {
    this.isEnabled = data.isEnabled;
    this.htmlContent = data.htmlContent;
    this.jsContent = data.jsContent;
    this.cssContent = data.cssContent;
  }
  setLoading(loading: boolean): void { this.loading.set(loading); }
  setSaving(saving: boolean): void { this.saving.set(saving); }
}
```

- [ ] **Step 7: Write the page template**

Create `fsldk-web/src/app/modules/welcomepopup/pages/form/welcomepopup.form.page.html`:

```html
<div class="page-head">
  <h1>Welcome Popup</h1>
  <p class="text-muted">Kelola popup selamat datang di halaman Beranda. Perubahan berlaku langsung di situs publik setelah disimpan.</p>
</div>

@if (loading()) {
  <div class="card card-pad"><span class="skel skel-line" style="width:60%;height:20px"></span></div>
} @else {
  <div class="card card-pad form-card">
    <label class="switch">
      <input type="checkbox" [(ngModel)]="isEnabled" name="isEnabled">
      <span class="switch-track"></span>
      <span class="switch-label">Aktif — tampilkan di halaman Beranda</span>
    </label>

    <div class="form-group">
      <label class="form-label">Field Index (HTML)</label>
      <textarea class="form-control popup-code-field" rows="14" [(ngModel)]="htmlContent" name="htmlContent" spellcheck="false" placeholder="<div id=&quot;wrp-backdrop&quot;>...</div>"></textarea>
      <p class="form-hint">Markup popup. Tag &lt;script&gt; di dalam field ini TIDAK akan dieksekusi browser — taruh javascript-nya di Field Javascript di bawah.</p>
    </div>

    <div class="form-group">
      <label class="form-label">Field Javascript</label>
      <textarea class="form-control popup-code-field" rows="14" [(ngModel)]="jsContent" name="jsContent" spellcheck="false" placeholder="(function () { ... }());"></textarea>
    </div>

    <div class="form-group mb-0">
      <label class="form-label">Field CSS</label>
      <textarea class="form-control popup-code-field" rows="14" [(ngModel)]="cssContent" name="cssContent" spellcheck="false" placeholder="#wrp-backdrop { ... }"></textarea>
    </div>

    <div class="form-actions">
      <button class="btn btn-primary" [disabled]="saving()" (click)="save()">
        @if (saving()) { <span class="spinner"></span> } @else { <app-icon name="check" [size]="14" /> Simpan }
      </button>
    </div>
  </div>
}
```

- [ ] **Step 8: Write the path constants**

Create `fsldk-web/src/app/modules/welcomepopup/welcomepopup.path.ts`:

```ts
export const welcomepopupPath = {
  index: '/cms/welcome-popup',
};
```

- [ ] **Step 9: Write the routes**

Create `fsldk-web/src/app/modules/welcomepopup/welcomepopup.routes.ts`:

```ts
import { Routes } from '@angular/router';
import { verifiedGuard, permissionGuard } from '../../core/guards/guards';

/** Rute Welcome Popup CMS — dipasang sebagai children dari CmsLayoutComponent
 *  (shell FSLDK/Portal Admin saja, lihat app.routes.ts). */
export const welcomepopupRoutes: () => Routes = () => [
  {
    path: 'welcome-popup',
    canActivate: [verifiedGuard, permissionGuard],
    data: { permission: 'welcomepopup.view' },
    title: 'Welcome Popup',
    loadComponent: () => import('./pages/form/welcomepopup.form.page').then((m) => m.WelcomepopupFormPage),
  },
];
```

- [ ] **Step 10: Register the route in app.routes.ts**

In `fsldk-web/src/app/app.routes.ts`, find this exact import line:

```ts
import { settingRoutes } from './modules/setting/setting.routes';
```

Replace it with:

```ts
import { settingRoutes } from './modules/setting/setting.routes';
import { welcomepopupRoutes } from './modules/welcomepopup/welcomepopup.routes';
```

Find this exact line (inside the `/cms` shell's `children` array):

```ts
      ...settingRoutes(),
```

Replace it with:

```ts
      ...settingRoutes(),
      ...welcomepopupRoutes(),
```

- [ ] **Step 11: Verify the frontend builds**

Run: `cd fsldk-web && npx ng build`
Expected: `Application bundle generation complete`, no new errors (pre-existing bundle-budget warnings on unrelated files are fine).

- [ ] **Step 12: Commit**

```bash
cd fsldk-web
git add src/app/modules/welcomepopup/pages src/app/modules/welcomepopup/welcomepopup.path.ts src/app/modules/welcomepopup/welcomepopup.routes.ts src/app/app.routes.ts
git commit -m "feat(welcomepopup): add CMS form page and wire /cms/welcome-popup route"
```

---

### Task 6: Public homepage injector + final verification

**Files:**
- Create: `fsldk-web/src/app/modules/home/components/welcome-popup.component.ts`
- Modify: `fsldk-web/src/app/modules/home/pages/index/home.index.page.ts` (import + add to `imports` array)
- Modify: `fsldk-web/src/app/modules/home/pages/index/home.index.page.html` (mount the component)

**Interfaces:**
- Consumes: `WelcomepopupRepository.getPublic(): Observable<WelcomePopup>` (Task 4).
- Produces: `WelcomePopupComponent` standalone component, selector `app-welcome-popup`, empty template (renders nothing itself — all visible content is injected directly into `document.body`/`document.head` on init).

This component is pure DOM-manipulation glue with no branching worth a unit test in isolation (its one decision — `isEnabled && htmlContent.trim()` — is exercised end-to-end in the manual QA step below, which is also the only way to confirm a `<script>` tag genuinely executes, since that requires a real browser). This matches the design spec's own testing section.

- [ ] **Step 1: Write the injector component**

Create `fsldk-web/src/app/modules/home/components/welcome-popup.component.ts`:

```ts
import { Component, OnInit, inject } from '@angular/core';
import { WelcomepopupRepository } from '../../welcomepopup/repositories/welcomepopup.repository';

/**
 * Injector generik Welcome Popup untuk halaman Beranda — komponen ini TIDAK
 * tahu apa pun soal "dismiss"/"backdrop"/dst.; semua itu ada di dalam
 * htmlContent/jsContent/cssContent yang diisi Super Admin lewat CMS
 * (/cms/welcome-popup, lihat WelcomepopupFormPage). Tugas komponen ini
 * murni mengambil kontennya lalu menyuntikkannya ke DOM.
 *
 * Template SENGAJA kosong — semua konten disuntik lewat DOM API langsung
 * (bukan binding [innerHTML] Angular), karena:
 * 1. <script> yang ikut ter-parse di dalam innerHTML TIDAK PERNAH dieksekusi
 *    browser (batasan platform, bukan sesuatu yang bisa "diperbaiki" lewat
 *    DomSanitizer) — satu-satunya cara javascript-nya beneran jalan adalah
 *    document.createElement('script') lalu appendChild, seperti di bawah.
 * 2. Konten ini memang sepenuhnya dipercaya (trust boundary-nya adalah
 *    permission welcomepopup.update di CMS, Super Admin only) — mem-bypass
 *    sanitizer Angular pun tidak menambah risiko baru di sini.
 */
@Component({
  selector: 'app-welcome-popup',
  standalone: true,
  template: '',
})
export class WelcomePopupComponent implements OnInit {
  private repo = inject(WelcomepopupRepository);

  ngOnInit(): void {
    this.repo.getPublic().subscribe({
      next: (popup) => {
        if (!popup.isEnabled || !popup.htmlContent?.trim()) return;
        this.inject(popup.htmlContent, popup.cssContent, popup.jsContent);
      },
      error: () => {},
    });
  }

  private inject(html: string, css: string, js: string): void {
    const wrap = document.createElement('div');
    wrap.innerHTML = html;
    document.body.appendChild(wrap);

    if (css?.trim()) {
      const style = document.createElement('style');
      style.textContent = css;
      document.head.appendChild(style);
    }
    if (js?.trim()) {
      const script = document.createElement('script');
      script.textContent = js;
      document.body.appendChild(script);
    }
  }
}
```

- [ ] **Step 2: Mount it on the homepage**

In `fsldk-web/src/app/modules/home/pages/index/home.index.page.ts`, find this exact import line:

```ts
import { IconComponent } from '../../../../shared/icon.component';
```

Replace it with:

```ts
import { IconComponent } from '../../../../shared/icon.component';
import { WelcomePopupComponent } from '../../components/welcome-popup.component';
```

Find this exact line:

```ts
  imports: [RouterLink, DatePipe, IconComponent],
```

Replace it with:

```ts
  imports: [RouterLink, DatePipe, IconComponent, WelcomePopupComponent],
```

In `fsldk-web/src/app/modules/home/pages/index/home.index.page.html`, find this exact first line:

```html
<section class="hero">
```

Replace it with:

```html
<app-welcome-popup />
<section class="hero">
```

- [ ] **Step 3: Verify the frontend builds**

Run: `cd fsldk-web && npx ng build`
Expected: `Application bundle generation complete`, no new errors.

- [ ] **Step 4: Verify the backend builds and its tests still pass**

Run: `cd fsldk-api && go build ./... && go vet ./... && go test ./...`
Expected: exit code 0, all tests PASS.

- [ ] **Step 5: Manual end-to-end verification**

This is the one part of this feature that genuinely needs a real browser (confirming a `<script>` tag executes cannot be verified by `ng build` or a unit test):

1. Start the backend: `cd fsldk-api && go run .` (this runs the new migration automatically on boot — confirm the log shows migration `0040_welcomepopup` applied, or check `schema_migrations` in MySQL).
2. Start the frontend: `cd fsldk-web && npm start`.
3. Log in as the seeded Super Admin account (see `fsldk-api/docs/INSTALLATION.md` for default credentials) and confirm **Welcome Popup** now appears in the Portal Admin sidebar (it must NOT appear for a non-Super-Admin account — check with a second, non-superadmin login if one is available).
4. Open `/cms/welcome-popup`, fill in the three fields with a minimal smoke-test payload, e.g.:
   - Field Index: `<div id="wp-test" style="position:fixed;top:20px;right:20px;background:#0a0;color:#fff;padding:12px;z-index:9999">Popup OK</div>`
   - Field Javascript: `console.log('welcome popup JS executed');`
   - Field CSS: `#wp-test { border-radius: 8px; }`
   - Toggle **Aktif** on, click **Simpan**, confirm the success toast appears.
5. Open the public homepage (`/`) in a new tab (or incognito window, to rule out any CMS session state bleeding through). Confirm:
   - The green "Popup OK" box appears in the corner (proves HTML + CSS injection worked).
   - The browser DevTools console shows `welcome popup JS executed` (proves the `<script>` node actually ran — this is the step that would fail silently if the injector used `[innerHTML]` instead of `document.createElement('script')`).
6. Go back to `/cms/welcome-popup`, toggle **Aktif** off, save. Reload the public homepage and confirm the box no longer appears.
7. Clear the three fields entirely, save, confirm no console errors on the public homepage (covers the "empty content" path).

- [ ] **Step 6: Commit**

```bash
cd fsldk-web
git add src/app/modules/home/components/welcome-popup.component.ts src/app/modules/home/pages/index/home.index.page.ts src/app/modules/home/pages/index/home.index.page.html
git commit -m "feat(welcomepopup): inject CMS-managed popup on the public homepage"
```
