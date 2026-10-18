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

## Why extend `modules/user`, not a new backend module

The core mutation is `ms_user.roleID` + `organizationID` — that's
`modules/user`'s data, not a new domain. A brand new module would own no
table of its own (no new schema beyond permission seed rows) and would
just be a thin wrapper around `user_repository.Update` plus a read into
`modules/submission`'s `ms_kader` — not enough independent shape to
justify the standard module scaffold. Two new narrow cross-module
interfaces (below) do the actual boundary work, following the same
pattern already established by `OrgScopeChecker`/`OrgScopeResolver`
(interfaces defined in the consuming module, implemented by the owning
module, wired manually in root `router.go` — no new DI mechanism).

The **frontend**, by contrast, gets its own new module (`modules/pengurus/`)
— the audience and access model (org-scoped, non-Super-Admin,
promote/demote only) is different enough from Manajemen Pengguna
(Super-Admin global CRUD) that bolting new pages onto `modules/user`'s
frontend structure would blur two genuinely different features.

## Backend (fsldk-api)

### New cross-module interfaces

```go
// modules/user/user_service/user_service.go — consumer-defined, like OrgScopeChecker.

// PengurusCandidate is one row eligible for the Pengurus table: an ACTIVE
// Kader (whatever their current role), enriched with display fields.
type PengurusCandidate struct {
	UserID               int64
	FullName             string
	Email                string
	RoleID               int64
	RoleName             string
	HomeOrganizationID   int64  // ms_kader.organizationID — never changes
	HomeOrganizationName string
}

// KaderScopeReader lets user_service list/validate Pengurus candidates
// without importing modules/submission directly. Implemented by
// submission_service, wired manually in router.go.
type KaderScopeReader interface {
	ActivePengurusCandidates(ctx context.Context, organizationIDs []int64) ([]PengurusCandidate, error)
}
```

```go
// modules/submission/submission_service/submission_service.go — consumer-defined.

// UserRoleDemoter lets submission_service revert a deactivated Kader's
// role without importing modules/user directly. Implemented by
// user_service, wired manually in router.go.
type UserRoleDemoter interface {
	// RevertToKaderRole is a no-op if the user's current role isn't one
	// of LDK Admin / Puskomda Verifikator / Puskomnas Verifikator.
	RevertToKaderRole(ctx context.Context, userID int64) error
}
```

`ActivePengurusCandidates` backs onto one new repository query in
`modules/submission` (owns `ms_kader`):

```sql
SELECT u.userID, u.fullName, u.email, u.roleID, r.roleName,
       k.organizationID AS homeOrganizationID, o.organizationName AS homeOrganizationName
FROM ms_kader k
JOIN ms_user u ON u.userID = k.userID
JOIN ms_role r ON r.roleID = u.roleID
JOIN ms_organization o ON o.organizationID = k.organizationID
WHERE k.status = 'ACTIVE' AND k.organizationID IN (?)
ORDER BY u.fullName ASC
```

Filtering by `k.organizationID` (home LDK) rather than `u.organizationID`
is deliberate: it stays correct regardless of whether a row's current
`organizationID` has been reassigned by a prior promotion, because a home
LDK that's accessible to the caller is always a subset of (or equal to)
the caller's accessible scope by construction of the org hierarchy —
unlike `u.organizationID`, which only happens to agree after a Puskomda/
Puskomnas promotion because the reassigned org is the caller's own.

### `ListPengurus` (service method backing `GET`)

1. Resolve `caller.AccessibleOrganizationIDs` (existing `OrgScopeChecker`
   cascade — LDK: self; Puskomda: self + children; Puskomnas: all).
2. Call `kaderScope.ActivePengurusCandidates(ctx, accessibleOrgIDs)`.
3. Return as-is — no extra filtering needed, the query already scopes to
   accessible home LDKs.

### `UpdatePengurusRole` (service method backing `PATCH`)

Request body: `{"roleName": "LDK Admin"}` — reuses existing role-name
strings as the contract (matches how the rest of the codebase already
refers to roles by name, e.g. `constants.RoleLDKAdmin`), no new enum.

