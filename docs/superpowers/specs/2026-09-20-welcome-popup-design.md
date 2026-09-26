# Welcome Popup (CMS-managed, dynamic) — Design

Status: approved by user, pending implementation plan.
Spans both repos: `fsldk-api` (backend) and `fsldk-web` (frontend). This
file lives in `fsldk-api/docs/` because the backend module is the source of
truth for the data shape; `fsldk-web`'s implementation plan should link
back here.

## Background

The public FSLDK site currently has no "welcome popup" mechanism. The user
wants one modeled on an existing reference in a sibling Laravel app,
`ldksyahid-app/resources/views/components/welcome-revamp-popup/` — three
files: `index.blade.php` (markup/content), `styles.blade.php` (CSS),
`scripts.blade.php` (JS: show-once-per-browser via `localStorage`, 800ms
delay, scroll-lock, close/dismiss, Web Share API). That reference is
redeployed by hand for every seasonal campaign (Eid, Qurban, Milad, ...) —
evidenced by a `LS_KEYS_OLD` cleanup list of eight past campaign keys in
its script.

The ask: make this dynamic. A Super-Admin-only CMS form with exactly three
raw-content fields — **Field Index** (HTML), **Field Javascript**,
**Field CSS** — that map 1:1 to the three reference files. Saving updates
the live public popup immediately. No new campaign should ever require a
code deploy again.

## Scope decisions (confirmed with user)

1. **Single active slot**, not a list of campaigns/drafts. One row, always
   edited in place — like the existing generic `setting` module, not a
   CRUD-with-list module.
2. **Homepage only** (`/`), not site-wide. Rendered once inside
   `modules/home`'s index page.
3. **Explicit enabled/disabled toggle**, separate from the three content
   fields, so Super Admin can pull the popup without clearing its content.

The show-once-per-visitor logic, close button, dismiss, backdrop, share
button, animations — all of that lives entirely inside whatever HTML/CSS/JS
the Super Admin pastes in, exactly like the reference component does today.
The Angular/Go side is a **generic delivery mechanism** with zero
popup-specific business logic baked in — it does not know what "dismiss"
or "backdrop" mean.

## Why not reuse the existing `setting` module?

`ms_setting` (migration `0008_setting.up.sql`) already has almost the right
shape: `TEXT` value column, `isHide` flag to keep entries out of the
generic CMS list, and its `setting.view`/`setting.update` permissions are
*already* Super-Admin-only — the exact same trust boundary this feature
needs.

Rejected anyway, because:
- `setting.index.page.html` renders every value as a single-line
  `<input>` (or a Yes/No `<select>` for booleans). That's unusable for
  pasting multi-line HTML/JS/CSS — the user explicitly asked for a
  dedicated form, not a shoehorned generic list.
- Lumping "raw code injection for the public site" under the generic
  "runtime config" permission (`setting.*`) muddies what that permission
  actually grants. A dedicated `welcomepopup.*` permission is clearer in
  permission audits and in the CMS sidebar.
- A dedicated table lets the public read endpoint be narrowly scoped to
  exactly 4 columns, with no risk of ever accidentally exposing an
  unrelated (possibly sensitive) `ms_setting` row to the public internet.

So: new **minimal singleton module**, following the same "trimmed shape"
precedent this codebase already uses for `zakat` (no repository at all,
stateless) and `upload` (no `_model`/`_repository`) — i.e. it is fine and
established practice to skip parts of the standard per-module shape when
the resource genuinely isn't a list.

## Backend (fsldk-api) — new module `welcomepopup`

