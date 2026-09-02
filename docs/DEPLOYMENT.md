# Deployment — Kantong Amal (Crowdfunding)

Panduan go-live untuk fitur Kantong Amal, ditambahkan Phase 14 development. Untuk setup dasar server (non-Kantong Amal) lihat [INSTALLATION.md](./INSTALLATION.md).

Fitur ini **selalu aktif** sejak revisi 2026-09-01 (item 9 revision-prompt-2.md) — flag `KANTONG_AMAL_ENABLED` yang sebelumnya menggerbang registrasi route sudah **dihapus sepenuhnya** dari kode (config, router, docs). Tidak ada lagi langkah "nyalakan flag setelah checklist tuntas"; seluruh route `campaign`/`donation`/`wallet`/`withdrawal`/`reports/kantong-amal/*` terdaftar tanpa syarat begitu binary berjalan.

## 1. Checklist Registrasi Template WhatsApp (Kirimdev/Meta Business Manager)

Sejak Phase 08, seluruh titik notifikasi memanggil `pkg/kirimdev` dengan **nama template literal** berikut (dikonfirmasi ada di kode, bukan hanya di techspec — lihat `grep` di setiap file). Tanpa registrasi, `kirimdevClient.Deliver()` akan gagal (template not found) untuk semua job WhatsApp, retry 5x lalu masuk dead-letter — pengguna tidak pernah menerima notifikasi apa pun.

**Update 2026-08-30 (revisi round 1)**: alur submission/review campaign dan maker-checker withdrawal dihapus dari produk — 4 template (`campaign_disetujui`, `campaign_revisi`, `campaign_ditolak`, `withdrawal_disetujui`) **sudah tidak dipanggil kode manapun lagi**.

**Update 2026-09-01 (revisi round 2)**: dua template tambahan menjadi dead:
- `kode_otp_kantong_amal` — OTP withdrawal sekarang dikirim **via email** (`pkg/mailer.SendOtpEmail`, ke alamat yang dikonfigurasi di `ms_setting` grup `kantong_amal` key `withdrawal_otp_email`, baris `isHide=true` sehingga tidak muncul di App Settings UI), bukan lagi WhatsApp. Template AUTHENTICATION ini boleh dibiarkan dead di Kirimdev (lihat catatan status di bawah — bahkan belum pernah berhasil tersubmit).
- `rekening_diubah` — alert keamanan "rekening penerima diubah" tidak relevan lagi karena campaign tidak lagi menyimpan rekening penerima sama sekali (dipindah ke saat pengajuan withdrawal, diinput ulang & diverifikasi live tiap kali, tidak ada mekanisme "ganti rekening tersimpan" untuk dialertkan).

Sudah dicoba dihapus dari Kirimdev tapi API mereka **tidak menyediakan endpoint DELETE untuk template** (hanya create/read/edit terdokumentasi di `docs.kirimdev.com/sending/create-templates/`) — seluruh template dead di atas tertinggal di akun Kirimdev tanpa pernah dipakai lagi. **Tindak lanjut manual diperlukan**: hapus lewat Kirimdev dashboard (bukan API), atau biarkan (harmless — template yang tidak pernah dikirim tidak punya konsekuensi selain baris tak terpakai di daftar template).

7 dari 9 template asli masih relevan dan sudah **disubmit ke Kirimdev/Meta** (`POST /v1/{phone_number_id}/templates`, kategori UTILITY, bahasa `id`) dan berstatus **`pending`** — menunggu review Meta, belum `approved`. Isi/wording final tunduk pada hasil review Meta (bisa diminta revisi).

