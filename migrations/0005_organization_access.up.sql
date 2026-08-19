-- ============================================================
-- FSLDK API — Organisasi & Fondasi Kontrol Akses (LDK/Puskomda/Puskomnas)
-- Tabel lk_organization_type, ms_organization; perluasan ms_user dengan
-- organizationID + wildcardTierAccess; role & permission tier organisasi.
-- Idempoten: aman dijalankan ulang (CREATE TABLE IF NOT EXISTS / INSERT IGNORE).
-- ============================================================

CREATE TABLE IF NOT EXISTS lk_organization_type (
    organizationTypeCode VARCHAR(20) PRIMARY KEY,
    typeName              VARCHAR(100) NOT NULL,
    levelOrder             TINYINT NOT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

INSERT IGNORE INTO lk_organization_type (organizationTypeCode, typeName, levelOrder) VALUES
('PUSKOMNAS', 'Puskomnas', 1),
('PUSKOMDA',  'Puskomda',  2),
('LDK',       'LDK',       3);

CREATE TABLE IF NOT EXISTS ms_organization (
    organizationID       BIGINT AUTO_INCREMENT PRIMARY KEY,
    organizationTypeCode VARCHAR(20) NOT NULL,
    parentOrganizationID BIGINT NULL,
    organizationName     VARCHAR(200) NOT NULL,
    organizationCode     VARCHAR(50) NOT NULL UNIQUE,
    provinceName         VARCHAR(100) NULL,
    cityName             VARCHAR(100) NULL,
    contactEmail         VARCHAR(100) NULL,
    contactPhone         VARCHAR(30) NULL,
    isActive             TINYINT(1) NOT NULL DEFAULT 1,
    createdDate          DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    createdBy            BIGINT NULL,
    updatedDate           DATETIME NULL,
    updatedBy             BIGINT NULL,
    CONSTRAINT fk_organization_type FOREIGN KEY (organizationTypeCode) REFERENCES lk_organization_type(organizationTypeCode),
    CONSTRAINT fk_organization_parent FOREIGN KEY (parentOrganizationID) REFERENCES ms_organization(organizationID),
    INDEX idx_organization_parent (parentOrganizationID),
    INDEX idx_organization_type_active (organizationTypeCode, isActive)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- Root tunggal Puskomnas (parentOrganizationID NULL, satu-satunya baris PUSKOMNAS).
INSERT INTO ms_organization (organizationTypeCode, parentOrganizationID, organizationName, organizationCode, isActive, createdDate)
SELECT 'PUSKOMNAS', NULL, 'Puskomnas', 'PUSKOMNAS', 1, NOW()
WHERE NOT EXISTS (SELECT 1 FROM ms_organization WHERE organizationTypeCode = 'PUSKOMNAS');

ALTER TABLE ms_user
    ADD COLUMN organizationID BIGINT NULL AFTER roleID,
    ADD COLUMN wildcardTierAccess SET('LDK','PUSKOMDA','PUSKOMNAS') NULL AFTER organizationID;

ALTER TABLE ms_user
    ADD CONSTRAINT fk_user_organization FOREIGN KEY (organizationID) REFERENCES ms_organization(organizationID),
    ADD INDEX idx_user_organization (organizationID);

-- Akun Super Admin bawaan diberi akses lintas seluruh tier (menggantikan
-- kasus khusus organizationID=NULL yang sebelumnya tersirat).
UPDATE ms_user u
JOIN ms_role r ON r.roleID = u.roleID
SET u.wildcardTierAccess = 'LDK,PUSKOMDA,PUSKOMNAS'
WHERE r.roleName = 'Super Admin';

-- Role tier organisasi baru.
INSERT IGNORE INTO ms_role (roleName, roleDescription, isSystemRole, isActive) VALUES
('LDK Admin', 'Mengisi & mengelola pendataan atas nama satu LDK.', 1, 1),
('Puskomda Verifikator', 'Memverifikasi & menyetujui data LDK di wilayahnya.', 1, 1),
('Puskomnas Verifikator', 'Verifikasi akhir, penetapan level, dan publikasi hasil secara nasional.', 1, 1),
('Kader', 'Mengisi & memantau data sensus kader miliknya sendiri.', 1, 1);

-- Permission modul organization (permission aksi tanpa menu memakai NULL,
-- mengikuti pola shortlink.create/update/delete pada 0004_shortlink.up.sql).
INSERT IGNORE INTO lk_permission (permissionCode, permissionName, moduleName, menuLabel, menuIcon, menuRoute, sortOrder) VALUES
('organization.create',           'Tambah Organisasi',          'organization', NULL, NULL, NULL, NULL),
('organization.deactivate',       'Nonaktifkan Organisasi',     'organization', NULL, NULL, NULL, NULL),
('organization.profile.manage',   'Kelola Profil LDK',          'organization', 'Profil LDK', 'building', '/cms/organization/profile', 10),
('organization.ldk.list',         'Lihat LDK Wilayah',          'organization', 'LDK Wilayah', 'building-2', '/cms/organizations/ldk', 10),
('organization.ldk.list.national','Lihat Seluruh LDK Nasional', 'organization', 'Seluruh LDK Nasional', 'building-2', '/cms/organizations/ldk-nasional', 10),
('organization.puskomda.list',    'Lihat Puskomda',             'organization', 'Puskomda', 'landmark', '/cms/organizations/puskomda', 11),
('user.provision',                'Kelola Pengguna Organisasi', 'user', NULL, NULL, NULL, NULL);

-- Super Admin: seluruh permission baru (wildcard 0002_seed sudah lewat).
INSERT IGNORE INTO map_role_permission (roleID, permissionID)
SELECT r.roleID, p.permissionID FROM ms_role r JOIN lk_permission p
WHERE r.roleName = 'Super Admin'
  AND p.permissionCode IN (
    'organization.create','organization.deactivate','organization.profile.manage',
    'organization.ldk.list','organization.ldk.list.national','organization.puskomda.list',
    'user.provision'
  );

-- LDK Admin: profil LDK sendiri + kelola user LDK sendiri + akses menu Pengguna existing.
INSERT IGNORE INTO map_role_permission (roleID, permissionID)
SELECT r.roleID, p.permissionID FROM ms_role r JOIN lk_permission p
WHERE r.roleName = 'LDK Admin' AND p.permissionCode IN (
  'organization.profile.manage','user.provision','user.view','user.create','user.update'
);

-- Puskomda Verifikator: kelola LDK wilayah + user LDK di bawahnya.
INSERT IGNORE INTO map_role_permission (roleID, permissionID)
SELECT r.roleID, p.permissionID FROM ms_role r JOIN lk_permission p
WHERE r.roleName = 'Puskomda Verifikator' AND p.permissionCode IN (
  'organization.create','organization.deactivate','organization.ldk.list',
  'user.provision','user.view','user.create','user.update'
);

-- Puskomnas Verifikator: kelola Puskomda + LDK nasional + user lintas organisasi.
INSERT IGNORE INTO map_role_permission (roleID, permissionID)
SELECT r.roleID, p.permissionID FROM ms_role r JOIN lk_permission p
WHERE r.roleName = 'Puskomnas Verifikator' AND p.permissionCode IN (
  'organization.create','organization.deactivate','organization.ldk.list.national','organization.puskomda.list',
  'user.provision','user.view','user.create','user.update'
);
