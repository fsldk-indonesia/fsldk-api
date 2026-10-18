-- ============================================================
-- FSLDK API — Super Admin: permission Pengurus (0051_pengurus_menu.up.sql)
-- 0051 cuma menggrant pengurus.ldk.manage/puskomda.manage/puskomnas.manage
-- ke LDK Admin/Puskomda Verifikator/Puskomnas Verifikator — lupa Super Admin,
-- sehingga permission-nya tidak lagi superset dari ketiga role kanonik itu
-- (melanggar invarian yang dipakai validateWildcardAgainstRole/
-- maxAllowedTierRank di user_service_impl.go: Super Admin tetap harus lolos
-- hasAllPermissions untuk tiap role kanonik wildcardTierOrder, meski
-- wildcard-nya sendiri dihitung lewat effectiveWildcardTierAccess, bukan
-- kolom ini) — bug-nya: menyimpan wildcardTierAccess LDK/PUSKOMDA/PUSKOMNAS
-- ke akun Super Admin manapun gagal dengan "Role ini belum memiliki semua
-- permission yang dibutuhkan untuk akses lintas tier LDK".
-- Idempoten: aman dijalankan ulang (INSERT IGNORE).
-- ============================================================

INSERT IGNORE INTO map_role_permission (roleID, permissionID)
SELECT r.roleID, p.permissionID FROM ms_role r JOIN lk_permission p
WHERE r.roleName = 'Super Admin'
  AND p.permissionCode IN ('pengurus.ldk.manage', 'pengurus.puskomda.manage', 'pengurus.puskomnas.manage');
