# FSLDK API (`fsldk-api`)

REST API untuk Website FSLDK Indonesia, dibangun dengan **Golang (Gin + GORM)** dan **MySQL**, mengikuti [Technical Specification](../.claude/techspec/Technical%20Specification%20-%20FSLDK%20Website.md).

## Arsitektur

Arsitektur berlapis (layered) mengikuti pola **`go-core-api`**: setiap modul memiliki subfolder per layer, dengan pemisahan **interface** (kontrak) dan **`_impl.go`** (implementasi).

```
handler  →  service  →  repository (GORM)  →  MySQL
```

Aturan lapisan yang dijaga konsisten di seluruh modul:

- **`_model/` & `_dto/`** — murni struct data (tanpa function/method apa pun).
- **`_handler/`, `_service/`, `_repository/`** — murni logika (tanpa deklarasi struct data; struct baru yang dibutuhkan diletakkan di `_dto`/`_model`). Satu-satunya struct yang boleh ada di sini adalah receiver DI (`XxxImpl`) pada berkas `_impl.go`, karena itu bagian dari pola interface+implementasi itu sendiri.
- **Repository** mengakses data via **GORM** (`.Table()/.Where()/.Joins()/.Select()`), bukan raw SQL string manual.

Struktur direktori:

```
fsldk-api/
├── main.go                 # Titik masuk
├── router.go                # Dependency injection manual + registrasi route
├── config/                  # Pemuatan konfigurasi (viper)
├── database/                # Koneksi MySQL (GORM)
├── constants/                # Kode error, role, permission, nama tabel
├── assets/                   # Aset runtime (dimuat via os.ReadFile, pola go-core-api)
│   ├── logo-fsldk.png        #   → disematkan inline (CID) pada email
│   └── email_template/       #   → verification.html, password_reset.html
├── base/                     # Fondasi reusable
│   ├── apperror/              # Tipe error terstruktur
│   ├── httphelper/            # Format response standar
│   ├── security/               # bcrypt & token acak
│   ├── token/                  # JWT access & refresh
│   ├── validation/             # Validasi struct
│   ├── appctx/                 # Pembaca identitas dari context
│   ├── dto/                    # DTO umum (paginasi)
│   ├── ptr/                    # Helper konversi ke pointer
│   └── slug/                   # Generator slug
├── middlewares/               # Recovery, CORS, Auth, Verified, Permission, RateLimit
├── pkg/
│   ├── mailer/                 # Pengiriman email (SMTP) — memuat template dari assets/
│   └── googleauth/             # Verifikasi Google ID Token
├── migrations/                 # Skema & seed (embed SQL + runner GORM)
└── modules/                    # Modul fitur — tiap modul contoh struktur (auth):
    └── auth/
        ├── auth_model/          # Struct entitas (mis. EmailToken)
        ├── auth_dto/            # Struct request/response
        ├── auth_repository/     # Interface Repository + RepositoryImpl (GORM)
        ├── auth_service/        # Interface Service + ServiceImpl
        ├── auth_handler/        # Interface Handler + HandlerImpl
        └── router.go            # Registrasi route modul
    # Pola yang sama berlaku pada: user, role, permission, news, article, content, dashboard
```

> **Catatan runtime:** Folder `assets/` dibaca via path relatif (`os.ReadFile("assets/...")`) saat aplikasi berjalan — jalankan `go run .` / binary dari root proyek ini (atau salin folder `assets/` bersebelahan dengan binary saat deploy), persis seperti konvensi `go-core-api`.

## Prasyarat

- Go 1.24+
- MySQL 8.0+ (buat database kosong, mis. `fsldk_db`)
- (Opsional) SMTP untuk verifikasi email & reset password

## Menjalankan

```bash
# 1. Salin & sesuaikan konfigurasi
cp .env.example app.env   # lalu sesuaikan kredensial DB

# 2. Pastikan database sudah dibuat di MySQL
#    CREATE DATABASE fsldk_db CHARACTER SET utf8mb4;

# 3. Jalankan (migration & seed berjalan otomatis saat start)
go run .
# atau: make run
```

Server berjalan pada `http://localhost:8080`. Migration dan seed (role, permission, kategori, konten awal, akun Super Admin) dijalankan otomatis saat aplikasi start.

### Akun Super Admin Awal

| Field | Nilai default | Override via env |
|---|---|---|
| Email | `admin@fsldk-indonesia.com` | `SEED_ADMIN_EMAIL` |
| Password | `Admin@123` | `SEED_ADMIN_PASSWORD` |

> Segera ganti kata sandi setelah login pertama.

### Catatan Email (Pengembangan)

Bila `SMTP_HOST` kosong, tautan verifikasi & reset **dicetak ke log** (tidak dikirim), sehingga alur registrasi tetap dapat diuji tanpa server email.

## Konsep Autentikasi

Mengikuti pola `ldksyahid-app`:

- **Password lokal** (registrasi mandiri) — wajib verifikasi email sebelum akses penuh.
- **Google OAuth** — auto-link ke akun email yang cocok / auto-provision akun baru, langsung terverifikasi. Endpoint verifikasi token diatur via `GOOGLE_TOKENINFO_URL` (default `https://oauth2.googleapis.com/tokeninfo`), tidak di-hardcode di kode.
- Satu akun dapat memiliki keduanya (dual-login); metode ditentukan dari kolom `password` & `googleID` (tanpa enum `authProvider`).

## Ringkasan Endpoint

Base URL: `/api/v1`. Detail lengkap pada [`06-API.md`](../.claude/techspec/06-API.md).

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
