-- ============================================================
-- FSLDK API — App Settings (ms_setting)
-- Modul platform generik: konfigurasi runtime key-value, bukan spesifik
-- shortlink — dibuat lebih dulu dari 0009_shortlink_request.up.sql supaya
-- baris seed PIC shortlink tersedia sebelum fitur request dipakai.
-- Idempoten: aman dijalankan ulang (CREATE TABLE IF NOT EXISTS / INSERT IGNORE).
-- ============================================================

CREATE TABLE IF NOT EXISTS ms_setting (
    settingID    BIGINT AUTO_INCREMENT PRIMARY KEY,
    settingGroup VARCHAR(100) NOT NULL,
    settingKey   VARCHAR(100) NOT NULL,
    settingLabel VARCHAR(255) NOT NULL,
    settingValue TEXT NULL,
    createdDate  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    createdBy    BIGINT NULL,
    updatedDate  DATETIME NULL,
    updatedBy    BIGINT NULL,

    UNIQUE KEY uq_setting_group_key (settingGroup, settingKey)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

INSERT IGNORE INTO ms_setting (settingGroup, settingKey, settingLabel, settingValue) VALUES
  ('layanan', 'shortlink_pic_name', 'Nama Penanggung Jawab Shortlink', 'Admin FSLDK'),
  ('layanan', 'shortlink_pic_whatsapp', 'No. WhatsApp Penanggung Jawab Shortlink (format 62xxxxxxxxxx)', '');

-- App Settings — Superadmin only.
INSERT IGNORE INTO lk_permission (permissionCode, permissionName, moduleName, menuLabel, menuIcon, menuRoute, sortOrder)
VALUES ('setting.view',   'Lihat App Settings', 'setting', 'App Settings', 'settings', '/cms/settings', 99),
       ('setting.update', 'Ubah App Settings',  'setting', NULL, NULL, NULL, 99);

INSERT IGNORE INTO map_role_permission (roleID, permissionID)
SELECT r.roleID, p.permissionID FROM ms_role r, lk_permission p
WHERE r.roleName = 'Super Admin' AND p.permissionCode IN ('setting.view', 'setting.update');
-- HANYA Super Admin — Editor/Kontributor tidak dapat akses App Settings sama sekali
