-- ============================================================
-- FSLDK API — Fitur Shortlink (pemendek URL)
-- Tabel ms_shortlink + permission shortlink.* + pemetaan role.
-- Idempoten: aman dijalankan ulang (CREATE TABLE IF NOT EXISTS / INSERT IGNORE).
-- ============================================================

CREATE TABLE IF NOT EXISTS ms_shortlink (
    shortLinkID    BIGINT AUTO_INCREMENT PRIMARY KEY,
    shortKey       VARCHAR(30) NOT NULL UNIQUE,
    destinationURL VARCHAR(1000) NOT NULL,
    visitCount     BIGINT NOT NULL DEFAULT 0,
    createdDate    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    createdBy      BIGINT NULL,
    updatedDate    DATETIME NULL,
    updatedBy      BIGINT NULL,
    INDEX idx_shortlink_key (shortKey)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- Permission (menu di sidebar CMS, sortOrder 5 — slot yang ditinggalkan
-- permission content.* yang sudah dihapus)
INSERT IGNORE INTO lk_permission (permissionCode, permissionName, moduleName, menuLabel, menuIcon, menuRoute, sortOrder) VALUES
('shortlink.view',   'Lihat Shortlink',  'shortlink', 'Shortlink', 'link', '/cms/shortlinks', 5),
('shortlink.create', 'Tambah Shortlink', 'shortlink', NULL, NULL, NULL, NULL),
('shortlink.update', 'Ubah Shortlink',   'shortlink', NULL, NULL, NULL, NULL),
('shortlink.delete', 'Hapus Shortlink',  'shortlink', NULL, NULL, NULL, NULL);

-- Super Admin: seluruh shortlink.* (0002_seed.up.sql sudah pernah diterapkan
-- sebelum permission ini ada, jadi wildcard-nya tidak menjangkau baris baru —
-- perlu di-insert eksplisit di sini juga)
INSERT IGNORE INTO map_role_permission (roleID, permissionID)
SELECT r.roleID, p.permissionID FROM ms_role r JOIN lk_permission p
WHERE r.roleName = 'Super Admin' AND p.permissionCode LIKE 'shortlink.%';

-- Editor: seluruh shortlink.*
INSERT IGNORE INTO map_role_permission (roleID, permissionID)
SELECT r.roleID, p.permissionID FROM ms_role r JOIN lk_permission p
WHERE r.roleName = 'Editor' AND p.permissionCode IN (
  'shortlink.view','shortlink.create','shortlink.update','shortlink.delete'
);