1. Derive caller's allow-list from `caller.OrganizationTypeCode` (table in
   Scope decision #2). Reject with `Forbidden` if `roleName` isn't in it
   — this is a hard server-side allow-list, not just a UI constraint.
2. Reject with `Forbidden` if `targetUserID == caller.UserID`.
3. Call `kaderScope.ActivePengurusCandidates` scoped to caller's
   accessible org IDs (same call `ListPengurus` uses) and find the target
   row — `NotFound` if absent (covers both "not an active Kader" and
   "outside caller's scope" with one check, same as the existing
   `DeactivateKader`/`ReactivateKader` scope-check pattern).
4. Compute the new `organizationID` from the *requested* `roleName`
   (Scope decision #3):
   - Kader / LDK Admin → `candidate.HomeOrganizationID`
   - Puskomda Verifikator → `organizationRepo.FindByID(candidate.HomeOrganizationID).ParentOrganizationID`
   - Puskomnas Verifikator → `organizationRepo.RootPuskomnasID(ctx)` (already exists, `organization_repository_impl.go:139-147`)
5. Look up `roleID` for `roleName`, update `ms_user.roleID` +
   `organizationID` via the existing `user_repository.Update`-style call.
6. `BumpTokenVersion` (existing mechanism, already used by Manajemen
   Pengguna's role/org changes) — forces the affected account to
   re-login so their menu/permissions refresh.

`RevertToKaderRole` (implemented by `user_service`, called by
`submission_service.DeactivateKader` after it sets `ms_kader.status =
INACTIVE`) is steps 4-6 above with `roleName` hardcoded to `"Kader"` —
same underlying function, just invoked from the cascade instead of the
HTTP handler, so the two call sites can never drift apart.

### Migration `0051_pengurus_menu.up.sql`

```sql
INSERT IGNORE INTO lk_permission (permissionCode, permissionName, moduleName, menuLabel, menuIcon, menuRoute, sortOrder) VALUES
('pengurus.ldk.manage',       'Kelola Pengurus LDK',       'user', 'Pengurus', 'users-round', '/cms-ldk/pengurus', 15),
('pengurus.puskomda.manage',  'Kelola Pengurus Puskomda',  'user', 'Pengurus', 'users-round', '/cms-puskomda/pengurus', 15),
('pengurus.puskomnas.manage', 'Kelola Pengurus Puskomnas', 'user', 'Pengurus', 'users-round', '/cms-puskomnas/pengurus', 15);

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

New routes added to `modules/user/router.go`, next to the existing
`/users` group:
```go
g.GET("/users/pengurus", mw.RequirePermission(constants.PermPengurusLDKManage, constants.PermPengurusPuskomdaManage, constants.PermPengurusPuskomnasManage), h.ListPengurus)
g.PATCH("/users/:id/pengurus-role", mw.RequirePermission(constants.PermPengurusLDKManage, constants.PermPengurusPuskomdaManage, constants.PermPengurusPuskomnasManage), h.UpdatePengurusRole)
```

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
Data source resolved via `OrgContextService.organizationID$(route)` →
`PengurusRepository.list(organizationID)`, same pattern
`SubmissionKaderPersetujuanPresenter.loadAll()` already uses.

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
- `RevertToKaderRole` is a no-op (not an error) when the target's current
  role is already Kader or anything else outside the three promoted
  roles — keeps `DeactivateKader` safe to call unconditionally without
  the caller needing to check role first.
- Any successful role/org change bumps the affected account's token
  version, exactly like Manajemen Pengguna already does — the promoted/
  demoted user is forced to re-login and sees the updated menu only
  after that.

## Testing

- Backend: table test for `UpdatePengurusRole` covering the full
  allow-list matrix (3 caller tiers × 4 target roles, including rejected
  combinations), the self-action block, the out-of-scope target case,
  and the `organizationID` reassignment for all three target-role
  categories (Kader/LDK Admin → home LDK, Puskomda Verifikator → parent,
  Puskomnas Verifikator → root). Separate test for `RevertToKaderRole`
  being called from `DeactivateKader`, asserting both the no-op case
  (already Kader) and the revert case (role + organizationID both change).
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
