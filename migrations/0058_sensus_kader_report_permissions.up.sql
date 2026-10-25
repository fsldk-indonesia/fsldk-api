-- 0058_sensus_kader_report_permissions.up.sql
-- Laporan Sensus Kader (baru) — Portal LDK dapat akses laporan untuk
-- PERTAMA KALINYA. Puskomda/Puskomnas tidak butuh grant baru untuk tier
-- mereka sendiri (report.region.*/report.national.* yang sudah ada sudah
-- cukup, lihat router.go) — migration ini hanya menambah pasangan LDK-tier
-- dan grant lintas-tier untuk Puskomda/Puskomnas yang masuk ke konteks LDK
-- lewat org-switcher (sama pola dengan 0014_puskomda_puskomnas_full_ldk_menu).
INSERT IGNORE INTO lk_permission (permissionCode, permissionName, moduleName, menuLabel, menuIcon, menuRoute, sortOrder) VALUES
('report.ldk.view',   'Lihat Laporan LDK',  'report', 'Laporan', 'file-bar-chart', '/cms-ldk/reports', 16),
('report.ldk.export', 'Ekspor Laporan LDK', 'report', NULL, NULL, NULL, NULL);

INSERT IGNORE INTO map_role_permission (roleID, permissionID)
SELECT r.roleID, p.permissionID FROM ms_role r JOIN lk_permission p
WHERE r.roleName = 'LDK Admin'
  AND p.permissionCode IN ('report.ldk.view', 'report.ldk.export');

INSERT IGNORE INTO map_role_permission (roleID, permissionID)
SELECT r.roleID, p.permissionID FROM ms_role r JOIN lk_permission p
WHERE r.roleName IN ('Puskomda Verifikator', 'Puskomnas Verifikator')
  AND p.permissionCode IN ('report.ldk.view', 'report.ldk.export');

INSERT IGNORE INTO map_role_permission (roleID, permissionID)
SELECT r.roleID, p.permissionID FROM ms_role r JOIN lk_permission p
WHERE r.roleName = 'Super Admin'
  AND p.permissionCode IN ('report.ldk.view', 'report.ldk.export');
