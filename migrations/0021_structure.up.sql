CREATE TABLE IF NOT EXISTS ms_structure (
    structureID          BIGINT AUTO_INCREMENT PRIMARY KEY,
    batch                VARCHAR(50) NOT NULL         COMMENT 'Angkatan, contoh: "XXXII", "Ke-32"',
    period               VARCHAR(50) NOT NULL         COMMENT 'Periode, contoh: "2025/2026"',
    structureName        VARCHAR(255) NOT NULL        COMMENT 'Nama kepengurusan, contoh: "Kabinet Al-Fatih"',
    structureDescription LONGTEXT NOT NULL            COMMENT 'Deskripsi kepengurusan (rich text)',
    logoImage            VARCHAR(255) NULL            COMMENT 'Path gambar logo lembaga',
    structureImage       VARCHAR(255) NULL            COMMENT 'Path gambar bagan struktur organisasi',
    createdDate          DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    createdBy            BIGINT NULL,
    updatedDate          DATETIME NULL,
    updatedBy            BIGINT NULL,
    INDEX idx_structure_batch (batch),
    INDEX idx_structure_period (period)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- Seed Permissions
INSERT IGNORE INTO lk_permission (permissionCode, permissionName, moduleName, menuLabel, menuIcon, menuRoute, sortOrder) VALUES
('structure.view',   'Lihat Struktur Org',  'structure', 'Struktur Org', 'sitemap', '/cms/structures', 7),
('structure.create', 'Tambah Struktur Org', 'structure', NULL, NULL, NULL, NULL),
('structure.update', 'Ubah Struktur Org',   'structure', NULL, NULL, NULL, NULL),
('structure.delete', 'Hapus Struktur Org',  'structure', NULL, NULL, NULL, NULL);

-- Assign to Super Admin
INSERT IGNORE INTO map_role_permission (roleID, permissionID)
SELECT r.roleID, p.permissionID FROM ms_role r JOIN lk_permission p
WHERE r.roleName = 'Super Admin' AND p.permissionCode LIKE 'structure.%';