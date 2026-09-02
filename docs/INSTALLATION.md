# Panduan Instalasi — FSLDK API

[← Kembali ke README](../README.md) · [Lihat Arsitektur & Alur Sistem →](./ARCHITECTURE.md) · [Panduan Deployment Kantong Amal →](./DEPLOYMENT.md)

Panduan ini menjelaskan langkah lengkap menyiapkan dan menjalankan `fsldk-api` dari nol, mulai dari prasyarat hingga verifikasi server berjalan.

---

## 1. Prasyarat

| Kebutuhan | Versi | Keterangan |
|---|---|---|
| **Go** | 1.24+ | `go version` untuk memeriksa |
| **MySQL** | 8.0+ | Database kosong, mis. `fsldk_db` |
| **SMTP** | — | Opsional untuk pengembangan (lihat §5); **wajib** untuk produksi (verifikasi email & reset password bergantung padanya) |
| **Google OAuth Client ID** | — | Opsional, hanya diperlukan bila fitur "Masuk dengan Google" diaktifkan |

---

## 2. Kloning & Masuk ke Direktori

```bash
cd C:/Apache24/htdocs/fsldk-app-web/fsldk-api
```

---

## 3. Siapkan Database MySQL

Buat database kosong dengan charset `utf8mb4` (wajib, agar mendukung emoji/karakter Unicode penuh pada konten):

```sql
CREATE DATABASE fsldk_db CHARACTER SET utf8mb4 COLLATE utf8mb4_general_ci;
```

> Tidak perlu membuat tabel secara manual — seluruh skema dibuat otomatis oleh migration saat aplikasi pertama kali dijalankan (lihat §6 dan [`migrations/`](../migrations)).

---

## 4. Konfigurasi Environment

Salin berkas contoh konfigurasi, lalu sesuaikan:

```bash
cp .env.example app.env
```

Buka `app.env` dan sesuaikan minimal bagian berikut:

| Variabel | Contoh | Keterangan |
|---|---|---|
| `DB_HOST` | `127.0.0.1` | Host MySQL |
| `DB_PORT` | `3306` | Port MySQL |
| `DB_NAME` | `fsldk_db` | Nama database dari langkah 3 |
| `DB_USER` | `root` | Pengguna MySQL |
| `DB_PASSWORD` | *(sesuai lokal)* | Kata sandi MySQL |
| `JWT_SECRET` | *(acak, panjang)* | Kunci penandatanganan access token — **wajib diganti**, jangan pakai nilai contoh |
| `JWT_REFRESH_SECRET` | *(acak, panjang, berbeda dari `JWT_SECRET`)* | Kunci refresh token |

Variabel lain (rate limit, masa berlaku token verifikasi, dsb.) sudah memiliki default yang wajar — lihat komentar pada `.env.example` untuk daftar lengkap.

> **Keamanan:** `app.env` sudah masuk `.gitignore` — jangan pernah commit berkas ini karena berisi kredensial.

---

## 5. (Opsional) Konfigurasi SMTP

Diperlukan agar email verifikasi registrasi & reset password benar-benar terkirim.

```env
SMTP_HOST=smtp.gmail.com
SMTP_PORT=587
SMTP_USERNAME=akun-pengirim@gmail.com
SMTP_PASSWORD=app-password-16-digit
MAIL_FROM_ADDRESS=no-reply@fsldk-indonesia.com
MAIL_FROM_NAME=FSLDK Indonesia
```

**Mode pengembangan tanpa SMTP:** bila `SMTP_HOST` dibiarkan kosong, sistem **tidak gagal** — tautan verifikasi/reset akan **dicetak ke log** server alih-alih dikirim, sehingga alur registrasi tetap bisa diuji end-to-end tanpa server email sungguhan.

---

## 5a. (Opsional) Konfigurasi Kirimdev (WhatsApp)

Diperlukan agar notifikasi WhatsApp fitur Permintaan Shortlink (PIC saat ada permintaan baru; requester saat approve/reject) benar-benar terkirim.

```env
KIRIMDEV_API_KEY=kdv_live_xxxxxxxxxxxxxxxxxxxxxxxx
KIRIMDEV_PHONE_NUMBER_ID=1152989091241508
KIRIMDEV_BASE_URL=https://api.kirimdev.com/v1
KIRIMDEV_TEMPLATE_LANGUAGE=id
KIRIMDEV_WEBHOOK_SECRETS=whsec_xxxxxxxxxxxxxxxxxxxxxxxx
```

