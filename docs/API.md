# Referensi API — FSLDK API

[← Kembali ke README](../README.md) · [Panduan Instalasi](./INSTALLATION.md) · [Arsitektur & Alur Sistem](./ARCHITECTURE.md)

Daftar lengkap seluruh endpoint REST API, disusun langsung dari kode routing (`modules/*/router.go`) agar selalu akurat dengan implementasi. Base URL: **`/api/v1`**.

---

## Konvensi

| Aspek | Ketentuan |
|---|---|
| **Auth** | Header `Authorization: Bearer <accessToken>` pada endpoint bertanda ✅ |
| **Terverifikasi** | Endpoint bertanda 🔒 mensyaratkan email pengguna sudah terverifikasi (selain login berhasil) |
| **Permission** | Endpoint CMS memerlukan kode permission tertentu pada role pengguna — lihat kolom **Permission** |
| **Format Response** | Lihat [README — Standar Response](../README.md#standar-response) |

---

## 1. Auth (`/auth`)

| Method | Endpoint | Auth | Rate Limit | Deskripsi |
|---|---|:---:|---|---|
| POST | `/auth/register` | ❌ | 5 akun / 10 menit / IP | Registrasi mandiri |
| POST | `/auth/login` | ❌ | 5x / menit / IP | Login email + password |
| POST | `/auth/google` | ❌ | 10x / menit / IP | Login/registrasi via Google ID Token |
| GET | `/auth/email/verify/:token` | ❌ | 6x / menit / IP | Verifikasi email dari tautan |
| POST | `/auth/forgot-password` | ❌ | 5x / menit / IP | Minta tautan reset password |
| POST | `/auth/reset-password` | ❌ | 6x / menit / IP | Tetapkan password baru dari token reset |
| POST | `/auth/logout` | ✅ | — | Logout (client menghapus token) |
| POST | `/auth/refresh-token` | ✅ | — | Perbarui access token |
| GET | `/auth/me` | ✅ | — | Profil pengguna saat ini |
| POST | `/auth/email/resend` | ✅ | 6x / menit | Kirim ulang email verifikasi |
| POST | `/auth/change-password` | ✅ | — | Ubah password (hanya akun berpassword lokal) |

**`POST /auth/register`**
```json
{ "fullName": "Ahmad Fadli", "email": "ahmad@fsldk.id", "password": "••••••••", "passwordConfirmation": "••••••••" }
```
→ `201` `{ "userID": 1, "email": "...", "emailVerified": false, "message": "..." }`

**`POST /auth/login`**
```json
{ "email": "ahmad@fsldk.id", "password": "••••••••" }
```
→ `200` `{ "accessToken": "...", "refreshToken": "...", "expiresIn": 3600, "user": { "userID": 1, "fullName": "...", "email": "...", "emailVerified": true, "role": "Kontributor", "permissions": ["news.view", ...] } }`

**`POST /auth/google`** → `{ "idToken": "<Google ID Token>" }` — response sama seperti login.

**`POST /auth/refresh-token`** → `{ "refreshToken": "..." }`

**`POST /auth/change-password`** → `{ "oldPassword": "...", "newPassword": "..." }`

**`POST /auth/forgot-password`** → `{ "email": "..." }`

**`POST /auth/reset-password`** → `{ "token": "...", "password": "...", "passwordConfirmation": "..." }`

---

## 2. User (`/users`) — ✅🔒 + permission

| Method | Endpoint | Permission | Deskripsi |
|---|---|---|---|
| GET | `/users` | `user.view` | Daftar pengguna (query: `page`, `limit`, `search`, `sort`, `roleID`) |
| GET | `/users/:id` | `user.view` | Detail pengguna |
| POST | `/users` | `user.create` | Buat pengguna baru |
| PUT | `/users/:id` | `user.update` | Perbarui pengguna (nama, email, role, status, password opsional) |
| PATCH | `/users/:id/status` | `user.update` | Aktifkan/nonaktifkan |
| DELETE | `/users/:id` | `user.delete` | Hapus (soft delete) |

**`POST /users`**
```json
{ "fullName": "Siti Nurhaliza", "email": "siti@fsldk.id", "roleID": 3, "password": "••••••••", "isActive": true }
```

**`PUT /users/:id`** → `{ "fullName": "...", "email": "...", "roleID": 2, "isActive": true, "password": "" }`
`password` bersifat opsional — kosongkan (string kosong) untuk mempertahankan password saat ini; isi (min. 8 karakter) untuk menggantinya. Tidak ada lagi endpoint reset-password terpisah.
**`PATCH /users/:id/status`** → `{ "isActive": false }`

---

## 3. Role & Permission (`/roles`, `/permissions`, `/me/menus`) — ✅🔒

| Method | Endpoint | Permission | Deskripsi |
|---|---|---|---|
| GET | `/roles` | `role.view` | Daftar role (query: `search`) |
| GET | `/roles/:id` | `role.view` | Detail role + daftar permission-nya |
| GET | `/roles/:id/users` | `role.view` | Daftar pengguna pemilik role ini |
| POST | `/roles` | `role.create` | Buat role baru |
| PUT | `/roles/:id` | `role.update` | Perbarui nama/deskripsi/status role |
| PUT | `/roles/:id/permissions` | `role.update` | Set ulang seluruh permission role |
| DELETE | `/roles/:id` | `role.delete` | Hapus role (bukan role sistem, tanpa pengguna) |
| GET | `/permissions` | `role.view` | Daftar seluruh permission tersedia |
| GET | `/me/menus` | — (cukup login+terverifikasi) | Menu sidebar CMS dinamis sesuai role |

**`POST /roles`** → `{ "roleName": "Moderator", "roleDescription": "..." }`
**`PUT /roles/:id/permissions`** → `{ "permissionIDs": [1, 2, 5, 8] }`

---

## 4. News / Berita

### Publik (tanpa auth)

| Method | Endpoint | Deskripsi |
|---|---|---|
| GET | `/public/news` | Daftar berita terpublikasi (query: `page`, `limit`, `search`, `category`) |
| GET | `/public/news-featured` | Berita unggulan (query: `limit`) |
| GET | `/public/news-categories` | Daftar kategori berita |
| GET | `/public/news/:slug` | Detail berita (menambah `viewCount`) |

### CMS — ✅🔒 + permission

| Method | Endpoint | Permission | Deskripsi |
|---|---|---|---|
| GET | `/news` | `news.view` | Daftar berita (semua status; query: `status=draft\|published`, `categoryID`) |
| GET | `/news/:id` | `news.view` | Detail untuk pengelolaan |
| POST | `/news` | `news.create` | Buat berita (publish butuh `news.publish`) |
| PUT | `/news/:id` | `news.update` | Perbarui berita |
| PATCH | `/news/:id/publish` | `news.publish` | Publish/tarik publikasi |
| PATCH | `/news/:id/featured` | `news.update` | Set/lepas status unggulan |
| DELETE | `/news/:id` | `news.delete` | Hapus berita |

**`POST /news`**
```json
{
  "newsTitle": "FSLDKN Ke-21 Resmi Digelar",
  "newsExcerpt": "Ringkasan singkat...",
  "newsContent": "<p>Isi lengkap...</p>",
  "newsImage": "https://.../gambar.jpg",
  "newsPublisher": "FSLDK Indonesia",
  "newsReporter": "Nama Reporter",
  "newsEditor": "Nama Editor",
  "categoryID": 1,
  "isFeatured": false,
  "status": "draft"
}
```
**`PATCH /news/:id/publish`** → `{ "isPublished": true }`
**`PATCH /news/:id/featured`** → `{ "isFeatured": true }`

`newsImage` tetap berupa string URL — nilainya biasanya hasil unggahan lewat `POST /uploads/image` (lihat §7), bukan ditulis manual. `newsReporter` wajib diisi (byline wartawan/penulis liputan); `newsPublisher` (penerbit) dan `newsEditor` opsional.

---

## 5. Artikel

Berbeda dari Berita: Artikel tidak punya `isFeatured`/`viewCount`, tapi punya konsep publikasi berbasis PDF — `articleIntro` (dulu `articleContent`) hanya berupa pendahuluan singkat yang tampil di landing page, sedangkan naskah lengkapnya dibaca lewat berkas PDF (`articlePdf`). Field `articleExcerpt` (ringkasan) sudah dihapus sepenuhnya (kolom DB & migration turut disesuaikan, lihat `migrations/0001_init.up.sql`).

### Publik

| Method | Endpoint | Deskripsi |
|---|---|---|
| GET | `/public/articles` | Daftar artikel terpublikasi |
| GET | `/public/article-categories` | Daftar kategori artikel |
| GET | `/public/articles/:slug` | Detail artikel |

### CMS — ✅🔒 + permission

| Method | Endpoint | Permission | Deskripsi |
|---|---|---|---|
| GET | `/articles` | `article.view` | Daftar artikel (semua status) |
| GET | `/articles/:id` | `article.view` | Detail untuk pengelolaan |
| POST | `/articles` | `article.create` | Buat artikel (publish butuh `article.publish`) |
| PUT | `/articles/:id` | `article.update` | Perbarui artikel |
| PATCH | `/articles/:id/publish` | `article.publish` | Publish/tarik publikasi |
| DELETE | `/articles/:id` | `article.delete` | Hapus artikel |

**`POST /articles`**
```json
{
  "articleTitle": "...",
  "articleIntro": "<p>Pendahuluan singkat...</p>",
  "articleImage": "http://localhost:8080/uploads/xxx.jpg",
  "articleWriter": "Nama Penulis",
  "articleEditor": "Nama Editor",
  "articlePdf": "http://localhost:8080/uploads/xxx.pdf",
  "categoryID": 1,
  "status": "draft"
}
```

`articleWriter` wajib diisi (byline penulis); `articleEditor` opsional. `articleImage`/`articlePdf` tetap berupa string URL — nilainya hasil unggahan lewat `POST /uploads/image` / `POST /uploads/document` (lihat §7), bukan ditulis manual.

---

## 6. Shortlink

Pemendek URL. Redirect yang dilihat pengunjung terjadi di **domain frontend**
(`fsldk-web`), bukan di backend ini — endpoint publik di bawah hanya
mengembalikan `destinationURL` sebagai JSON; frontend-lah yang melakukan
`window.location.href` setelah menerimanya (rute publik `/:key` di repositori
`fsldk-web`, di luar cakupan dokumen ini).

| Method | Endpoint | Auth | Deskripsi |
|---|---|:---:|---|
| GET | `/public/shortlinks/:key` | ❌ | `{ "destinationURL": "..." }`, mencatat 1 kunjungan (`visitCount`) |

### CMS — ✅🔒 + permission

| Method | Endpoint | Permission | Deskripsi |
|---|---|---|---|
| GET | `/shortlinks` | `shortlink.view` | Daftar shortlink (query: `page`, `limit`, `search`) |
| GET | `/shortlinks/:id` | `shortlink.view` | Detail shortlink |
| POST | `/shortlinks` | `shortlink.create` | Buat shortlink baru |
| PUT | `/shortlinks/:id` | `shortlink.update` | Perbarui tujuan/kunci shortlink |
| DELETE | `/shortlinks/:id` | `shortlink.delete` | Hapus shortlink |

**`POST /shortlinks`**
```json
{ "destinationURL": "https://fsldk-indonesia.com/berita/artikel-panjang", "shortKey": "acara2026" }
```
`shortKey` opsional — bila kosong, kunci acak 8 karakter dibuatkan otomatis. Response menyertakan `shortURL` siap-pakai (`{FRONTEND_URL}/{shortKey}`, mis. `https://fsldk-indonesia.com/acara2026`) dan `visitCount`.

**`PUT /shortlinks/:id`** → `{ "destinationURL": "...", "shortKey": "..." }` (keduanya wajib — kunci boleh diganti, tapi harus tetap unik)

---

## 7. Organization (`/organizations`, `/me/organizations`) — ✅🔒

Hierarki 3 tingkat LDK → Puskomda → Puskomnas. Cakupan akses ("scope") caller diresolusi server-side dari `organizationID`/`organizationTypeCode` (cascade: LDK→diri sendiri, Puskomda→diri+LDK di bawahnya, Puskomnas→seluruh organisasi) **atau** `wildcardTierAccess` (mem-bypass cascade untuk akun seperti Super Admin) — **tidak pernah** dipercaya dari input klien. Endpoint bertanda `RequireOrganizationScope` menolak (`403`) permintaan ke `:id` di luar cakupan caller, terlepas dari apa yang ditampilkan UI.

| Method | Endpoint | Permission | Deskripsi |
|---|---|---|---|
| GET | `/me/organizations` | — (cukup login+terverifikasi) | Daftar organisasi yang dapat diakses caller (dashboard switcher) |
| GET | `/organizations/directory` | — (cukup login+terverifikasi) | Direktori organisasi **aktif** bertipe tertentu (query wajib `organizationTypeCode`), lintas cakupan akses — dipakai skenario pemilihan bebas (mis. Kader memilih LDK tujuan pendaftaran) |
| GET | `/organizations` | `organization.profile.manage` / `organization.ldk.list` / `organization.ldk.list.national` / `organization.puskomda.list` (salah satu) | Daftar organisasi (query: `page`, `limit`, `search`, `organizationTypeCode`) — hasil sudah tersaring cascade/wildcard caller |
| GET | `/organizations/:id` | sama seperti di atas + `RequireOrganizationScope` | Detail organisasi |
| GET | `/organizations/:id/children` | sama seperti di atas + `RequireOrganizationScope` | Daftar anak organisasi langsung |
| POST | `/organizations` | `organization.create` | Buat organisasi baru — aturan parent: Puskomnas bebas pilih (LDK butuh parent Puskomda eksplisit); Puskomda hanya boleh buat LDK, parent otomatis dikunci ke dirinya sendiri |
| PUT | `/organizations/:id` | `organization.profile.manage` + `RequireOrganizationScope` | Perbarui profil organisasi |
| POST | `/organizations/:id/deactivate` | `organization.deactivate` + `RequireOrganizationScope` | Nonaktifkan organisasi |
| POST | `/organizations/:id/reactivate` | `organization.deactivate` + `RequireOrganizationScope` | Aktifkan kembali |

**`POST /organizations`**
```json
{ "organizationTypeCode": "LDK", "organizationName": "LDK Contoh", "organizationCode": "CONTOH", "parentOrganizationID": 2, "provinceName": "...", "cityName": "...", "contactEmail": "...", "contactPhone": "..." }
```

---

## 8. Submission Form — Form Builder (`/submission-forms`) — ✅🔒

Mesin form metadata-driven yang dipakai dua form konkret (`LEVELISASI_LDK`, `SENSUS_KADER`) — hierarki Form → Version (DRAFT/PUBLISHED/ARCHIVED) → Section → Field (10 tipe: TEXT/TEXTAREA/NUMBER/DATE/SELECT/MULTISELECT/RADIO/CHECKBOX/FILE_DOCUMENT/FILE_IMAGE) → Option. Struktur version hanya bisa diubah selama `DRAFT`; setelah `PUBLISHED`, immutable (ditegakkan di service layer) — perubahan berikutnya lewat version baru (opsional clone dari version manapun).

| Method | Endpoint | Permission | Deskripsi |
|---|---|---|---|
| GET | `/submission-forms/by-code/:formCode/published` | — (cukup login+terverifikasi) | Struktur version **PUBLISHED** form tsb. — bukan data sensitif, dipakai UI pengisian form (LDK/Kader) tanpa perlu permission admin |
| GET | `/submission-forms` | `submission_form.view` | Daftar form |
| POST | `/submission-forms` | `submission_form.manage` | Buat form baru |
| GET | `/submission-forms/:formID` | `submission_form.view` | Detail form + ringkasan seluruh version |
| POST | `/submission-forms/:formID/versions` | `submission_form.manage` | Buat version baru (opsional `cloneFromVersionID`) |
| GET | `/submission-forms/versions/:versionID` | `submission_form.view` | Struktur lengkap satu version (section/field/option) |
| POST | `/submission-forms/versions/:versionID/publish` | `submission_form.manage` | Publikasikan version (mengunci struktur) |
| POST | `/submission-forms/versions/:versionID/sections` | `submission_form.manage` | Tambah section |
| PUT/DELETE | `/submission-forms/sections/:sectionID` | `submission_form.manage` | Ubah/hapus section |
| POST | `/submission-forms/sections/:sectionID/fields` | `submission_form.manage` | Tambah field |
| PUT/DELETE | `/submission-forms/fields/:fieldID` | `submission_form.manage` | Ubah/hapus field |
| POST | `/submission-forms/fields/:fieldID/options` | `submission_form.manage` | Tambah pilihan (SELECT/MULTISELECT/RADIO/CHECKBOX) |
| PUT/DELETE | `/submission-forms/options/:optionID` | `submission_form.manage` | Ubah/hapus pilihan |

`validationRuleJSON` field (opsional): `{"minLength":2,"maxLength":50}` (teks) atau `{"min":0,"max":100}` (angka). `conditionalRuleJSON` (opsional, field kondisional lewat `conditionalOnFieldID`): `{"operator":"equals"|"notEquals","value":"YA"}` — field disembunyikan/tidak divalidasi wajib selama trigger belum terjawab.

---

## 9. Submission — Pendataan & Review (`/submissions`, `/kaders`) — ✅🔒

Alur Levelisasi LDK (2 tier: Puskomda → Puskomnas) dan Sensus Kader (1 tier: LDK, keputusan final). Kepemilikan/cakupan data diperiksa di **service layer** (bukan `RequireOrganizationScope` generik) karena submission ber-subjek `ORGANIZATION` dikunci ke organisasi pemanggil sendiri, sedangkan submission ber-subjek `KADER` bebas menunjuk LDK manapun. Seluruh aksi mutasi (`review`/`establish-level`/`publish`/`reopen`/`reassess`) mensyaratkan `version` (optimistic locking) — selisih dengan versi tersimpan → `409 Conflict`.

| Method | Endpoint | Permission | Deskripsi |
|---|---|---|---|
| POST | `/submissions` | `submission.create` | Buat submission `DRAFT` baru (idempotent per organisasi/form — `409` bila sudah ada & bukan draft) |
| PUT | `/submissions/:id/answers` | `submission.update` | Simpan jawaban (dapat dipanggil berulang, draft) |
| POST | `/submissions/:id/submit` | `submission.create` | Kirim (validasi required/format/conditional di service layer) → `SUBMITTED` |
| POST | `/submissions/:id/cancel` | `submission.cancel` | Batalkan (hanya status `DRAFT`) |
| GET | `/submissions` | `submission.view` | Daftar (query: `formCode`, `status`, `page`, `limit`) — tersaring cakupan akses caller otomatis |
| GET | `/submissions/:id` | `submission.view` | Detail + jawaban + riwayat status + hasil levelisasi/kader bila ada |
| POST | `/submissions/:id/review` | `submission.review.ldk` / `.review.tier1` / `.approve.tier1` / `.review.tier2` (salah satu — tier sesungguhnya diresolusi dari status submission & tier caller, bukan dari body) | `{decision: APPROVED\|REVISION_REQUESTED\|REJECTED, note, checklist?, version}` — Puskomda tidak bisa REJECTED (hanya revisi), Puskomnas tidak bisa APPROVED (lihat `establish-level`) |
| POST | `/submissions/:id/establish-level` | `submission.level.establish` | Puskomnas menetapkan level (`{levelCode, justificationNote, version}`) — otomatis mendeteksi reassessment (insert baris `tr_levelisasi_result` baru) vs koreksi dalam siklus sama (update baris ada) |
| POST | `/submissions/:id/publish` | `submission.publish` | Puskomnas mempublikasikan hasil (hanya dari `LEVEL_ESTABLISHED`) |
| POST | `/submissions/:id/reopen` | `submission.reopen` | Puskomnas membuka kembali submission `PUBLISHED` untuk **koreksi administratif** (`{reason, version}`) → `REVISION_REQUESTED_PUSKOMNAS`, level lama tetap tampil resmi selama proses |
| POST | `/submissions/:id/reassess` | `submission.reassess` | LDK **atau** Puskomnas mengajukan **siklus reassessment baru** (`{version}`, hanya dari `PUBLISHED`) → reset ke `DRAFT`, LDK isi ulang dari awal |
| GET | `/kaders` | `submission.review.ldk` | Daftar kader (query: `status`) — tersaring cakupan akses (DL-11: Puskomda/Puskomnas tidak punya akses sama sekali ke data ini) |
| GET | `/kaders/:id/code` | — (kader pemilik atau org access) | Kartu digital kader (`uniqueCode`, `issuedDate`) |
| POST | `/kaders/:id/deactivate` | `kader.deactivate` | Nonaktifkan kader `ACTIVE` (hanya LDK pemilik, tidak ada cascade meski untuk Puskomda/Puskomnas — kecuali wildcard) |

**`POST /submissions/:id/review`**
```json
{ "decision": "REVISION_REQUESTED", "note": "Lengkapi dokumen Sarana & Prasarana.", "checklist": { "identitas": true, "saranaPrasarana": false }, "version": 3 }
```
`checklist` bersifat freeform JSON (tidak divalidasi bentuknya oleh backend) — konvensi frontend: `{sectionCode: boolean}` mengikuti section form yang sedang direview.

Status Levelisasi LDK: `DRAFT → SUBMITTED → PUSKOMDA_REVIEW → APPROVED_PUSKOMDA → APPROVED_PUSKOMNAS → LEVEL_ESTABLISHED → PUBLISHED` (dengan cabang `REVISION_REQUESTED_*` kembali ke `SUBMITTED` setelah diedit). `APPROVED_PUSKOMNAS` hanya dicapai lewat keputusan "Setujui" pada Verifikasi Akhir (tier Puskomnas) — Penetapan Levelisasi (`EstablishLevel`) mensyaratkan status ini, bukan `APPROVED_PUSKOMDA` langsung, supaya kedua tahap berurutan (bukan gerbang yang sama). Status Sensus Kader: `DRAFT → SUBMITTED → APPROVED_LDK` (otomatis lanjut ke `ACTIVE` + kode kader terbit) `/ REVISION_REQUESTED_LDK / REJECTED`.

---

## 10. Report (`/reports`) — ✅🔒

Ekspor laporan submission Levelisasi — **bukan** endpoint JSON, response berupa berkas biner (`Content-Disposition: attachment`), sinkron (tanpa job queue). Cakupan data mengikuti cascade organisasi caller yang sama seperti `/submissions`.

| Method | Endpoint | Permission | Deskripsi |
|---|---|---|---|
| GET | `/reports/submissions/export` | `report.region.export` / `report.national.export` (salah satu) | Ekspor Excel/CSV (query: `formCode` wajib, `status` opsional, `format=xlsx\|csv` default `xlsx`) |

Nama berkas: `laporan-{formCode}-{yyyyMMdd-HHmmss}.{ext}`. Setiap ekspor dicatat ke `tr_export_log` (audit trail).

---

## 11. Upload (`/uploads`) — ✅🔒

Unggah berkas gambar/dokumen — dipakai bersama oleh form Artikel/Berita CMS **dan** field `FILE_IMAGE`/`FILE_DOCUMENT` pada pengisian submission (§9), bukan endpoint khusus per-modul.

| Method | Endpoint | Auth | Deskripsi |
|---|---|:---:|---|
| POST | `/uploads/image` | ✅ (login + verified, tanpa permission khusus) | Unggah satu berkas gambar, `multipart/form-data` field `image` |
| POST | `/uploads/document` | ✅ (login + verified, tanpa permission khusus) | Unggah satu berkas dokumen, `multipart/form-data` field `document` |

**`POST /uploads/image`** (multipart/form-data, field `image`) → `{ "url": "http://localhost:8080/uploads/<nama-acak>.jpg" }`
Validasi: ekstensi `jpg`/`jpeg`/`png`/`webp`/`gif`, maksimal 5MB.

**`POST /uploads/document`** (multipart/form-data, field `document`) → `{ "url": "http://localhost:8080/uploads/<nama-acak>.pdf" }`
Validasi: ekstensi `pdf`/`docx`/`xlsx` (docx/xlsx ditambahkan untuk dokumen pendukung submission — lihat §9; naskah Artikel di §5 tetap PDF saja secara konvensi), maksimal 20MB.

Kedua endpoint menyimpan berkas ke `assets/uploads/` dengan nama acak (hex 16 byte + ekstensi asli) dan menyajikannya sebagai berkas statis publik di `/uploads/*`. `url` hasil unggahan inilah yang dikirim sebagai nilai `articleImage`/`newsImage`/`articlePdf` pada `POST`/`PUT` Artikel & Berita — kolom tersebut tetap berupa string URL di database, tidak ada perubahan skema tambahan di luar yang sudah dijelaskan di §5.

---

## 12. Dashboard (`/dashboard`) — ✅🔒

| Method | Endpoint | Deskripsi |
|---|---|---|
| GET | `/dashboard/summary` | Ringkasan **tier-aware** — bentuk response berbeda sesuai `organizationTypeCode` caller |

Response selalu `{ "organizationTypeCode": "LDK\|PUSKOMDA\|PUSKOMNAS", "ldk"?, "puskomda"?, "puskomnas"? }` — hanya satu dari tiga kunci opsional yang terisi:

- **`ldk`**: `{ submissionStatus, lastUpdatedDate?, levelCode?, levelLabel?, kaderPending, kaderActive, recentNotes: [{note, createdDate}] }`
- **`puskomda`**: `{ totalLDK, belumMengisi, menungguVerifikasi, perluRevisi, terverifikasi, totalKaderAktif }`
- **`puskomnas`**: `{ totalLDKNasional, belumMengisi, menungguVerifikasi, perluRevisi, terverifikasi, levelEstablishedCount, totalPuskomda, totalKaderAktifNasional, levelDistribution: [{levelCode, levelLabel, count}], perPuskomda: [{organizationID, organizationName, totalLDK, kaderAktif}] }`

Caller tanpa tier organisasi/wildcard (mis. Kader) mendapat `{organizationTypeCode: ""}` kosong (bukan error) — dashboard memang tidak dirancang untuk tier ini (Kader punya halaman sendiri, lihat modul `submission`).

---

## 13. Sistem (tanpa prefix `/api/v1`)

| Method | Endpoint | Deskripsi |
|---|---|---|
| GET | `/health` | `{ "status": "ok\|degraded", "service": "fsldk-api" }` |
| GET | `/version` | `{ "name", "version", "env" }` |

---

## Matriks Permission

| Kode | Modul | Kode | Modul |
|---|---|---|---|
| `news.view/create/update/delete/publish` | Berita | `article.view/create/update/delete/publish` | Artikel |
| `user.view/create/update/delete` | Pengguna | `role.view/create/update/delete` | Role |
| `shortlink.view/create/update/delete` | Shortlink | `organization.create/profile.manage/deactivate` | Organisasi |
| `organization.ldk.list/ldk.list.national/puskomda.list` | Organisasi (daftar) | `submission_form.view/manage` | Form Builder |
| `submission.create/update/cancel/view` | Pendataan (pemilik) | `submission.review.ldk` | Persetujuan Kader (LDK) |
| `submission.review.tier1/approve.tier1` | Verifikasi/Persetujuan Wilayah (Puskomda) | `submission.review.tier2` | Verifikasi Akhir (Puskomnas) |
| `submission.level.establish/publish/reopen/reassess` | Penetapan Level/Publikasi/Koreksi (Puskomnas) | `kader.deactivate` | Nonaktifkan Kader (LDK) |
| `report.region.view/export` | Laporan Wilayah (Puskomda) | `report.national.view/export` | Laporan Nasional (Puskomnas) |

Role bawaan pra-proyek: **Super Admin** (semua permission), **Editor** (news/article/shortlink penuh), **Kontributor** (news/article tanpa publish/delete, tanpa shortlink). Detail lengkap lihat [`migrations/0002_seed.up.sql`](../migrations/0002_seed.up.sql) dan [`migrations/0004_shortlink.up.sql`](../migrations/0004_shortlink.up.sql).

Role tambahan modul Submission Dashboard (hierarki organisasi) — satu role per akun, tanpa multi-role:

| Role | Cakupan | Permission utama |
|---|---|---|
| **LDK Admin** | Organisasi sendiri | `submission.create/update/cancel/view`, `submission.review.ldk`, `submission.reassess`, `kader.deactivate`, `organization.profile.manage` (diri sendiri) |
| **Puskomda Verifikator** | Diri + LDK di wilayahnya | `submission.review.tier1/approve.tier1`, `organization.ldk.list`, `report.region.view/export` |
| **Puskomnas Verifikator** | Seluruh organisasi | `submission.review.tier2/level.establish/publish/reopen/reassess`, `organization.ldk.list.national/puskomda.list`, `report.national.view/export` |
| **Kader** | Diri sendiri (bukan cascade organisasi) | `submission.create/update/cancel/view` (form Sensus Kader saja) |

`wildcardTierAccess` (kolom `SET('LDK','PUSKOMDA','PUSKOMNAS')` di `ms_user`) mem-bypass cascade organisasi untuk akun sepert Super Admin — lihat [Arsitektur §11](./ARCHITECTURE.md#11-organization-scope--cascade-access). Detail lengkap seed role/permission modul ini: [`migrations/0005_organization_access.up.sql`](../migrations/0005_organization_access.up.sql) s.d. [`0009_audit_reporting.up.sql`](../migrations/0009_audit_reporting.up.sql).

> Konten Landing Page (visi/misi/struktur organisasi/kontak) tidak dikelola via API/database — dikelola sebagai teks tetap (hardcoded) langsung di frontend `fsldk-web`.

---

[← Kembali ke README](../README.md) · [Panduan Instalasi](./INSTALLATION.md) · [Arsitektur & Alur Sistem](./ARCHITECTURE.md)
