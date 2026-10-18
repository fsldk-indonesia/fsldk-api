# Pengurus Menu (role self-promotion within own org) — Design

Status: approved by user, pending implementation plan.
Spans both repos: `fsldk-api` (backend) and `fsldk-web` (frontend). This
file lives in `fsldk-api/docs/` because the role-transition rules and
cross-module wiring are the backend's responsibility and the hardest part
to get right; `fsldk-web`'s implementation plan should link back here.

## Background

Today, turning an active Kader into an LDK Admin (or Puskomda/Puskomnas
Verifikator) requires a Super Admin to go through Portal Admin's
Manajemen Pengguna (`modules/user`, CMS-only, global CRUD). LDK/Puskomda/
Puskomnas staff who want to promote one of their own active members have
no self-service way to do this — they have to ask a Super Admin.

The ask: a new **"Pengurus"** menu, present in Portal LDK, Portal Puskomda,
and Portal Puskomnas (never Portal Admin), showing a table of active
Kader within the caller's own organization scope, with a per-row action
to change that person's role directly — promote toward the tier's
admin/verifikator role, or demote back down, including to plain Kader.
Deactivating a Kader who currently holds a promoted role must
automatically revert them to plain Kader.

## Scope decisions (confirmed with user)

1. **One table, both directions.** The Pengurus table lists every user
   with an **ACTIVE** `ms_kader` record in the caller's scope — whether
   they currently hold role Kader or an already-promoted role (LDK Admin
   etc.). The per-row role picker handles promote and demote in the same
   place, matching "ubah dari Kader ke LDK Admin atau sebaliknya."
2. **Role catalog per tier** (caller's own role, from their own
   `organizationTypeCode`, determines the allow-list — not a config
   table):
   - Portal LDK (LDK Admin): Kader, LDK Admin
   - Portal Puskomda (Puskomda Verifikator): Kader, LDK Admin, Puskomda
     Verifikator
   - Portal Puskomnas (Puskomnas Verifikator): Kader, LDK Admin, Puskomda
     Verifikator, Puskomnas Verifikator
   - Jumping directly between any two roles in the allow-list is allowed
     (e.g. Puskomnas caller can promote a Kader straight to Puskomda
     Verifikator without an intermediate LDK Admin step).
3. **`organizationID` always follows the *target* role**, regardless of
   direction (promote, demote, or the deactivate-cascade):
   - Kader or LDK Admin → the user's home LDK (`ms_kader.organizationID`,
     which never changes)
   - Puskomda Verifikator → that home LDK's parent Puskomda
     (`parentOrganizationID`)
   - Puskomnas Verifikator → the single Puskomnas root org
     (`RootPuskomnasID`, already exists in `organization_repository`)

   This is necessary, not a style choice: `ms_user.organizationTypeCode`
   is never a stored column, it's always joined live from
   `ms_organization.organizationTypeCode` through `organizationID`
   (`user_repository_impl.go:19`) and baked into the JWT at login
   (`middlewares/middleware.go:80`). Every access-scope branch in the
   backend (`organization_service`, `submission_service`, `user_service`)
   keys off that field, so a Puskomda Verifikator whose `organizationID`
   still pointed at their home LDK would resolve as an **LDK**-tier
   caller everywhere else in the system, despite the role name. The
   user's true home LDK is never lost — it stays on record permanently
   in `ms_kader.organizationID`, independent of what `ms_user.organizationID`
   is doing.
4. **Deactivate cascade lives in the existing flow, not a new button.**
   The "Nonaktifkan" action already exists at Kader Persetujuan
   (`DeactivateKader`, `submission_service_impl.go:1483-1506`) and today
   only flips `ms_kader.status` to `INACTIVE`. This design adds: if the
   target's current role is LDK Admin / Puskomda Verifikator / Puskomnas
   Verifikator, auto-revert them to Kader (and `organizationID` back to
   their home LDK) as part of that same call — reusing the identical
   role-reassignment function the promote/demote endpoint uses. No new
   "Nonaktifkan" button is added to the Pengurus page itself.
5. **Self-action is blocked.** A caller can never promote, demote, or
   (via the existing cascade) deactivate their own account through this
   feature — guards against accidental self-lockout.
