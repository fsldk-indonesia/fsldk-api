-- Menu "Profil Puskomda"/"Profil Puskomnas" — self-service edit profil
-- organisasi sendiri untuk Puskomda Verifikator & Puskomnas Verifikator,
-- setara dengan "Profil LDK" milik LDK Admin (organization.profile.manage,
-- 0005_organization_access.up.sql).
--
-- Permission code BARU di sini murni untuk identitas MENU (menuLabel/
-- menuRoute per tier — satu lk_permission row cuma bisa punya satu
-- menuRoute, lihat 0010_cms_shell_menu_routes.up.sql) — BUKAN permission
-- baru untuk aksi simpan. Endpoint PUT /organizations/:id tetap digerbang
-- organization.profile.manage seperti sebelumnya, yang sudah dimiliki
-- Puskomda Verifikator & Puskomnas Verifikator sejak
-- 0014_puskomda_puskomnas_full_ldk_menu.up.sql (semula untuk skenario
-- "masuk ke LDK binaan lewat org-switcher"); organization_service.Update()
-- tidak membedakan organizationTypeCode sama sekali, dan
-- RequireOrganizationScope sudah otomatis mengizinkan organisasi sendiri —
-- jadi tidak ada perubahan backend lain yang dibutuhkan.
--
-- sortOrder ditempatkan sebelum "Laporan" di tiap tier (pola sama seperti
-- "Profil LDK" yang berada di baris terakhir menu cms-ldk).
INSERT IGNORE INTO lk_permission (permissionCode, permissionName, moduleName, menuLabel, menuIcon, menuRoute, sortOrder) VALUES
('organization.profile.manage.puskomda',   'Kelola Profil Puskomda',   'organization', 'Profil Puskomda',   'building', '/cms-puskomda/organization/profile', 13),
('organization.profile.manage.puskomnas',  'Kelola Profil Puskomnas',  'organization', 'Profil Puskomnas',  'building', '/cms-puskomnas/organization/profile', 17);

INSERT IGNORE INTO map_role_permission (roleID, permissionID)
SELECT r.roleID, p.permissionID FROM ms_role r JOIN lk_permission p
WHERE r.roleName IN ('Puskomda Verifikator', 'Super Admin')
  AND p.permissionCode = 'organization.profile.manage.puskomda';

INSERT IGNORE INTO map_role_permission (roleID, permissionID)
SELECT r.roleID, p.permissionID FROM ms_role r JOIN lk_permission p
WHERE r.roleName IN ('Puskomnas Verifikator', 'Super Admin')
  AND p.permissionCode = 'organization.profile.manage.puskomnas';
