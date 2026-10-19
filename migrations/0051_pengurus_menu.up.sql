-- ============================================================
-- FSLDK API — Pengurus menu (promosi/demosi role kader aktif dalam
-- scope organisasi caller sendiri, tanpa lewat Portal Admin).
-- 3 permissionCode terpisah (satu per tier) karena permissionCode unik
-- dan tiap baris menu butuh menuRoute sendiri — pola sama dengan
-- migrations/0010_cms_shell_menu_routes.up.sql. Endpoint backend-nya
-- SATU saja (modules/submission), dijaga ketiga kode ini sekaligus
-- (any-match) — lihat RequirePermission di modules/submission/router.go.
-- Idempoten: aman dijalankan ulang (INSERT IGNORE).
-- ============================================================

INSERT IGNORE INTO lk_permission (permissionCode, permissionName, moduleName, menuLabel, menuIcon, menuRoute, sortOrder) VALUES
('pengurus.ldk.manage',       'Kelola Pengurus LDK',       'submission', 'Pengurus', 'users-round', '/cms-ldk/pengurus', 15),
('pengurus.puskomda.manage',  'Kelola Pengurus Puskomda',  'submission', 'Pengurus', 'users-round', '/cms-puskomda/pengurus', 15),
('pengurus.puskomnas.manage', 'Kelola Pengurus Puskomnas', 'submission', 'Pengurus', 'users-round', '/cms-puskomnas/pengurus', 15);

INSERT IGNORE INTO map_role_permission (roleID, permissionID)
SELECT r.roleID, p.permissionID FROM ms_role r JOIN lk_permission p ON TRUE
WHERE r.roleName = 'LDK Admin' AND p.permissionCode = 'pengurus.ldk.manage';

INSERT IGNORE INTO map_role_permission (roleID, permissionID)
SELECT r.roleID, p.permissionID FROM ms_role r JOIN lk_permission p ON TRUE
WHERE r.roleName = 'Puskomda Verifikator' AND p.permissionCode = 'pengurus.puskomda.manage';

INSERT IGNORE INTO map_role_permission (roleID, permissionID)
SELECT r.roleID, p.permissionID FROM ms_role r JOIN lk_permission p ON TRUE
WHERE r.roleName = 'Puskomnas Verifikator' AND p.permissionCode = 'pengurus.puskomnas.manage';
