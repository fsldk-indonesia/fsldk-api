# Deployment — Kantong Amal (Crowdfunding)

Panduan go-live untuk fitur Kantong Amal, ditambahkan Phase 14 development. Untuk setup dasar server (non-Kantong Amal) lihat [INSTALLATION.md](./INSTALLATION.md).

## 1. Feature Flag

Seluruh route publik/self-service/CMS modul `campaign`, `donation`, `wallet`, `withdrawal`, dan sub-route laporan finance `reports/kantong-amal/*` digerbang satu env var:

```env
KANTONG_AMAL_ENABLED=false   # default — WAJIB tetap false sampai checklist di bawah tuntas
```

Bila `false`, route-route tersebut **tidak didaftarkan sama sekali** ke Gin (404 generik, sama seperti route yang memang tidak pernah ada) — bukan didaftarkan lalu ditolak di handler, jadi tidak membocorkan bahwa fitur "ada tapi dimatikan". Dua scheduler latar belakangnya (`donation.expire_check` via `RunExpireScheduler`, `withdrawal.reconcile_check` via `RunReconcileScheduler`) juga tidak dijalankan. Modul lain (`user`, `role`, `news`, `article`, `event`, `shortlink`, dst.) sama sekali tidak terpengaruh oleh flag ini.

Set `KANTONG_AMAL_ENABLED=true` hanya setelah:
1. Migration `0018`–`0026` sudah dijalankan bersih di database production/staging (`schema_migrations` lengkap).
2. Kredensial BisaTopup/Bisabiller **live** (bukan sandbox) sudah diisi (`BISATOPUP_ENV_CROWDFUNDING=live`, `BISATOPUP_USERNAME_CROWDFUNDING`, `BISATOPUP_PASSWORD_API_CROWDFUNDING`, `BISATOPUP_CALLBACK_DISBURSEMENT_SECRET_CROWDFUNDING`).
3. Kredensial Kirimdev sudah diisi DAN seluruh template WhatsApp §2 di bawah sudah disetujui.
4. Smoke test §3 di bawah selesai di environment sandbox (`BISATOPUP_ENV_CROWDFUNDING=dev`).

## 2. Checklist Registrasi Template WhatsApp (Kirimdev/Meta Business Manager)

Sejak Phase 08, seluruh titik notifikasi memanggil `pkg/kirimdev` dengan **nama template literal** berikut (dikonfirmasi ada di kode, bukan hanya di techspec — lihat `grep` di setiap file). Tanpa registrasi, `kirimdevClient.Deliver()` akan gagal (template not found) untuk semua job WhatsApp, retry 5x lalu masuk dead-letter — pengguna tidak pernah menerima notifikasi apa pun.

**Update 2026-08-30**: 12 dari 13 template sudah **disubmit ke Kirimdev/Meta** (`POST /v1/{phone_number_id}/templates`, kategori UTILITY, bahasa `id`) dan berstatus **`pending`** — menunggu review Meta, belum `approved`. Isi/wording final tunduk pada hasil review Meta (bisa diminta revisi). Param di tabel di bawah sudah diverifikasi ulang langsung terhadap call site kode (bukan disalin dari §14.3 techspec — ditemukan `invoice_donasi_kantong_amal` sebenarnya 4 parameter di kode, techspec mencatat 3).

