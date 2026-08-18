-- Puskomda/Puskomnas Verifikator yang masuk ke Portal LDK salah satu LDK di
-- bawahnya (cms-ldk, lewat org-switcher) sebelumnya cuma melihat menu
-- "Dashboard" + "Status Pendataan" (satu-satunya permission LDK-tier yang
-- sudah mereka punya, submission.view dari migration 0007) — jauh lebih
-- sempit dari LDK Admin sendiri. Dikonfirmasi stakeholder (2026-08-18):
-- Puskomda/Puskomnas sebagai tier PENGAWAS di atas LDK semestinya bisa
-- melihat & bertindak selengkap LDK Admin saat masuk ke konteks LDK
-- tersebut (mengisi Pendataan, mengelola Profil LDK, memutuskan Persetujuan
-- Kader) — bukan dibatasi permission role asli mereka sendiri. Ini secara
-- eksplisit MEMBALIK DL-11 (TechSpec) yang sebelumnya melarang Puskomda
-- mereview data Kader individual — DL-11 dianggap tidak berlaku lagi untuk
-- kolom "Persetujuan Kader" per konfirmasi ini.
--
-- Aman dari sisi cakupan data: RequireOrganizationScope/checkOrgAccess yang
-- sudah ada tetap membatasi LDK spesifik mana yang boleh mereka sentuh
-- (Puskomda -> LDK anaknya saja, Puskomnas -> LDK manapun) — migration ini
-- HANYA menambah izin AKSI-nya, bukan cakupan organisasinya.
INSERT IGNORE INTO map_role_permission (roleID, permissionID)
SELECT r.roleID, p.permissionID FROM ms_role r JOIN lk_permission p
WHERE r.roleName IN ('Puskomda Verifikator', 'Puskomnas Verifikator')
  AND p.permissionCode IN (
    'submission.create', 'submission.update', 'submission.cancel',
    'organization.profile.manage', 'submission.review.ldk', 'kader.deactivate'
  );
