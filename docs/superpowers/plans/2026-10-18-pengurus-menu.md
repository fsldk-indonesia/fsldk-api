# Pengurus Menu Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Let LDK Admin / Puskomda Verifikator / Puskomnas Verifikator accounts promote an active Kader in their own organization scope directly to a higher role (and demote back), without going through Portal Admin's Manajemen Pengguna — including an automatic demote-to-Kader cascade when a promoted Kader is deactivated.

**Architecture:** Entirely new methods on the EXISTING `submission_service.Service` (no new backend module, no new cross-module interfaces — `submission_service.ServiceImpl` already has direct `userRepo`/`roleRepo`/`orgScope` dependencies, the same ones `promoteToKaderRole` already uses). One shared private mutation core (`setPengurusRole`) is reused by both the new promote/demote endpoint and a one-line addition to the existing `DeactivateKader`. Frontend gets a new Angular module `modules/pengurus/` with one page mounted into 3 CMS tier shells (`cms-ldk`/`cms-puskomda`/`cms-puskomnas`, never `cms`).

**Tech Stack:** Go 1.24 + Gin + GORM + MySQL (`fsldk-api`); Angular 19 standalone components + signals (`fsldk-web`).

**Spec:** `fsldk-api/docs/superpowers/specs/2026-10-18-pengurus-feature-design.md`

## Global Constraints

- **No new git branch** — continue committing directly on `feat/design-polish` in both repos (explicit user instruction, overrides the usual "new branch per feature" default).
- **No new cross-module Go interfaces.** Everything lives in `modules/submission`, reusing its existing `repo`/`orgRepo`/`userRepo`/`roleRepo`/`orgScope` dependencies. `submission_service.NewService`'s signature does not change.
- **`organizationID` always follows the *target* role**, never the role being left: Kader/LDK Admin → the user's home LDK (`ms_kader.organizationID`, immutable); Puskomda Verifikator → that LDK's parent Puskomda; Puskomnas Verifikator → the single Puskomnas root org.
- **Self-action is always blocked** — a caller can never change their own role through this feature (`targetUserID == caller.UserID` → `403`).
- **Role allow-list is derived from `caller.OrganizationTypeCode` server-side**, never trusted from the client: LDK → {Kader, LDK Admin}; Puskomda → {Kader, LDK Admin, Puskomda Verifikator}; Puskomnas → all four.
- **The `/pengurus` route group is a sibling of `/kaders`/`/submissions`**, not nested under `/kaders` — Gin's router cannot mix a static segment and a wildcard param (`:id`) at the same path depth, and `/kaders/:id/...` already occupies that depth.
- **Deactivating a promoted Kader auto-reverts their role** via a one-line addition to the existing `DeactivateKader` — no new "Nonaktifkan" button is added to the Pengurus page itself.
- Follow this codebase's module shape exactly: `_dto` is pure data (zero methods); `_service`/`_handler` are pure logic (zero data fields except the `ServiceImpl`/`HandlerImpl` DI receiver).
- No new third-party dependencies (no new Go modules, no new npm packages).

---

### Task 1: Backend data foundation — migration, permission constants, DTOs

**Files:**
- Create: `fsldk-api/migrations/0051_pengurus_menu.up.sql`
- Modify: `fsldk-api/constants/constants.go`
- Modify: `fsldk-api/modules/submission/submission_dto/submission_dto.go`

**Interfaces:**
- Produces: `constants.PermPengurusLDKManage`, `constants.PermPengurusPuskomdaManage`, `constants.PermPengurusPuskomnasManage` (all `string`); `submission_dto.PengurusEntry` struct (`UserID int64`, `FullName string`, `Email string`, `RoleName string`, `HomeOrganizationID int64`, `HomeOrganizationName string`); `submission_dto.UpdatePengurusRoleRequest` struct (`RoleName string`).

No independent test — pure data + SQL, mirrors `submission_dto.KaderResponse`'s own shape (no test file). Verified by compilation in Task 2 and by the migration actually running in Task 3.

- [ ] **Step 1: Write the migration**

Create `fsldk-api/migrations/0051_pengurus_menu.up.sql`:

```sql
-- ============================================================
-- FSLDK API — Pengurus menu (promosi/demosi role kader aktif dalam
-- scope organisasi caller sendiri, tanpa lewat Portal Admin).
-- 3 permissionCode terpisah (satu per tier) karena permissionCode unik
-- dan tiap baris menu butuh menuRoute sendiri — pola sama dengan
-- migrations/0010_cms_shell_menu_routes.up.sql. Endpoint backend-nya
-- SATU saja (modules/submission), dijaga ketiga kode ini sekaligus
-- (any-match) — lihat RequirePermission di modules/submission/router.go.
-- Idempoten: aman dijalankan ulang (INSERT IGNORE).
-- ============================================================

INSERT IGNORE INTO lk_permission (permissionCode, permissionName, moduleName, menuLabel, menuIcon, menuRoute, sortOrder) VALUES
('pengurus.ldk.manage',       'Kelola Pengurus LDK',       'submission', 'Pengurus', 'id-card', '/cms-ldk/pengurus', 15),
('pengurus.puskomda.manage',  'Kelola Pengurus Puskomda',  'submission', 'Pengurus', 'id-card', '/cms-puskomda/pengurus', 15),
('pengurus.puskomnas.manage', 'Kelola Pengurus Puskomnas', 'submission', 'Pengurus', 'id-card', '/cms-puskomnas/pengurus', 15);

INSERT IGNORE INTO map_role_permission (roleID, permissionID)
SELECT r.roleID, p.permissionID FROM ms_role r JOIN lk_permission p ON TRUE
WHERE r.roleName = 'LDK Admin' AND p.permissionCode = 'pengurus.ldk.manage';

INSERT IGNORE INTO map_role_permission (roleID, permissionID)
SELECT r.roleID, p.permissionID FROM ms_role r JOIN lk_permission p ON TRUE
WHERE r.roleName = 'Puskomda Verifikator' AND p.permissionCode = 'pengurus.puskomda.manage';

INSERT IGNORE INTO map_role_permission (roleID, permissionID)
SELECT r.roleID, p.permissionID FROM ms_role r JOIN lk_permission p ON TRUE
WHERE r.roleName = 'Puskomnas Verifikator' AND p.permissionCode = 'pengurus.puskomnas.manage';
```

- [ ] **Step 2: Add permission constants**

In `fsldk-api/constants/constants.go`, find this exact block:

```go
	PermSubmissionLevelEstablish = "submission.level.establish"
	PermSubmissionPublish        = "submission.publish"
	PermSubmissionReopen         = "submission.reopen"
	PermSubmissionReassess       = "submission.reassess"
	PermKaderDeactivate          = "kader.deactivate"
```

Replace it with:

```go
	PermSubmissionLevelEstablish = "submission.level.establish"
	PermSubmissionPublish        = "submission.publish"
	PermSubmissionReopen         = "submission.reopen"
	PermSubmissionReassess       = "submission.reassess"
	PermKaderDeactivate          = "kader.deactivate"

	PermPengurusLDKManage       = "pengurus.ldk.manage"
	PermPengurusPuskomdaManage  = "pengurus.puskomda.manage"
	PermPengurusPuskomnasManage = "pengurus.puskomnas.manage"
```

- [ ] **Step 3: Add the DTOs**

In `fsldk-api/modules/submission/submission_dto/submission_dto.go`, find this exact block (end of `KaderResponse`):

```go
type KaderResponse struct {
	KaderID                int64      `json:"kaderID"`
	SubmissionID           int64      `json:"submissionID"`
	OrganizationID         int64      `json:"organizationID"`
	OrganizationName       string     `json:"organizationName,omitempty"`
	ParentOrganizationName string     `json:"parentOrganizationName,omitempty"`
	UniqueCode             string     `json:"uniqueCode,omitempty"`
	FullName               string     `json:"fullName"`
	Status                 string     `json:"status"`
	IssuedDate             *time.Time `json:"issuedDate,omitempty"`
	StatusChangedDate      *time.Time `json:"statusChangedDate,omitempty"`
}
```

Replace it with:

```go
type KaderResponse struct {
	KaderID                int64      `json:"kaderID"`
	SubmissionID           int64      `json:"submissionID"`
	OrganizationID         int64      `json:"organizationID"`
	OrganizationName       string     `json:"organizationName,omitempty"`
	ParentOrganizationName string     `json:"parentOrganizationName,omitempty"`
	UniqueCode             string     `json:"uniqueCode,omitempty"`
	FullName               string     `json:"fullName"`
	Status                 string     `json:"status"`
	IssuedDate             *time.Time `json:"issuedDate,omitempty"`
	StatusChangedDate      *time.Time `json:"statusChangedDate,omitempty"`
}

// PengurusEntry adalah satu baris kader ACTIVE (apa pun role ms_user saat
// ini) dalam scope organisasi caller — dipakai fitur Pengurus (promosi/
// demosi role dalam organisasi sendiri, tanpa lewat Portal Admin).
// HomeOrganizationID/HomeOrganizationName SELALU LDK asal (ms_kader.
// organizationID, tidak pernah berubah) — BUKAN organizationID ms_user
// saat ini, yang bisa berbeda setelah promosi ke Puskomda/Puskomnas
// Verifikator (lihat setPengurusRole di submission_service_impl.go).
type PengurusEntry struct {
	UserID               int64  `json:"userID"`
	FullName             string `json:"fullName"`
	Email                string `json:"email"`
	RoleName             string `json:"roleName"`
	HomeOrganizationID   int64  `json:"homeOrganizationID"`
	HomeOrganizationName string `json:"homeOrganizationName"`
}

// UpdatePengurusRoleRequest adalah body mengubah role pengurus. Tidak ada
// validate:"oneof=..." dengan sengaja — daftar role yang valid bergantung
// pada tier caller (LDK/Puskomda/Puskomnas), bukan daftar statis, jadi
// validasi sesungguhnya ada di service (pengurusRolesByTier), bukan di sini.
type UpdatePengurusRoleRequest struct {
	RoleName string `json:"roleName" validate:"required"`
}
```

- [ ] **Step 4: Verify it compiles**

Run: `cd fsldk-api && go build ./... && go vet ./...`
Expected: no output, exit code 0.

- [ ] **Step 5: Commit**

```bash
cd fsldk-api
git add migrations/0051_pengurus_menu.up.sql constants/constants.go modules/submission/submission_dto/submission_dto.go
git commit -m "feat(submission): add Pengurus menu migration, permissions, and DTOs"
```

---

### Task 2: Backend service logic — ListPengurus, UpdatePengurusRole, setPengurusRole, DeactivateKader cascade (TDD)

**Files:**
- Modify: `fsldk-api/modules/submission/submission_service/submission_service.go`
- Modify: `fsldk-api/modules/submission/submission_service/submission_service_impl.go`
- Create: `fsldk-api/modules/submission/submission_service/pengurus_test.go`

