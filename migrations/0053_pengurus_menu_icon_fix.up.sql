-- ============================================================
-- FSLDK API — Pengurus menu: fix menuIcon (0051_pengurus_menu.up.sql)
-- 0051 seeded menuIcon = 'users-round', yang TIDAK terdaftar di
-- fsldk-web ICONS map (shared/icon.component.ts) — jatuh ke fallback
-- ikon tanda tanya di sidebar. 'id-card' dipilih karena belum dipakai
-- sidebar menu manapun (users/shield-check/user-group sudah dipakai
-- Pengguna/Role/Moderasi Campaign — reuse akan membuat ikon menu
-- kembar), dan 'fas fa-id-card' sudah ada di ICONS map.
-- Pakai UPDATE, bukan edit 0051 langsung — 0051 memakai INSERT IGNORE
-- dan sudah tercatat idempoten, jadi edit filenya tidak berefek pada
-- baris yang sudah terlanjur ter-seed dengan nilai lama.
-- Idempoten: aman dijalankan ulang (UPDATE ke nilai yang sama).
-- ============================================================

UPDATE lk_permission SET menuIcon = 'id-card'
WHERE permissionCode IN ('pengurus.ldk.manage', 'pengurus.puskomda.manage', 'pengurus.puskomnas.manage');