| # | Nama Template | Event | Call site | Params (urutan, terverifikasi dari kode) | Kategori | Status |
|---|---|---|---|---|---|---|
| 1 | `invoice_donasi_kantong_amal` | Donasi dibuat (instruksi QRIS, WhatsApp) | `donation_service_impl.go` `Create()` | nama donor, nominal, nama campaign, link QR | UTILITY | **pending** (submitted) |
| 2 | `donasi_berhasil_kantong_amal` | Donasi lunas (ke donor, WhatsApp) | `donation_service_impl.go` `notifyDonationPaid()` | nama donor, nominal, nama campaign | UTILITY | **pending** (submitted) |
| 3 | `notifikasi_pic_donasi_kantong_amal` | Donasi lunas (ke PIC campaign, WhatsApp) | `donation_service_impl.go` `notifyDonationPaid()` | nominal, nama donor (respect anonymous), nama campaign | UTILITY | **pending** (submitted) |
| 4 | `withdrawal_diajukan` | Withdrawal diajukan (ke PIC campaign, WhatsApp) | `withdrawal_service_impl.go` `Request()` | nominal, nama campaign | UTILITY | **pending** (submitted) |
| 5 | `withdrawal_diproses` | Withdrawal diproses (ke PIC campaign, WhatsApp) | `withdrawal_service_impl.go` `Process()` | nominal, estimasi waktu | UTILITY | **pending** (submitted) |
| 6 | `withdrawal_berhasil` | Withdrawal berhasil cair (ke PIC campaign, WhatsApp) | `withdrawal_service_impl.go` `processCallbackTx()` | nominal, rekening tujuan | UTILITY | **pending** (submitted) |
| 7 | `withdrawal_gagal` | Withdrawal gagal (penolakan gateway/callback — bukan lagi aksi admin, maker-checker dihapus) | `withdrawal_service_impl.go` (gateway reject + callback failed) | nominal, alasan | UTILITY | **pending** (submitted) |

**Template dead/tidak terpakai lagi** (masih ada di akun Kirimdev, tapi tidak dipanggil kode manapun): `campaign_disetujui`, `campaign_revisi`, `campaign_ditolak`, `withdrawal_disetujui` (round 1), `kode_otp_kantong_amal`, `rekening_diubah` (round 2 — lihat catatan di atas).

**Catatan historis template `kode_otp_kantong_amal`** (kategori AUTHENTICATION, sekarang dead — dipertahankan sebagai catatan): berbeda dari template UTILITY lain, kategori `AUTHENTICATION` di Meta **tidak menerima body text bebas** — 2 percobaan dengan `text` custom ditolak (`400 invalid_field_value`/`invalid_request_error`). Percobaan ketiga memakai skema resmi Meta untuk template OTP tapi mendapat `502` dari Kirimdev. Template ini **tidak pernah benar-benar tercipta** di Kirimdev (`404` saat dicek) — sekarang moot karena OTP withdrawal sudah pindah ke email (lihat §1 update round 2 di atas), tidak perlu ditindaklanjuti lagi.

**Email transaksional** (`donation_invoice.html`, `donation_receipt.html`, `otp_kantong_amal.html` — lihat §5): dikirim lewat SMTP langsung (`pkg/mailer`), **bukan** lewat Kirimdev/WhatsApp — tidak perlu registrasi template pihak ketiga, cukup `SMTP_HOST`/`SMTP_USERNAME`/`SMTP_PASSWORD` terisi di production.

**Sebelum go-live**:
- [x] 7 dari 7 template yang masih relevan disubmit ke Kirimdev/Meta (2026-08-30) — status `pending`, menunggu review Meta.
- [ ] Hapus manual 6 template dead (lihat daftar di atas) lewat Kirimdev dashboard — opsional (harmless bila dibiarkan), tapi lebih rapi.
- [ ] Pantau status seluruh 7 template sampai **`approved`** (poll `GET /v1/{phone_number_id}/templates/{name}`) — `pending` BUKAN `approved`.
- [ ] Nama template hasil approval Meta **sama persis** dengan string literal di tabel (bila Meta memaksa nama berbeda saat review, kode Go harus diupdate menyamakan — jangan diam-diam berasumsi sama).
- [ ] `KIRIMDEV_TEMPLATE_LANGUAGE=id` cocok dengan bahasa template yang didaftarkan (sudah cocok — seluruh submission memakai `"language":"id"`).
- [ ] Kirim 1 pesan uji per template lewat Kirimdev dashboard (bukan lewat aplikasi) untuk konfirmasi rendering parameter sebelum smoke test §4.
- [ ] Review wording 7 template WhatsApp yang sudah disubmit (draft otomatis) — belum direview oleh tim/PIC FSLDK.
- [ ] Review wording 3 template email (`donation_invoice.html`, `donation_receipt.html`, `otp_kantong_amal.html`) — draft baru, belum direview PIC FSLDK.
- [ ] Isi baris `ms_setting` (`settingGroup='kantong_amal'`, `settingKey='withdrawal_otp_email'`) dengan alamat email produksi yang benar-benar dipantau tim keuangan — migration `0029` men-seed nilai default `yusufwijaya3@gmail.com`, **wajib diverifikasi/diganti** sebelum go-live production (baris ini sengaja `isHide=true`, tidak muncul di App Settings UI — ubah langsung lewat DB atau endpoint `PUT /settings/:id` bila diekspos internal).