6. **Design doc location**: single combined doc here in `fsldk-api/docs/`
   (user's explicit choice over splitting one doc per repo), because the
   role-transition rules are the part most likely to be gotten wrong and
   backend is where they live.

## Why implement entirely within `modules/submission`, not `modules/user`

Earlier draft of this doc said "extend `modules/user`," reasoning that the
core mutation (`roleID`/`organizationID`) is that module's data. Deeper
research while writing the implementation plan found this doesn't hold up:
`submission_service.ServiceImpl` **already has direct `userRepo
user_repository.Repository` and `roleRepo role_repository.Repository`
dependencies** (`submission_service_impl.go:44-62`), used today by the
existing `promoteToKaderRole` function — fetch user → look up roleID by
name → `userRepo.Update(...)`, the exact shape this feature needs. It also
already has `repo.ListKadersByOrganizations(ctx, organizationIDs, status)`
and `orgScope.AccessibleOrganizationIDs(...)` — the two calls needed to
list "active kader in my scope."

This means the whole feature fits inside `modules/submission` reusing
dependencies that are **already wired in `router.go`** — zero new
cross-module interfaces in either direction, and zero risk of the circular
construction problem the original two-interface design would have hit
(`user_service` needing `submission_service`'s data while
`submission_service` needs `user_service` for the cascade — neither could
be constructed first without two-phase setter injection, a pattern this
codebase has no precedent for). Simpler and lower-risk wins.

The **frontend** module placement decision is unaffected by this — it's
about UX/audience, not backend module boundaries. `modules/pengurus/`
(frontend) still calls a `/pengurus` endpoint that now happens to be
registered by `modules/submission/router.go` instead of
`modules/user/router.go`.

## Backend (fsldk-api) — new methods on `submission_service`

No new cross-module interfaces. `submission_dto` gets one new read shape
and one new request shape:

```go
// PengurusEntry adalah satu baris kader ACTIVE (apa pun role ms_user saat
// ini) dalam scope organisasi caller — dipakai fitur Pengurus (promosi/
// demosi role tanpa lewat Portal Admin).
type PengurusEntry struct {
	UserID               int64  `json:"userID"`
	FullName             string `json:"fullName"`
	Email                string `json:"email"`
	RoleName             string `json:"roleName"`
	HomeOrganizationID   int64  `json:"homeOrganizationID"`   // ms_kader.organizationID — never changes
	HomeOrganizationName string `json:"homeOrganizationName"`
}

type UpdatePengurusRoleRequest struct {
	RoleName string `json:"roleName" validate:"required"`
}
```

`UpdatePengurusRoleRequest` deliberately has no `oneof` validator — the
valid set isn't static (it depends on the caller's own tier), so real
validation happens in the service against a caller-tier-derived allow-list,
not a fixed DTO-level list.

### `ListPengurus` (service method backing `GET /pengurus`)

1. `orgIDs := orgScope.AccessibleOrganizationIDs(ctx, caller.OrganizationID, caller.OrganizationTypeCode, caller.WildcardTierAccess)` —
   existing cascade (LDK: self; Puskomda: self + children; Puskomnas: all).
2. `kaders := repo.ListKadersByOrganizations(ctx, orgIDs, constants.KaderStatusActive)` —
   existing repository method, already used elsewhere in this file.
3. For each Kader row, enrich with `userRepo.FindByID(kader.UserID)` (role
   name, email) and `orgRepo.FindByID(kader.OrganizationID)` (LDK name),
   both already-injected dependencies. Rows whose user/org lookup fails
   are skipped (defensive against a dangling FK during a migration window)
   rather than failing the whole list.

No org-switcher narrowing in v1 (`caller.RequestedOrganizationID` is not
consulted) — the page always shows the caller's **full** accessible scope.
YAGNI; can be added later if asked.

### `UpdatePengurusRole` (service method backing `PATCH /pengurus/:userID/role`)

