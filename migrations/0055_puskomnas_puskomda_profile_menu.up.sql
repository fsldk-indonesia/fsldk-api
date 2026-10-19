-- ============================================================
-- FSLDK API — Puskomnas Verifikator: akses menu "Profil Puskomda"
-- organization.profile.manage.puskomda (menuLabel "Profil Puskomda", route
-- /cms-puskomda/organization/profile) dibuat di
-- 0050_puskomda_puskomnas_own_profile_menu.up.sql tapi cuma digrant ke
-- Puskomda Verifikator & Super Admin -- Puskomnas Verifikator cuma dapat
-- organization.profile.manage.puskomnas (profil Puskomnas miliknya sendiri),
-- jadi saat masuk Portal Puskomda lewat org-switcher, menu Profil Puskomda
-- tidak muncul sama sekali.
--
-- Sejalan dengan filosofi 0014_puskomda_puskomnas_full_ldk_menu.up.sql &
-- 0054_pengurus_menu_cross_tier_access.up.sql: tier pengawas di atas
-- (Puskomnas) semestinya bisa bertindak selengkap role asli tier di
-- bawahnya (Puskomda) saat masuk ke konteks tier tersebut. Hanya menambah
-- izin AKSES MENU (route guard) -- endpoint simpan tetap digerbang
-- organization.profile.manage yang sudah dimiliki Puskomnas Verifikator
-- sejak 0014, dan cakupan organisasi tetap dibatasi logic scope yang sudah
-- ada (lihat juga komentar 0050 soal permission ini murni identitas menu).
-- Idempoten: aman dijalankan ulang (INSERT IGNORE).
-- ============================================================

INSERT IGNORE INTO map_role_permission (roleID, permissionID)
SELECT r.roleID, p.permissionID FROM ms_role r JOIN lk_permission p
WHERE r.roleName = 'Puskomnas Verifikator'
  AND p.permissionCode = 'organization.profile.manage.puskomda';
