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
| PUT | `/users/:id` | `user.update` | Perbarui pengguna |
| PATCH | `/users/:id/status` | `user.update` | Aktifkan/nonaktifkan |
| POST | `/users/:id/reset-password` | `user.update` | Reset password → password sementara |
| DELETE | `/users/:id` | `user.delete` | Hapus (soft delete) |

**`POST /users`**
```json
{ "fullName": "Siti Nurhaliza", "email": "siti@fsldk.id", "roleID": 3, "password": "••••••••", "isActive": true }
```

**`PUT /users/:id`** → `{ "fullName": "...", "roleID": 2, "isActive": true }`
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
  "categoryID": 1,
  "isFeatured": false,
  "status": "draft"
}
```
**`PATCH /news/:id/publish`** → `{ "isPublished": true }`
**`PATCH /news/:id/featured`** → `{ "isFeatured": true }`

---

## 5. Artikel

Struktur identik dengan Berita (tanpa `isFeatured`/`viewCount`).

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
{ "articleTitle": "...", "articleExcerpt": "...", "articleContent": "...", "articleImage": "...", "categoryID": 1, "status": "draft" }
```

---

## 6. Konten Landing Page & Struktur Organisasi

### Publik

| Method | Endpoint | Deskripsi |
|---|---|---|
| GET | `/public/contents` | Seluruh konten aktif (key-value) |
| GET | `/public/contents/:key` | Satu konten by key (mis. `about.vision`) |
| GET | `/public/profile` | Semua konten sebagai map `{ key: body }` |
| GET | `/public/organization-structure` | Struktur organisasi aktif |

### CMS — ✅🔒 + permission

| Method | Endpoint | Permission | Deskripsi |
|---|---|---|---|
| GET | `/contents` | `content.view` | Semua konten (termasuk nonaktif) |
| PUT | `/contents/:key` | `content.update` | Perbarui judul/isi konten |
| GET | `/organization-structure` | `content.view` | Semua struktur organisasi |
| POST | `/organization-structure` | `content.update` | Tambah pengurus |
| PUT | `/organization-structure/:id` | `content.update` | Perbarui pengurus |
| DELETE | `/organization-structure/:id` | `content.update` | Hapus pengurus |

**`PUT /contents/:key`** → `{ "contentTitle": "...", "contentBody": "..." }`
**`POST /organization-structure`** → `{ "memberName": "...", "position": "...", "photoURL": "...", "level": "Puskomnas", "sortOrder": 1 }`

---

## 7. Dashboard (`/dashboard`) — ✅🔒

| Method | Endpoint | Deskripsi |
|---|---|---|
| GET | `/dashboard/summary` | `{ totalNews, publishedNews, draftNews, totalUsers }` |
| GET | `/dashboard/recent-news` | Berita terbaru (query: `limit`, default 5) |

---

## 8. Sistem (tanpa prefix `/api/v1`)

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
| `content.view/update` | Konten Landing Page | | |

Role bawaan: **Super Admin** (semua permission), **Editor** (news/article penuh + content), **Kontributor** (news/article tanpa publish/delete, tanpa content). Detail lengkap lihat [`migrations/0002_seed.up.sql`](../migrations/0002_seed.up.sql).

---

[← Kembali ke README](../README.md) · [Panduan Instalasi](./INSTALLATION.md) · [Arsitektur & Alur Sistem](./ARCHITECTURE.md)
