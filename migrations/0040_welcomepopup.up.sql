-- ============================================================
-- FSLDK API — Welcome Popup (ms_welcome_popup)
-- Modul singleton: satu baris (id=1) berisi HTML/JS/CSS mentah yang
-- disuntikkan apa adanya ke halaman Beranda publik. Super Admin only —
-- tidak ada sanitasi di mana pun (lihat design spec, bagian Security).
-- Idempoten: aman dijalankan ulang (CREATE TABLE IF NOT EXISTS / INSERT IGNORE).
-- ============================================================

CREATE TABLE IF NOT EXISTS ms_welcome_popup (
    id          TINYINT UNSIGNED PRIMARY KEY,
    isEnabled   BOOLEAN NOT NULL DEFAULT FALSE,
    htmlContent LONGTEXT NULL,
    jsContent   LONGTEXT NULL,
    cssContent  LONGTEXT NULL,
    updatedDate DATETIME NULL,
    updatedBy   BIGINT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

INSERT IGNORE INTO ms_welcome_popup (id, isEnabled) VALUES (1, FALSE);

INSERT IGNORE INTO lk_permission (permissionCode, permissionName, moduleName, menuLabel, menuIcon, menuRoute, sortOrder)
VALUES ('welcomepopup.view',   'Lihat Welcome Popup', 'welcomepopup', 'Welcome Popup', 'megaphone', '/cms/welcome-popup', 98),
       ('welcomepopup.update', 'Ubah Welcome Popup',  'welcomepopup', NULL, NULL, NULL, 98);

INSERT IGNORE INTO map_role_permission (roleID, permissionID)
SELECT r.roleID, p.permissionID FROM ms_role r, lk_permission p
WHERE r.roleName = 'Super Admin' AND p.permissionCode IN ('welcomepopup.view', 'welcomepopup.update');
-- HANYA Super Admin — raw HTML/JS/CSS ini dieksekusi apa adanya di public
-- site tanpa sanitasi, sama seperti setting.* (0008_setting.up.sql).
