-- ============================================================
-- FSLDK API - Event Module
-- Table ms_event + event.* permissions + role mapping.
-- Idempotent: safe to re-run (CREATE TABLE IF NOT EXISTS / INSERT IGNORE).
-- ============================================================

CREATE TABLE IF NOT EXISTS ms_event (
    eventID          BIGINT AUTO_INCREMENT PRIMARY KEY,
    eventTitle       VARCHAR(255) NOT NULL,
    eventSlug        VARCHAR(255) NOT NULL UNIQUE,
    eventDivision    VARCHAR(150) NOT NULL            COMMENT 'Organizing division/department',
    eventContent     LONGTEXT NOT NULL                COMMENT 'Rich-text description (TinyMCE)',
    eventImage       VARCHAR(255) NULL                COMMENT 'Poster image path (via upload module)',
    startDate        DATETIME NULL,
    endDate          DATETIME NULL,
    closeRegistDate  DATETIME NULL                    COMMENT 'Registration deadline',
    location         VARCHAR(255) NULL                COMMENT 'City or platform (online)',
    place            VARCHAR(255) NULL                COMMENT 'Specific venue name',
    locationLink     VARCHAR(500) NULL                COMMENT 'Google Maps URL',
    registrationLink VARCHAR(500) NULL                COMMENT 'External registration URL',
    documentLink     VARCHAR(500) NULL                COMMENT 'Google Drive docs/photos link',
    presentationLink VARCHAR(500) NULL                COMMENT 'Google Drive presentation/materials link',
    contactPerson1   VARCHAR(30) NULL                 COMMENT 'WhatsApp number of CP 1 (without +62)',
    nameCp1          VARCHAR(100) NULL                COMMENT 'Name of CP 1',
    contactPerson2   VARCHAR(30) NULL                 COMMENT 'WhatsApp number of CP 2',
    nameCp2          VARCHAR(100) NULL                COMMENT 'Name of CP 2',
    tag              VARCHAR(255) NULL                COMMENT 'Comma-separated tags/categories',
    isPublished      TINYINT(1) NOT NULL DEFAULT 0,
    viewCount        BIGINT NOT NULL DEFAULT 0,
    authorID         BIGINT NOT NULL,
    createdDate      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    createdBy        BIGINT NULL,
    updatedDate      DATETIME NULL,
    updatedBy        BIGINT NULL,
    CONSTRAINT fk_event_author FOREIGN KEY (authorID) REFERENCES ms_user(userID),
    INDEX idx_event_slug (eventSlug),
    INDEX idx_event_start (startDate),
    INDEX idx_event_published (isPublished)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- Permissions (menu slot 6 in sidebar)
INSERT IGNORE INTO lk_permission (permissionCode, permissionName, moduleName, menuLabel, menuIcon, menuRoute, sortOrder) VALUES
('event.view',   'Lihat Event',  'event', 'Event', 'calendar-days', '/cms/events', 6),
('event.create', 'Tambah Event', 'event', NULL, NULL, NULL, NULL),
('event.update', 'Ubah Event',   'event', NULL, NULL, NULL, NULL),
('event.delete', 'Hapus Event',  'event', NULL, NULL, NULL, NULL);

-- Grant all event.* to Super Admin
INSERT IGNORE INTO map_role_permission (roleID, permissionID)
SELECT r.roleID, p.permissionID FROM ms_role r JOIN lk_permission p
WHERE r.roleName = 'Super Admin' AND p.permissionCode LIKE 'event.%';

-- Grant all event.* to Editor
INSERT IGNORE INTO map_role_permission (roleID, permissionID)
SELECT r.roleID, p.permissionID FROM ms_role r JOIN lk_permission p
WHERE r.roleName = 'Editor' AND p.permissionCode IN (
  'event.view','event.create','event.update','event.delete'
);
