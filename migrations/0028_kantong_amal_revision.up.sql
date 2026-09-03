-- Revisi produk (2026-08-30): campaign & withdrawal murni CRUD/permission
-- gate, tanpa alur submission/review campaign atau maker-checker withdrawal
-- terpisah. Migration ini murni additive terhadap seed permission/menu
-- (0018_kantong_amal_lookup.up.sql tidak diubah, sesuai konvensi proyek) —
-- tidak menyentuh data transaksi (donasi/withdrawal/ledger).

-- kantong_amal.campaign.review dan kantong_amal.withdrawal.reject sudah
-- sepenuhnya orphan (tidak ada route yang mereferensikannya lagi) — dihapus
-- bersih beserta role grant-nya. Ini permission/config row, bukan data
-- finansial, jadi hapus langsung aman (bukan tabel transaksi yang wajib
-- permanen).
DELETE mrp FROM map_role_permission mrp
JOIN lk_permission p ON p.permissionID = mrp.permissionID
WHERE p.permissionCode IN ('kantong_amal.campaign.review', 'kantong_amal.withdrawal.reject');

DELETE FROM lk_permission
WHERE permissionCode IN ('kantong_amal.campaign.review', 'kantong_amal.withdrawal.reject');

-- Sidebar CMS Kantong Amal digabung jadi satu grup dropdown ("Kantong
-- Amal" > Campaign/Donasi/Penarikan/Laporan Kantong Amal/Audit Log) —
-- pengelompokan sendiri dikerjakan di frontend (cms-layout.component.ts,
-- tidak ada field "parent group" di skema ini), migration ini hanya
-- merapikan label/route tiap baris agar konsisten dengan konsep baru.
UPDATE lk_permission SET menuLabel = 'Campaign'
WHERE permissionCode = 'kantong_amal.campaign.view';

-- kantong_amal.withdrawal.approve sebelumnya menggerbang aksi "Setujui
-- Penarikan" — sekarang murni menggerbang lihat/kelola daftar withdrawal
-- ("Penarikan" di sidebar); aksi approve terpisah sudah dihapus dari kode
-- (lihat withdrawal_service_impl.go), pencairan kini permission-gated
-- langsung lewat kantong_amal.withdrawal.process tanpa approve dahulu.
UPDATE lk_permission SET
  menuLabel = 'Penarikan',
  menuRoute = '/cms/kantong-amal/penarikan',
  permissionName = 'Kelola Penarikan Saldo'
WHERE permissionCode = 'kantong_amal.withdrawal.approve';

-- Dipersingkat karena akan tampil sebagai child di bawah parent "Kantong
-- Amal" yang sudah menyebut nama modulnya.
UPDATE lk_permission SET menuLabel = 'Audit Log'
WHERE permissionCode = 'kantong_amal.audit.view';

-- Queue Kantong Amal dihilangkan dari sidebar sama sekali (job queue
-- Kantong Amal & shortlink sudah berbagi satu halaman generik "Job Queue"
-- sejak Phase 8/12) — menuRoute NULL membuat baris ini tidak lagi
-- dikembalikan GET /me/menus (lihat permission_repository.MenuByRole,
-- filter menuRoute IS NOT NULL). Permission row & grant-nya tetap
-- dipertahankan (bukan dihapus) untuk berjaga jika suatu saat perlu
-- diaktifkan kembali.
UPDATE lk_permission SET menuRoute = NULL, menuLabel = NULL, menuIcon = NULL
WHERE permissionCode = 'kantong_amal.queue.view';
