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

## 7. Upload (`/uploads`) — ✅🔒

Unggah berkas gambar/dokumen untuk form Artikel & Berita CMS — dipakai bersama oleh kedua modul, bukan endpoint khusus per-modul.

| Method | Endpoint | Auth | Deskripsi |
|---|---|:---:|---|
| POST | `/uploads/image` | ✅ (login + verified, tanpa permission khusus) | Unggah satu berkas gambar, `multipart/form-data` field `image` |
| POST | `/uploads/document` | ✅ (login + verified, tanpa permission khusus) | Unggah satu berkas dokumen (PDF), `multipart/form-data` field `document` |

**`POST /uploads/image`** (multipart/form-data, field `image`) → `{ "url": "http://localhost:8080/uploads/<nama-acak>.jpg" }`
Validasi: ekstensi `jpg`/`jpeg`/`png`/`webp`/`gif`, maksimal 5MB.

**`POST /uploads/document`** (multipart/form-data, field `document`) → `{ "url": "http://localhost:8080/uploads/<nama-acak>.pdf" }`
Validasi: ekstensi `pdf`, maksimal 20MB. Dipakai sebagai `articlePdf` — naskah lengkap Artikel yang dibaca lewat PDF di landing page (lihat §5).

Kedua endpoint menyimpan berkas ke `assets/uploads/` dengan nama acak (hex 16 byte + ekstensi asli) dan menyajikannya sebagai berkas statis publik di `/uploads/*`. `url` hasil unggahan inilah yang dikirim sebagai nilai `articleImage`/`newsImage`/`articlePdf` pada `POST`/`PUT` Artikel & Berita — kolom tersebut tetap berupa string URL di database, tidak ada perubahan skema tambahan di luar yang sudah dijelaskan di §5.

---

## 8. Dashboard (`/dashboard`) — ✅🔒

| Method | Endpoint | Deskripsi |
|---|---|---|
| GET | `/dashboard/summary` | `{ totalNews, totalArticles, totalUsers }` |

---

## 9. Sistem (tanpa prefix `/api/v1`)

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
| `shortlink.view/create/update/delete` | Shortlink | | |

Role bawaan: **Super Admin** (semua permission), **Editor** (news/article/shortlink penuh), **Kontributor** (news/article tanpa publish/delete, tanpa shortlink). Detail lengkap lihat [`migrations/0002_seed.up.sql`](../migrations/0002_seed.up.sql) dan [`migrations/0004_shortlink.up.sql`](../migrations/0004_shortlink.up.sql).

> Konten Landing Page (visi/misi/struktur organisasi/kontak) tidak dikelola via API/database — dikelola sebagai teks tetap (hardcoded) langsung di frontend `fsldk-web`.

---

[← Kembali ke README](../README.md) · [Panduan Instalasi](./INSTALLATION.md) · [Arsitektur & Alur Sistem](./ARCHITECTURE.md)
