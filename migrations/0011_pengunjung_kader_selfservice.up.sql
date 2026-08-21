-- Pengunjung (role default akun baru, lihat REGISTER_DEFAULT_ROLE) sebelumnya
-- tidak punya permission apapun, termasuk submission.create — artinya tidak
-- ada jalur sama sekali bagi akun baru untuk mendaftar Sensus Kader ("Daftar
-- Kader" di navbar akun, miss-development poin 5). Diberi 3 permission
-- swadaya (self-service) yang sama seperti role Kader (migrations/
-- 0007_submission_core.up.sql) — bukan permission LDK/tier manapun. Endpoint
-- POST /submissions sendiri sudah menolak subjek non-Kader untuk caller tanpa
-- organizationTypeCode=LDK (lihat submission_service_impl.go Create()), jadi
-- pemberian ini tidak membuka form Levelisasi ke Pengunjung.
INSERT IGNORE INTO map_role_permission (roleID, permissionID)
SELECT r.roleID, p.permissionID FROM ms_role r JOIN lk_permission p
WHERE r.roleName = 'Pengunjung' AND p.permissionCode IN ('submission.create','submission.update','submission.cancel','submission.view');
