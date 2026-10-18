-- ============================================================
-- FSLDK API — Puskomda/Puskomnas Verifikator: akses penuh menu Pengurus
-- 0051_pengurus_menu.up.sql cuma menggrant tiap permission Pengurus ke role
-- pemiliknya sendiri (LDK Admin -> pengurus.ldk.manage, Puskomda Verifikator
-- -> pengurus.puskomda.manage, Puskomnas Verifikator -> pengurus.puskomnas.manage),
-- jadi saat Puskomda/Puskomnas Verifikator masuk ke Portal LDK (dan, khusus
-- Puskomnas, juga Portal Puskomda) lewat org-switcher, menu Pengurus tidak
-- muncul sama sekali -- padahal secara data scope mereka sudah boleh melihat
-- & mengubah role pengurus di LDK/Puskomda manapun di bawah mereka
-- (resolveScopedOrganizationIDs + pengurusRoleAllowed sudah pakai
-- caller.OrganizationTypeCode milik role mereka sendiri, yang sudah superset
-- dari tier di bawahnya).
--
-- Sejalan dengan filosofi 0014_puskomda_puskomnas_full_ldk_menu.up.sql:
-- tier pengawas di atas (Puskomda, Puskomnas) semestinya bisa bertindak
-- selengkap role asli tier di bawahnya saat masuk ke konteks tier tersebut,
-- bukan dibatasi permission role aslinya sendiri. Hanya menambah izin AKSES
-- MENU (route guard) -- cakupan organisasi tetap dibatasi
-- resolveScopedOrganizationIDs yang sudah ada.
-- Idempoten: aman dijalankan ulang (INSERT IGNORE).
-- ============================================================

INSERT IGNORE INTO map_role_permission (roleID, permissionID)
SELECT r.roleID, p.permissionID FROM ms_role r JOIN lk_permission p
WHERE r.roleName = 'Puskomnas Verifikator'
  AND p.permissionCode IN ('pengurus.ldk.manage', 'pengurus.puskomda.manage');

INSERT IGNORE INTO map_role_permission (roleID, permissionID)
SELECT r.roleID, p.permissionID FROM ms_role r JOIN lk_permission p
WHERE r.roleName = 'Puskomda Verifikator'
  AND p.permissionCode = 'pengurus.ldk.manage';