## 2. Revisi Alur Campaign & Withdrawal (2026-08-30, round 1)

Perubahan produk signifikan pasca go-live-readiness Phase 14 — dicatat di sini karena mengubah apa yang perlu diverifikasi di smoke test §4:

- **Campaign**: alur submission/review (`DRAFT→SUBMITTED→APPROVED→PUBLISHED`, endpoint `Submit`/`Review`/`ReviewHistory`, permission `kantong_amal.campaign.review`) **dihapus sepenuhnya**. Campaign kini murni CRUD: `DRAFT→PUBLISHED` langsung (gerbang `kantong_amal.campaign.publish`). `PUBLISHED`/`PAUSED` juga bisa langsung diarsipkan.
- **Withdrawal**: maker-checker (`Approve`/`Reject`, syarat "approver ≠ requester") **dihapus sepenuhnya**. Setelah verifikasi keamanan (password/OTP) lolos, withdrawal langsung `APPROVED` — siap diproses lewat `Process()`. Aksi `Reject` dihapus.
- Migration `0028_kantong_amal_revision.up.sql` merapikan seed permission/menu terkait.

## 3. Revisi Round 2 — Campaign/Donation/Withdrawal Murni CMS (2026-09-01)

Perubahan arsitektur lebih besar, mengikuti model celengan syahid (`ldksyahid-app`):

- **Campaign tidak lagi punya kepemilikan** — `ownerUserID` dan seluruh endpoint `/me/campaigns/...` dihapus. `Create`/`Update`/`Delete` murni permission-gated (`kantong_amal.campaign.create/.update/.delete`), siapapun dengan akses boleh mengelola campaign manapun. `Delete` diblokir oleh guard bisnis (donasi PAID belum ditarik, withdrawal aktif, donasi PENDING aktif).
- **Rekening penerima pindah dari campaign ke withdrawal-request** — kolom `beneficiary*`/`beneficiaryLockedUntil` di `ms_campaign` dibiarkan menganggur (tidak di-drop), tidak dipakai lagi. Tidak ada lagi cooling period 24 jam — rekening tujuan diinput ulang & diverifikasi live (inquiry gateway) setiap pengajuan withdrawal.
- **Withdrawal juga murni CMS** — `/me/campaigns/:id/withdrawals`, `/me/withdrawals/...` dihapus, digantikan `/campaigns/:id/withdrawals` (POST) dan `/withdrawals/:id/...` (cancel/security-verify/process), semua permission-gated (`kantong_amal.withdrawal.request`), bukan lagi harus pengaju asli.
- **Donasi manual/offline** — `POST/PUT/DELETE /donations` (permission `kantong_amal.donation.create/.update/.delete`) mencatat donasi yang tidak lewat Amdigipay/Bisatopup (`gateway='manual'`). Baris ini **tidak pernah** menyentuh `tr_wallet_ledger`, sehingga saldo yang bisa ditarik (`GetBalance`/`ReserveWithdrawal`) otomatis tetap murni dari donasi Bisatopup saja — tidak ada perubahan di `wallet_service` yang diperlukan untuk menjamin ini.
- **Email donasi ganda** — `donation_invoice.html` dikirim saat donasi dibuat (sebelum bayar), `donation_receipt.html` setelah PAID.
- **OTP withdrawal via email** — lihat §1 update round 2.
- **Laporan baru**: `GET /reports/kantong-amal/ledger-global` (debit/kredit global, khusus Bisatopup by construction) dan `GET /reports/kantong-amal/analytics` (distribusi nominal donasi + usia donor + progres campaign).
- Migration `0029_kantong_amal_revision_2.up.sql` menambah kolom campaign baru (province/city/goals/pic/organisasi), kolom `paymentMethod` di `tr_donation`, kolom `isHide` di `ms_setting`, dan permission baru.