1. Reject with `Forbidden` if `targetUserID == caller.UserID`.
2. Reject with `Forbidden` if `req.RoleName` isn't in the allow-list for
   `caller.OrganizationTypeCode` (table in Scope decision #2) — a package-
   level `map[string][]string` (`pengurusRolesByTier`), hard server-side
   check, not just a UI constraint.
3. Resolve the same `orgIDs`/`ListKadersByOrganizations` call `ListPengurus`
   uses, and find the row where `UserID == targetUserID` — `NotFound` if
   absent (covers both "not an active Kader" and "outside caller's scope"
   in one check, same pattern `DeactivateKader`/`ReactivateKader` already
   use).
4. Call `setPengurusRole(ctx, targetUserID, homeOrganizationID, req.RoleName, caller.UserID)` —
   see below.
5. Return the refreshed `PengurusEntry` for that user.

**`setPengurusRole`** (new unexported method, the single mutation core
reused by both the endpoint above and the deactivate cascade below):

1. Compute the new `organizationID` from the *target* `roleName` (Scope
   decision #3):
   - Kader / LDK Admin → `homeOrganizationID` unchanged
   - Puskomda Verifikator → `orgRepo.FindByID(homeOrganizationID).ParentOrganizationID`
   - Puskomnas Verifikator → `orgRepo.RootPuskomnasID(ctx)` (already exists,
     `organization_repository_impl.go:139-147`)
2. `roleID := roleRepo.IDByName(ctx, roleName)` (already exists, used by
   `promoteToKaderRole` today).
3. `userRepo.Update(ctx, userID, u.FullName, u.Email, roleID, u.IsActive, newOrgID, u.WildcardTierAccess, actorID)` —
   existing repository method, preserving every field except role/org.
4. `userRepo.BumpTokenVersion(ctx, userID)` — existing mechanism, already
   used by Manajemen Pengguna's role/org changes, forces re-login so the
   account's menu/permissions refresh.

### Deactivate cascade — modify the existing `DeactivateKader`

After `repo.UpdateKaderStatus(ctx, kaderID, constants.KaderStatusInactive, ...)`
succeeds, fetch the Kader's current user (`userRepo.FindByID(k.UserID)`);
if their role is LDK Admin / Puskomda Verifikator / Puskomnas Verifikator,
call `setPengurusRole(ctx, k.UserID, k.OrganizationID, roleNameKader, caller.UserID)`
— the exact same mutation core `UpdatePengurusRole` uses, so the two call
sites can never drift apart. Best-effort: a failure here does not roll
back the Kader deactivation itself, matching the precedent already set by
`promoteToKaderRole`'s own doc comment ("Kegagalan di sini tidak
membatalkan... dicatat, bukan menggagalkan transaksi").

### Migration `0051_pengurus_menu.up.sql`

```sql
INSERT IGNORE INTO lk_permission (permissionCode, permissionName, moduleName, menuLabel, menuIcon, menuRoute, sortOrder) VALUES
('pengurus.ldk.manage',       'Kelola Pengurus LDK',       'submission', 'Pengurus', 'users-round', '/cms-ldk/pengurus', 15),
('pengurus.puskomda.manage',  'Kelola Pengurus Puskomda',  'submission', 'Pengurus', 'users-round', '/cms-puskomda/pengurus', 15),
('pengurus.puskomnas.manage', 'Kelola Pengurus Puskomnas', 'submission', 'Pengurus', 'users-round', '/cms-puskomnas/pengurus', 15);

INSERT IGNORE INTO map_role_permission (roleID, permissionID)
SELECT r.roleID, p.permissionID FROM ms_role r JOIN lk_permission p
WHERE r.roleName = 'LDK Admin' AND p.permissionCode = 'pengurus.ldk.manage';

INSERT IGNORE INTO map_role_permission (roleID, permissionID)
SELECT r.roleID, p.permissionID FROM ms_role r JOIN lk_permission p
WHERE r.roleName = 'Puskomda Verifikator' AND p.permissionCode = 'pengurus.puskomda.manage';

INSERT IGNORE INTO map_role_permission (roleID, permissionID)
SELECT r.roleID, p.permissionID FROM ms_role r JOIN lk_permission p
WHERE r.roleName = 'Puskomnas Verifikator' AND p.permissionCode = 'pengurus.puskomnas.manage';
```

Three separate `permissionCode` rows (not one shared code) because
`permissionCode` is unique and each menu row needs its own `menuRoute` —
mirrors the existing per-shell split done for other CMS menus in
`migrations/0010_cms_shell_menu_routes.up.sql`. Both backend API routes
(list + update) are guarded with `mw.RequirePermission(PermPengurusLDKManage,
PermPengurusPuskomdaManage, PermPengurusPuskomnasManage)` (existing
variadic any-match) — one route pair serves all three tiers; only the
*menu* is split three ways.

### Routing

New route GROUP added to `modules/submission/router.go` (`RegisterRoutes`),
as a sibling of the existing `/submissions` and `/kaders` groups — **not**
nested under `/kaders`, because Gin's router (httprouter-based) cannot have
a static segment and a wildcard param at the same path depth, and `/kaders`
already has `:id` at that depth:
```go
p := rg.Group("/pengurus")
p.Use(mw.Auth())
{
	p.GET("", mw.RequirePermission(constants.PermPengurusLDKManage, constants.PermPengurusPuskomdaManage, constants.PermPengurusPuskomnasManage), h.ListPengurus)
	p.PATCH("/:userID/role", mw.RequirePermission(constants.PermPengurusLDKManage, constants.PermPengurusPuskomdaManage, constants.PermPengurusPuskomnasManage), h.UpdatePengurusRole)
}
```
No root `router.go` changes needed — `submission.RegisterRoutes(api, subH, mw)`
is already called there, and `submission_service.NewService`'s signature is
unchanged (same 6 existing dependencies).

## Frontend CMS (fsldk-web) — new module `modules/pengurus/`

```
modules/pengurus/
├── entities/pengurus.ts               # PengurusEntry interface (plain)
├── services/pengurus-api.service.ts
├── repositories/pengurus.repository.ts
├── pages/index/
│   ├── pengurus.index.page.ts / .html
│   ├── pengurus.index.presenter.ts
│   └── pengurus.index.view.ts
└── pengurus.routes.ts                  # factory, mounted in 3 tiers
```

`pengurus.routes.ts` follows the shape of `organization.routes.ts` (a
factory spread into multiple tiers' `children` arrays), but — unlike
`organizationRoutes()` — it takes the tier's permission code as a
parameter, because `permissionGuard` only reads a single
`route.data['permission']` string and each tier needs a *different* code
checked:
```ts
export const pengurusRoutes: (permissionCode: string) => Routes = (permissionCode) => [
  { path: 'pengurus', canActivate: [permissionGuard],
    data: { permission: permissionCode },
    loadComponent: () => import('./pages/index/pengurus.index.page').then(m => m.PengurusIndexPage) },
];
```
Each of the three `app.routes.ts` call sites passes its own code —
`...pengurusRoutes('pengurus.ldk.manage')` under `cms-ldk`,
`...pengurusRoutes('pengurus.puskomda.manage')` under `cms-puskomda`,
`...pengurusRoutes('pengurus.puskomnas.manage')` under `cms-puskomnas` —
**never** under `cms`.

**Page**: a plain list (not `app-cms-index` — no bulk-delete/column-
visibility need here), modeled on `submission.kader-persetujuan.page.ts`'s
lightweight pattern. Columns: Nama, Email, LDK Asal, Role Saat Ini, Aksi.
Data source is `PengurusRepository.list()` — no arguments; per the
`ListPengurus` v1 scope decision (no org-switcher narrowing), the backend
resolves the caller's full scope from the JWT alone, so the page doesn't
need `OrgContextService`/`ActivatedRoute` for data fetching (it still reads
`route.snapshot.data['tier']` separately, synchronously, just to pick which
hardcoded role catalog to show in the dropdown).

**Aksi column**: `app-select` (small, non-searchable) per row, pre-filled
with the current role, options restricted client-side to the tier's
catalog (Scope decision #2 — a UX convenience; the real enforcement is
server-side in step 1 of `UpdatePengurusRole`). On change:
`AlertService.confirm()` ("Ubah role {nama} dari {lama} jadi {baru}?") →
if confirmed, `PengurusRepository.updateRole(userID, roleName)` → on
success, toast + reload the list.

## Edge cases & validation summary

- Target not found in caller's accessible active-Kader set → `404`
  (covers "not a Kader," "Kader not ACTIVE," and "outside my org scope"
  uniformly, consistent with `DeactivateKader`/`ReactivateKader`'s
  existing scope-check pattern).
- Requested `roleName` outside caller's tier allow-list → `403`, checked
  server-side regardless of what the frontend dropdown shows.
- `targetUserID == caller.UserID` → `403`.
- The `DeactivateKader` cascade only calls `setPengurusRole` when the
  target's current role is one of the three promoted roles — a plain
  Kader being deactivated never touches `ms_user` at all.
- Any successful role/org change bumps the affected account's token
  version, exactly like Manajemen Pengguna already does — the promoted/
  demoted user is forced to re-login and sees the updated menu only
  after that.

## Testing

- Backend: a pure-function table test for the `pengurusRolesByTier`
  allow-list check (all 3 caller tiers × 4 target roles, including
  rejected combinations) — no fakes needed. A second test, using hand-
  written fakes for the 6 `submission_service.ServiceImpl` dependencies
  (precedent: `user_service_impl_test.go`'s `fakeUserRepository`/
  `fakeRoleRepository`), covering `UpdatePengurusRole`'s self-action
  block, out-of-scope target case, and `organizationID` reassignment for
  all three target-role categories; plus `DeactivateKader`'s cascade,
  asserting both the no-op case (target already Kader) and the revert
  case (role + organizationID both change).
- Frontend: presenter test for `PengurusIndexPresenter` (load →
  role-change happy path, confirm-then-act), and one per-tier smoke test
  that the role dropdown options differ (2 vs 3 vs 4 entries).

## Out of scope (YAGNI, can be added later without reshaping this design)

- Bulk promote/demote (one row at a time only, matching the existing
  Kader Persetujuan pattern).
- A capacity cap on how many LDK Admin / Verifikator a single org can
  have — unconstrained, matching the system's existing lack of such caps.
- Audit log of who promoted/demoted whom beyond the existing
  `updatedBy`/`updatedDate` columns already on `ms_user`.
- Letting wildcard-tier-access accounts (e.g. a Puskomnas Verifikator
  with LDK wildcard access) use a *lower* tier's Pengurus page — the
  three permission codes are granted to exactly one role each; extending
  this is a seed-data change, not a code change, if ever needed.