**Migration** `NNNN_welcomepopup.up.sql`:
```sql
CREATE TABLE IF NOT EXISTS ms_welcome_popup (
    id           TINYINT UNSIGNED PRIMARY KEY,
    isEnabled    BOOLEAN NOT NULL DEFAULT FALSE,
    htmlContent  LONGTEXT NULL,
    jsContent    LONGTEXT NULL,
    cssContent   LONGTEXT NULL,
    updatedDate  DATETIME NULL,
    updatedBy    BIGINT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

INSERT IGNORE INTO ms_welcome_popup (id, isEnabled) VALUES (1, FALSE);

INSERT IGNORE INTO lk_permission (permissionCode, permissionName, moduleName, menuLabel, menuIcon, menuRoute, sortOrder)
VALUES ('welcomepopup.view',   'Lihat Welcome Popup', 'welcomepopup', 'Welcome Popup', 'megaphone', '/cms/welcome-popup', 98),
       ('welcomepopup.update', 'Ubah Welcome Popup',  'welcomepopup', NULL, NULL, NULL, 98);

INSERT IGNORE INTO map_role_permission (roleID, permissionID)
SELECT r.roleID, p.permissionID FROM ms_role r, lk_permission p
WHERE r.roleName = 'Super Admin' AND p.permissionCode IN ('welcomepopup.view', 'welcomepopup.update');
-- HANYA Super Admin, sama seperti setting.* (0008_setting.up.sql) — raw
-- HTML/JS/CSS ini dieksekusi apa adanya di public site, tanpa sanitasi.
```
The row is always `id = 1`; repository methods operate on that fixed id
(no list, no auto-increment, no risk of a second row ever existing).

