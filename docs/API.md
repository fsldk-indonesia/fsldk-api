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
| GET | `/users/mention-search` | — (cukup login+terverifikasi) | Cari pengguna aktif untuk autocomplete @mention komentar (query: `q`, `limit` maks. 20, default 8); boleh mencari & mem-mention diri sendiri — lihat [Arsitektur §12](./ARCHITECTURE.md#12-komentar-kedalaman-balasan-moderasi-dan-mention) |
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

`newsImage` tetap berupa string URL — nilainya biasanya hasil unggahan lewat `POST /uploads/image` (lihat §13), bukan ditulis manual. `newsReporter` wajib diisi (byline wartawan/penulis liputan); `newsPublisher` (penerbit) dan `newsEditor` opsional.

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

`articleWriter` wajib diisi (byline penulis); `articleEditor` opsional. `articleImage`/`articlePdf` tetap berupa string URL — nilainya hasil unggahan lewat `POST /uploads/image` / `POST /uploads/document` (lihat §13), bukan ditulis manual.

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
`PUT /events/:id` memakai bentuk body yang sama persis (semua field wajib dikirim ulang, bukan partial update). Semua field tanggal menerima ISO8601 atau `YYYY-MM-DD[ HH:mm[:ss]]`, boleh dikosongkan (`""` → `null`). `eventImage` string URL hasil `POST /uploads/image` (lihat §13), bukan ditulis manual. `eventSlug` dibuat otomatis dari `eventTitle` (unik, re-slug hanya kalau judul berubah).

---

## 6a. Format Keuangan (`/finance-formats`)

Repositori publik template Excel (`.xlsx`) format laporan keuangan, dikelompokkan per **9 kategori tetap** (`lk_finance_format_type`, seed-only — tidak ada CRUD kategori). Berkas diunggah lewat `POST /uploads/document` bersama (§13); `financeformat_service` menambahkan validasi khusus: `fileURL` wajib berakhiran `.xlsx`.

### Publik (tanpa auth)

| Method | Endpoint | Deskripsi |
|---|---|---|
| GET | `/public/finance-formats` | Payload gabungan `{ formatTypes, formats, cpName, cpPhone }` — `formatTypes` = 9 kategori urut `sortOrder` (selalu lengkap, termasuk yang belum punya berkas), `formats` = seluruh berkas `isActive=true` urut `formatTypeID`, lalu `createdDate` DESC; frontend yang mengelompokkan. `cpName`/`cpPhone` dari App Settings grup `format_keuangan` (opsional — string kosong bila belum diisi) |
| GET | `/public/finance-formats/:id/download[/:name]` | Unduh berkas `.xlsx` (hanya `isActive=true`). Disajikan dengan `Content-Disposition: attachment; filename="<fileName>.xlsx"` — nama unduhan **persis** `fileName` yang diinput admin (mis. field `Format RAB` → berkas tersimpan sebagai `Format RAB.xlsx`), `.xlsx` ditambahkan bila belum ada. Berkas fisik di disk tetap memakai nama token acak dari `fileURL`; admin tidak perlu me-rename berkas atau mengubah `fileURL`. Segmen `:name` opsional & hanya dekoratif (slug agar tautan yang disalin enak dibaca); nama berkas selalu diambil dari DB |

### CMS — ✅🔒 + permission

| Method | Endpoint | Permission | Deskripsi |
|---|---|---|---|
| GET | `/finance-formats` | `financeformat.view` | Daftar seluruh format, aktif atau tidak (query: `page`, `limit`, `search` nama file, `formatTypeID`, `dateFrom`/`dateTo` `YYYY-MM-DD`, `sort` — whitelist `fileName`/`createdDate`) |
| GET | `/finance-formats/types` | `financeformat.view` | 9 kategori format tetap (dropdown form) |
| GET | `/finance-formats/:id` | `financeformat.view` | Detail untuk pengelolaan |
| POST | `/finance-formats` | `financeformat.create` | Buat format baru |
| PUT | `/finance-formats/:id` | `financeformat.update` | Perbarui — berkas lama dihapus dari disk (best-effort) bila `fileURL` berubah |
| PATCH | `/finance-formats/:id/publish` | `financeformat.publish` | Body `{ "isActive": bool }` — toggle tampil di halaman publik |
| DELETE | `/finance-formats/:id` | `financeformat.delete` | Hapus baris DB lalu berkas dari disk (best-effort) |

**`POST /finance-formats`** (`PUT` memakai bentuk sama)
```json
{
  "fileName": "Format Arus Kas 2026",
  "fileURL": "http://localhost:8080/uploads/xxx.xlsx",
  "formatTypeID": 1
}
```
`fileURL` wajib hasil `POST /uploads/document` dan berakhiran `.xlsx` (ditolak `400` "Berkas wajib berformat Excel (.xlsx)" bila bukan) serta maks. **10MB** (ditolak `400` "Ukuran berkas melebihi 10MB" — batas khusus modul ini, lebih ketat dari limit dokumen bersama 20MB di §13). `formatTypeID` wajib salah satu dari 9 kategori seed. Diperbolehkan lebih dari satu berkas aktif per kategori (mis. revisi terbaru + arsip).

---

## 6b. FSLDK Goods (`/goods`, `/goods-categories`)

Marketplace/product catalog — **bukan e-commerce**: tidak ada cart/checkout/payment/order. Tombol "Beli Sekarang" di frontend murni redirect ke `purchaseUrl` yang dikonfigurasi per produk (biasanya link WhatsApp). Gambar utama & gallery diunggah lewat `POST /uploads/image` bersama (§13), lalu URL-nya dikirim sebagai bagian body create/update.

### Publik (tanpa auth)

| Method | Endpoint | Deskripsi |
|---|---|---|
| GET | `/public/goods` | Daftar produk `isPublished=true` (query: `page`, `limit`, `search` nama/SKU, `category` slug, `categoryID`, `availability` `available`\|`out_of_stock`\|`coming_soon`, `featured` `1`/`true`, `sort` `newest`\|`name`\|`price_asc`\|`price_desc`\|`featured`) |
| GET | `/public/goods-categories` | Kategori `isActive=true`, urut `sortOrder` |
| GET | `/public/goods/:slug` | Detail produk published (`{ ...produk, images: string[] }`) — 404 bila unpublished/tidak ada |

### CMS — ✅🔒 + permission

| Method | Endpoint | Permission | Deskripsi |
|---|---|---|---|
| GET | `/goods` | `goods.view` | Daftar seluruh produk (query sama seperti publik minus `featured`/`sort` kurasi; `sort` bebas whitelist `goodsName`/`price`/`sortOrder`/`createdDate`) |
| GET | `/goods/:id` | `goods.view` | Detail untuk pengelolaan (termasuk `images`) |
| POST | `/goods` | `goods.create` | Buat produk baru (selalu draft & non-unggulan) |
| PUT | `/goods/:id` | `goods.update` | Perbarui — gambar gallery lama yang sudah tidak dipakai otomatis diganti (replace-all), gambar utama lama dihapus dari disk (best-effort) bila berubah |
| PATCH | `/goods/:id/publish` | `goods.publish` | Body `{ "isPublished": bool }` |
| PATCH | `/goods/:id/featured` | `goods.update` | Body `{ "isFeatured": bool }` |
| DELETE | `/goods/:id` | `goods.delete` | Hapus baris DB, lalu gambar utama + seluruh gallery dari disk (best-effort) |
| GET | `/goods-categories` | `goodscategory.view` | Seluruh kategori (aktif & nonaktif) |
| POST | `/goods-categories` | `goodscategory.create` | Buat kategori baru |
| PUT | `/goods-categories/:id` | `goodscategory.update` | Perbarui (nama/status aktif/urutan) |
| DELETE | `/goods-categories/:id` | `goodscategory.delete` | Ditolak `409` bila kategori masih dipakai produk manapun |

**`POST /goods`** (`PUT` memakai bentuk sama)
```json
{
  "goodsName": "Kaos FSLDK Edisi Munas",
  "skuCode": "GDS-KAOS-01",
  "goodsCategoryID": 1,
  "shortDescription": "Kaos katun combed 30s, unisex.",
  "fullDescription": "<p>Deskripsi lengkap…</p>",
  "price": 120000,
  "mainImageUrl": "http://localhost:8080/uploads/xxx.jpg",
  "imageUrls": ["http://localhost:8080/uploads/yyy.jpg"],
  "availabilityStatus": "available",
  "purchaseUrl": "https://wa.me/6281234567890",
  "purchaseButtonLabel": "Beli Sekarang"
}
```
`purchaseUrl` hanya menerima scheme `http`/`https` (ditolak `400` untuk `javascript:`/`data:`/dsb.). `fullDescription` disanitasi HTML (`bluemonday.UGCPolicy()`) di service layer sebelum disimpan — tidak ada mekanisme sanitasi rich-text lain di project ini, jadi Goods menerapkannya sendiri. `imageUrls` maksimal 10 URL.

---

## 6c. Perpustakaan / Katalog Buku (`/catalog-books`)

| Method | Endpoint | Deskripsi |
|---|---|---|
| GET | `/public/catalog-books` | Daftar buku published (query: `page`, `limit`, `search`, `category`, `language`, `authorType`, `availability`) |
| GET | `/public/catalog-books/:slug` | Detail buku |
| POST | `/public/catalog-books/:id/like` | Like/dukung buku (rate limit 20/5 menit, tanpa auth) |
| GET | `/public/catalog-book-categories` \| `-languages` \| `-author-types` \| `-availability-types` | Daftar nilai referensi untuk filter/form |

### CMS — ✅🔒 + permission

| Method | Endpoint | Permission | Deskripsi |
|---|---|---|---|
| GET | `/catalog-books` | `catalogbook.view` | Daftar seluruh buku (semua status) |
| GET | `/catalog-books/:id` | `catalogbook.view` | Detail untuk pengelolaan |
| POST | `/catalog-books` | `catalogbook.create` | Buat entri buku baru |
| PUT | `/catalog-books/:id` | `catalogbook.update` | Perbarui |
| PATCH | `/catalog-books/:id/publish` | `catalogbook.publish` | Publish/tarik publikasi |
| DELETE | `/catalog-books/:id` | `catalogbook.delete` | Hapus |

---

## 6d. Jadwal Kegiatan (`/schedules`)

Agenda/jadwal kegiatan organisasi (kajian, rapat, daurah, dst. — lihat `constants.ScheduleCategories`), bukan jadwal sholat (jadwal sholat di frontend berasal dari API publik pihak ketiga `api.myquran.com`, tanpa melibatkan backend ini sama sekali).

| Method | Endpoint | Deskripsi |
|---|---|---|
| GET | `/public/schedules` | Daftar jadwal published (query: `page`, `limit`, `category`, `search`) |

### CMS — ✅🔒 + permission

| Method | Endpoint | Permission | Deskripsi |
|---|---|---|---|
| GET | `/schedules` | `schedule.view` | Daftar seluruh jadwal (semua status) |
| GET | `/schedules/:id` | `schedule.view` | Detail untuk pengelolaan |
| POST | `/schedules` | `schedule.create` | Buat jadwal baru (`category` wajib salah satu `ScheduleCategories`) |
| PUT | `/schedules/:id` | `schedule.update` | Perbarui |
| PATCH | `/schedules/:id/publish` | `schedule.publish` | Publish/tarik publikasi |
| DELETE | `/schedules/:id` | `schedule.delete` | Hapus |

---

## 6e. Struktur Organisasi (`/structures`)

Arsip **kepengurusan per periode** (batch/period + nama/deskripsi struktur + logo & gambar bagan) — bukan visi/misi/struktur organisasi statis yang tampil di section "Tentang" Beranda (itu tetap teks hardcoded di frontend, lihat catatan di penutup dokumen ini). Fitur ini murni CRUD sederhana, tanpa status draft/publish.

| Method | Endpoint | Deskripsi |
|---|---|---|
| GET | `/public/structures` | Daftar seluruh struktur kepengurusan (urut periode terbaru) |

### CMS — ✅🔒 + permission

| Method | Endpoint | Permission | Deskripsi |
|---|---|---|---|
| GET | `/structures` | `structure.view` | Daftar untuk pengelolaan |
| GET | `/structures/:id` | `structure.view` | Detail |
| POST | `/structures` | `structure.create` | Buat entri struktur baru — `logoImage`/`structureImage` wajib (hasil `POST /uploads/image`, §13) |
| PUT | `/structures/:id` | `structure.update` | Perbarui |
| DELETE | `/structures/:id` | `structure.delete` | Hapus |

---

## 6f. Galeri (`/galleries`)

Galeri foto per album (satu `gallery` = satu album/event, berisi banyak `photo`).

| Method | Endpoint | Deskripsi |
|---|---|---|
| GET | `/public/galleries` | Daftar album published |
| GET | `/public/galleries/:id` | Detail album |
| GET | `/public/galleries/:id/photos` | Foto dalam album |

### CMS — ✅🔒 + permission

| Method | Endpoint | Permission | Deskripsi |
|---|---|---|---|
| GET | `/galleries` | `gallery.view` | Daftar album (semua status) |
| GET | `/galleries/:id` | `gallery.view` | Detail album |
| POST | `/galleries` | `gallery.create` | Buat album baru |
| PUT | `/galleries/:id` | `gallery.update` | Perbarui album |
| DELETE | `/galleries/:id` | `gallery.delete` | Hapus album (+ seluruh foto di dalamnya) |
| GET | `/galleries/:id/photos` | `gallery.view` | Daftar foto (semua status) untuk pengelolaan |
| POST | `/galleries/:id/photos` | `gallery.update` | Tambah foto (URL hasil `POST /uploads/image`, §13) |
| PUT | `/galleries/:id/photos/:photoID` | `gallery.update` | Perbarui caption/foto |
| DELETE | `/galleries/:id/photos/:photoID` | `gallery.update` | Hapus foto |
| POST | `/galleries/:id/photos/reorder` | `gallery.update` | Susun ulang urutan tampil foto |

---

## 7. Komentar (`/comments`)

Dipakai bersama oleh Artikel, Berita, dan Event — `contentType` (`article`/`news`/`event`) + `contentID` menunjuk ke konten manapun tanpa foreign key (lihat [Arsitektur §12](./ARCHITECTURE.md#12-komentar-kedalaman-balasan-moderasi-dan-mention)). Balasan dibatasi **1 level** (tidak bisa membalas balasan).

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

**`GET /public/shortlink-requests/pic`** tidak meng-expose endpoint `/settings` penuh ke publik (App Settings tetap CMS-only, Super Admin only, §14a) — hanya dua nilai `shortlink_pic_name`/`shortlink_pic_whatsapp` dari grup `layanan`.

### CMS — ✅🔒 + permission

| Method | Endpoint | Permission | Deskripsi |
|---|---|---|---|
| GET | `/shortlink-requests` | `shortlink.view` | Daftar permintaan (query: `page`, `limit`, `search`, `status=pending\|approved\|rejected`) |
| GET | `/shortlink-requests/:id` | `shortlink.view` | Detail satu permintaan |
| POST | `/shortlink-requests/:id/approve` | `shortlink.approve` | Setujui — buat shortlink baru + notifikasi requester |
| POST | `/shortlink-requests/:id/reject` | `shortlink.approve` | Tolak — `{ "rejectionReason": "..." }` + notifikasi requester |

Approve/Reject menolak (`409 Conflict`) bila permintaan sudah pernah diproses (`status != pending`) — berlaku untuk KEDUA jalur (CMS maupun balasan WhatsApp, Arsitektur §12), bukan cuma jalur CMS. Response `Response` juga menyertakan `reviewedVia` (`"cms"` | `"whatsapp"`) untuk membedakan jalur mana yang menyelesaikan permintaan.

---

## 8c. QR Code (`/qrcodes`)

Generator kode QR yang **menggantikan** peran "tautan pendek" dengan gambar QR. Isi QR adalah `destinationURL` **langsung** — TIDAK ada kunci/slug, tidak ada redirect lewat domain ini, tidak ada pelacakan pindaian (kalau ada kunci + redirect, ini sama saja dengan Shortlink §8). Setiap baris menyimpan opsi kustomisasi gambar: warna depan/belakang (`#RRGGBB`), URL ikon tengah (di-upload lewat `POST /uploads/image`), dan teks caption yang dirender di bawah kode. Gambar PNG dirender server (skip2/go-qrcode + compositing) — level koreksi kesalahan dinaikkan otomatis ke tertinggi saat ada ikon tengah.

| Method | Endpoint | Auth | Deskripsi |
|---|---|:---:|---|
| GET | `/public/qrcodes/:id/image` | ❌ | Gambar PNG QR (`Content-Type: image/png`) untuk baris `:id`, sudah menerapkan warna/ikon/caption. Query `size` (piksel sisi area QR, 128–1024, default 512). |

### CMS — ✅🔒 + permission

| Method | Endpoint | Permission | Deskripsi |
|---|---|---|---|
| GET | `/qrcodes` | `qrcode.view` | Daftar QR Code (query: `page`, `limit`, `search`) |
| GET | `/qrcodes/:id` | `qrcode.view` | Detail QR Code |
| POST | `/qrcodes` | `qrcode.create` | Buat QR Code baru |
| PUT | `/qrcodes/:id` | `qrcode.update` | Perbarui tujuan/label/kustomisasi |
| DELETE | `/qrcodes/:id` | `qrcode.delete` | Hapus QR Code |

**`POST /qrcodes`** (bentuk sama untuk `PUT /qrcodes/:id`)
```json
{
  "destinationURL": "https://fsldk-indonesia.com/agenda",
  "label": "Banner Agenda 2026",
  "foregroundColor": "#00933B",
  "backgroundColor": "#FFFFFF",
  "centerIconKey": "instagram",
  "centerIconURL": "data:image/png;base64,iVBOR...",
  "captionText": "Scan untuk info lengkap"
}
```
Hanya `destinationURL` wajib. `foregroundColor`/`backgroundColor` divalidasi `hexcolor` (default `#16211C`/`#FFFFFF` — selaras tema). `captionText` maks 120 karakter.

**Ikon tengah** — dua bentuk `centerIconURL` didukung (maks 40000 char):
- **data URI PNG** — ikon *preset* yang dikomposisi di frontend (kotak putih ber-radius + border & glyph outline berwarna = warna QR). `centerIconKey` menyertai (`fsldk` | `link` | `browser` | `instagram` | `tiktok` | `youtube` | `x` | `facebook`) hanya sebagai penanda supaya form CMS memulihkan pilihan & meng-generate ulang saat warna berubah — backend tidak menafsirkannya.
- **URL `/uploads/...`** — ikon kustom (unggah via `POST /uploads/image`, CMS-only). Rasio aspek asli dipertahankan (tidak digepengkan).

Bentuk lain diabaikan (QR dirender tanpa ikon). Response menyertakan `imageURL` absolut (`{APP_URL}/api/v1/public/qrcodes/{id}/image`).

---

## 8d. Permintaan QR Code (`/qrcode-requests`)

Alur permintaan publik + persetujuan admin di atas modul QR Code (§8c) — cerminan penuh Permintaan Shortlink (§8a), termasuk **dua jalur penyelesaian race-safe** (CMS `qrcode.approve` ATAU balasan WhatsApp PIC via webhook Kirimdev yang sama, `POST /public/webhooks/kirimdev` — di-fan-out ke modul ini bila balasan bukan milik shortlink), transaksi atomik pembuatan `ms_qrcode` saat approve, dan notifikasi WhatsApp + email lewat job queue.

> **Catatan template WhatsApp:** 3 template (`qrcode_request_notice`, `qrcode_approved`, `qrcode_rejected`) harus didaftarkan dulu di Meta/Kirimdev. Sampai itu dilakukan, jalur WhatsApp dorman (enqueue tetap dicoba lalu gagal async) — jalur CMS approve/reject + notifikasi email tidak terpengaruh.

### Publik (tanpa auth)

| Method | Endpoint | Rate Limit | Deskripsi |
|---|---|---|---|
| POST | `/public/qrcode-requests` | 3x / menit / IP | Ajukan permintaan QR Code baru (status awal `pending`) |
| GET | `/public/qrcode-requests/pic` | — | `{ "picName": "...", "picWhatsapp": "..." }` dari `qrcode_pic_name`/`qrcode_pic_whatsapp` grup `layanan`; `picWhatsapp` bisa `""` bila belum dikonfigurasi (bukan error) |

Body `POST /public/qrcode-requests`: `requesterName`, `requesterEmail`, `requesterWhatsapp`, `destinationURL`, `note` (wajib) + `foregroundColor`, `backgroundColor`, `centerIconKey`, `centerIconURL`, `captionText` (opsional — sama seperti §8c). **Tidak ada `requestedKey`.** Form publik punya editor kustomisasi + pratinjau (pratinjau memakai tautan CONTOH `https://fsldk.or.id`, bukan URL tujuan). Ikon di form publik hanya lewat **preset** (data URI — endpoint `/uploads` butuh login). Nilai kustomisasi disimpan di `ms_qrcode_request` dan disalin ke `ms_qrcode` saat approve — barulah QR mengarah ke `destinationURL` asli. Notifikasi WhatsApp ke pemohon berisi **tautan unduh** gambar QR hasil approve (`imageURL?size=1024`).

### CMS — ✅🔒 + permission

| Method | Endpoint | Permission | Deskripsi |
|---|---|---|---|
| GET | `/qrcode-requests` | `qrcode.view` | Daftar permintaan (query: `page`, `limit`, `search`, `status=pending\|approved\|rejected`) |
| GET | `/qrcode-requests/:id` | `qrcode.view` | Detail satu permintaan |
| POST | `/qrcode-requests/:id/approve` | `qrcode.approve` | Setujui — buat baris `ms_qrcode` (warna default, tanpa ikon/caption; admin menyesuaikan di §8c) + notifikasi requester |
| POST | `/qrcode-requests/:id/reject` | `qrcode.approve` | Tolak — `{ "rejectionReason": "..." }` + notifikasi requester |

Approve/Reject menolak (`409 Conflict`) bila `status != pending`. Response menyertakan `reviewedVia` (`"cms"` | `"whatsapp"`) dan `imageURL` (terisi setelah approved).

---

## 8b. Kalkulator Zakat (`/zakat`)

Halaman publik `/kalkulator-zakat` di frontend menghitung 7 jenis zakat sepenuhnya di browser (data & rumus hardcoded). Satu-satunya endpoint backend adalah proxy harga emas Antam ber-cache — tanpa tabel DB, permission, atau menu CMS.

### Publik (tanpa auth)

| Method | Endpoint | Rate limit | Deskripsi |
|---|---|---|---|
| GET | `/public/zakat/gold-price` | `30x / menit / IP` (burst 10) | Harga emas batangan 1 gr (Antam). Query opsional `refresh=1` memaksa fetch ulang ke upstream (dipakai tombol "Perbarui") |

Response `result`: `{ "success": true, "price": 2750000, "source": "antam-live", "cachedAt": "2026-08-31 10:00:00" }`.

- **Selalu `200 OK`** — bahkan saat upstream (`logam-mulia-api`, proyek pihak ketiga tanpa SLA) gagal: `success` menjadi `false`, `price` berisi nilai fallback (`ZAKAT_GOLD_PRICE_FALLBACK`, default `2600000`), `source` menjadi `"fallback"`. Frontend membedakan lewat `success`, bukan status HTTP.
- Hasil sukses di-cache in-memory selama `ZAKAT_GOLD_PRICE_CACHE_MINUTES` (default 60) — satu pemanggilan upstream per jam untuk semua pengunjung. Hasil fallback **tidak** di-cache sebagai "segar": permintaan berikutnya langsung mencoba upstream lagi begitu provider pulih.

---

## 8c. Kantong Amal — Ringkasan

Sistem crowdfunding donasi (empat modul backend terpisah — `campaign`, `donation`, `wallet`, `withdrawal` — dipetakan ke **satu** modul frontend `kantong-amal`, lihat ARCHITECTURE.md/`fsldk-web` §2). Payment gateway: **BisaTopup/Bisabiller** (QRIS), kredensial & alur go-live di [`docs/DEPLOYMENT.md`](./DEPLOYMENT.md). Permission-nya diprefix `kantong_amal.*` walau kode Go modulnya terpisah per entity.

## 8d. Kantong Amal — Campaign (`/campaigns`)

| Method | Endpoint | Deskripsi |
|---|---|---|
| GET | `/public/campaigns` | Daftar campaign published (query: `page`, `limit`, `search`, `category`) |
| GET | `/public/campaigns/:slug` | Detail campaign + progres donasi terkumpul |
| GET | `/public/campaign-categories` | Daftar kategori campaign |

### CMS — ✅🔒 + permission

| Method | Endpoint | Permission | Deskripsi |
|---|---|---|---|
| GET | `/campaigns` | `kantong_amal.campaign.view` | Daftar seluruh campaign (semua status) |
| GET | `/campaigns/lite` | `kantong_amal.campaign.view` | Daftar ringkas (untuk dropdown/pemilihan campaign di form lain) |
| GET | `/campaigns/:id` | `kantong_amal.campaign.view` | Detail untuk pengelolaan |
| POST | `/campaigns` | `kantong_amal.campaign.create` | Buat campaign baru (draft) |
| PUT | `/campaigns/:id` | `kantong_amal.campaign.update` | Perbarui |
| DELETE | `/campaigns/:id` | `kantong_amal.campaign.delete` | Hapus |
| POST | `/campaigns/:id/publish` | `kantong_amal.campaign.publish` | Publikasikan |
| POST | `/campaigns/:id/pause` \| `/resume` \| `/archive` | `kantong_amal.campaign.moderate` | Jeda/lanjutkan/arsipkan penggalangan |

## 8e. Kantong Amal — Donation (`/donations`, `/campaigns/:slug/donate`)

| Method | Endpoint | Deskripsi |
|---|---|---|
| POST | `/public/campaigns/:slug/donate` | Buat donasi (login opsional — donatur anonim didukung), rate limit 30/5 menit. Merespons instruksi pembayaran QRIS BisaTopup |
| GET | `/public/campaigns/:slug/donations` | Donasi terbaru pada campaign tsb (untuk feed "Donasi Terkini") |
| GET | `/public/donations/:publicRef` | Detail donasi via referensi publik (bukan ID DB internal) |
| GET | `/public/donations/:publicRef/receipt.pdf` | Unduh kwitansi PDF |
| GET | `/public/donations/:publicRef/status` | Polling status pembayaran (rate limit 60/10 menit) |
| POST | `/public/payments/callback` | Webhook callback pembayaran dari BisaTopup (signature-verified) |
| GET | `/me/donations` | ✅🔒 (login+verified) Riwayat donasi milik akun sendiri |

### CMS — ✅🔒 + permission

| Method | Endpoint | Permission | Deskripsi |
|---|---|---|---|
| GET | `/donations` | `kantong_amal.donation.view` | Daftar seluruh donasi |
| GET | `/donations/:id` | `kantong_amal.donation.view` | Detail donasi |
| POST | `/donations` | `kantong_amal.donation.create` | Catat donasi manual (mis. transfer di luar sistem) |
| PUT | `/donations/:id` | `kantong_amal.donation.update` | Koreksi data donasi |
| DELETE | `/donations/:id` | `kantong_amal.donation.delete` | Hapus |

## 8f. Kantong Amal — Wallet (`/wallet`) — ✅🔒

Saldo & buku besar (ledger) internal per campaign — read-only dari CMS, ditulis otomatis oleh sistem saat donasi settle/withdrawal disetujui.

| Method | Endpoint | Permission | Deskripsi |
|---|---|---|---|
| GET | `/wallet/balance` | `kantong_amal.wallet.view` | Saldo saat ini |
| GET | `/wallet/ledger` | `kantong_amal.wallet.view` | Riwayat mutasi (debit/kredit) |

## 8g. Kantong Amal — Withdrawal / Penarikan Dana (`/withdrawals`, `/transfer`)

| Method | Endpoint | Auth | Deskripsi |
|---|---|:---:|---|
| POST | `/campaigns/:id/withdrawals` | ✅🔒 `kantong_amal.withdrawal.request` | Ajukan penarikan dana campaign |
| GET | `/transfer/banks` | ✅🔒 (login+verified) | Daftar bank tujuan transfer yang didukung |
| POST | `/transfer/inquiry` | ✅🔒 (login+verified) | Validasi nomor rekening tujuan sebelum submit |
| POST | `/withdrawals/callback/:secret` | Publik (secret di path, bukan header) | Callback status disbursement dari BisaTopup/Bisabiller |

### CMS — ✅🔒 + permission

| Method | Endpoint | Permission | Deskripsi |
|---|---|---|---|
| GET | `/withdrawals` | `kantong_amal.withdrawal.approve` | Daftar permintaan penarikan (**catatan**: kode permission dipertahankan `approve` walau maker-checker sudah dihapus 2026-08-30 — kini menggerbang akses lihat/kelola, bukan aksi approve terpisah) |
| GET | `/withdrawals/:id` | `kantong_amal.withdrawal.approve` | Detail |
| POST | `/withdrawals/:id/cancel` | `kantong_amal.withdrawal.request` | Batalkan permintaan (pemohon) |
| POST | `/withdrawals/:id/security-verify/otp` | `kantong_amal.withdrawal.request` | Kirim OTP verifikasi keamanan (rate limit 1/5 menit) |
| POST | `/withdrawals/:id/security-verify` | `kantong_amal.withdrawal.request` | Verifikasi OTP (rate limit 1/5 menit) |
| POST | `/withdrawals/:id/process` | `kantong_amal.withdrawal.process` | Proses pencairan ke bank tujuan |

## 8h. Kantong Amal — Laporan Keuangan (`/reports/*`) — ✅🔒

Bagian dari modul Go `report` yang sama dengan §12 (Laporan Pendataan), tapi prefix/permission terpisah (`kantong_amal.report.*`/`kantong_amal.audit.*`) karena domainnya finansial, bukan submission.

| Method | Endpoint | Permission | Deskripsi |
|---|---|---|---|
| GET | `/reports/balance` \| `/reports/balance/export` | `kantong_amal.report.view` / `.export` | Laporan saldo |
| GET | `/reports/campaigns` \| `/reports/campaigns/export` | `kantong_amal.report.view` / `.export` | Laporan per campaign |
| GET | `/reports/donations` \| `/reports/donations/export` | `kantong_amal.report.view` / `.export` | Laporan donasi |
| GET | `/reports/withdrawals` \| `/reports/withdrawals/export` | `kantong_amal.report.view` / `.export` | Laporan penarikan dana |
| GET | `/reports/reconciliation` | `kantong_amal.report.view` | Rekonsiliasi saldo vs. wallet gateway (toleransi `BISATOPUP_SETTLEMENT_MINUTES_CROWDFUNDING`) |
| GET | `/reports/ledger-global` | `kantong_amal.report.view` | Buku besar gabungan seluruh campaign |
| GET | `/reports/analytics` | `kantong_amal.report.view` | Ringkasan analitik (tren donasi, dsb.) |
| GET | `/reports/audit-log` | `kantong_amal.audit.view` | Log audit aksi finansial |

---

## 9. Organization (`/organizations`, `/me/organizations`) — ✅🔒

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

## 9a. Kontak — Pesan Masuk (`/contact`)

Kotak masuk pesan dari form "Hubungi Kami" publik — **bukan** section "Kontak" (alamat/media sosial) di Beranda, yang tetap teks hardcoded di frontend (lihat catatan penutup dokumen ini).

| Method | Endpoint | Rate limit | Deskripsi |
|---|---|---|---|
| POST | `/public/contact` | 0.5x/menit/IP (burst 5) | Kirim pesan dari form kontak publik |

### CMS — ✅🔒 + permission

| Method | Endpoint | Permission | Deskripsi |
|---|---|---|---|
| GET | `/contact` | `contact.view` | Daftar pesan masuk (query: `page`, `limit`, `search`, `isRead`) |
| GET | `/contact/:id` | `contact.view` | Detail pesan |
| PATCH | `/contact/:id/read` | `contact.view` | Tandai sudah dibaca |
| POST | `/contact/:id/reply` | `contact.view` | Balas pesan lewat email (`subject`+`message`) |
| DELETE | `/contact/:id` | `contact.delete` | Hapus pesan |

## 9b. Newsletter / Subscriber (`/subscribers`)

| Method | Endpoint | Rate limit | Deskripsi |
|---|---|---|---|
| POST | `/public/subscribers` | 0.5x/menit/IP (burst 5) | Berlangganan newsletter (email) |
| POST | `/public/subscribers/unsubscribe` | 0.5x/menit/IP (burst 5) | Berhenti berlangganan |

### CMS — ✅🔒 + permission

| Method | Endpoint | Permission | Deskripsi |
|---|---|---|---|
| GET | `/subscribers` | `subscription.view` | Daftar subscriber |
| GET | `/subscribers/:id` | `subscription.view` | Detail |
| POST | `/subscribers/bulk` | `subscription.create` | Tambah banyak email sekaligus (impor) |
| PUT | `/subscribers/:id` | `subscription.create` | Perbarui |
| DELETE | `/subscribers/:id` | `subscription.delete` | Hapus |
| POST | `/subscribers/bulk-delete` | `subscription.delete` | Hapus massal |

## 9c. Statistik Jaringan (Publik) (`/network-stats`)

Sepenuhnya publik, tanpa permission/menu CMS — dipakai landing page untuk menampilkan sebaran LDK/Puskomda secara agregat.

| Method | Endpoint | Deskripsi |
|---|---|---|
| GET | `/public/network-stats` | Ringkasan jumlah LDK/Puskomda/Puskomnas aktif secara nasional |
| GET | `/public/network-stats/directory` | Direktori sebaran organisasi (untuk peta/daftar publik) |

---

## 10. Submission Form — Form Builder (`/submission-forms`) — ✅🔒

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

## 11. Submission — Pendataan & Review (`/submissions`, `/kaders`) — ✅🔒

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

## 12. Report (`/reports`) — ✅🔒

Ekspor laporan submission Levelisasi — **bukan** endpoint JSON, response berupa berkas biner (`Content-Disposition: attachment`), sinkron (tanpa job queue). Cakupan data mengikuti cascade organisasi caller yang sama seperti `/submissions`.

| Method | Endpoint | Permission | Deskripsi |
|---|---|---|---|
| GET | `/reports/submissions/export` | `report.region.export` / `report.national.export` (salah satu) | Ekspor Excel/CSV (query: `formCode` wajib, `status` opsional, `format=xlsx\|csv` default `xlsx`) |

Nama berkas: `laporan-{formCode}-{yyyyMMdd-HHmmss}.{ext}`. Setiap ekspor dicatat ke `tr_export_log` (audit trail).

---

## 13. Upload (`/uploads`) — ✅🔒

Unggah berkas gambar/dokumen — dipakai bersama oleh form Artikel/Berita CMS **dan** field `FILE_IMAGE`/`FILE_DOCUMENT` pada pengisian submission (§9), bukan endpoint khusus per-modul.

| Method | Endpoint | Auth | Deskripsi |
|---|---|:---:|---|
| POST | `/uploads/image` | ✅ (login + verified, tanpa permission khusus) | Unggah satu berkas gambar, `multipart/form-data` field `image` |
| POST | `/uploads/document` | ✅ (login + verified, tanpa permission khusus) | Unggah satu berkas dokumen, `multipart/form-data` field `document` |

**`POST /uploads/image`** (multipart/form-data, field `image`) → `{ "url": "http://localhost:8080/uploads/<nama-acak>.jpg" }`
Validasi: ekstensi `jpg`/`jpeg`/`png`/`webp`/`gif`, maksimal 5MB.

**`POST /uploads/document`** (multipart/form-data, field `document`) → `{ "url": "http://localhost:8080/uploads/<nama-acak>.pdf" }`
Validasi: ekstensi `pdf`/`docx`/`xlsx` (docx/xlsx ditambahkan untuk dokumen pendukung submission — lihat §11; naskah Artikel di §5 tetap PDF saja secara konvensi), maksimal 20MB.

Kedua endpoint menyimpan berkas ke `assets/uploads/` dengan nama acak (hex 16 byte + ekstensi asli) dan menyajikannya sebagai berkas statis publik di `/uploads/*`. `url` hasil unggahan inilah yang dikirim sebagai nilai `articleImage`/`newsImage`/`articlePdf` pada `POST`/`PUT` Artikel & Berita — kolom tersebut tetap berupa string URL di database, tidak ada perubahan skema tambahan di luar yang sudah dijelaskan di §5.

---

## 14. Dashboard (`/dashboard`) — ✅🔒

| Method | Endpoint | Deskripsi |
|---|---|---|
| GET | `/dashboard/summary` | Ringkasan **tier-aware** — bentuk response berbeda sesuai `organizationTypeCode` caller |

Response selalu `{ "organizationTypeCode": "LDK\|PUSKOMDA\|PUSKOMNAS", "ldk"?, "puskomda"?, "puskomnas"? }` — hanya satu dari tiga kunci opsional yang terisi:

- **`ldk`**: `{ submissionStatus, lastUpdatedDate?, levelCode?, levelLabel?, kaderPending, kaderActive, recentNotes: [{note, createdDate}] }`
- **`puskomda`**: `{ totalLDK, belumMengisi, menungguVerifikasi, perluRevisi, terverifikasi, totalKaderAktif }`
- **`puskomnas`**: `{ totalLDKNasional, belumMengisi, menungguVerifikasi, perluRevisi, terverifikasi, levelEstablishedCount, totalPuskomda, totalKaderAktifNasional, levelDistribution: [{levelCode, levelLabel, count}], perPuskomda: [{organizationID, organizationName, totalLDK, kaderAktif}] }`

Caller tanpa tier organisasi/wildcard (mis. Kader) mendapat `{organizationTypeCode: ""}` kosong (bukan error) — dashboard memang tidak dirancang untuk tier ini (Kader punya halaman sendiri, lihat modul `submission`).

---

## 14a. App Settings (`/settings`) — ✅🔒, Super Admin only

Konfigurasi runtime key-value generik (bukan spesifik satu fitur) — dipakai lintas fitur lewat `setting_service.GetValue(group, key)`, mis. nomor/nama PIC yang menerima notifikasi permintaan shortlink (§8a).

| Method | Endpoint | Permission | Deskripsi |
|---|---|---|---|
| GET | `/settings` | `setting.view` | Daftar seluruh setting (tidak dipaginasi) |
| PUT | `/settings/:id` | `setting.update` | Perbarui `settingValue` satu setting — `{ "settingValue": "..." }` |

---

## 14b. Job Queue (`/job-queue`) — ✅🔒, Super Admin only

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

## 15. Sistem (tanpa prefix `/api/v1`)

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
| `financeformat.view/create/update/delete/publish` | Format Keuangan | `goods.view/create/update/delete/publish` | FSLDK Goods (Produk) |
| `goodscategory.view/create/update/delete` | FSLDK Goods (Kategori) | | |
| `comment.view/update/delete` | Komentar | `setting.view/update` | App Settings |
| `jobqueue.view/retry/delete` | Job Queue | `organization.create/profile.manage/deactivate` | Organisasi |
| `organization.ldk.list/ldk.list.national/puskomda.list` | Organisasi (daftar) | `submission_form.view/manage` | Form Builder |
| `submission.create/update/cancel/view` | Pendataan (pemilik) | `submission.review.ldk` | Persetujuan Kader (LDK) |
| `submission.review.tier1/approve.tier1` | Verifikasi/Persetujuan Wilayah (Puskomda) | `submission.review.tier2` | Verifikasi Akhir (Puskomnas) |
| `submission.level.establish/publish/reopen/reassess` | Penetapan Level/Publikasi/Koreksi (Puskomnas) | `kader.deactivate` | Nonaktifkan Kader (LDK) |
| `report.region.view/export` | Laporan Wilayah (Puskomda) | `report.national.view/export` | Laporan Nasional (Puskomnas) |
| `catalogbook.view/create/update/delete/publish` | Perpustakaan / Katalog Buku | `schedule.view/create/update/delete/publish` | Jadwal Kegiatan |
| `structure.view/create/update/delete` | Struktur Organisasi (kepengurusan per periode) | `gallery.view/create/update/delete` | Galeri |
| `contact.view/delete` | Kontak (pesan masuk) | `subscription.view/create/delete` | Newsletter / Subscriber |
| `dynamicform.view/create/update/delete/publish/manage.all` | Formulir Dinamis | `kantong_amal.campaign.view/create/update/delete/publish/moderate` | Kantong Amal — Campaign |
| `kantong_amal.donation.view/create/update/delete` | Kantong Amal — Donasi | `kantong_amal.wallet.view` | Kantong Amal — Wallet |
| `kantong_amal.withdrawal.request/approve/process` | Kantong Amal — Penarikan Dana | `kantong_amal.report.view/export`, `kantong_amal.audit.view` | Kantong Amal — Laporan & Audit |

`comment.*` beda pola dari modul lain: **tidak ada** `comment.create` (siapa pun yang login+verified boleh berkomentar, tanpa permission apa pun). `comment.view` membuka menu sidebar "Komentar" (moderasi/listing); `comment.update` dan `comment.delete` *action-only* (tanpa menu) dan hanya jadi jalur **tambahan** di atas hak pemilik komentar yang selalu ada — lihat [Arsitektur §12](./ARCHITECTURE.md#12-komentar-kedalaman-balasan-moderasi-dan-mention).

`shortlink.approve` adalah permission terpisah dari `shortlink.create/update/delete` — dipegang **Super Admin & Editor**, bukan Kontributor (§8a). `setting.*` dan `jobqueue.*` **hanya** Super Admin — Editor/Kontributor tidak dapat akses App Settings maupun Job Queue sama sekali (§14a/§14b) — keduanya modul operasional platform, bukan konten editorial.

Role bawaan pra-proyek: **Super Admin** (semua permission), **Editor** (news/article/shortlink/event/financeformat penuh termasuk `shortlink.approve` + moderasi komentar `comment.view/update/delete`, tanpa `setting.*`/`jobqueue.*`), **Kontributor** (news/article tanpa publish/delete, tanpa shortlink/event/komentar/setting/jobqueue), **Member** (pendaftar publik — bisa berkomentar, tanpa akses CMS apa pun). Detail lengkap lihat [`migrations/0002_seed.up.sql`](../migrations/0002_seed.up.sql), [`0004_shortlink.up.sql`](../migrations/0004_shortlink.up.sql), [`0005_comment.up.sql`](../migrations/0005_comment.up.sql), [`0005_event.up.sql`](../migrations/0005_event.up.sql), [`0006_comment_update_permission.up.sql`](../migrations/0006_comment_update_permission.up.sql), [`0008_setting.up.sql`](../migrations/0008_setting.up.sql), [`0009_shortlink_request.up.sql`](../migrations/0009_shortlink_request.up.sql), [`0010_job_queue.up.sql`](../migrations/0010_job_queue.up.sql), dan [`0011_shortlink_request_whatsapp_reply.up.sql`](../migrations/0011_shortlink_request_whatsapp_reply.up.sql).

Role tambahan modul Submission Dashboard (hierarki organisasi) — satu role per akun, tanpa multi-role:

| Role | Cakupan | Permission utama |
|---|---|---|
| **LDK Admin** | Organisasi sendiri | `submission.create/update/cancel/view`, `submission.review.ldk`, `submission.reassess`, `kader.deactivate`, `organization.profile.manage` (diri sendiri) |
| **Puskomda Verifikator** | Diri + LDK di wilayahnya | `submission.review.tier1/approve.tier1`, `organization.ldk.list`, `report.region.view/export` |
| **Puskomnas Verifikator** | Seluruh organisasi | `submission.review.tier2/level.establish/publish/reopen/reassess`, `organization.ldk.list.national/puskomda.list`, `report.national.view/export` |
| **Kader** | Diri sendiri (bukan cascade organisasi) | `submission.create/update/cancel/view` (form Sensus Kader saja) |

`wildcardTierAccess` (kolom `SET('LDK','PUSKOMDA','PUSKOMNAS')` di `ms_user`) mem-bypass cascade organisasi untuk akun sepert Super Admin — lihat [Arsitektur §11](./ARCHITECTURE.md#11-organization-scope--cascade-access). Detail lengkap seed role/permission modul ini: [`migrations/0005_organization_access.up.sql`](../migrations/0005_organization_access.up.sql) s.d. [`0009_audit_reporting.up.sql`](../migrations/0009_audit_reporting.up.sql).

> Section "Tentang" (visi/misi) & "Kontak" (alamat/media sosial) pada Beranda tetap teks tetap (hardcoded) langsung di frontend `fsldk-web`, **tidak** dikelola via API/database — jangan disamakan dengan modul `structure` (§6e, arsip kepengurusan per periode) atau `contact` (§9a, kotak masuk pesan form "Hubungi Kami"), yang meski namanya mirip adalah fitur CMS asli dan berbeda tujuan.

---

[← Kembali ke README](../README.md) · [Panduan Instalasi](./INSTALLATION.md) · [Arsitektur & Alur Sistem](./ARCHITECTURE.md)
