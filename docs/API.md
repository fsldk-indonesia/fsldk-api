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

## 2. User (`/users`) — ✅🔒

| Method | Endpoint | Permission | Deskripsi |
|---|---|---|---|
| GET | `/users/mention-search` | — (cukup login+terverifikasi) | Cari pengguna aktif untuk autocomplete @mention komentar (query: `q`, `limit` maks. 20, default 8); boleh mencari & mem-mention diri sendiri — lihat [Arsitektur §11](./ARCHITECTURE.md#11-komentar-kedalaman-balasan-moderasi-dan-mention) |
| GET | `/users` | `user.view` | Daftar pengguna (query: `page`, `limit`, `search`, `sort`, `roleID`) |
| GET | `/users/:id` | `user.view` | Detail pengguna |
| POST | `/users` | `user.create` | Buat pengguna baru |
| PUT | `/users/:id` | `user.update` | Perbarui pengguna (nama, email, role, status, password opsional) |
| PATCH | `/users/:id/status` | `user.update` | Aktifkan/nonaktifkan |
| DELETE | `/users/:id` | `user.delete` | Hapus (soft delete) |

**`GET /users/mention-search`** → `[{ "userID": 1, "fullName": "Ahmad Fadli", "photoURL": "..." }, ...]` — hanya field minimal ini (bukan `user_dto.Response` penuh), karena endpoint ini bisa dipanggil siapa pun yang login+verified, bukan hanya pemegang `user.view`.

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

`newsImage` tetap berupa string URL — nilainya biasanya hasil unggahan lewat `POST /uploads/image` (lihat §9), bukan ditulis manual. `newsReporter` wajib diisi (byline wartawan/penulis liputan); `newsPublisher` (penerbit) dan `newsEditor` opsional.

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

`articleWriter` wajib diisi (byline penulis); `articleEditor` opsional. `articleImage`/`articlePdf` tetap berupa string URL — nilainya hasil unggahan lewat `POST /uploads/image` / `POST /uploads/document` (lihat §9), bukan ditulis manual.

---

## 6. Event

### Publik (tanpa auth)

| Method | Endpoint | Deskripsi |
|---|---|---|
| GET | `/public/events` | Daftar event terpublikasi (query: `page`, `limit` default 9 maks. 100, `search`, `division`, `year`, `status` — masing-masing boleh multi-nilai dipisah koma, mis. `status=upcoming,ongoing`, `sort` default `newest`, alternatif `title`) |
| GET | `/public/events/:slug` | Detail event (menambah `viewCount`) — turunan `status` (`upcoming`/`ongoing`/`past`) & `registOpen` dihitung dari `startDate`/`endDate`/`closeRegistDate` saat request, bukan disimpan |

### CMS — ✅🔒 + permission

| Method | Endpoint | Permission | Deskripsi |
|---|---|---|---|
| GET | `/events` | `event.view` | Daftar seluruh event, published atau tidak (query: `page`, `limit`, `search`, `division`, `sort`) |
| GET | `/events/:id` | `event.view` | Detail untuk pengelolaan |
| POST | `/events` | `event.create` | Buat event baru |
| PUT | `/events/:id` | `event.update` | Perbarui event |
| DELETE | `/events/:id` | `event.delete` | Hapus event — komentar terkait (`contentType="event"`) ikut dibersihkan (lihat §7) |

**`POST /events`**
```json
{
  "eventTitle": "FSLDKN Ke-21",
  "eventDivision": "Departemen Jaringan",
  "eventContent": "<p>Deskripsi lengkap acara...</p>",
  "eventImage": "http://localhost:8080/uploads/xxx.jpg",
  "startDate": "2026-09-01T08:00:00",
  "endDate": "2026-09-03T17:00:00",
  "closeRegistDate": "2026-08-25T23:59:00",
  "location": "Bandung",
  "place": "Gedung Sate",
  "locationLink": "https://maps.google.com/...",
  "registrationLink": "https://forms.gle/...",
  "documentLink": "https://drive.google.com/...",
  "presentationLink": "https://drive.google.com/...",
  "contactPerson1": "81234567890",
  "nameCp1": "Ahmad",
  "contactPerson2": "81234567891",
  "nameCp2": "Siti",
  "tag": "nasional,jaringan",
  "isPublished": false
}
```
`PUT /events/:id` memakai bentuk body yang sama persis (semua field wajib dikirim ulang, bukan partial update). Semua field tanggal menerima ISO8601 atau `YYYY-MM-DD[ HH:mm[:ss]]`, boleh dikosongkan (`""` → `null`). `eventImage` string URL hasil `POST /uploads/image` (lihat §9), bukan ditulis manual. `eventSlug` dibuat otomatis dari `eventTitle` (unik, re-slug hanya kalau judul berubah).

---

## 7. Komentar (`/comments`)

Dipakai bersama oleh Artikel, Berita, dan Event — `contentType` (`article`/`news`/`event`) + `contentID` menunjuk ke konten manapun tanpa foreign key (lihat [Arsitektur §11](./ARCHITECTURE.md#11-komentar-kedalaman-balasan-moderasi-dan-mention)). Balasan dibatasi **1 level** (tidak bisa membalas balasan).

### Publik (tanpa auth, tapi login opsional ikut mengisi `isOwner`/reaksi milik-sendiri)

| Method | Endpoint | Deskripsi |
|---|---|---|
| GET | `/public/comments` | Thread komentar satu konten (query wajib: `contentType`, `contentID`) — array bersarang `replies` |

### Aksi milik-sendiri — ✅🔒 (cukup login+terverifikasi, tanpa permission khusus)

| Method | Endpoint | Deskripsi |
|---|---|---|
| POST | `/comments` | Buat komentar/balasan baru |
| PUT | `/comments/:id` | Ubah komentar — **pemilik selalu boleh**; bukan-pemilik butuh permission `comment.update` |
| DELETE | `/comments/:id` | Hapus komentar (balasan & reaksi ikut terhapus via `ON DELETE CASCADE`) — **pemilik selalu boleh**; bukan-pemilik butuh permission `comment.delete` |
| POST | `/comments/:id/react` | Toggle reaksi emoji (like/dislike/love/heart_eyes/laughing/rage/slight_smile) — kirim ulang `reactionType` yang sama untuk membatalkan |
| GET | `/comments/gif-search` | Proxy pencarian GIF/sticker via GIPHY (query: `q`, `tab=gifs\|stickers`) — array kosong bila `GIPHY_API_KEY` tidak diisi, tidak pernah error |
| GET | `/comments/gif-categories` | Proxy kategori GIF trending GIPHY (maks. 8) |

### Moderasi admin — ✅🔒 + permission

| Method | Endpoint | Permission | Deskripsi |
|---|---|---|---|
| GET | `/comments` | `comment.view` | Daftar komentar top-level lintas konten (query: `page`, `limit`, `search`, `contentType`, `sort`) |
| GET | `/comments/:id` | `comment.view` | Detail satu komentar + seluruh subtree balasannya |
| POST | `/comments/bulk-delete` | `comment.delete` | Hapus banyak komentar sekaligus — `{ "ids": [1,2,3] }`, tanpa konsep kepemilikan (murni permission) |

**`POST /comments`**
```json
{
  "contentType": "article",
  "contentID": 12,
  "parentID": null,
  "commentText": "Terima kasih atas informasinya @Ahmad Fadli!",
  "mediaURL": "",
  "mediaType": "",
  "mentionedUserIDs": [7]
}
```
`parentID` diisi `commentID` induk untuk membalas (`null` untuk komentar top-level). `commentText`/`mediaURL` setidaknya satu wajib diisi. `mentionedUserIDs` opsional (maks. 20) — userID yang benar-benar dipilih lewat autocomplete `GET /users/mention-search` (§2), **bukan** hasil parsing pola teks; teks `@Nama` di `commentText` yang tidak ada di `mentionedUserIDs` tidak dianggap mention. `PUT /comments/:id` memakai bentuk body yang sama (tanpa `contentType`/`contentID`/`parentID`).

**Response** (`Response`, dipakai di semua endpoint di atas):
```json
{
  "commentID": 5, "contentType": "article", "contentID": 12, "commentText": "...",
  "mediaURL": "", "mediaType": "", "parentID": null, "isOwner": true,
  "createdDate": "2026-08-16 10:00:00",
  "author": { "userID": 3, "name": "...", "photo": "..." },
  "reactions": { "counts": { "like": 2 }, "userTypes": ["like"] },
  "mentions": [{ "userID": 7, "name": "Ahmad Fadli", "photo": "..." }],
  "replies": []
}
```

---

## 8. Shortlink

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

## 8a. Permintaan Shortlink (`/shortlink-requests`)

Alur permintaan publik + persetujuan admin di atas modul Shortlink (§8) — pengunjung tanpa akun mengajukan tautan. **Dua jalur bisa menyelesaikan sebuah permintaan**: admin (`shortlink.approve`) meninjau lalu approve/reject lewat CMS, ATAU PIC membalas notifikasi WhatsApp ("YES"/"NO"/tombol quick-reply) — keduanya melewati mekanisme atomik yang sama (`UPDATE ... WHERE status='pending'`, lihat Arsitektur §12), jadi tidak pernah diproses dobel dari jalur mana pun. Approve membuat baris `ms_shortlink` baru dalam satu transaksi (reuse logic generate-key `shortlink_service`). Notifikasi WhatsApp (via Kirimdev) + email dikirim lewat job queue (Arsitektur §13) — retry otomatis dengan backoff kalau gagal, bukan sekali-coba.

### Publik (tanpa auth)

| Method | Endpoint | Rate Limit | Deskripsi |
|---|---|---|---|
| POST | `/public/shortlink-requests` | 3x / menit / IP | Ajukan permintaan shortlink baru (status awal `pending`) |
| GET | `/public/shortlink-requests/pic` | — | `{ "picName": "...", "picWhatsapp": "..." }` — subset read-only App Settings untuk kartu "Konfirmasi via WhatsApp" di halaman pengajuan; `picWhatsapp` bisa `""` bila belum dikonfigurasi (bukan error) |
| POST | `/public/webhooks/kirimdev` | — | Balasan WhatsApp inbound dari PIC (signature+timestamp HMAC diverifikasi di handler) — **bisa memicu approve/reject** lewat `HandleWhatsAppReply` (jalur approval kedua, Arsitektur §12); selalu `200 OK` kecuali signature gagal |

**`POST /public/shortlink-requests`**
```json
{
  "requesterName": "Ahmad Fadli",
  "requesterEmail": "ahmad@fsldk.id",
  "requesterWhatsapp": "081234567890",
  "destinationURL": "https://fsldk-indonesia.com/berita/artikel-panjang",
  "requestedKey": "acara2026",
  "note": "Untuk poster acara nasional"
}
```
Seluruh field wajib diisi termasuk `requestedKey` & `note` (mengikuti perilaku form referensi). `requesterWhatsapp` dinormalisasi ke format `62xxxxxxxxxx` di service. Fallback generate-key otomatis saat approve tetap ada di kode untuk baris lama yang `requestedKey`-nya masih `NULL`, tapi tidak lagi bisa tercapai lewat submission baru.

**`GET /public/shortlink-requests/pic`** tidak meng-expose endpoint `/settings` penuh ke publik (App Settings tetap CMS-only, Super Admin only, §10a) — hanya dua nilai `shortlink_pic_name`/`shortlink_pic_whatsapp` dari grup `layanan`.

### CMS — ✅🔒 + permission

| Method | Endpoint | Permission | Deskripsi |
|---|---|---|---|
| GET | `/shortlink-requests` | `shortlink.view` | Daftar permintaan (query: `page`, `limit`, `search`, `status=pending\|approved\|rejected`) |
| GET | `/shortlink-requests/:id` | `shortlink.view` | Detail satu permintaan |
| POST | `/shortlink-requests/:id/approve` | `shortlink.approve` | Setujui — buat shortlink baru + notifikasi requester |
| POST | `/shortlink-requests/:id/reject` | `shortlink.approve` | Tolak — `{ "rejectionReason": "..." }` + notifikasi requester |

Approve/Reject menolak (`409 Conflict`) bila permintaan sudah pernah diproses (`status != pending`) — berlaku untuk KEDUA jalur (CMS maupun balasan WhatsApp, Arsitektur §12), bukan cuma jalur CMS. Response `Response` juga menyertakan `reviewedVia` (`"cms"` | `"whatsapp"`) untuk membedakan jalur mana yang menyelesaikan permintaan.

---

## 9. Upload (`/uploads`) — ✅🔒

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

## 10. Dashboard (`/dashboard`) — ✅🔒

| Method | Endpoint | Deskripsi |
|---|---|---|
| GET | `/dashboard/summary` | `{ totalNews, totalArticles, totalUsers }` |

---

## 10a. App Settings (`/settings`) — ✅🔒, Super Admin only

Konfigurasi runtime key-value generik (bukan spesifik satu fitur) — dipakai lintas fitur lewat `setting_service.GetValue(group, key)`, mis. nomor/nama PIC yang menerima notifikasi permintaan shortlink (§8a).

| Method | Endpoint | Permission | Deskripsi |
|---|---|---|---|
| GET | `/settings` | `setting.view` | Daftar seluruh setting (tidak dipaginasi) |
| PUT | `/settings/:id` | `setting.update` | Perbarui `settingValue` satu setting — `{ "settingValue": "..." }` |

---

## 10b. Job Queue (`/job-queue`) — ✅🔒, Super Admin only

Dashboard monitoring antrian pengiriman WhatsApp/email asinkron (Arsitektur §13) — dipakai `shortlinkrequest_service` untuk notifikasi Permintaan Shortlink, didesain reusable untuk fitur lain.

| Method | Endpoint | Permission | Deskripsi |
|---|---|---|---|
| GET | `/job-queue` | `jobqueue.view` | Daftar job (query: `page`, `limit`, `search`, `status=pending\|processing\|completed\|failed`, `queue=whatsapp\|email`) |
| GET | `/job-queue/stats` | `jobqueue.view` | `{ "pending", "delayed", "processing", "stuck", "failed", "completed" }` — jumlah job per bucket |
| GET | `/job-queue/:id` | `jobqueue.view` | Detail satu job |
| POST | `/job-queue/:id/retry` | `jobqueue.retry` | Coba ulang — hanya dari status `failed`, mengembalikan `attempts=0` |
| DELETE | `/job-queue/:id` | `jobqueue.delete` | Hapus — hanya dari status `failed`/`completed` |

Retry/Delete menolak (`409 Conflict`) bila job tidak dalam status yang sesuai.

---

## 11. Sistem (tanpa prefix `/api/v1`)

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
| `shortlink.view/create/update/delete/approve` | Shortlink (+ Permintaan Shortlink) | `event.view/create/update/delete` | Event |
| `comment.view/update/delete` | Komentar | `setting.view/update` | App Settings |
| `jobqueue.view/retry/delete` | Job Queue | | |

`comment.*` beda pola dari modul lain: **tidak ada** `comment.create` (siapa pun yang login+verified boleh berkomentar, tanpa permission apa pun). `comment.view` membuka menu sidebar "Komentar" (moderasi/listing); `comment.update` dan `comment.delete` *action-only* (tanpa menu) dan hanya jadi jalur **tambahan** di atas hak pemilik komentar yang selalu ada — lihat [Arsitektur §11](./ARCHITECTURE.md#11-komentar-kedalaman-balasan-moderasi-dan-mention).

`shortlink.approve` adalah permission terpisah dari `shortlink.create/update/delete` — dipegang **Super Admin & Editor**, bukan Kontributor (§8a). `setting.*` dan `jobqueue.*` **hanya** Super Admin — Editor/Kontributor tidak dapat akses App Settings maupun Job Queue sama sekali (§10a/§10b) — keduanya modul operasional platform, bukan konten editorial.

Role bawaan: **Super Admin** (semua permission), **Editor** (news/article/shortlink/event penuh termasuk `shortlink.approve` + moderasi komentar `comment.view/update/delete`, tanpa `setting.*`/`jobqueue.*`), **Kontributor** (news/article tanpa publish/delete, tanpa shortlink/event/komentar/setting/jobqueue), **Member** (pendaftar publik — bisa berkomentar, tanpa akses CMS apa pun). Detail lengkap lihat [`migrations/0002_seed.up.sql`](../migrations/0002_seed.up.sql), [`0004_shortlink.up.sql`](../migrations/0004_shortlink.up.sql), [`0005_comment.up.sql`](../migrations/0005_comment.up.sql), [`0005_event.up.sql`](../migrations/0005_event.up.sql), [`0006_comment_update_permission.up.sql`](../migrations/0006_comment_update_permission.up.sql), [`0008_setting.up.sql`](../migrations/0008_setting.up.sql), [`0009_shortlink_request.up.sql`](../migrations/0009_shortlink_request.up.sql), [`0010_job_queue.up.sql`](../migrations/0010_job_queue.up.sql), dan [`0011_shortlink_request_whatsapp_reply.up.sql`](../migrations/0011_shortlink_request_whatsapp_reply.up.sql).

> Konten Landing Page (visi/misi/struktur organisasi/kontak) tidak dikelola via API/database — dikelola sebagai teks tetap (hardcoded) langsung di frontend `fsldk-web`.

---

[← Kembali ke README](../README.md) · [Panduan Instalasi](./INSTALLATION.md) · [Arsitektur & Alur Sistem](./ARCHITECTURE.md)
