-- Form Pendataan (Form Builder) dipindah ke CMS Puskomnas di migration 0012
-- (menuRoute-nya), tapi permission submission_form.view/.manage sebelumnya
-- HANYA digrant ke Super Admin (migrations/0006_submission_form_engine.up.sql)
-- — akun Puskomnas Verifikator asli (bukan wildcard) jadi tetap ditolak
-- permissionGuard walau menu-nya sudah muncul di sidebar-nya (miss-
-- development-prompt-3.md: "padahal ada akses puskomnas tetapi gk bisa
-- akses form data"). Puskomnas Verifikator memang pemilik CMS ini sekarang,
-- jadi wajar mereka juga yang mengelola struktur form penilaian nasional.
INSERT IGNORE INTO map_role_permission (roleID, permissionID)
SELECT r.roleID, p.permissionID FROM ms_role r JOIN lk_permission p
WHERE r.roleName = 'Puskomnas Verifikator' AND p.permissionCode IN ('submission_form.view','submission_form.manage');