**Layers** (mirrors the standard shape, trimmed to singleton — no list/
pagination anywhere):
- `welcomepopup_model`: `WelcomePopup` struct — pure data, gorm tags.
- `welcomepopup_dto`:
  - `Response` (CMS): `isEnabled`, `htmlContent`, `jsContent`, `cssContent`,
    `updatedDate`.
  - `UpdateRequest`: `isEnabled`, `htmlContent`, `jsContent`, `cssContent`
    (no length cap via `validate:"max=..."` the way `setting.UpdateRequest`
    does at 1000 — HTML/JS/CSS blocks are expected to run well past that;
    rely on `LONGTEXT`'s own ~4GB ceiling, no artificial app-level limit).
  - `PublicResponse`: `isEnabled`, `htmlContent`, `jsContent`, `cssContent`
    only — deliberately excludes `updatedDate`/`updatedBy` so the public
    endpoint never leaks CMS metadata.
- `welcomepopup_repository`: `Get() (*model, error)`, `Update(*model) error`
  — both just `WHERE id = 1`.
- `welcomepopup_service`: `Get()` (CMS), `GetPublic()` (maps to
  `PublicResponse`), `Update(req)`.
- `welcomepopup_handler`: `Get`, `Update` (CMS, both behind permission
  middleware), `GetPublic` (public, no middleware).

**Routing** (`welcomepopup/router.go`), following the `zakat`/`setting`
split exactly:
```go
func RegisterCMSRoutes(rg *gin.RouterGroup, h welcomepopup_handler.Handler, mw *middlewares.Middleware) {
    g := rg.Group("/welcome-popup")
    g.Use(mw.Auth(), mw.RequireVerified())
    g.GET("", mw.RequirePermission(constants.PermWelcomePopupView), h.Get)
    g.PUT("", mw.RequirePermission(constants.PermWelcomePopupUpdate), h.Update)
}

func RegisterPublicRoutes(pub *gin.RouterGroup, h welcomepopup_handler.Handler) {
    pub.GET("/welcome-popup", middlewares.RateLimit(30, 10), h.GetPublic)
}
```
Wired into root `router.go` alongside the other modules (repo → service →
handler → both `Register*Routes` calls), same as every existing module.
New `constants.PermWelcomePopupView` / `PermWelcomePopupUpdate` added to
`constants/constants.go` next to `PermSettingView`/`PermSettingUpdate`.

## Frontend CMS (fsldk-web) — new module `modules/welcomepopup/`

Structure mirrors the established module shape (`entities/`, `services/`,
`repositories/`), but the **page** is a single edit form (no index/list —
there's nothing to list), closer in spirit to a settings page than a CRUD
page:

```
modules/welcomepopup/
├── entities/welcome-popup.ts          # WelcomePopup interface (plain)
├── services/welcomepopup-api.service.ts
├── repositories/welcomepopup.repository.ts
├── pages/form/
│   ├── welcomepopup.form.page.ts / .html
│   ├── welcomepopup.form.presenter.ts
│   └── welcomepopup.form.view.ts
└── welcomepopup.routes.ts             # '/cms/welcome-popup', permissionGuard
```

Form fields: a switch ("Aktif — tampilkan di halaman Beranda"), then three
large `<textarea>`s (`rows` generous, e.g. 12–16, monospace font) labeled
exactly **Field Index**, **Field Javascript**, **Field CSS** — matching
the user's own naming, which itself maps to the three reference files. One
"Simpan" button PUTs all four fields together. No live preview in this
first version (YAGNI — the public homepage itself *is* the preview, one
tab away).

Sidebar entry needs no `cms-layout.component.ts` change — it's driven
entirely by the `lk_permission.menuLabel/menuIcon/menuRoute` row from the
migration, exactly like every other CMS menu item, and will only render
for accounts holding `welcomepopup.view` (Super Admin, by the seed above).

## Frontend public site — new component `modules/home/components/welcome-popup.component.ts`

Included once in `home.index.page.html` (homepage only, per the scope
decision). On init:
1. `GET /public/welcome-popup` (unauthenticated; goes through the normal
   `ApiService` — no interaction with the auth/session machinery).
2. If `!isEnabled || !htmlContent.trim()`, do nothing further.
3. Otherwise, inject via direct DOM APIs (**not** Angular's `[innerHTML]`
   binding, and not `DomSanitizer.bypassSecurityTrustHtml` + a template
   binding either) — because a `<script>` tag parsed into `innerHTML`
   never executes, by browser design, regardless of Angular's sanitizer:
   ```ts
   const wrap = document.createElement('div');
   wrap.innerHTML = htmlContent;                 // parses markup; embedded <script> stays inert
   document.body.appendChild(wrap);
   if (cssContent) {
     const style = document.createElement('style');
     style.textContent = cssContent;
     document.head.appendChild(style);
   }
   if (jsContent) {
     const script = document.createElement('script');
     script.textContent = jsContent;              // appending via DOM API DOES execute it
     document.body.appendChild(script);
   }
   ```
   Guard this whole block behind `isPlatformBrowser` (this app doesn't
   currently appear to use Angular SSR, but the guard is one line and
   removes any ambiguity about `document` at bootstrap time).

## Security note (explicit, not a gap to "fix")

This feature is intentionally arbitrary HTML/CSS/**JavaScript** execution
on the live public site, editable by anyone holding `welcomepopup.update`.
There is no sanitization anywhere in this design — sanitizing would break
the JS entirely, and defeats the feature's purpose (it exists specifically
so Super Admin can paste share buttons, AJAX calls, animations, etc.,
exactly like the `ldksyahid-app` reference does). The only safeguard is the
permission gate, which is why the migration seeds it to Super Admin only
and nothing else — this must not be loosened without deliberate review.
`updatedBy`/`updatedDate` are kept (mirroring `ms_setting`) as a minimal
audit trail of who last changed it.

## Testing

- Backend: table test for `welcomepopup_service` covering `Get`/`Update`/
  `GetPublic` (particularly that `GetPublic`'s response never contains
  `updatedBy`/`updatedDate` fields even if the DTO struct changes shape
  later — a field-list assertion, not just a status-code check).
- Frontend: presenter test for the CMS form (load → edit → save happy
  path); the public injector component is DOM-manipulation-heavy and is
  best verified manually (load homepage with a small test payload in all
  three fields, confirm the script actually runs) rather than over-mocked
  in a unit test.

## Out of scope (YAGNI, can be added later without reshaping this design)

- Multiple saved campaigns / scheduling / history of past popups.
- Live preview inside the CMS form.
- Per-page or per-tier targeting beyond "homepage only."
- Rich HTML/code editor (syntax highlighting, etc.) — plain `<textarea>`
  for v1, matching the reference's own plain-file editing.
