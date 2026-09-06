-- ============================================================
-- FSLDK API — Kelompokkan menu sidebar CMS "Shortlink" & "Permintaan
-- Shortlink" jadi satu grup dropdown "Shortlink", mengikuti pola yang
-- sudah dipakai "Kantong Amal" (lihat fsldk-web
-- src/app/layouts/cms-layout.component.ts SIDEBAR_GROUPS — pengelompokan
-- murni berdasar prefix menuRoute yang sama, bukan kolom/tabel baru).
-- Menyamakan preseden 0028_kantong_amal_revision.up.sql, yang juga
-- memendekkan menuLabel anak pertama grup supaya tidak mengulang nama
-- grup itu sendiri.
-- Idempoten: UPDATE ber-WHERE, aman dijalankan ulang.
-- ============================================================

-- shortlink.view jadi anak pertama grup "Shortlink" (sortOrder tetap 5,
-- menentukan posisi grup di sidebar — lihat komentar SIDEBAR_GROUPS).
UPDATE lk_permission
SET menuLabel = 'Daftar Shortlink', menuRoute = '/cms/shortlink/list'
WHERE permissionCode = 'shortlink.view';

-- shortlink.approve jadi anak kedua grup "Shortlink".
UPDATE lk_permission
SET menuRoute = '/cms/shortlink/permintaan'
WHERE permissionCode = 'shortlink.approve';
