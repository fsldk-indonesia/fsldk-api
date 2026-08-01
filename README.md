# FSLDK API (`fsldk-api`)

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

| Grup | Contoh |
|---|---|
| Auth | `POST /auth/register`, `/auth/login`, `/auth/google`, `GET /auth/email/verify/:token`, `POST /auth/email/resend`, `/auth/refresh-token`, `GET /auth/me` |
| User | `GET/POST /users`, `PUT /users/:id`, `PATCH /users/:id/status`, `POST /users/:id/reset-password` |
| Role | `GET/POST /roles`, `PUT /roles/:id/permissions`, `GET /roles/:id/users` |
| Menu | `GET /me/menus`, `GET /permissions` |
| Berita | `GET /public/news`, `GET /public/news/:slug`, `GET/POST /news`, `PATCH /news/:id/publish` |
| Artikel | `GET /public/articles`, `GET/POST /articles`, `PATCH /articles/:id/publish` |
| Konten | `GET /public/contents`, `GET /public/profile`, `PUT /contents/:key`, `.../organization-structure` |
| Dashboard | `GET /dashboard/summary`, `GET /dashboard/recent-news` |
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