**Interfaces:**
- Consumes (all already exist, already injected into `ServiceImpl`): `s.orgScope.AccessibleOrganizationIDs(ctx, callerOrganizationID *int64, callerOrganizationTypeCode, wildcardTierAccess string) ([]int64, error)`; `s.repo.ListKadersByOrganizations(ctx, organizationIDs []int64, status string) ([]submission_model.Kader, error)` (`submission_model.Kader` has `UserID int64`, `OrganizationID int64`, `KaderID int64`); `s.userRepo.FindByID(ctx, id int64) (user_model.User, error)` (`user_model.User` has `UserID`, `FullName`, `Email`, `RoleID int64`, `RoleName string`, `IsActive bool`, `OrganizationID sql.NullInt64`, `WildcardTierAccess sql.NullString`); `s.userRepo.Update(ctx, id int64, fullName, email string, roleID int64, isActive bool, organizationID sql.NullInt64, wildcardTierAccess sql.NullString, updatedBy int64) error`; `s.userRepo.BumpTokenVersion(ctx, id int64) error`; `s.roleRepo.IDByName(ctx, name string) (int64, error)`; `s.orgRepo.FindByID(ctx, id int64) (organization_model.Organization, error)` (`organization_model.Organization` has `OrganizationName string`, `ParentOrganizationID sql.NullInt64`); `s.orgRepo.RootPuskomnasID(ctx) (int64, error)`; `apperror.Forbidden/NotFound/Internal/BadRequest(msg string) *AppError`; `constants.KaderStatusActive`, `constants.KaderStatusInactive`, `constants.OrgTypeLDK/OrgTypePuskomda/OrgTypePuskomnas`, `constants.RoleLDKAdmin/RolePuskomdaVerifikator/RolePuskomnasVerifikator`; the file's own existing `roleNameKader = "Kader"` constant (`submission_service_impl.go:33`).
- Produces: `submission_service.Service` gains `ListPengurus(ctx context.Context, caller CallerScope) ([]submission_dto.PengurusEntry, error)` and `UpdatePengurusRole(ctx context.Context, targetUserID int64, caller CallerScope, req submission_dto.UpdatePengurusRoleRequest) (submission_dto.PengurusEntry, error)`; package-level `pengurusRolesByTier map[string][]string` and `pengurusRoleAllowed(callerOrgType, roleName string) bool` (both unexported, used by Task 3's test too via same-package access).

This is the first test file for `submission_service.ServiceImpl` as a whole (the only existing test, `scoring_test.go`, tests pure scoring functions with no `ServiceImpl` instance) — so this task writes a full fake harness for the 6 `ServiceImpl` dependencies. Most fake methods are no-ops (never called by these scenarios); only the ones this task's tests actually exercise behave meaningfully, mirroring `user_service_impl_test.go`'s `fakeUserRepository` precedent exactly.

The full test file below (including every fake method) was written against, and verified to compile and pass against, the actual codebase as of 2026-10-18 — including `submission_repository.Repository`'s `CountByFormAndStatuses`/`ListByFormAndStatuses`/`WipeSubmission` (added by unrelated concurrent work partway through writing this plan). **This repo has other work landing on the same branch concurrently** — if `go build` reports a fake missing a method when you reach Step 2, that interface gained a method after this plan was written; add a stub for it following the exact same pattern as its neighbors (return zero values, matching the real signature from the interface file) and continue — it is not a sign anything else in this plan is wrong.

- [ ] **Step 1: Write the failing tests**

Create `fsldk-api/modules/submission/submission_service/pengurus_test.go`:

```go
package submission_service

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"fsldk-api/constants"
	"fsldk-api/modules/organization/organization_dto"
	"fsldk-api/modules/organization/organization_model"
	"fsldk-api/modules/role/role_model"
	"fsldk-api/modules/submission/submission_dto"
	"fsldk-api/modules/submission/submission_model"
	"fsldk-api/modules/submission_form/submission_form_model"
	"fsldk-api/modules/user/user_dto"
	"fsldk-api/modules/user/user_model"
)

// ---------- pure allow-list table test (no fakes needed) ----------

func TestPengurusRoleAllowed(t *testing.T) {
	tests := []struct {
		name     string
		orgType  string
		roleName string
		want     bool
	}{
		{"LDK caller can assign Kader", constants.OrgTypeLDK, roleNameKader, true},
		{"LDK caller can assign LDK Admin", constants.OrgTypeLDK, constants.RoleLDKAdmin, true},
		{"LDK caller cannot assign Puskomda Verifikator", constants.OrgTypeLDK, constants.RolePuskomdaVerifikator, false},
		{"LDK caller cannot assign Puskomnas Verifikator", constants.OrgTypeLDK, constants.RolePuskomnasVerifikator, false},
		{"Puskomda caller can assign Kader", constants.OrgTypePuskomda, roleNameKader, true},
		{"Puskomda caller can assign LDK Admin", constants.OrgTypePuskomda, constants.RoleLDKAdmin, true},
		{"Puskomda caller can assign Puskomda Verifikator", constants.OrgTypePuskomda, constants.RolePuskomdaVerifikator, true},
		{"Puskomda caller cannot assign Puskomnas Verifikator", constants.OrgTypePuskomda, constants.RolePuskomnasVerifikator, false},
		{"Puskomnas caller can assign Kader", constants.OrgTypePuskomnas, roleNameKader, true},
		{"Puskomnas caller can assign LDK Admin", constants.OrgTypePuskomnas, constants.RoleLDKAdmin, true},
		{"Puskomnas caller can assign Puskomda Verifikator", constants.OrgTypePuskomnas, constants.RolePuskomdaVerifikator, true},
		{"Puskomnas caller can assign Puskomnas Verifikator", constants.OrgTypePuskomnas, constants.RolePuskomnasVerifikator, true},
		{"unknown org type allows nothing", "", constants.RoleLDKAdmin, false},
		{"unrecognized role name is never allowed", constants.OrgTypePuskomnas, "Super Admin", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := pengurusRoleAllowed(tt.orgType, tt.roleName); got != tt.want {
				t.Errorf("pengurusRoleAllowed(%q, %q) = %v, want %v", tt.orgType, tt.roleName, got, tt.want)
			}
		})
	}
}

// ---------- fakes for ServiceImpl's 6 dependencies ----------
// Only methods exercised by this file's scenarios behave meaningfully;
// every other method is a no-op stub required to satisfy the interface
// (same approach as user_service_impl_test.go's fakeUserRepository).

type fakeSubmissionRepo struct {
	kaders         map[int64]submission_model.Kader // by kaderID
	kaderStatusSet map[int64]string                 // kaderID -> last status written by UpdateKaderStatus
}

func (f *fakeSubmissionRepo) Create(ctx context.Context, p submission_model.CreateParams) (int64, error) {
	return 0, nil
}
func (f *fakeSubmissionRepo) FindByID(ctx context.Context, id int64) (submission_model.Submission, error) {
	return submission_model.Submission{}, errors.New("not found")
}
func (f *fakeSubmissionRepo) FindByOwnerOrganization(ctx context.Context, organizationID, formID int64) (submission_model.Submission, error) {
	return submission_model.Submission{}, errors.New("not found")
}
func (f *fakeSubmissionRepo) FindByUserAndForm(ctx context.Context, userID, formID int64) (submission_model.Submission, error) {
	return submission_model.Submission{}, errors.New("not found")
}
func (f *fakeSubmissionRepo) List(ctx context.Context, filt submission_dto.ListFilter) ([]submission_model.Submission, int64, error) {
	return nil, 0, nil
}
func (f *fakeSubmissionRepo) UpdateStatus(ctx context.Context, id int64, toStatus string, submittedDate sql.NullTime, updatedBy int64) error {
	return nil
}
func (f *fakeSubmissionRepo) TouchVersion(ctx context.Context, id int64, updatedBy int64) error {
	return nil
}
func (f *fakeSubmissionRepo) SetSubjectReference(ctx context.Context, id int64, subjectReferenceID int64) error {
	return nil
}
func (f *fakeSubmissionRepo) UpdateFormVersion(ctx context.Context, id int64, formVersionID int64) error {
	return nil
}
func (f *fakeSubmissionRepo) UpsertAnswer(ctx context.Context, submissionID int64, p submission_model.AnswerParams) error {
	return nil
}
func (f *fakeSubmissionRepo) ListAnswersBySubmission(ctx context.Context, submissionID int64) ([]submission_model.Answer, error) {
	return nil, nil
}
func (f *fakeSubmissionRepo) RemapAnswerField(ctx context.Context, submissionID, oldFieldID, newFieldID int64) error {
	return nil
}
func (f *fakeSubmissionRepo) InsertStatusHistory(ctx context.Context, submissionID int64, fromStatus sql.NullString, toStatus string, actorUserID int64, actorOrganizationID sql.NullInt64, note sql.NullString) error {
	return nil
}
func (f *fakeSubmissionRepo) ListStatusHistory(ctx context.Context, submissionID int64) ([]submission_model.StatusHistory, error) {
	return nil, nil
}
func (f *fakeSubmissionRepo) CreateKader(ctx context.Context, p submission_model.KaderParams) (int64, error) {
	return 0, nil
}
func (f *fakeSubmissionRepo) FindKaderBySubmission(ctx context.Context, submissionID int64) (submission_model.Kader, error) {
	return submission_model.Kader{}, errors.New("not found")
}
func (f *fakeSubmissionRepo) FindKaderByID(ctx context.Context, id int64) (submission_model.Kader, error) {
	k, ok := f.kaders[id]
	if !ok {
		return submission_model.Kader{}, errors.New("not found")
	}
	return k, nil
}
func (f *fakeSubmissionRepo) UpdateKaderStatus(ctx context.Context, id int64, status string, uniqueCode sql.NullString, issuedDate sql.NullTime) error {
	if f.kaderStatusSet == nil {
		f.kaderStatusSet = map[int64]string{}
	}
	f.kaderStatusSet[id] = status
	k := f.kaders[id]
	k.Status = status
	f.kaders[id] = k
	return nil
}
func (f *fakeSubmissionRepo) ListKadersByOrganizations(ctx context.Context, organizationIDs []int64, status string) ([]submission_model.Kader, error) {
	inScope := map[int64]bool{}
	for _, id := range organizationIDs {
		inScope[id] = true
	}
	var out []submission_model.Kader
	for _, k := range f.kaders {
		if inScope[k.OrganizationID] && (status == "" || k.Status == status) {
			out = append(out, k)
		}
	}
	return out, nil
}
func (f *fakeSubmissionRepo) CountKaderCodesForOrgYear(ctx context.Context, codePrefix string) (int, error) {
	return 0, nil
}
func (f *fakeSubmissionRepo) CreateReview(ctx context.Context, p submission_model.ReviewParams) (int64, error) {
	return 0, nil
}
func (f *fakeSubmissionRepo) ListReviewsBySubmission(ctx context.Context, submissionID int64) ([]submission_model.Review, error) {
	return nil, nil
}
func (f *fakeSubmissionRepo) UpsertFieldScore(ctx context.Context, submissionID, fieldID int64, rawScore float64, scoredByUserID int64) error {
	return nil
}
func (f *fakeSubmissionRepo) ListFieldScoresBySubmission(ctx context.Context, submissionID int64) ([]submission_model.FieldScore, error) {
	return nil, nil
}
func (f *fakeSubmissionRepo) FindCurrentLevelResult(ctx context.Context, organizationID int64) (submission_model.LevelisasiResult, error) {
	return submission_model.LevelisasiResult{}, errors.New("not found")
}
func (f *fakeSubmissionRepo) FindLevelResultBySubmission(ctx context.Context, submissionID int64) (submission_model.LevelisasiResult, error) {
	return submission_model.LevelisasiResult{}, errors.New("not found")
}
func (f *fakeSubmissionRepo) FindLevelResultHistory(ctx context.Context, organizationID int64) ([]submission_model.LevelisasiResult, error) {
	return nil, nil
}
func (f *fakeSubmissionRepo) ClearCurrentLevelResult(ctx context.Context, organizationID int64) error {
	return nil
}
func (f *fakeSubmissionRepo) CreateLevelResult(ctx context.Context, submissionID, organizationID int64, levelCode, justificationNote string, establishedByUserID int64) (int64, error) {
	return 0, nil
}
func (f *fakeSubmissionRepo) UpdateLevelResult(ctx context.Context, resultID int64, levelCode, justificationNote string, establishedByUserID int64) error {
	return nil
}
func (f *fakeSubmissionRepo) PublishLevelResult(ctx context.Context, resultID int64) error {
	return nil
}
func (f *fakeSubmissionRepo) LevelLabel(ctx context.Context, levelCode string) (string, error) {
	return "", nil
}
func (f *fakeSubmissionRepo) CountByFormAndStatuses(ctx context.Context, formID int64, statuses []string) (int, error) {
	return 0, nil
}
func (f *fakeSubmissionRepo) ListByFormAndStatuses(ctx context.Context, formID int64, statuses []string) ([]submission_model.Submission, error) {
	return nil, nil
}
func (f *fakeSubmissionRepo) WipeSubmission(ctx context.Context, sub submission_model.Submission, newVersionID int64) error {
	return nil
}

type fakeFormRepo struct{}

func (fakeFormRepo) CreateForm(ctx context.Context, formCode, formName, description string, createdBy sql.NullInt64) (int64, error) {
	return 0, nil
}
func (fakeFormRepo) ExistsFormCode(ctx context.Context, code string) (bool, error) { return false, nil }
func (fakeFormRepo) ListForms(ctx context.Context) ([]submission_form_model.Form, error) {
	return nil, nil
}
func (fakeFormRepo) FindFormByID(ctx context.Context, id int64) (submission_form_model.Form, error) {
	return submission_form_model.Form{}, errors.New("not found")
}
func (fakeFormRepo) FindFormByCode(ctx context.Context, code string) (submission_form_model.Form, error) {
	return submission_form_model.Form{}, errors.New("not found")
}
func (fakeFormRepo) CreateVersion(ctx context.Context, formID int64, versionNumber int, createdBy sql.NullInt64) (int64, error) {
	return 0, nil
}
func (fakeFormRepo) MaxVersionNumber(ctx context.Context, formID int64) (int, error) { return 0, nil }
func (fakeFormRepo) FindVersionByID(ctx context.Context, id int64) (submission_form_model.Version, error) {
	return submission_form_model.Version{}, errors.New("not found")
}
func (fakeFormRepo) FindPublishedVersionByForm(ctx context.Context, formID int64) (submission_form_model.Version, error) {
	return submission_form_model.Version{}, errors.New("not found")
}
func (fakeFormRepo) ListVersionsByForm(ctx context.Context, formID int64) ([]submission_form_model.Version, error) {
	return nil, nil
}
func (fakeFormRepo) PublishVersion(ctx context.Context, id int64, publishedBy int64) error { return nil }
func (fakeFormRepo) ArchiveOtherPublished(ctx context.Context, formID, exceptVersionID int64) error {
	return nil
}
func (fakeFormRepo) CreateSection(ctx context.Context, versionID int64, code, label string, sortOrder int, description sql.NullString) (int64, error) {
	return 0, nil
}
func (fakeFormRepo) FindSectionByID(ctx context.Context, id int64) (submission_form_model.Section, error) {
	return submission_form_model.Section{}, errors.New("not found")
}
func (fakeFormRepo) UpdateSection(ctx context.Context, id int64, label string, sortOrder int, description sql.NullString) error {
	return nil
}
func (fakeFormRepo) DeleteSection(ctx context.Context, id int64) error { return nil }
func (fakeFormRepo) ListSectionsByVersion(ctx context.Context, versionID int64) ([]submission_form_model.Section, error) {
	return nil, nil
}
func (fakeFormRepo) CreateField(ctx context.Context, p submission_form_model.FieldParams) (int64, error) {
	return 0, nil
}
func (fakeFormRepo) FindFieldByID(ctx context.Context, id int64) (submission_form_model.Field, error) {
	return submission_form_model.Field{}, errors.New("not found")
}
func (fakeFormRepo) UpdateField(ctx context.Context, id int64, p submission_form_model.FieldParams) error {
	return nil
}
func (fakeFormRepo) DeleteField(ctx context.Context, id int64) error { return nil }
func (fakeFormRepo) ListFieldsByVersion(ctx context.Context, versionID int64) ([]submission_form_model.Field, error) {
	return nil, nil
}
func (fakeFormRepo) CreateOption(ctx context.Context, fieldID int64, value, label string, sortOrder int, score sql.NullFloat64) (int64, error) {
	return 0, nil
}
func (fakeFormRepo) FindOptionByID(ctx context.Context, id int64) (submission_form_model.Option, error) {
	return submission_form_model.Option{}, errors.New("not found")
}
func (fakeFormRepo) UpdateOption(ctx context.Context, id int64, value, label string, sortOrder int, isActive bool, score sql.NullFloat64) error {
	return nil
}
func (fakeFormRepo) DeleteOption(ctx context.Context, id int64) error { return nil }
func (fakeFormRepo) ListOptionsByVersion(ctx context.Context, versionID int64) ([]submission_form_model.Option, error) {
	return nil, nil
}
func (fakeFormRepo) ListOptionsByField(ctx context.Context, fieldID int64) ([]submission_form_model.Option, error) {
	return nil, nil
}
func (fakeFormRepo) VersionStatusBySectionID(ctx context.Context, sectionID int64) (string, error) {
	return "", nil
}
func (fakeFormRepo) VersionStatusByFieldID(ctx context.Context, fieldID int64) (string, error) {
	return "", nil
}
func (fakeFormRepo) VersionIDByFieldID(ctx context.Context, fieldID int64) (int64, error) {
	return 0, nil
}
func (fakeFormRepo) CloneVersionStructure(ctx context.Context, fromVersionID, toVersionID int64) error {
	return nil
}

type fakeOrgRepoForPengurus struct {
	byID map[int64]organization_model.Organization
	root int64
}

func newTestOrgRepo() *fakeOrgRepoForPengurus {
	return &fakeOrgRepoForPengurus{byID: map[int64]organization_model.Organization{}}
}

func (f *fakeOrgRepoForPengurus) FindByID(ctx context.Context, id int64) (organization_model.Organization, error) {
	o, ok := f.byID[id]
	if !ok {
		return organization_model.Organization{}, errors.New("not found")
	}
	return o, nil
}
func (f *fakeOrgRepoForPengurus) ExistsByCode(ctx context.Context, code string) (bool, error) {
	return false, nil
}
func (f *fakeOrgRepoForPengurus) TypeCodeByID(ctx context.Context, id int64) (string, error) {
	return f.byID[id].OrganizationTypeCode, nil
}
func (f *fakeOrgRepoForPengurus) IsChildOf(ctx context.Context, targetID, parentID int64) (bool, error) {
	return false, nil
}
func (f *fakeOrgRepoForPengurus) ListAll(ctx context.Context, filt organization_dto.ListFilter) ([]organization_model.Organization, int64, error) {
	return nil, 0, nil
}
func (f *fakeOrgRepoForPengurus) ListByTypes(ctx context.Context, types []string, filt organization_dto.ListFilter) ([]organization_model.Organization, int64, error) {
	return nil, 0, nil
}
func (f *fakeOrgRepoForPengurus) ListSelfAndChildren(ctx context.Context, id int64, filt organization_dto.ListFilter) ([]organization_model.Organization, int64, error) {
	return nil, 0, nil
}
func (f *fakeOrgRepoForPengurus) Children(ctx context.Context, id int64) ([]organization_model.Organization, error) {
	return nil, nil
}
func (f *fakeOrgRepoForPengurus) RootPuskomnasID(ctx context.Context) (int64, error) {
	return f.root, nil
}
func (f *fakeOrgRepoForPengurus) ListActiveByType(ctx context.Context, typeCode string) ([]organization_model.Organization, error) {
	return nil, nil
}
func (f *fakeOrgRepoForPengurus) Create(ctx context.Context, p organization_model.CreateParams) (int64, error) {
	return 0, nil
}
func (f *fakeOrgRepoForPengurus) Update(ctx context.Context, id int64, name, province, city, email, phone, website, photoURL string, updatedBy int64) error {
	return nil
}
func (f *fakeOrgRepoForPengurus) SetActive(ctx context.Context, id int64, active bool, updatedBy int64) error {
	return nil
}

type fakeUserRepoForPengurus struct {
	byID    map[int64]user_model.User
	updates []userUpdateCall
	bumped  []int64
}

type userUpdateCall struct {
	id             int64
	roleID         int64
	organizationID sql.NullInt64
}

func newTestUserRepo() *fakeUserRepoForPengurus {
	return &fakeUserRepoForPengurus{byID: map[int64]user_model.User{}}
}

func (f *fakeUserRepoForPengurus) FindByID(ctx context.Context, id int64) (user_model.User, error) {
	u, ok := f.byID[id]
	if !ok {
		return user_model.User{}, errors.New("not found")
	}
	return u, nil
}
func (f *fakeUserRepoForPengurus) FindByEmail(ctx context.Context, email string) (user_model.User, error) {
	return user_model.User{}, errors.New("not found")
}
func (f *fakeUserRepoForPengurus) FindByGoogleID(ctx context.Context, googleID string) (user_model.User, error) {
	return user_model.User{}, errors.New("not found")
}
func (f *fakeUserRepoForPengurus) ExistsByEmail(ctx context.Context, email string) (bool, error) {
	return false, nil
}
func (f *fakeUserRepoForPengurus) ExistsByEmailExcept(ctx context.Context, email string, exceptID int64) (bool, error) {
	return false, nil
}
func (f *fakeUserRepoForPengurus) Create(ctx context.Context, p user_model.CreateParams) (int64, error) {
	return 0, nil
}
func (f *fakeUserRepoForPengurus) List(ctx context.Context, filt user_dto.ListFilter) ([]user_model.User, int64, error) {
	return nil, 0, nil
}
func (f *fakeUserRepoForPengurus) SearchActive(ctx context.Context, search string, limit int) ([]user_model.User, error) {
	return nil, nil
}
func (f *fakeUserRepoForPengurus) Update(ctx context.Context, id int64, fullName, email string, roleID int64, isActive bool, organizationID sql.NullInt64, wildcardTierAccess sql.NullString, updatedBy int64) error {
	f.updates = append(f.updates, userUpdateCall{id: id, roleID: roleID, organizationID: organizationID})
	u := f.byID[id]
	u.RoleID = roleID
	u.OrganizationID = organizationID
	f.byID[id] = u
	return nil
}
func (f *fakeUserRepoForPengurus) SetOrganizationID(ctx context.Context, id int64, organizationID int64) error {
	return nil
}
func (f *fakeUserRepoForPengurus) UpdateContactInfo(ctx context.Context, id int64, phoneNumber, address sql.NullString) error {
	return nil
}
func (f *fakeUserRepoForPengurus) SetActive(ctx context.Context, id int64, active bool, updatedBy int64) error {
	return nil
}
func (f *fakeUserRepoForPengurus) SetPassword(ctx context.Context, id int64, hashed string, mustChange bool) error {
	return nil
}
func (f *fakeUserRepoForPengurus) LinkGoogle(ctx context.Context, id int64, googleID string, markVerified bool) error {
	return nil
}
func (f *fakeUserRepoForPengurus) UpdatePhoto(ctx context.Context, id int64, photoURL string) error {
	return nil
}
func (f *fakeUserRepoForPengurus) UpdateCustomPhoto(ctx context.Context, id int64, photoURL string) error {
	return nil
}
func (f *fakeUserRepoForPengurus) MarkEmailVerified(ctx context.Context, id int64) error { return nil }
func (f *fakeUserRepoForPengurus) SetEmailVerified(ctx context.Context, id int64, verified bool) error {
	return nil
}
func (f *fakeUserRepoForPengurus) SoftDelete(ctx context.Context, id int64, updatedBy int64) error {
	return nil
}
func (f *fakeUserRepoForPengurus) LogLogin(ctx context.Context, userID int64, ip, ua, status string) error {
	return nil
}
func (f *fakeUserRepoForPengurus) TokenVersion(ctx context.Context, id int64) (int, error) { return 0, nil }
func (f *fakeUserRepoForPengurus) BumpTokenVersion(ctx context.Context, id int64) error {
	f.bumped = append(f.bumped, id)
	return nil
}

type fakeRoleRepoForPengurus struct{ idsByName map[string]int64 }

func newTestRoleRepo() *fakeRoleRepoForPengurus {
	return &fakeRoleRepoForPengurus{idsByName: map[string]int64{
		roleNameKader: 1, constants.RoleLDKAdmin: 2,
		constants.RolePuskomdaVerifikator: 3, constants.RolePuskomnasVerifikator: 4,
	}}
}

func (f *fakeRoleRepoForPengurus) IDByName(ctx context.Context, name string) (int64, error) {
	id, ok := f.idsByName[name]
	if !ok {
		return 0, errors.New("role not found")
	}
	return id, nil
}
func (f *fakeRoleRepoForPengurus) FindByID(ctx context.Context, id int64) (role_model.Role, error) {
	return role_model.Role{}, nil
}
func (f *fakeRoleRepoForPengurus) ListWithUserCount(ctx context.Context, search string) ([]role_model.Role, error) {
	return nil, nil
}
func (f *fakeRoleRepoForPengurus) Create(ctx context.Context, name, desc string) (int64, error) {
	return 0, nil
}
func (f *fakeRoleRepoForPengurus) Update(ctx context.Context, id int64, name, desc string, isActive bool) error {
	return nil
}
func (f *fakeRoleRepoForPengurus) Delete(ctx context.Context, id int64) error { return nil }
func (f *fakeRoleRepoForPengurus) CountUsers(ctx context.Context, id int64) (int, error) { return 0, nil }
func (f *fakeRoleRepoForPengurus) PermissionIDs(ctx context.Context, roleID int64) ([]int64, error) {
	return nil, nil
}
func (f *fakeRoleRepoForPengurus) PermissionNames(ctx context.Context, roleID int64) ([]string, error) {
	return nil, nil
}
func (f *fakeRoleRepoForPengurus) SetPermissions(ctx context.Context, roleID int64, permissionIDs []int64) error {
	return nil
}
func (f *fakeRoleRepoForPengurus) UsersByRole(ctx context.Context, roleID int64) ([]role_model.RoleUser, error) {
	return nil, nil
}
func (f *fakeRoleRepoForPengurus) BumpTokenVersionForRole(ctx context.Context, roleID int64) error {
	return nil
}

type fakeOrgScopeForPengurus struct{ accessibleIDs []int64 }

func (f *fakeOrgScopeForPengurus) IsAccessible(ctx context.Context, callerOrganizationID *int64, callerOrganizationTypeCode, wildcardTierAccess string, targetOrganizationID int64) (bool, error) {
	for _, id := range f.accessibleIDs {
		if id == targetOrganizationID {
			return true, nil
		}
	}
	return false, nil
}
func (f *fakeOrgScopeForPengurus) AccessibleOrganizationIDs(ctx context.Context, callerOrganizationID *int64, callerOrganizationTypeCode, wildcardTierAccess string) ([]int64, error) {
	return f.accessibleIDs, nil
}
func (f *fakeOrgScopeForPengurus) AccessibleOrganizationIDsForTarget(ctx context.Context, targetOrganizationID int64) ([]int64, error) {
	return f.accessibleIDs, nil
}

// ---------- scenario fixtures ----------

const (
	ldkOrgID       int64 = 10
	puskomdaOrgID  int64 = 20
	puskomnasOrgID int64 = 30
	kaderUserID    int64 = 101
	callerUserID   int64 = 999
)

func buildTestService() (Service, *fakeSubmissionRepo, *fakeUserRepoForPengurus, *fakeOrgRepoForPengurus) {
	subRepo := &fakeSubmissionRepo{kaders: map[int64]submission_model.Kader{
		1: {KaderID: 1, UserID: kaderUserID, OrganizationID: ldkOrgID, FullName: "Budi", Status: constants.KaderStatusActive},
	}}
	userRepo := newTestUserRepo()
	userRepo.byID[kaderUserID] = user_model.User{UserID: kaderUserID, FullName: "Budi", Email: "budi@example.com", RoleID: 1, RoleName: roleNameKader, IsActive: true, OrganizationID: sql.NullInt64{Int64: ldkOrgID, Valid: true}}
	userRepo.byID[callerUserID] = user_model.User{UserID: callerUserID, FullName: "Caller", Email: "caller@example.com", RoleID: 2, RoleName: constants.RoleLDKAdmin, IsActive: true, OrganizationID: sql.NullInt64{Int64: ldkOrgID, Valid: true}}

	orgRepo := newTestOrgRepo()
	orgRepo.byID[ldkOrgID] = organization_model.Organization{OrganizationID: ldkOrgID, OrganizationTypeCode: constants.OrgTypeLDK, OrganizationName: "LDK Syahid", ParentOrganizationID: sql.NullInt64{Int64: puskomdaOrgID, Valid: true}}
	orgRepo.byID[puskomdaOrgID] = organization_model.Organization{OrganizationID: puskomdaOrgID, OrganizationTypeCode: constants.OrgTypePuskomda, OrganizationName: "Puskomda Banten"}
	orgRepo.byID[puskomnasOrgID] = organization_model.Organization{OrganizationID: puskomnasOrgID, OrganizationTypeCode: constants.OrgTypePuskomnas, OrganizationName: "Puskomnas"}
	orgRepo.root = puskomnasOrgID

	roleRepo := newTestRoleRepo()
	orgScope := &fakeOrgScopeForPengurus{accessibleIDs: []int64{ldkOrgID, puskomdaOrgID, puskomnasOrgID}}

	svc := NewService(subRepo, fakeFormRepo{}, orgRepo, userRepo, roleRepo, orgScope)
	return svc, subRepo, userRepo, orgRepo
}

func ldkCaller() CallerScope {
	id := ldkOrgID
	return CallerScope{UserID: callerUserID, OrganizationID: &id, OrganizationTypeCode: constants.OrgTypeLDK}
}

// ---------- UpdatePengurusRole scenarios ----------

func TestUpdatePengurusRole_RejectsSelfAction(t *testing.T) {
	svc, _, _, _ := buildTestService()
	caller := ldkCaller()
	caller.UserID = kaderUserID // caller targeting themselves

	_, err := svc.UpdatePengurusRole(context.Background(), kaderUserID, caller, submission_dto.UpdatePengurusRoleRequest{RoleName: constants.RoleLDKAdmin})
	if err == nil {
		t.Fatal("UpdatePengurusRole() error = nil, want Forbidden for self-action")
	}
}

func TestUpdatePengurusRole_RejectsRoleOutsideCallerTierAllowList(t *testing.T) {
	svc, _, _, _ := buildTestService()

	_, err := svc.UpdatePengurusRole(context.Background(), kaderUserID, ldkCaller(), submission_dto.UpdatePengurusRoleRequest{RoleName: constants.RolePuskomdaVerifikator})
	if err == nil {
		t.Fatal("UpdatePengurusRole() error = nil, want Forbidden — LDK caller cannot assign Puskomda Verifikator")
	}
}

func TestUpdatePengurusRole_RejectsTargetOutsideCallerScope(t *testing.T) {
	svc, subRepo, _, _ := buildTestService()
	subRepo.kaders[2] = submission_model.Kader{KaderID: 2, UserID: 777, OrganizationID: 999999, Status: constants.KaderStatusActive}

	_, err := svc.UpdatePengurusRole(context.Background(), 777, ldkCaller(), submission_dto.UpdatePengurusRoleRequest{RoleName: constants.RoleLDKAdmin})
	if err == nil {
		t.Fatal("UpdatePengurusRole() error = nil, want NotFound — target's org is outside caller's accessible scope")
	}
}

func TestUpdatePengurusRole_PromoteToLDKAdmin_KeepsHomeOrganizationID(t *testing.T) {
	svc, _, userRepo, _ := buildTestService()

	entry, err := svc.UpdatePengurusRole(context.Background(), kaderUserID, ldkCaller(), submission_dto.UpdatePengurusRoleRequest{RoleName: constants.RoleLDKAdmin})
	if err != nil {
		t.Fatalf("UpdatePengurusRole() error = %v", err)
	}
	if entry.HomeOrganizationID != ldkOrgID {
		t.Errorf("entry.HomeOrganizationID = %d, want %d (unchanged)", entry.HomeOrganizationID, ldkOrgID)
	}
	if len(userRepo.updates) != 1 || userRepo.updates[0].organizationID.Int64 != ldkOrgID {
		t.Errorf("userRepo.Update() organizationID = %+v, want %d", userRepo.updates, ldkOrgID)
	}
	if len(userRepo.bumped) != 1 || userRepo.bumped[0] != kaderUserID {
		t.Errorf("expected BumpTokenVersion(%d) exactly once, got %v", kaderUserID, userRepo.bumped)
	}
}

func TestUpdatePengurusRole_PromoteToPuskomdaVerifikator_ReassignsToParentOrg(t *testing.T) {
	svc, _, userRepo, _ := buildTestService()
	caller := CallerScope{UserID: callerUserID, OrganizationID: func() *int64 { id := puskomdaOrgID; return &id }(), OrganizationTypeCode: constants.OrgTypePuskomda}

	_, err := svc.UpdatePengurusRole(context.Background(), kaderUserID, caller, submission_dto.UpdatePengurusRoleRequest{RoleName: constants.RolePuskomdaVerifikator})
	if err != nil {
		t.Fatalf("UpdatePengurusRole() error = %v", err)
	}
	if len(userRepo.updates) != 1 || userRepo.updates[0].organizationID.Int64 != puskomdaOrgID {
		t.Errorf("userRepo.Update() organizationID = %+v, want %d (LDK's parent Puskomda)", userRepo.updates, puskomdaOrgID)
	}
}

func TestUpdatePengurusRole_PromoteToPuskomnasVerifikator_ReassignsToRoot(t *testing.T) {
	svc, _, userRepo, _ := buildTestService()
	caller := CallerScope{UserID: callerUserID, OrganizationID: func() *int64 { id := puskomnasOrgID; return &id }(), OrganizationTypeCode: constants.OrgTypePuskomnas}

	_, err := svc.UpdatePengurusRole(context.Background(), kaderUserID, caller, submission_dto.UpdatePengurusRoleRequest{RoleName: constants.RolePuskomnasVerifikator})
	if err != nil {
		t.Fatalf("UpdatePengurusRole() error = %v", err)
	}
	if len(userRepo.updates) != 1 || userRepo.updates[0].organizationID.Int64 != puskomnasOrgID {
		t.Errorf("userRepo.Update() organizationID = %+v, want %d (Puskomnas root)", userRepo.updates, puskomnasOrgID)
	}
}

// ---------- ListPengurus ----------

func TestListPengurus_ReturnsActiveKaderInScope(t *testing.T) {
	svc, _, _, _ := buildTestService()

	entries, err := svc.ListPengurus(context.Background(), ldkCaller())
	if err != nil {
		t.Fatalf("ListPengurus() error = %v", err)
	}
	if len(entries) != 1 || entries[0].UserID != kaderUserID {
		t.Fatalf("ListPengurus() = %+v, want exactly one entry for userID %d", entries, kaderUserID)
	}
	if entries[0].RoleName != roleNameKader {
		t.Errorf("entries[0].RoleName = %q, want %q", entries[0].RoleName, roleNameKader)
	}
	if entries[0].HomeOrganizationName != "LDK Syahid" {
		t.Errorf("entries[0].HomeOrganizationName = %q, want %q", entries[0].HomeOrganizationName, "LDK Syahid")
	}
}

// ---------- DeactivateKader cascade ----------

func TestDeactivateKader_RevertsPromotedUserToKader(t *testing.T) {
	svc, subRepo, userRepo, _ := buildTestService()
	userRepo.byID[kaderUserID] = user_model.User{UserID: kaderUserID, FullName: "Budi", Email: "budi@example.com", RoleID: 2, RoleName: constants.RoleLDKAdmin, IsActive: true, OrganizationID: sql.NullInt64{Int64: ldkOrgID, Valid: true}}

	if err := svc.DeactivateKader(context.Background(), 1, ldkCaller()); err != nil {
		t.Fatalf("DeactivateKader() error = %v", err)
	}
	if subRepo.kaderStatusSet[1] != constants.KaderStatusInactive {
		t.Errorf("kader status = %q, want %q", subRepo.kaderStatusSet[1], constants.KaderStatusInactive)
	}
	if len(userRepo.updates) != 1 || userRepo.updates[0].roleID != 1 {
		t.Fatalf("expected exactly one role revert to Kader (roleID 1), got %+v", userRepo.updates)
	}
	if userRepo.updates[0].organizationID.Int64 != ldkOrgID {
		t.Errorf("reverted organizationID = %d, want %d (home LDK)", userRepo.updates[0].organizationID.Int64, ldkOrgID)
	}
}

func TestDeactivateKader_PlainKader_NeverTouchesUserRecord(t *testing.T) {
	svc, _, userRepo, _ := buildTestService()
	// kaderUserID is already role Kader in buildTestService()'s fixture.

	if err := svc.DeactivateKader(context.Background(), 1, ldkCaller()); err != nil {
		t.Fatalf("DeactivateKader() error = %v", err)
	}
	if len(userRepo.updates) != 0 {
		t.Errorf("expected no userRepo.Update() calls for a plain Kader, got %+v", userRepo.updates)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd fsldk-api && go test ./modules/submission/submission_service/... -run 'TestPengurusRoleAllowed|TestUpdatePengurusRole|TestListPengurus|TestDeactivateKader' -v`
Expected: FAIL — compile errors (`pengurusRoleAllowed`, `ListPengurus`, `UpdatePengurusRole` undefined on `Service`/`ServiceImpl`; everything else in the test file compiles cleanly since all fakes already implement their real interfaces in full).

- [ ] **Step 3: Add `pengurusRolesByTier` and `pengurusRoleAllowed`**

In `fsldk-api/modules/submission/submission_service/submission_service_impl.go`, find this exact block:

```go
const (
	roleNamePengunjung = "Pengunjung"
	roleNameKader      = "Kader"
)
```

Replace it with:

```go
const (
	roleNamePengunjung = "Pengunjung"
	roleNameKader      = "Kader"
)

// pengurusRolesByTier memetakan organizationTypeCode caller ke daftar role
// yang boleh ditetapkan lewat fitur Pengurus (promosi/demosi kader aktif
// dalam scope organisasi caller sendiri) — index menaik = rank lebih
// tinggi. Dipakai pengurusRoleAllowed (validasi UpdatePengurusRole) dan
// TIDAK dipakai arah sebaliknya oleh DeactivateKader (yang selalu
// menurunkan ke Kader tanpa pengecekan allow-list, karena itu aksi sistem
// otomatis, bukan permintaan caller).
var pengurusRolesByTier = map[string][]string{
	constants.OrgTypeLDK:       {roleNameKader, constants.RoleLDKAdmin},
	constants.OrgTypePuskomda:  {roleNameKader, constants.RoleLDKAdmin, constants.RolePuskomdaVerifikator},
	constants.OrgTypePuskomnas: {roleNameKader, constants.RoleLDKAdmin, constants.RolePuskomdaVerifikator, constants.RolePuskomnasVerifikator},
}

func pengurusRoleAllowed(callerOrgType, roleName string) bool {
	for _, r := range pengurusRolesByTier[callerOrgType] {
		if r == roleName {
			return true
		}
	}
	return false
}
```

- [ ] **Step 4: Add `setPengurusRole`, `ListPengurus`, `UpdatePengurusRole`**

In the same file, find this exact block (end of `ReactivateKader`):

```go
	if err := s.repo.UpdateKaderStatus(ctx, kaderID, constants.KaderStatusActive, sql.NullString{}, sql.NullTime{}); err != nil {
		return apperror.Internal("")
	}
	return nil
}
```

Replace it with:

```go
	if err := s.repo.UpdateKaderStatus(ctx, kaderID, constants.KaderStatusActive, sql.NullString{}, sql.NullTime{}); err != nil {
		return apperror.Internal("")
	}
	return nil
}

// setPengurusRole adalah inti mutasi role pengurus, dipakai UpdatePengurusRole
// (promosi/demosi manual lewat endpoint) MAUPUN cascade DeactivateKader
// (revert otomatis ke Kader) — satu fungsi supaya kedua jalur itu tidak
// pernah berbeda perilaku. organizationID BARU dihitung dari roleName
// TUJUAN (bukan role asal): Kader/LDK Admin selalu balik ke
// homeOrganizationID (ms_kader.organizationID, tidak pernah berubah);
// Puskomda Verifikator pindah ke Puskomda induk LDK itu; Puskomnas
// Verifikator pindah ke Puskomnas tunggal. Wajib begini karena
// ms_user.organizationTypeCode SELALU di-join live dari organizationID
// (lihat user_repository_impl.go selectCols) dan itulah yang menentukan
// cakupan akses caller di seluruh backend — kalau organizationID tidak
// ikut pindah, role Verifikator-nya tidak akan berfungsi penuh.
func (s *ServiceImpl) setPengurusRole(ctx context.Context, userID, homeOrganizationID int64, roleName string, actorID int64) error {
	var targetOrgID int64
	switch roleName {
	case roleNameKader, constants.RoleLDKAdmin:
		targetOrgID = homeOrganizationID
	case constants.RolePuskomdaVerifikator:
		homeOrg, err := s.orgRepo.FindByID(ctx, homeOrganizationID)
		if err != nil {
			return apperror.Internal("")
		}
		if !homeOrg.ParentOrganizationID.Valid {
			return apperror.Internal("LDK ini tidak memiliki Puskomda induk")
		}
		targetOrgID = homeOrg.ParentOrganizationID.Int64
	case constants.RolePuskomnasVerifikator:
		rootID, err := s.orgRepo.RootPuskomnasID(ctx)
		if err != nil {
			return apperror.Internal("")
		}
		targetOrgID = rootID
	default:
		return apperror.BadRequest("Role tidak dikenali")
	}
	roleID, err := s.roleRepo.IDByName(ctx, roleName)
	if err != nil {
		return apperror.BadRequest("Role tidak ditemukan")
	}
	u, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		return apperror.NotFound("Pengguna tidak ditemukan")
	}
	if err := s.userRepo.Update(ctx, userID, u.FullName, u.Email, roleID, u.IsActive, sql.NullInt64{Int64: targetOrgID, Valid: true}, u.WildcardTierAccess, actorID); err != nil {
		return apperror.Internal("")
	}
	return s.userRepo.BumpTokenVersion(ctx, userID)
}

func (s *ServiceImpl) pengurusEntryFor(ctx context.Context, userID, homeOrganizationID int64) (submission_dto.PengurusEntry, error) {
	u, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		return submission_dto.PengurusEntry{}, apperror.Internal("")
	}
	org, err := s.orgRepo.FindByID(ctx, homeOrganizationID)
	if err != nil {
		return submission_dto.PengurusEntry{}, apperror.Internal("")
	}
	return submission_dto.PengurusEntry{
		UserID: u.UserID, FullName: u.FullName, Email: u.Email, RoleName: u.RoleName,
		HomeOrganizationID: homeOrganizationID, HomeOrganizationName: org.OrganizationName,
	}, nil
}

func (s *ServiceImpl) ListPengurus(ctx context.Context, caller CallerScope) ([]submission_dto.PengurusEntry, error) {
	orgIDs, err := s.orgScope.AccessibleOrganizationIDs(ctx, caller.OrganizationID, caller.OrganizationTypeCode, caller.WildcardTierAccess)
	if err != nil {
		return nil, apperror.Internal("")
	}
	kaders, err := s.repo.ListKadersByOrganizations(ctx, orgIDs, constants.KaderStatusActive)
	if err != nil {
		return nil, apperror.Internal("")
	}
	out := make([]submission_dto.PengurusEntry, 0, len(kaders))
	for _, k := range kaders {
		entry, err := s.pengurusEntryFor(ctx, k.UserID, k.OrganizationID)
		if err != nil {
			continue
		}
		out = append(out, entry)
	}
	return out, nil
}

func (s *ServiceImpl) UpdatePengurusRole(ctx context.Context, targetUserID int64, caller CallerScope, req submission_dto.UpdatePengurusRoleRequest) (submission_dto.PengurusEntry, error) {
	if targetUserID == caller.UserID {
		return submission_dto.PengurusEntry{}, apperror.Forbidden("Anda tidak dapat mengubah role akun Anda sendiri")
	}
	if !pengurusRoleAllowed(caller.OrganizationTypeCode, req.RoleName) {
		return submission_dto.PengurusEntry{}, apperror.Forbidden("Role tersebut tidak dapat ditetapkan dari portal ini")
	}
	orgIDs, err := s.orgScope.AccessibleOrganizationIDs(ctx, caller.OrganizationID, caller.OrganizationTypeCode, caller.WildcardTierAccess)
	if err != nil {
		return submission_dto.PengurusEntry{}, apperror.Internal("")
	}
	kaders, err := s.repo.ListKadersByOrganizations(ctx, orgIDs, constants.KaderStatusActive)
	if err != nil {
		return submission_dto.PengurusEntry{}, apperror.Internal("")
	}
	var homeOrgID int64
	found := false
	for _, k := range kaders {
		if k.UserID == targetUserID {
			homeOrgID = k.OrganizationID
			found = true
			break
		}
	}
	if !found {
		return submission_dto.PengurusEntry{}, apperror.NotFound("Kader tidak ditemukan atau di luar jangkauan akses Anda")
	}
	if err := s.setPengurusRole(ctx, targetUserID, homeOrgID, req.RoleName, caller.UserID); err != nil {
		return submission_dto.PengurusEntry{}, err
	}
	return s.pengurusEntryFor(ctx, targetUserID, homeOrgID)
}
```

- [ ] **Step 5: Add the `DeactivateKader` cascade**

In the same file, find this exact block:

```go
	if k.Status != constants.KaderStatusActive {
		return apperror.InvalidStatusTransition("Hanya kader berstatus ACTIVE yang dapat dinonaktifkan")
	}
	if err := s.repo.UpdateKaderStatus(ctx, kaderID, constants.KaderStatusInactive, sql.NullString{}, sql.NullTime{}); err != nil {
		return apperror.Internal("")
	}
	return nil
}
```

Replace it with:

```go
	if k.Status != constants.KaderStatusActive {
		return apperror.InvalidStatusTransition("Hanya kader berstatus ACTIVE yang dapat dinonaktifkan")
	}
	if err := s.repo.UpdateKaderStatus(ctx, kaderID, constants.KaderStatusInactive, sql.NullString{}, sql.NullTime{}); err != nil {
		return apperror.Internal("")
	}
	// Cascade: kalau kader ini sedang memegang role pengurus (LDK Admin/
	// Puskomda/Puskomnas Verifikator), otomatis turunkan balik ke Kader +
	// organizationID balik ke LDK asalnya — tidak masuk akal akun nonaktif
	// tetap pegang akses admin. Best-effort seperti promoteToKaderRole:
	// kegagalan di sini TIDAK membatalkan nonaktifkan kader-nya sendiri.
	if u, err := s.userRepo.FindByID(ctx, k.UserID); err == nil {
		if u.RoleName == constants.RoleLDKAdmin || u.RoleName == constants.RolePuskomdaVerifikator || u.RoleName == constants.RolePuskomnasVerifikator {
			_ = s.setPengurusRole(ctx, k.UserID, k.OrganizationID, roleNameKader, caller.UserID)
		}
	}
	return nil
}
```

- [ ] **Step 6: Add the 2 new methods to the `Service` interface**

In `fsldk-api/modules/submission/submission_service/submission_service.go`, find this exact block:

```go
	ListKaders(ctx context.Context, caller CallerScope, q dto.ListQuery, status string) ([]submission_dto.KaderResponse, int, error)
	GetKaderCode(ctx context.Context, kaderID int64, caller CallerScope) (submission_dto.KaderResponse, error)
	DeactivateKader(ctx context.Context, kaderID int64, caller CallerScope) error
	ReactivateKader(ctx context.Context, kaderID int64, caller CallerScope) error
	ReassessKader(ctx context.Context, id int64, caller CallerScope, req submission_dto.VersionedRequest) (submission_dto.Response, error)
	ReinstateKader(ctx context.Context, kaderID int64, caller CallerScope) error
}
```

Replace it with:

```go
	ListKaders(ctx context.Context, caller CallerScope, q dto.ListQuery, status string) ([]submission_dto.KaderResponse, int, error)
	GetKaderCode(ctx context.Context, kaderID int64, caller CallerScope) (submission_dto.KaderResponse, error)
	DeactivateKader(ctx context.Context, kaderID int64, caller CallerScope) error
	ReactivateKader(ctx context.Context, kaderID int64, caller CallerScope) error
	ReassessKader(ctx context.Context, id int64, caller CallerScope, req submission_dto.VersionedRequest) (submission_dto.Response, error)
	ReinstateKader(ctx context.Context, kaderID int64, caller CallerScope) error

	ListPengurus(ctx context.Context, caller CallerScope) ([]submission_dto.PengurusEntry, error)
	UpdatePengurusRole(ctx context.Context, targetUserID int64, caller CallerScope, req submission_dto.UpdatePengurusRoleRequest) (submission_dto.PengurusEntry, error)
}
```

- [ ] **Step 7: Run tests to verify they pass**

Run: `cd fsldk-api && go test ./modules/submission/submission_service/... -run 'TestPengurusRoleAllowed|TestUpdatePengurusRole|TestListPengurus|TestDeactivateKader' -v`
Expected: PASS — all table cases and scenario tests green.

- [ ] **Step 8: Run the full existing submission_service test suite**

Run: `cd fsldk-api && go test ./modules/submission/... -v`
Expected: PASS — `scoring_test.go`'s existing tests unaffected, plus the new `pengurus_test.go` tests.

- [ ] **Step 9: Commit**

```bash
cd fsldk-api
git add modules/submission/submission_service
git commit -m "feat(submission): add Pengurus list/promote/demote + auto-revert-on-deactivate cascade"
```

---

### Task 3: Backend handler, router wiring, and build verification

**Files:**
- Modify: `fsldk-api/modules/submission/submission_handler/submission_handler.go`
- Modify: `fsldk-api/modules/submission/submission_handler/submission_handler_impl.go`
- Modify: `fsldk-api/modules/submission/router.go`

**Interfaces:**
- Consumes: `submission_service.Service.ListPengurus`/`UpdatePengurusRole` (Task 2); existing `callerScope(c)` helper (`submission_handler_impl.go:32-40`); `httphelper.Success`/`Error`; `apperror.BadRequest`; `validation.Struct`.
- Produces: `submission_handler.Handler` gains `ListPengurus(c *gin.Context)`, `UpdatePengurusRole(c *gin.Context)`; `submission_handler_impl.go` gains a `userIDParam(c) (int64, bool)` helper (parallel to the existing `idParam`, reading `c.Param("userID")` instead of `c.Param("id")`).

No new test — these are thin pass-throughs, identical shape to `DeactivateKader`'s/`ListKaders`' own handlers (neither has a test). Verified by `go build`/`go vet` plus Task 6's manual E2E.

- [ ] **Step 1: Add the 2 methods to the `Handler` interface**

In `fsldk-api/modules/submission/submission_handler/submission_handler.go`, find this exact block:

```go
	ListKaders(c *gin.Context)
	GetKaderCode(c *gin.Context)
	DeactivateKader(c *gin.Context)
	ReactivateKader(c *gin.Context)
	ReassessKader(c *gin.Context)
	ReinstateKader(c *gin.Context)
}
```

Replace it with:

```go
	ListKaders(c *gin.Context)
	GetKaderCode(c *gin.Context)
	DeactivateKader(c *gin.Context)
	ReactivateKader(c *gin.Context)
	ReassessKader(c *gin.Context)
	ReinstateKader(c *gin.Context)

	ListPengurus(c *gin.Context)
	UpdatePengurusRole(c *gin.Context)
}
```

- [ ] **Step 2: Add the `userIDParam` helper**

In `fsldk-api/modules/submission/submission_handler/submission_handler_impl.go`, find this exact block:

```go
func idParam(c *gin.Context) (int64, bool) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		httphelper.Error(c, apperror.BadRequest("ID tidak valid"))
		return 0, false
	}
	return id, true
}
```

Replace it with:

```go
func idParam(c *gin.Context) (int64, bool) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		httphelper.Error(c, apperror.BadRequest("ID tidak valid"))
		return 0, false
	}
	return id, true
}

func userIDParam(c *gin.Context) (int64, bool) {
	id, err := strconv.ParseInt(c.Param("userID"), 10, 64)
	if err != nil || id <= 0 {
		httphelper.Error(c, apperror.BadRequest("ID pengguna tidak valid"))
		return 0, false
	}
	return id, true
}
```

- [ ] **Step 3: Add the 2 handler functions**

In the same file, find this exact block (end of `ReactivateKader`):

```go
func (h *HandlerImpl) ReactivateKader(c *gin.Context) {
	id, ok := idParam(c)
	if !ok {
		return
	}
	if err := h.svc.ReactivateKader(c.Request.Context(), id, callerScope(c)); err != nil {
		httphelper.Error(c, err)
		return
	}
	httphelper.Success(c, "Kader berhasil diaktifkan kembali", nil)
}
```

Replace it with:

```go
func (h *HandlerImpl) ReactivateKader(c *gin.Context) {
	id, ok := idParam(c)
	if !ok {
		return
	}
	if err := h.svc.ReactivateKader(c.Request.Context(), id, callerScope(c)); err != nil {
		httphelper.Error(c, err)
		return
	}
	httphelper.Success(c, "Kader berhasil diaktifkan kembali", nil)
}

func (h *HandlerImpl) ListPengurus(c *gin.Context) {
	data, err := h.svc.ListPengurus(c.Request.Context(), callerScope(c))
	if err != nil {
		httphelper.Error(c, err)
		return
	}
	httphelper.Success(c, "", data)
}

func (h *HandlerImpl) UpdatePengurusRole(c *gin.Context) {
	userID, ok := userIDParam(c)
	if !ok {
		return
	}
	var req submission_dto.UpdatePengurusRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httphelper.Error(c, apperror.BadRequest("Format permintaan tidak valid"))
		return
	}
	if err := validation.Struct(req); err != nil {
		httphelper.Error(c, err)
		return
	}
	data, err := h.svc.UpdatePengurusRole(c.Request.Context(), userID, callerScope(c), req)
	if err != nil {
		httphelper.Error(c, err)
		return
	}
	httphelper.Success(c, "Role pengurus berhasil diperbarui", data)
}
```

- [ ] **Step 4: Add the `/pengurus` route group**

In `fsldk-api/modules/submission/router.go`, find this exact block:

```go
	k := rg.Group("/kaders")
	k.Use(mw.Auth())
	{
		k.GET("", mw.RequirePermission(constants.PermSubmissionReviewLDK), h.ListKaders)
		k.GET("/:id/code", h.GetKaderCode)
		k.POST("/:id/deactivate", mw.RequirePermission(constants.PermKaderDeactivate), h.DeactivateKader)
		k.POST("/:id/reactivate", mw.RequirePermission(constants.PermKaderDeactivate), h.ReactivateKader)
		k.POST("/:id/reinstate", mw.RequirePermission(constants.PermKaderDeactivate), h.ReinstateKader)
	}
}
```

Replace it with:

```go
	k := rg.Group("/kaders")
	k.Use(mw.Auth())
	{
		k.GET("", mw.RequirePermission(constants.PermSubmissionReviewLDK), h.ListKaders)
		k.GET("/:id/code", h.GetKaderCode)
		k.POST("/:id/deactivate", mw.RequirePermission(constants.PermKaderDeactivate), h.DeactivateKader)
		k.POST("/:id/reactivate", mw.RequirePermission(constants.PermKaderDeactivate), h.ReactivateKader)
		k.POST("/:id/reinstate", mw.RequirePermission(constants.PermKaderDeactivate), h.ReinstateKader)
	}

	// /pengurus adalah GROUP TERPISAH dari /kaders (bukan nested di
	// bawahnya) — Gin/httprouter tidak bisa campur segmen statis dengan
	// wildcard (:id) di kedalaman path yang sama, dan /kaders/:id/... sudah
	// menempati kedalaman itu.
	p := rg.Group("/pengurus")
	p.Use(mw.Auth())
	{
		pengurusPerms := []string{constants.PermPengurusLDKManage, constants.PermPengurusPuskomdaManage, constants.PermPengurusPuskomnasManage}
		p.GET("", mw.RequirePermission(pengurusPerms...), h.ListPengurus)
		p.PATCH("/:userID/role", mw.RequirePermission(pengurusPerms...), h.UpdatePengurusRole)
	}
}
```

- [ ] **Step 5: Verify it builds and vets clean**

Run: `cd fsldk-api && go build ./... && go vet ./...`
Expected: no output, exit code 0.

- [ ] **Step 6: Run the full backend test suite**

Run: `cd fsldk-api && go test ./...`
Expected: PASS across all packages (including Task 2's new tests).

- [ ] **Step 7: Commit**

```bash
cd fsldk-api
git add modules/submission/submission_handler modules/submission/router.go
git commit -m "feat(submission): wire Pengurus handler and /pengurus routes"
```

---

### Task 4: Frontend data layer — entity, API service, repository

**Files:**
- Create: `fsldk-web/src/app/modules/pengurus/entities/pengurus.ts`
- Create: `fsldk-web/src/app/modules/pengurus/services/pengurus-api.service.ts`
- Create: `fsldk-web/src/app/modules/pengurus/repositories/pengurus.repository.ts`

**Interfaces:**
- Consumes: `fsldk-web/src/app/core/services/api.service.ts` — `ApiService.get<T>(path, query?)`, `ApiService.patch<T>(path, body?)` (both already unwrap the `{result}` envelope).
- Produces: `PengurusEntry` interface (`userID: number; fullName: string; email: string; roleName: string; homeOrganizationID: number; homeOrganizationName: string;`); `PengurusApiService` with `list(): Observable<PengurusEntry[]>`, `updateRole(userID: number, roleName: string): Observable<PengurusEntry>`; `PengurusRepository` with the same two methods, same signatures.

No test for this task — direct pass-through with zero branching logic, matching `kader-api.service.ts`/`kader.repository.ts` in this codebase, neither of which has a spec file.

- [ ] **Step 1: Write the entity**

Create `fsldk-web/src/app/modules/pengurus/entities/pengurus.ts`:

```ts
/** Satu baris kader ACTIVE (apa pun role saat ini) dalam scope organisasi
 *  caller — homeOrganizationID/homeOrganizationName SELALU LDK asal
 *  (tidak berubah walau role-nya sudah dipromosikan ke Puskomda/Puskomnas
 *  Verifikator, lihat submission_service_impl.go setPengurusRole di backend). */
export interface PengurusEntry {
  userID: number;
  fullName: string;
  email: string;
  roleName: string;
  homeOrganizationID: number;
  homeOrganizationName: string;
}
```

- [ ] **Step 2: Write the API service**

Create `fsldk-web/src/app/modules/pengurus/services/pengurus-api.service.ts`:

```ts
import { Injectable, inject } from '@angular/core';
import { Observable } from 'rxjs';
import { ApiService } from '../../../core/services/api.service';
import { PengurusEntry } from '../entities/pengurus';

/** Panggilan HTTP mentah untuk fitur Pengurus (/pengurus) — promosi/demosi
 *  role kader aktif dalam scope organisasi caller sendiri. */
@Injectable({ providedIn: 'root' })
export class PengurusApiService {
  private api = inject(ApiService);

  list(): Observable<PengurusEntry[]> { return this.api.get('/pengurus'); }
  updateRole(userID: number, roleName: string): Observable<PengurusEntry> {
    return this.api.patch(`/pengurus/${userID}/role`, { roleName });
  }
}
```

- [ ] **Step 3: Write the repository**

Create `fsldk-web/src/app/modules/pengurus/repositories/pengurus.repository.ts`:

```ts
import { Injectable, inject } from '@angular/core';
import { Observable } from 'rxjs';
import { PengurusApiService } from '../services/pengurus-api.service';
import { PengurusEntry } from '../entities/pengurus';

@Injectable({ providedIn: 'root' })
export class PengurusRepository {
  private api = inject(PengurusApiService);

  list(): Observable<PengurusEntry[]> { return this.api.list(); }
  updateRole(userID: number, roleName: string): Observable<PengurusEntry> { return this.api.updateRole(userID, roleName); }
}
```

- [ ] **Step 4: Verify it compiles**

Run: `cd fsldk-web && npx tsc -p tsconfig.app.json --noEmit`
Expected: no errors.

- [ ] **Step 5: Commit**

```bash
cd fsldk-web
git add src/app/modules/pengurus/entities src/app/modules/pengurus/services src/app/modules/pengurus/repositories
git commit -m "feat(pengurus): add frontend entity, API service, repository"
```

---

### Task 5: Frontend presenter (TDD) + view interface

**Files:**
- Create: `fsldk-web/src/app/modules/pengurus/pages/index/pengurus.index.view.ts`
- Create: `fsldk-web/src/app/modules/pengurus/pages/index/pengurus.index.presenter.ts`
- Test: `fsldk-web/src/app/modules/pengurus/pages/index/pengurus.index.presenter.spec.ts`

**Interfaces:**
- Consumes: `PengurusRepository` (Task 4) — `list()`, `updateRole(userID, roleName)`; `BasePresenter<TView>` (`attachView(view)`, `protected view!: TView`); `ToastService.success(msg)`/`.error(msg)`.
- Produces: `PengurusIndexView` interface (`setItems(items: PengurusEntry[]): void; setLoading(loading: boolean): void; setBusy(userID: number | null): void;`); `PengurusIndexPresenter extends BasePresenter<PengurusIndexView>` with `loadAll(): void` and `updateRole(userID: number, roleName: string): void`.

- [ ] **Step 1: Write the view interface**

Create `fsldk-web/src/app/modules/pengurus/pages/index/pengurus.index.view.ts`:

```ts
import { PengurusEntry } from '../../entities/pengurus';

export interface PengurusIndexView {
  setItems(items: PengurusEntry[]): void;
  setLoading(loading: boolean): void;
  /** userID sedang diproses (dipakai page untuk disable baris itu saja
   *  selagi request jalan), null = tidak ada yang sedang diproses. */
  setBusy(userID: number | null): void;
}
```

- [ ] **Step 2: Write the failing presenter test**

Create `fsldk-web/src/app/modules/pengurus/pages/index/pengurus.index.presenter.spec.ts`:

```ts
import { TestBed } from '@angular/core/testing';
import { of, throwError } from 'rxjs';
import { PengurusIndexPresenter } from './pengurus.index.presenter';
import { PengurusRepository } from '../../repositories/pengurus.repository';
import { ToastService } from '../../../../core/services/toast.service';
import { PengurusIndexView } from './pengurus.index.view';
import { PengurusEntry } from '../../entities/pengurus';

describe('PengurusIndexPresenter', () => {
  let presenter: PengurusIndexPresenter;
  let repo: jasmine.SpyObj<PengurusRepository>;
  let toast: jasmine.SpyObj<ToastService>;
  let view: jasmine.SpyObj<PengurusIndexView>;

  const sample: PengurusEntry[] = [
    { userID: 1, fullName: 'Budi', email: 'budi@example.com', roleName: 'Kader', homeOrganizationID: 10, homeOrganizationName: 'LDK Syahid' },
  ];

  beforeEach(() => {
    repo = jasmine.createSpyObj('PengurusRepository', ['list', 'updateRole']);
    toast = jasmine.createSpyObj('ToastService', ['success', 'error']);

    TestBed.configureTestingModule({
      providers: [
        PengurusIndexPresenter,
        { provide: PengurusRepository, useValue: repo },
        { provide: ToastService, useValue: toast },
      ],
    });
    presenter = TestBed.inject(PengurusIndexPresenter);
    view = jasmine.createSpyObj('PengurusIndexView', ['setItems', 'setLoading', 'setBusy']);
    presenter.attachView(view);
  });

  it('loadAll() populates the view and clears loading on success', () => {
    repo.list.and.returnValue(of(sample));

    presenter.loadAll();

    expect(view.setLoading).toHaveBeenCalledWith(true);
    expect(view.setItems).toHaveBeenCalledWith(sample);
    expect(view.setLoading).toHaveBeenCalledWith(false);
  });

  it('loadAll() clears loading even when the request fails', () => {
    repo.list.and.returnValue(throwError(() => new Error('network')));

    presenter.loadAll();

    expect(view.setItems).not.toHaveBeenCalled();
    expect(view.setLoading).toHaveBeenCalledWith(false);
  });

  it('updateRole() marks the row busy, shows a success toast, and reloads the list on success', () => {
    repo.updateRole.and.returnValue(of(sample[0]));
    repo.list.and.returnValue(of(sample));

    presenter.updateRole(1, 'LDK Admin');

    expect(view.setBusy).toHaveBeenCalledWith(1);
    expect(repo.updateRole).toHaveBeenCalledWith(1, 'LDK Admin');
    expect(toast.success).toHaveBeenCalled();
    expect(repo.list).toHaveBeenCalled();
    expect(view.setBusy).toHaveBeenCalledWith(null);
  });

  it('updateRole() clears busy without a success toast when the request fails', () => {
    repo.updateRole.and.returnValue(throwError(() => new Error('network')));

    presenter.updateRole(1, 'LDK Admin');

    expect(toast.success).not.toHaveBeenCalled();
    expect(view.setBusy).toHaveBeenCalledWith(null);
  });
});
```

- [ ] **Step 3: Run the test to verify it fails**

Run: `cd fsldk-web && npx ng test --watch=false --include='**/pengurus.index.presenter.spec.ts'`
Expected: FAIL — `pengurus.index.presenter` module not found. (If Karma cannot launch a browser in this environment, note that explicitly rather than assuming the test passed — the test file is still correct and will run wherever `ng test` has a launcher configured.)

- [ ] **Step 4: Write the presenter**

Create `fsldk-web/src/app/modules/pengurus/pages/index/pengurus.index.presenter.ts`:

```ts
import { Injectable, inject } from '@angular/core';
import { BasePresenter } from '../../../../core/mvp/base.presenter';
import { ToastService } from '../../../../core/services/toast.service';
import { PengurusRepository } from '../../repositories/pengurus.repository';
import { PengurusIndexView } from './pengurus.index.view';

@Injectable()
export class PengurusIndexPresenter extends BasePresenter<PengurusIndexView> {
  private repo = inject(PengurusRepository);
  private toast = inject(ToastService);

  loadAll(): void {
    this.view.setLoading(true);
    this.repo.list().subscribe({
      next: (items) => { this.view.setItems(items); this.view.setLoading(false); },
      error: () => this.view.setLoading(false),
    });
  }

  updateRole(userID: number, roleName: string): void {
    this.view.setBusy(userID);
    this.repo.updateRole(userID, roleName).subscribe({
      next: () => {
        this.toast.success('Role pengurus berhasil diperbarui');
        this.view.setBusy(null);
        this.loadAll();
      },
      error: () => this.view.setBusy(null),
    });
  }
}
```

- [ ] **Step 5: Run the test to verify it passes**

Run: `cd fsldk-web && npx ng test --watch=false --include='**/pengurus.index.presenter.spec.ts'`
Expected: PASS — all 4 specs green.

- [ ] **Step 6: Commit**

```bash
cd fsldk-web
git add src/app/modules/pengurus/pages/index/pengurus.index.view.ts src/app/modules/pengurus/pages/index/pengurus.index.presenter.ts src/app/modules/pengurus/pages/index/pengurus.index.presenter.spec.ts
git commit -m "feat(pengurus): add index presenter with tests"
```

---

### Task 6: Frontend page component, routing, and 3-tier wiring

**Files:**
- Create: `fsldk-web/src/app/modules/pengurus/pages/index/pengurus.index.page.ts`
- Create: `fsldk-web/src/app/modules/pengurus/pages/index/pengurus.index.page.html`
- Create: `fsldk-web/src/app/modules/pengurus/pengurus.routes.ts`
- Modify: `fsldk-web/src/app/app.routes.ts`

**Interfaces:**
- Consumes: `PengurusIndexPresenter` (Task 5); `fsldk-web/src/app/shared/select.component.ts` — `SelectComponent` (selector `app-select`), `SelectOption` (`{value: unknown; label: string}`), inputs `[options]`, `[ngModel]`/`(ngModelChange)`, `[disabled]`; `fsldk-web/src/app/shared/icon.component.ts` — `IconComponent` (selector `app-icon`, `[name]`/`[size]`); `fsldk-web/src/app/core/services/alert.service.ts` — `AlertService.confirm(message, opts?, event?): Promise<boolean>`; `fsldk-web/src/app/core/guards/guards.ts` — `permissionGuard` (`CanActivateFn`, reads `route.data['permission']`).
- Produces: `PengurusIndexPage` component (standalone, selector `app-pengurus-index-page`); `pengurusRoutes: (permissionCode: string) => Routes`.

- [ ] **Step 1: Write the page component**

Create `fsldk-web/src/app/modules/pengurus/pages/index/pengurus.index.page.ts`:

```ts
import { Component, OnInit, inject, signal } from '@angular/core';
import { FormsModule } from '@angular/forms';
import { ActivatedRoute } from '@angular/router';
import { AlertService } from '../../../../core/services/alert.service';
import { SelectComponent, SelectOption } from '../../../../shared/select.component';
import { IconComponent } from '../../../../shared/icon.component';
import { PengurusEntry } from '../../entities/pengurus';
import { PengurusIndexPresenter } from './pengurus.index.presenter';
import { PengurusIndexView } from './pengurus.index.view';

/** Katalog role per tier — HARUS ikut diubah di backend
 *  (submission_service_impl.go pengurusRolesByTier) bila daftar ini
 *  berubah, pola sama dengan WILDCARD_TIERS/TIER_REQUIRED_ROLE di
 *  user.index.page.ts. Enforcement sesungguhnya tetap di backend — daftar
 *  ini murni supaya dropdown tidak menampilkan opsi yang pasti ditolak
 *  server. */
const PENGURUS_ROLES_BY_TIER: Record<string, SelectOption[]> = {
  LDK: [
    { value: 'Kader', label: 'Kader' },
    { value: 'LDK Admin', label: 'LDK Admin' },
  ],
  PUSKOMDA: [
    { value: 'Kader', label: 'Kader' },
    { value: 'LDK Admin', label: 'LDK Admin' },
    { value: 'Puskomda Verifikator', label: 'Puskomda Verifikator' },
  ],
  PUSKOMNAS: [
    { value: 'Kader', label: 'Kader' },
    { value: 'LDK Admin', label: 'LDK Admin' },
    { value: 'Puskomda Verifikator', label: 'Puskomda Verifikator' },
    { value: 'Puskomnas Verifikator', label: 'Puskomnas Verifikator' },
  ],
};

@Component({
  selector: 'app-pengurus-index-page',
  standalone: true,
  templateUrl: './pengurus.index.page.html',
  imports: [FormsModule, SelectComponent, IconComponent],
  providers: [PengurusIndexPresenter],
  styles: [`
    .page-head { margin-bottom: 20px; } .page-head h1 { margin-bottom: 2px; }
    .pengurus-table { width: 100%; border-collapse: collapse; }
    .pengurus-table th { text-align: left; padding: 10px 12px; font-size: .8rem; text-transform: uppercase; letter-spacing: .04em; color: var(--color-muted); border-bottom: 1px solid var(--color-border); }
    .pengurus-table td { padding: 10px 12px; border-bottom: 1px solid var(--color-border); vertical-align: middle; }
    .role-cell { max-width: 220px; }
    .empty-state-inline { padding: 36px 20px 24px; }
  `],
})
export class PengurusIndexPage implements OnInit, PengurusIndexView {
  private presenter = inject(PengurusIndexPresenter);
  private route = inject(ActivatedRoute);
  private alert = inject(AlertService);

  items = signal<PengurusEntry[]>([]);
  loading = signal(true);
  busyUserID = signal<number | null>(null);
  /** Dibump setiap kali onRoleChange() berjalan (confirm ATAU cancel) —
   *  dipakai sebagai bagian dari @for's track supaya app-select tiap baris
   *  dipaksa remount dan membaca ulang [ngModel] dari item.roleName.
   *  Perlu: SelectComponent.choose() langsung menyetel state internalnya
   *  SEBELUM konfirmasi dialog ditampilkan (lihat select.component.ts),
   *  jadi kalau pengguna membatalkan konfirmasi, dropdown akan tetap
   *  menampilkan pilihan yang dibatalkan itu kecuali kita paksa resync. */
  resyncTick = signal(0);

  roleOptions: SelectOption[] = PENGURUS_ROLES_BY_TIER[(this.route.snapshot.data['tier'] as string) ?? 'LDK'] ?? [];

  ngOnInit(): void {
    this.presenter.attachView(this);
    this.presenter.loadAll();
  }

  rowKey(item: PengurusEntry): string {
    return `${item.userID}_${this.resyncTick()}`;
  }

  async onRoleChange(item: PengurusEntry, newRole: unknown, event?: Event): Promise<void> {
    const roleName = newRole as string;
    if (roleName === item.roleName) return;
    const ok = await this.alert.confirm(
      `Ubah role "${item.fullName}" dari ${item.roleName} menjadi ${roleName}?`,
      { title: 'Ubah Role Pengurus', confirmLabel: 'Ya, Ubah' }, event,
    );
    this.resyncTick.update((n) => n + 1);
    if (!ok) return;
    this.presenter.updateRole(item.userID, roleName);
  }

  setItems(items: PengurusEntry[]): void { this.items.set(items); }
  setLoading(loading: boolean): void { this.loading.set(loading); }
  setBusy(userID: number | null): void { this.busyUserID.set(userID); }
}
```

- [ ] **Step 2: Write the page template**

Create `fsldk-web/src/app/modules/pengurus/pages/index/pengurus.index.page.html`:

```html
<div class="page-head">
  <h1>Pengurus</h1>
  <p class="text-muted">Kelola role pengurus (promosi/demosi) untuk kader aktif di organisasi Anda.</p>
</div>

@if (loading()) {
  <div class="card card-pad"><span class="skel skel-line" style="width:60%;height:20px"></span></div>
} @else if (items().length === 0) {
  <div class="card card-pad">
    <div class="empty-state empty-state-inline">
      <span class="icon-badge lg icon-badge-soft"><app-icon name="id-card" [size]="26" /></span>
      <h4>Belum ada kader aktif</h4>
      <p class="text-muted">Kader aktif di organisasi Anda akan muncul di sini.</p>
    </div>
  </div>
} @else {
  <div class="card card-pad">
    <table class="pengurus-table">
      <thead>
        <tr>
          <th>Nama</th>
          <th>Email</th>
          <th>LDK Asal</th>
          <th>Role Saat Ini</th>
          <th class="role-cell">Aksi</th>
        </tr>
      </thead>
      <tbody>
        @for (item of items(); track rowKey(item)) {
          <tr>
            <td>{{ item.fullName }}</td>
            <td class="text-muted">{{ item.email }}</td>
            <td>{{ item.homeOrganizationName }}</td>
            <td>{{ item.roleName }}</td>
            <td class="role-cell">
              <app-select
                [options]="roleOptions"
                [ngModel]="item.roleName"
                (ngModelChange)="onRoleChange(item, $event, $event)"
                [disabled]="busyUserID() === item.userID" />
            </td>
          </tr>
        }
      </tbody>
    </table>
  </div>
}
```

- [ ] **Step 3: Write the routes**

Create `fsldk-web/src/app/modules/pengurus/pengurus.routes.ts`:

```ts
import { Routes } from '@angular/router';
import { permissionGuard } from '../../core/guards/guards';

/** Rute modul Pengurus — dipasang sebagai children dari CmsLayoutComponent
 *  di 3 tier (cms-ldk/cms-puskomda/cms-puskomnas), TIDAK PERNAH cms (Portal
 *  Admin). Beda dari organizationRoutes(): factory ini menerima
 *  permissionCode sebagai parameter karena permissionGuard hanya membaca
 *  SATU route.data['permission'], dan tiap tier butuh kode izin berbeda
 *  (pengurus.ldk.manage / pengurus.puskomda.manage / pengurus.puskomnas.manage). */
export const pengurusRoutes: (permissionCode: string) => Routes = (permissionCode) => [
  {
    path: 'pengurus',
    canActivate: [permissionGuard],
    data: { permission: permissionCode },
    title: 'Pengurus',
    loadComponent: () => import('./pages/index/pengurus.index.page').then((m) => m.PengurusIndexPage),
  },
];
```

- [ ] **Step 4: Wire the route into all 3 tiers in `app.routes.ts`**

In `fsldk-web/src/app/app.routes.ts`, find this exact import line:

```ts
import { organizationRoutes } from './modules/organization/organization.routes';
```

Replace it with:

```ts
import { organizationRoutes } from './modules/organization/organization.routes';
import { pengurusRoutes } from './modules/pengurus/pengurus.routes';
```

Find this exact block (`cms-ldk` children):

```ts
    path: 'cms-ldk',
    component: CmsLayoutComponent,
    canActivate: [authGuard],
    data: { tier: 'LDK' },
    children: [
      { path: '', pathMatch: 'full', redirectTo: 'dashboard' },
      ...dashboardRoutes(),
      ...organizationRoutes(),
      ...submissionRoutes(),
    ],
  },
```

Replace it with:

```ts
    path: 'cms-ldk',
    component: CmsLayoutComponent,
    canActivate: [authGuard],
    data: { tier: 'LDK' },
    children: [
      { path: '', pathMatch: 'full', redirectTo: 'dashboard' },
      ...dashboardRoutes(),
      ...organizationRoutes(),
      ...submissionRoutes(),
      ...pengurusRoutes('pengurus.ldk.manage'),
    ],
  },
```

Find this exact block (`cms-puskomda` children):

```ts
    path: 'cms-puskomda',
    component: CmsLayoutComponent,
    canActivate: [authGuard],
    data: { tier: 'PUSKOMDA' },
    children: [
      { path: '', pathMatch: 'full', redirectTo: 'dashboard' },
      ...dashboardRoutes(),
      ...organizationRoutes(),
      ...submissionRoutes(),
      ...reportRoutes(),
    ],
  },
```

Replace it with:

```ts
    path: 'cms-puskomda',
    component: CmsLayoutComponent,
    canActivate: [authGuard],
    data: { tier: 'PUSKOMDA' },
    children: [
      { path: '', pathMatch: 'full', redirectTo: 'dashboard' },
      ...dashboardRoutes(),
      ...organizationRoutes(),
      ...submissionRoutes(),
      ...reportRoutes(),
      ...pengurusRoutes('pengurus.puskomda.manage'),
    ],
  },
```

Find this exact block (`cms-puskomnas` children):

```ts
    path: 'cms-puskomnas',
    component: CmsLayoutComponent,
    canActivate: [authGuard],
    data: { tier: 'PUSKOMNAS' },
    children: [
      { path: '', pathMatch: 'full', redirectTo: 'dashboard' },
      ...dashboardRoutes(),
      ...organizationRoutes(),
      ...submissionRoutes(),
      ...reportRoutes(),
      ...submissionFormRoutes(),
    ],
  },
```

Replace it with:

```ts
    path: 'cms-puskomnas',
    component: CmsLayoutComponent,
    canActivate: [authGuard],
    data: { tier: 'PUSKOMNAS' },
    children: [
      { path: '', pathMatch: 'full', redirectTo: 'dashboard' },
      ...dashboardRoutes(),
      ...organizationRoutes(),
      ...submissionRoutes(),
      ...reportRoutes(),
      ...submissionFormRoutes(),
      ...pengurusRoutes('pengurus.puskomnas.manage'),
    ],
  },
```

- [ ] **Step 5: Verify the frontend builds**

Run: `cd fsldk-web && npx ng build`
Expected: `Application bundle generation complete`, no new errors (pre-existing bundle-budget warnings on unrelated files are fine).

- [ ] **Step 6: Commit**

```bash
cd fsldk-web
git add src/app/modules/pengurus/pages/index/pengurus.index.page.ts src/app/modules/pengurus/pages/index/pengurus.index.page.html src/app/modules/pengurus/pengurus.routes.ts src/app/app.routes.ts
git commit -m "feat(pengurus): add index page and wire /pengurus route into 3 CMS tiers"
```

---

### Task 7: Manual end-to-end verification across all 3 tiers

This is the part that genuinely needs a real browser and real accounts in each of the 3 roles — role/org transitions, the deactivate cascade, and the dropdown resync-on-cancel behavior (Task 6 Step 1's `resyncTick` workaround) cannot be confirmed by `go build`/`ng build` alone.

- [ ] **Step 1: Final full build + test pass, both repos**

Run: `cd fsldk-api && go build ./... && go vet ./... && go test ./...`
Expected: exit code 0, all tests PASS.

Run: `cd fsldk-web && npx tsc -p tsconfig.app.json --noEmit && npx ng build`
Expected: no type errors; `Application bundle generation complete`.

- [ ] **Step 2: Start both servers**

`cd fsldk-api && go run .` — confirm the boot log shows migration `0051_pengurus_menu` applied (or check `schema_migrations` in MySQL).
`cd fsldk-web && npm start`.

- [ ] **Step 3: Seed/confirm 3 test accounts exist**

One active Kader each in an LDK, that LDK's parent Puskomda, and Puskomnas — i.e. you need at minimum: one LDK Admin account, one Puskomda Verifikator account (for a Puskomda with at least one child LDK that has an active Kader), one Puskomnas Verifikator account, and at least one plain active Kader under that LDK. Use the existing Kader Persetujuan flow (`/cms-ldk/kaders/persetujuan`) to get a Kader to ACTIVE status if none exists yet.

- [ ] **Step 4: Portal LDK — verify the 2-role catalog and promote/demote**

Log in as the LDK Admin. Confirm **Pengurus** appears in the sidebar at `/cms-ldk/pengurus` (and does NOT appear for an account without `pengurus.ldk.manage`). Open it, confirm the table shows the active Kader(s) of that LDK only. In the Aksi dropdown for a Kader row, confirm only **Kader** and **LDK Admin** are offered (not Puskomda/Puskomnas Verifikator). Select **LDK Admin**, confirm the confirmation dialog text, confirm — toast success, row's Role Saat Ini updates to LDK Admin after reload. Try selecting a role on your OWN row (if you appear in the list) — the backend must reject with 403; confirm the UI surfaces an error (not a silent success).

- [ ] **Step 5: Portal LDK — verify dropdown cancel does not leave stale UI**

Pick a row, open its Aksi dropdown, select a different role, then click **Batal** on the confirmation dialog. Confirm the dropdown's displayed value reverts to the row's actual current role (not stuck showing the cancelled selection) — this exercises the `resyncTick` workaround from Task 6.

- [ ] **Step 6: Portal Puskomda — verify the 3-role catalog and cross-LDK promotion**

Log in as the Puskomda Verifikator. Open `/cms-puskomda/pengurus`, confirm the table shows active Kader across ALL LDKs under that Puskomda (not just one), with a correct **LDK Asal** column per row. Confirm the Aksi dropdown offers Kader/LDK Admin/Puskomda Verifikator (not Puskomnas Verifikator). Promote a Kader to **Puskomda Verifikator** — after the page reloads, log in as THAT promoted account separately (or check its token-forced-relogin behavior) and confirm it can now access `/cms-puskomda/...` pages generally (proving `organizationID` really was reassigned to the Puskomda, not just the role label changing).

- [ ] **Step 7: Portal Puskomnas — verify the 4-role catalog and the Puskomnas-root reassignment**

Log in as the Puskomnas Verifikator. Open `/cms-puskomnas/pengurus`, confirm the table shows active Kader nationally. Confirm the Aksi dropdown offers all 4 roles. Promote a Kader straight to **Puskomnas Verifikator** (skipping LDK Admin/Puskomda Verifikator) — confirm it succeeds in one step, and that the promoted account's effective organization is now the single Puskomnas org.

- [ ] **Step 8: Verify the deactivate cascade**

From `/cms-ldk/kaders/persetujuan` (or the Puskomda/Puskomnas equivalent), find a Kader who is CURRENTLY an LDK Admin (promoted in Step 4/6/7) and click **Nonaktifkan**. Confirm the Kader's status becomes INACTIVE as usual. Then go back to the Pengurus page (or Manajemen Pengguna in Portal Admin) and confirm that account's role has automatically reverted to **Kader** — not left stuck as LDK Admin on a now-inactive account.

- [ ] **Step 9: Verify Portal Admin is unaffected**

Log in as Super Admin, confirm **Pengurus** does NOT appear anywhere in Portal Admin's sidebar (`/cms/...`), and that Manajemen Pengguna (`/cms/users`) still works exactly as before — this feature adds a new surface, it does not touch the existing Super-Admin-only flow.
