-- ============================================================
-- FSLDK API — Shortlink Request (permintaan publik + persetujuan admin)
-- Tabel ms_shortlink_request + permission shortlink.approve + pemetaan role.
-- Idempoten: aman dijalankan ulang (CREATE TABLE IF NOT EXISTS / INSERT IGNORE).
-- ============================================================

CREATE TABLE IF NOT EXISTS ms_shortlink_request (
    shortLinkRequestID BIGINT AUTO_INCREMENT PRIMARY KEY,
    requesterName       VARCHAR(255) NOT NULL,
    requesterEmail      VARCHAR(255) NOT NULL,
    requesterWhatsapp   VARCHAR(50) NOT NULL,
    destinationURL      VARCHAR(1000) NOT NULL,
    requestedKey        VARCHAR(30) NULL,
    note                TEXT NULL,
    status              VARCHAR(20) NOT NULL DEFAULT 'pending',
    shortLinkID         BIGINT NULL,
    rejectionReason     VARCHAR(500) NULL,
    reviewedBy          BIGINT NULL,
    reviewedDate        DATETIME NULL,
    createdDate         DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,

    INDEX idx_shortlink_request_status (status),
    CONSTRAINT fk_shortlink_request_link     FOREIGN KEY (shortLinkID) REFERENCES ms_shortlink(shortLinkID),
    CONSTRAINT fk_shortlink_request_reviewer FOREIGN KEY (reviewedBy)  REFERENCES ms_user(userID)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- Permission approve (menu terpisah dari CRUD shortlink langsung).
INSERT IGNORE INTO lk_permission (permissionCode, permissionName, moduleName, menuLabel, menuIcon, menuRoute, sortOrder)
VALUES ('shortlink.approve', 'Setujui Permintaan Shortlink', 'shortlink', 'Permintaan Shortlink', 'clock', '/cms/shortlink-requests', 6);

-- Super Admin & Editor (mengikuti preseden shortlink.* lain, lihat 0004_shortlink.up.sql);
-- Kontributor TIDAK diberi, konsisten dengan shortlink.create/update/delete.
INSERT IGNORE INTO map_role_permission (roleID, permissionID)
SELECT r.roleID, p.permissionID FROM ms_role r, lk_permission p
WHERE r.roleName IN ('Super Admin', 'Editor') AND p.permissionCode = 'shortlink.approve';