| # | Nama Template | Event | Call site | Params (urutan, terverifikasi dari kode) | Kategori | Status submit |
|---|---|---|---|---|---|---|
| 1 | `invoice_donasi_kantong_amal` | Donasi dibuat (instruksi QRIS) | `donation_service_impl.go:160` | nama donor, nominal, nama campaign, link QR | UTILITY | **pending** (submitted) |
| 2 | `donasi_berhasil_kantong_amal` | Donasi lunas (ke donor) | `donation_service_impl.go:343` | nama donor, nominal, nama campaign | UTILITY | **pending** (submitted) |
| 3 | `notifikasi_pic_donasi_kantong_amal` | Donasi lunas (ke owner campaign) | `donation_service_impl.go:359` | nominal, nama donor (respect anonymous), nama campaign | UTILITY | **pending** (submitted) |
| 4 | `campaign_disetujui` | Campaign disetujui | `campaign_service_impl.go:394` | nama campaign | UTILITY | **pending** (submitted) |
| 5 | `campaign_revisi` | Campaign perlu revisi | `campaign_service_impl.go:396` | nama campaign, catatan reviewer | UTILITY | **pending** (submitted) |
| 6 | `campaign_ditolak` | Campaign ditolak | `campaign_service_impl.go:398` | nama campaign, catatan reviewer | UTILITY | **pending** (submitted) |
| 7 | `withdrawal_diajukan` | Withdrawal diajukan | `withdrawal_service_impl.go:252` | nominal, nama campaign | UTILITY | **pending** (submitted) |
| 8 | `kode_otp_kantong_amal` | Verifikasi keamanan (OTP) | `withdrawal_service_impl.go:340` | kode OTP, masa berlaku (5 menit) | AUTHENTICATION | **BELUM tersubmit** — lihat catatan di bawah |
| 9 | `withdrawal_disetujui` | Withdrawal disetujui | `withdrawal_service_impl.go:446` | nominal | UTILITY | **pending** (submitted) |
| 10 | `withdrawal_diproses` | Withdrawal diproses | `withdrawal_service_impl.go:546` | nominal, estimasi waktu | UTILITY | **pending** (submitted) |
| 11 | `withdrawal_berhasil` | Withdrawal berhasil cair | `withdrawal_service_impl.go:616` | nominal, rekening tujuan | UTILITY | **pending** (submitted) |
| 12 | `withdrawal_gagal` | Withdrawal gagal/ditolak | `withdrawal_service_impl.go:478,515,623` | nominal, alasan | UTILITY | **pending** (submitted) |
| 13 | `rekening_diubah` | Beneficiary diubah (alert keamanan) | `campaign_service_impl.go:304` | rekening baru (masked), waktu | UTILITY | **pending** (submitted) |

**Catatan template #8 (`kode_otp_kantong_amal`, kategori AUTHENTICATION)**: berbeda dari 12 template lain, kategori `AUTHENTICATION` di Meta **tidak menerima body text bebas** — 2 percobaan dengan `text` custom ditolak (`400 invalid_field_value`/`invalid_request_error`). Percobaan ketiga memakai skema resmi Meta untuk template OTP (`{"type":"BODY","add_security_recommendation":true}` + `{"type":"FOOTER","code_expiration_minutes":5}` + tombol `{"type":"BUTTONS","buttons":[{"type":"OTP","otp_type":"COPY_CODE",...}]}`) tapi mendapat `502` dari Kirimdev (bukan validasi — kemungkinan bug di sisi Kirimdev untuk kombinasi field ini, atau field yang masih salah tapi errornya tidak terungkap jelas). Dikonfirmasi via `GET /v1/{phone}/templates/kode_otp_kantong_amal` bahwa template ini **belum benar-benar tercipta** (404). **Belum diselesaikan di sesi ini** — akses dokumentasi Kirimdev untuk skema AUTHENTICATION tidak sempat terverifikasi lebih lanjut (quota lookup habis). Perlu ditindaklanjuti manual lewat dashboard Kirimdev, atau coba lagi API dengan skema yang dikonfirmasi lebih dulu dari support Kirimdev.

**Sebelum go-live**:
- [x] 12 dari 13 template disubmit ke Kirimdev/Meta (2026-08-30) — status `pending`, menunggu review Meta.
- [ ] **`kode_otp_kantong_amal` (AUTHENTICATION) belum tersubmit** — lihat catatan di atas. Blocker tersendiri untuk withdrawal (OTP tidak akan pernah terkirim tanpa ini).
- [ ] Pantau status seluruh 13 template sampai **`approved`** (poll `GET /v1/{phone_number_id}/templates/{name}`) — `pending` BUKAN `approved`, jangan set `KANTONG_AMAL_ENABLED=true` di production sebelum seluruhnya `approved`.
- [ ] Nama template hasil approval Meta **sama persis** dengan 13 string literal di tabel (bila Meta memaksa nama berbeda saat review, kode Go harus diupdate menyamakan — jangan diam-diam berasumsi sama).
- [ ] `KIRIMDEV_TEMPLATE_LANGUAGE=id` cocok dengan bahasa template yang didaftarkan (sudah cocok — seluruh submission memakai `"language":"id"`).
- [ ] Kirim 1 pesan uji per template lewat Kirimdev dashboard (bukan lewat aplikasi) untuk konfirmasi rendering parameter sebelum smoke test §3.
- [ ] Review wording 12 template yang sudah disubmit (draft otomatis, lihat isi lengkap di `GET /v1/{phone_number_id}/templates/{name}`) — belum direview oleh tim/PIC FSLDK, hanya ditulis untuk memenuhi kebutuhan submit teknis. Sesuaikan bila ada preferensi bahasa/nada yang berbeda (revisi template yang masih `pending` kemungkinan butuh dibuat ulang dengan nama sama setelah `rejected`, atau diedit setelah `approved` dengan batas 1x/24 jam atau 10x/30 hari).