`KIRIMDEV_WEBHOOK_SECRETS` mendukung banyak nilai (comma-separated) untuk rotasi tanpa downtime — simpan secret lama & baru sekaligus selama masa transisi. Bila `KIRIMDEV_API_KEY` dibiarkan kosong, pengiriman WhatsApp akan gagal secara *best-effort* (dicatat ke log, tidak menggagalkan aksi submit/approve/reject) — lihat [Arsitektur §12](./ARCHITECTURE.md#12-permintaan-shortlink-satu-transaksi-atomik-notifikasi-best-effort). Setelah kredensial diisi, nomor & nama PIC penerima notifikasi diatur lewat halaman **App Settings** di CMS (`/cms/settings`, permission `setting.view`/`setting.update`, khusus Super Admin) — bukan dari `app.env`.

---

## 6. Jalankan Server

```bash
go run .
# atau
make run
```

Saat pertama kali dijalankan, aplikasi otomatis:

1. Terhubung ke MySQL (`DB_HOST`/`DB_NAME` dari `app.env`).
2. Menjalankan seluruh migration di [`migrations/`](../migrations) secara berurutan (idempoten — aman dijalankan berkali-kali):
   - `0001_init.up.sql` — skema seluruh tabel.
   - `0002_seed.up.sql` — role bawaan, permission, kategori berita/artikel.
   - `0003_seed_admin.up.sql` — **1 akun Super Admin awal** (lihat §7).
3. Menyalakan HTTP server pada `APP_HOST:APP_PORT` (default `0.0.0.0:8080`).

Output log akan menampilkan:

```
[migration] diterapkan: 0001_init.up.sql
[migration] diterapkan: 0002_seed.up.sql
[migration] diterapkan: 0003_seed_admin.up.sql
FSLDK API berjalan pada http://0.0.0.0:8080 (env: development)
```

---

## 7. Kredensial Admin FSLDK (Bawaan)

Akun Super Admin awal dibuat otomatis oleh migration [`0003_seed_admin.up.sql`](../migrations/0003_seed_admin.up.sql) — **gunakan ini untuk login pertama kali** ke CMS:

| Field | Nilai |
|---|---|
| **Email** | `noreplyfsldkindonesia@gmail.com` |
| **Password** | `abc123` |
| **Role** | Super Admin (akses penuh) |

> ⚠️ **Wajib diganti** setelah login pertama (`POST /auth/change-password`). Kredensial ini hanya cocok untuk lingkungan pengembangan/staging awal.
>
> Untuk lingkungan produksi, **jangan** memakai kredensial bawaan ini — edit nilai email/password (hash bcrypt baru) di berkas migration tersebut sebelum menjalankannya pertama kali di server produksi, atau buat migration terpisah khusus produksi.

---

## 8. Verifikasi Server Berjalan

```bash
curl http://localhost:8080/health
# {"status":"ok","service":"fsldk-api"}

curl http://localhost:8080/version
# {"name":"fsldk-api","version":"1.0.0","env":"development"}
```

Uji login dengan kredensial admin bawaan:

```bash
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"noreplyfsldkindonesia@gmail.com","password":"abc123"}'
```

Response sukses berisi `accessToken`, `refreshToken`, dan profil pengguna (lihat format lengkap pada [README — Standar Response](../README.md#standar-response)).

---

## 9. (Opsional) Konfigurasi Google OAuth

Jika fitur "Masuk dengan Google" ingin diaktifkan:

1. Buat kredensial OAuth 2.0 Client ID di [Google Cloud Console](https://console.cloud.google.com/apis/credentials).
2. Isi `GOOGLE_CLIENT_ID` pada `app.env`.
3. (Opsional) Batasi domain email yang diizinkan via `GOOGLE_ALLOWED_DOMAINS` (kosongkan untuk mengizinkan semua domain).
4. `GOOGLE_TOKENINFO_URL` sudah memiliki default resmi Google — tidak perlu diubah kecuali untuk kebutuhan pengujian/mock.

---

## 10. Build untuk Produksi

```bash
make build
# menghasilkan binary: fsldk-api.exe
```

> **Penting:** folder [`assets/`](../assets) (template email & logo) dibaca via path relatif saat runtime — pastikan folder ini **disalin bersebelahan dengan binary** saat deploy, atau jalankan binary dari root proyek. Detail lihat [Arsitektur — Pemuatan Aset Runtime](./ARCHITECTURE.md#8-pemuatan-aset-runtime).

---

## Pemecahan Masalah (Troubleshooting)

| Gejala | Kemungkinan Penyebab | Solusi |
|---|---|---|
| `gagal terhubung ke database` | MySQL belum jalan / kredensial salah | Periksa `DB_HOST`/`DB_USER`/`DB_PASSWORD`, pastikan MySQL aktif |
| `gagal menjalankan migration` | Database belum dibuat / user tidak punya hak CREATE TABLE | Buat database (§3), periksa hak akses user MySQL |
| Login admin gagal (`401`) | Migration `0003_seed_admin` belum sempat jalan | Cek log saat start — pastikan migration diterapkan; cek `SELECT * FROM ms_user` |
| Email verifikasi tidak sampai | SMTP belum dikonfigurasi | Cek log server (tautan dicetak di sana) atau isi kredensial SMTP (§5) |
| `assets/email_template/...: no such file` | Binary dijalankan dari luar root proyek | Jalankan dari root `fsldk-api/`, atau salin `assets/` bersebelahan dengan binary |

---

[← Kembali ke README](../README.md) · [Lihat Arsitektur & Alur Sistem →](./ARCHITECTURE.md) · [Panduan Deployment Kantong Amal →](./DEPLOYMENT.md)
