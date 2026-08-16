-- ============================================================
-- FSLDK API — Audit Trail Tambahan & Reporting
-- tr_form_audit_log, tr_user_audit_log, tr_export_log (Section 29/30).
-- Idempoten: aman dijalankan ulang (CREATE TABLE IF NOT EXISTS / INSERT IGNORE).
-- ============================================================

CREATE TABLE IF NOT EXISTS tr_form_audit_log (
    logID               BIGINT AUTO_INCREMENT PRIMARY KEY,
    actorUserID         BIGINT NOT NULL,
    actorOrganizationID BIGINT NULL,
    action              VARCHAR(50) NOT NULL,
    entity              VARCHAR(50) NOT NULL,
    entityID            BIGINT NOT NULL,
    beforeJSON          TEXT NULL,
    afterJSON           TEXT NULL,
    ipAddress           VARCHAR(45) NULL,
    metadata            TEXT NULL,
    createdDate         DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_form_audit_entity (entity, entityID)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS tr_user_audit_log (
    logID               BIGINT AUTO_INCREMENT PRIMARY KEY,
    actorUserID         BIGINT NOT NULL,
    actorOrganizationID BIGINT NULL,
    action              VARCHAR(50) NOT NULL,
    entity              VARCHAR(50) NOT NULL,
    entityID            BIGINT NOT NULL,
    beforeJSON          TEXT NULL,
    afterJSON           TEXT NULL,
    ipAddress           VARCHAR(45) NULL,
    metadata            TEXT NULL,
    createdDate         DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_user_audit_entity (entity, entityID)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS tr_export_log (
    logID               BIGINT AUTO_INCREMENT PRIMARY KEY,
    actorUserID         BIGINT NOT NULL,
    actorOrganizationID BIGINT NULL,
    action              VARCHAR(50) NOT NULL,
    entity              VARCHAR(50) NOT NULL,
    entityID            BIGINT NOT NULL DEFAULT 0,
    beforeJSON          TEXT NULL,
    afterJSON           TEXT NULL,
    ipAddress           VARCHAR(45) NULL,
    metadata            TEXT NULL,
    createdDate         DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_export_audit_actor (actorUserID, createdDate)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

INSERT IGNORE INTO lk_permission (permissionCode, permissionName, moduleName, menuLabel, menuIcon, menuRoute, sortOrder) VALUES
('report.region.view',     'Lihat Laporan Wilayah',   'report', 'Laporan Wilayah',   'file-bar-chart', '/cms/reports/wilayah', 17),
('report.region.export',   'Ekspor Laporan Wilayah',  'report', NULL, NULL, NULL, NULL),
('report.national.view',   'Lihat Laporan Nasional',  'report', 'Laporan Nasional',  'file-bar-chart', '/cms/reports/nasional', 18),
('report.national.export', 'Ekspor Laporan Nasional', 'report', NULL, NULL, NULL, NULL);

INSERT IGNORE INTO map_role_permission (roleID, permissionID)
SELECT r.roleID, p.permissionID FROM ms_role r JOIN lk_permission p
WHERE r.roleName = 'Super Admin' AND p.permissionCode IN (
  'report.region.view','report.region.export','report.national.view','report.national.export'
);

INSERT IGNORE INTO map_role_permission (roleID, permissionID)
SELECT r.roleID, p.permissionID FROM ms_role r JOIN lk_permission p
WHERE r.roleName = 'Puskomda Verifikator' AND p.permissionCode IN ('report.region.view','report.region.export');

INSERT IGNORE INTO map_role_permission (roleID, permissionID)
SELECT r.roleID, p.permissionID FROM ms_role r JOIN lk_permission p
WHERE r.roleName = 'Puskomnas Verifikator' AND p.permissionCode IN ('report.national.view','report.national.export');