## 3. Smoke Test (sandbox, sebelum flag `true` di production)

Jalankan dengan `BISATOPUP_ENV_CROWDFUNDING=dev` (sandbox Bisabiller, **bukan** live) dan `KANTONG_AMAL_ENABLED=true` di environment staging:

- [ ] Buat campaign baru → submit → approve (CMS) → publish; verifikasi notifikasi WA `campaign_disetujui` diterima.
- [ ] Donasi nominal kecil ke campaign tsb → verifikasi `invoice_donasi_kantong_amal` diterima berisi QR sandbox.
- [ ] Callback sandbox PAID diterima → status donasi jadi `PAID`, saldo campaign bertambah sesuai `d.Amount` (net, bukan gross) → `donasi_berhasil_kantong_amal` + `notifikasi_pic_donasi_kantong_amal` terkirim.
- [ ] Ajukan withdrawal dari saldo tsb → verifikasi `withdrawal_diajukan` terkirim, status masuk `SECURITY_CHECK`.
- [ ] Minta OTP (`kode_otp_kantong_amal` terkirim) → verifikasi kode salah 5x dari SATU challenge memicu 429 (`TooManyRequests`) sesuai Phase 13; verifikasi minta ulang kode ke-4 kalinya (setelah 3 challenge) juga ditolak 429 (batas kumulatif Phase 13).
- [ ] Verifikasi OTP benar → approve (CMS, aktor ≠ requester) → `withdrawal_disetujui` terkirim.
- [ ] Proses pencairan (CMS) → `withdrawal_diproses` terkirim → callback disbursement sandbox SUCCESS → `withdrawal_berhasil` terkirim, status withdrawal `SUCCESS`.
- [ ] Cek `GET /reports/kantong-amal/balance` & `.../audit-log` (CMS) mencerminkan seluruh transaksi di atas secara akurat.
- [ ] Verifikasi worker queue tidak menumpuk `PENDING` (`tr_job_queue` — dicek lewat `GET /cms/job-queue` existing) selama smoke test berjalan.
- [ ] Verifikasi rate limiter WhatsApp aktual (`JOBQUEUE_WHATSAPP_RATE_PER_MINUTE=8`) tidak dilanggar di bawah beban smoke test (load test empirik sudah dilakukan Phase 08, di sini cukup sanity-check tidak ada regresi).

## 4. Rollback Plan

Fitur finansial **tidak** mengandalkan `git revert` semata — per komponen:

| Komponen | Strategi |
|---|---|
| **Kode aplikasi** | Revert deploy ke binary versi sebelumnya. Modul Kantong Amal murni additive — binary lama tetap berjalan normal walau tidak mengenali tabel baru (tidak ada FK dari tabel lama ke tabel baru). |
| **Database** | **Tidak pernah** `DROP TABLE` untuk rollback — data finansial nyata mungkin sudah ada. Migration bersifat forward-only; koreksi skema selalu lewat migration baru (`00NN_...`), bukan `down.sql` yang menghapus data. |
| **Fitur bermasalah pasca-live** | Set `KANTONG_AMAL_ENABLED=false` kembali — route berhenti menerima trafik baru. **Kecuali** endpoint callback (`POST /donations/callback`, `POST /withdrawals/callback/:secret`) yang **wajib tetap gated di jalur berbeda** bila memungkinkan (lihat catatan di bawah) karena transaksi yang sudah berjalan sebelum flag dimatikan tetap butuh menerima callback-nya. |
| **Withdrawal yang sudah `PROCESSING`** | **Tidak bisa** dibatalkan sepihak (gateway sudah menerima request) — tunggu callback definitif. Withdrawal yang belum `PROCESSING` aman di-`Cancel` (melepas reservasi saldo). |
| **Ledger (`tr_wallet_ledger`)** | **Tidak pernah** di-`UPDATE`/`DELETE` langsung. Koreksi kesalahan selalu lewat **compensating transaction** — entry `ADJUSTMENT_CREDIT`/`ADJUSTMENT_DEBIT` baru yang membalikkan efek entry salah, `referenceType=ADJUSTMENT`, catat `ledgerID` asli + alasan di `note`, dan audit `balance.adjusted` mencatat before/after. `balanceAfter` seluruh entry setelahnya otomatis benar karena dihitung berantai, tidak perlu migrasi data historis. |

**Catatan penting soal mematikan flag pasca-live**: implementasi Phase 14 saat ini mengontrol registrasi route lewat SATU flag `KANTONG_AMAL_ENABLED` untuk seluruh modul termasuk callback route. Jika fitur sudah pernah `true` di production dan ada donasi/withdrawal yang sedang berjalan, mematikan flag kembali ke `false` juga akan memutus endpoint callback pembayaran/disbursement untuk transaksi yang sudah terlanjur dibuat. **Sebelum mematikan flag di production yang sudah live**, pastikan tidak ada donasi `PENDING` atau withdrawal `PROCESSING` yang masih menunggu callback — atau terima risiko callback tsb baru terproses setelah flag dinyalakan lagi (job tetap aman tersimpan di gateway, bukan hilang, hanya tertunda). Ini konsisten dengan §18.7 techspec yang mensyaratkan webhook tetap aktif menerima request meski fitur "paused" — kalau kondisi ini terjadi di production sungguhan, evaluasi memisahkan route callback dari flag utama sebagai perbaikan lanjutan, bukan diasumsikan aman diam-diam.

## 5. Monitoring 48 Jam Pertama Pasca Go-Live

- [ ] `schema_migrations` tercatat lengkap untuk seluruh migration `0018`–`0026` di production.
- [ ] Worker queue berjalan normal — `tr_job_queue` tidak menumpuk status `PENDING` lebih dari beberapa menit.
- [ ] Rate limiter WhatsApp aktual tidak terlampaui (pantau log `[BISATOPUP:CALLBACK]`/job WhatsApp, tidak ada lonjakan job `whatsapp.send_template` yang stuck `PENDING` karena `ErrRateLimited` reschedule berulang).
- [ ] Reconciliation Report H+1 pertama (`GET /reports/kantong-amal/reconciliation`) dipantau manual — belum dipercaya berjalan otomatis tanpa pengecekan sampai minimal 2-3 siklus berjalan mulus.
- [ ] Tidak ada donasi berstatus `AMOUNT_MISMATCH` yang tidak terjelaskan (jika ada, investigasi manual — bukan auto-resolve).
- [ ] Tidak ada withdrawal `PROCESSING` yang stale melewati `staleProcessingThreshold` (10 menit) tanpa masuk antrian reconcile.
- [ ] Spot-check `tr_finance_audit_log` mencatat aksi finansial sesuai ekspektasi (donation.callback.processed, withdrawal approval/process/success, dst.) — tidak ada gap.
- [ ] Notifikasi WhatsApp benar-benar diterima end user (bukan hanya job berstatus `SUCCESS` di internal) — cross-check manual minimal beberapa sampel nyata di 48 jam pertama.

## 6. Data Retention (referensi)

`tr_donation`, `tr_withdrawal`, `tr_wallet_ledger`, `tr_finance_audit_log` — **permanen, tidak pernah dihapus** (kebutuhan audit/reconciliation jangka panjang, termasuk PII donor yang melekat pada record finansialnya). `tr_job_queue`/log status final — retensi 90 hari lalu cleanup terjadwal (bukan data finansial primer, ledger tetap sumber kebenaran).