## 4. Smoke Test (sandbox)

Jalankan dengan `BISATOPUP_ENV_CROWDFUNDING=dev` (sandbox Bisabiller, **bukan** live) di environment staging:

- [ ] Buat campaign baru lewat CMS (gerbang `campaign.create`, tanpa kepemilikan) → publish langsung dari DRAFT (gerbang `campaign.publish`).
- [ ] Donasi nominal kecil ke campaign tsb → verifikasi WA `invoice_donasi_kantong_amal` **dan** email `donation_invoice` diterima berisi QR sandbox (cek log `[MAILER:DEV]` bila SMTP belum dikonfigurasi sandbox).
- [ ] Callback sandbox PAID diterima → status donasi jadi `PAID`, saldo campaign bertambah sesuai `d.Amount` (net, bukan gross) → WA `donasi_berhasil_kantong_amal` + `notifikasi_pic_donasi_kantong_amal` terkirim, email `donation_receipt` terkirim.
- [ ] Catat 1 donasi manual (`POST /donations`, `paymentMethod=CASH`, `paymentStatus=PAID`) ke campaign yang sama → verifikasi tampil di "collected" tapi **tidak** menambah `GetBalance().availableBalance` (hanya donasi Bisatopup yang menambah saldo bisa ditarik).
- [ ] Ajukan withdrawal (`POST /campaigns/:id/withdrawals`, gerbang `withdrawal.request`) dengan bank code + nomor rekening baru → verifikasi inquiry live berhasil, WA `withdrawal_diajukan` terkirim, status masuk `SECURITY_CHECK`.
- [ ] Minta OTP (`POST /withdrawals/:id/security-verify/otp`) → verifikasi email OTP terkirim ke alamat `ms_setting.withdrawal_otp_email` (bukan WhatsApp); verifikasi kode salah 5x dari SATU challenge memicu 429; verifikasi minta ulang kode ke-4 kalinya (setelah 3 challenge) juga ditolak 429.
- [ ] Verifikasi OTP benar → status withdrawal langsung `APPROVED` → **tidak ada** notifikasi WA di langkah ini.
- [ ] Proses pencairan (gerbang `withdrawal.process`) → `withdrawal_diproses` terkirim → callback disbursement sandbox SUCCESS → `withdrawal_berhasil` terkirim, status withdrawal `SUCCESS`.
- [ ] Coba hapus campaign yang masih punya saldo belum ditarik → harus ditolak (`Unprocessable`); ulangi setelah saldo ditarik habis → berhasil.
- [ ] Cek `GET /reports/kantong-amal/balance`, `.../ledger-global`, `.../analytics`, dan `.../audit-log` (CMS) mencerminkan seluruh transaksi di atas secara akurat.
- [ ] Verifikasi worker queue tidak menumpuk `PENDING` (`tr_job_queue` — dicek lewat `GET /cms/job-queue` existing) selama smoke test berjalan.
- [ ] Verifikasi rate limiter WhatsApp aktual (`JOBQUEUE_WHATSAPP_RATE_PER_MINUTE=8`) tidak dilanggar di bawah beban smoke test.

## 5. Rollback Plan

Fitur finansial **tidak** mengandalkan `git revert` semata — per komponen:

