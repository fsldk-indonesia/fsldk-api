# FSLDK API (`fsldk-api`)

[![CI](https://github.com/fsldk-indonesia/fsldk-api/actions/workflows/ci.yml/badge.svg)](https://github.com/fsldk-indonesia/fsldk-api/actions/workflows/ci.yml)
[![Deploy](https://github.com/fsldk-indonesia/fsldk-api/actions/workflows/deploy.yml/badge.svg)](https://github.com/fsldk-indonesia/fsldk-api/actions/workflows/deploy.yml)

REST API untuk Website FSLDK Indonesia, dibangun dengan **Golang (Gin + GORM)** dan **MySQL**.

📖 **Dokumentasi lengkap:**

| Dokumen | Isi |
|---|---|
| [**docs/INSTALLATION.md**](./docs/INSTALLATION.md) | Panduan instalasi langkah-demi-langkah — prasyarat, konfigurasi, menjalankan server, **kredensial Admin FSLDK**, troubleshooting |
| [**docs/ARCHITECTURE.md**](./docs/ARCHITECTURE.md) | Penjelasan arsitektur & alur sistem — pola berlapis, struktur modul, dependency injection, request lifecycle, alur autentikasi |
| [**docs/API.md**](./docs/API.md) | Referensi lengkap seluruh endpoint REST — request/response, permission, rate limit |

---

## Ringkasan Cepat

Arsitektur berlapis (layered): setiap modul memiliki subfolder per layer (`_model`/`_dto`/`_repository`/`_service`/`_handler`) dengan pemisahan **interface** (kontrak) dan **`_impl.go`** (implementasi), akses data via **GORM**.

```
handler  →  service  →  repository (GORM)  →  MySQL
```

Penjelasan detail struktur direktori, aturan pemisahan struct/logika, dan diagram alur request ada di **[docs/ARCHITECTURE.md](./docs/ARCHITECTURE.md)**.

## Menjalankan Cepat

```bash
cp .env.example app.env   # sesuaikan kredensial DB
# buat database: CREATE DATABASE fsldk_db CHARACTER SET utf8mb4;
go run .                  # migration & seed berjalan otomatis
```

Langkah lengkap + kredensial Admin FSLDK bawaan ada di **[docs/INSTALLATION.md](./docs/INSTALLATION.md)**.

## Konsep Autentikasi

Mengikuti pola `ldksyahid-app` — password lokal (wajib verifikasi email) & Google OAuth (auto-link/auto-provision, langsung terverifikasi) dapat dimiliki bersamaan oleh satu akun. Detail alur lengkap di **[docs/ARCHITECTURE.md §6](./docs/ARCHITECTURE.md#6-konsep-autentikasi)**.

## Ringkasan Endpoint

Base URL: `/api/v1`. Detail lengkap pada [`docs/API.md`](./docs/API.md).

Sekitar 30 modul terdaftar di [`docs/API.md`](./docs/API.md) §1–§15 (termasuk lettered sub-section §6a–§6f, §8a–§8h, §9a–§9c, §14a–§14b) — tabel di bawah cuma cuplikan, bukan daftar lengkap.

| Grup | Contoh |
|---|---|
| Auth | `POST /auth/register`, `/auth/login`, `/auth/google`, `GET /auth/email/verify/:token`, `POST /auth/email/resend`, `/auth/refresh-token`, `GET /auth/me` |
| User / Role / Permission | `GET/POST /users`, `PUT /users/:id`, `GET /users/mention-search` (@mention), `GET/POST /roles`, `PUT /roles/:id/permissions`, `GET /me/menus` |
| Konten editorial | Berita (`/news`), Artikel (`/articles`), Event (`/events`), Perpustakaan (`/catalog-books`), Format Keuangan (`/finance-formats`), FSLDK Goods (`/goods`), Jadwal (`/schedules`), Galeri (`/galleries`), Struktur Organisasi (`/structures`) — pola serupa: `GET /public/<modul>` (list/detail) + CRUD CMS + (untuk sebagian besar) `PATCH .../publish` |
| Komentar | `GET /public/comments`, `POST /comments`, `PUT/DELETE /comments/:id` (pemilik atau `comment.update`/`comment.delete`), `POST /comments/:id/react` |
| Shortlink | `GET /public/shortlinks/:key` (redirect publik), `GET/POST /shortlinks`, `PUT/DELETE /shortlinks/:id`, `/shortlink-requests` (pengajuan publik + approval CMS) |
| Kalkulator Zakat | `GET /public/zakat/gold-price` — satu-satunya endpoint, kalkulasi 7 jenis zakat di browser |
| Kantong Amal (crowdfunding) | Campaign (`/campaigns`), Donasi (`/campaigns/:slug/donate`, `/donations`), Wallet (`/wallet/balance`), Penarikan Dana (`/withdrawals`), Laporan Keuangan (`/reports/balance`, `/reports/reconciliation`, dst.) — lihat [API.md §8c–§8h](./docs/API.md#8c-kantong-amal--ringkasan) |
| Formulir Dinamis | `GET /public/dynamicforms/:slug`, `POST .../submit`, `GET/POST /dynamicforms` (builder CMS), `.../analytics`, `.../gsheet/connect` |
| Organisasi & Pendataan | `/organizations`, `/me/organizations` (hierarki LDK/Puskomda/Puskomnas), `/submission-forms` (form builder Levelisasi/Sensus Kader), `/submissions` + `/kaders` (pengisian/review/persetujuan), `/reports/submissions/export` |
| Kontak & Newsletter | `POST /public/contact` (kotak masuk pesan), `/contact` (CMS), `POST /public/subscribers` (langganan), `/subscribers` (CMS) |
| Statistik Publik | `GET /public/network-stats`, `GET /public/network-stats/directory` |
| Upload | `POST /uploads/image`, `POST /uploads/document` |
| Dashboard | `GET /dashboard/summary` — tier-aware (bentuk response beda per `organizationTypeCode`) |
| App Settings / Job Queue | `/settings`, `/job-queue` — Super Admin only |
| Sistem | `GET /health`, `GET /version` |

## Standar Response

```json
{ "path": "...", "timestamp": "...", "status": "ok|fail", "code": "00", "message": "...", "result": {}, "errors": null }
```

## Perintah

| Perintah | Fungsi |
|---|---|
| `make run` | Menjalankan server |
| `make build` | Membangun binary |
| `make vet` | Analisis statik |
| `make tidy` | Merapikan dependensi |