| Komponen | Strategi |
|---|---|
| **Kode aplikasi** | Revert deploy ke binary versi sebelumnya. Modul Kantong Amal murni additive — binary lama tetap berjalan normal walau tidak mengenali tabel/kolom baru (tidak ada FK dari tabel lama ke tabel baru). Sejak flag `KANTONG_AMAL_ENABLED` dihapus (§0), **tidak ada lagi cara mematikan fitur tanpa revert deploy** — bila fitur bermasalah pasca-live, revert ke binary versi sebelumnya adalah satu-satunya jalur cepat, bukan flip flag. |
| **Database** | **Tidak pernah** `DROP TABLE`/`DROP COLUMN` untuk rollback — data finansial nyata mungkin sudah ada. Migration bersifat forward-only; koreksi skema selalu lewat migration baru (`00NN_...`), bukan `down.sql` yang menghapus data. |
| **Withdrawal yang sudah `PROCESSING`** | **Tidak bisa** dibatalkan sepihak (gateway sudah menerima request) — tunggu callback definitif. Withdrawal yang belum `PROCESSING` aman di-`Cancel` (melepas reservasi saldo). |
| **Ledger (`tr_wallet_ledger`)** | **Tidak pernah** di-`UPDATE`/`DELETE` langsung. Koreksi kesalahan selalu lewat **compensating transaction** — entry `ADJUSTMENT_CREDIT`/`ADJUSTMENT_DEBIT` baru yang membalikkan efek entry salah, `referenceType=ADJUSTMENT`, catat `ledgerID` asli + alasan di `note`, dan audit `balance.adjusted` mencatat before/after. `balanceAfter` seluruh entry setelahnya otomatis benar karena dihitung berantai, tidak perlu migrasi data historis. |
| **Donasi manual salah input** | Boleh di-`DELETE`/`PUT` langsung lewat `AdminDelete`/`AdminUpdate` (`gateway='manual'` saja) — beda dari donasi Bisatopup yang merupakan catatan finansial gateway sungguhan dan **ditolak** dihapus/diubah lewat endpoint ini. |
| **Campaign salah hapus** | **Tidak bisa** — hard delete, tidak ada soft-delete/undo. Guard bisnis (donasi/withdrawal aktif) mencegah penghapusan campaign yang masih ada aktivitas finansial, tapi campaign kosong yang terhapus keliru harus dibuat ulang manual. |

## 6. Monitoring 48 Jam Pertama Pasca Go-Live

- [ ] `schema_migrations` tercatat lengkap untuk seluruh migration `0018`–`0029` di production.
- [ ] Worker queue berjalan normal — `tr_job_queue` tidak menumpuk status `PENDING` lebih dari beberapa menit.
- [ ] Rate limiter WhatsApp aktual tidak terlampaui (pantau log `[BISATOPUP:CALLBACK]`/job WhatsApp, tidak ada lonjakan job `whatsapp.send_template` yang stuck `PENDING` karena `ErrRateLimited` reschedule berulang).
- [ ] Reconciliation Report H+1 pertama (`GET /reports/kantong-amal/reconciliation`) dipantau manual — belum dipercaya berjalan otomatis tanpa pengecekan sampai minimal 2-3 siklus berjalan mulus.
- [ ] Tidak ada donasi berstatus `AMOUNT_MISMATCH` yang tidak terjelaskan (jika ada, investigasi manual — bukan auto-resolve).
- [ ] Tidak ada withdrawal `PROCESSING` yang stale melewati `staleProcessingThreshold` (10 menit) tanpa masuk antrian reconcile.
- [ ] Spot-check `tr_finance_audit_log` mencatat aksi finansial sesuai ekspektasi (donation.callback.processed, campaign.deleted, donation.manual_created/updated/deleted, withdrawal process/success, dst.) — tidak ada gap.
- [ ] Notifikasi WhatsApp **dan** email benar-benar diterima end user (bukan hanya job/pemanggilan berstatus sukses di internal) — cross-check manual minimal beberapa sampel nyata di 48 jam pertama, termasuk email OTP withdrawal ke alamat `ms_setting.withdrawal_otp_email`.
- [ ] Saldo campaign yang punya donasi manual dicek manual sekali: "collected" (tampilan publik) mencakup donasi manual, tapi saldo yang bisa ditarik (`GetBalance`) tidak — pastikan tidak ada laporan pengguna bingung soal ini di awal go-live.

## 7. Data Retention (referensi)

`tr_donation`, `tr_withdrawal`, `tr_wallet_ledger`, `tr_finance_audit_log` — **permanen, tidak pernah dihapus** (kebutuhan audit/reconciliation jangka panjang, termasuk PII donor yang melekat pada record finansialnya) — **kecuali** donasi manual (`gateway='manual'`) yang boleh dihapus admin lewat `AdminDelete` (data entri, bukan catatan gateway). `tr_job_queue`/log status final — retensi 90 hari lalu cleanup terjadwal (bukan data finansial primer, ledger tetap sumber kebenaran).
