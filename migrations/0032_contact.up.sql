-- Migration 0031_contact: Contact messages table and permissions
CREATE TABLE IF NOT EXISTS tr_contact_message (
    messageID   BIGINT AUTO_INCREMENT PRIMARY KEY,
    senderName  VARCHAR(100) NOT NULL      COMMENT 'Sender name',
    email       VARCHAR(255) NOT NULL      COMMENT 'Sender email',
    subject     VARCHAR(200) NOT NULL      COMMENT 'Message subject',
    message     TEXT NOT NULL              COMMENT 'Message body (max 1000 chars)',
    ipAddress   VARCHAR(45) NULL           COMMENT 'Client IP address for audit',
    isRead      TINYINT(1) NOT NULL DEFAULT 0,
    createdDate DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_contact_email (email),
    INDEX idx_contact_created (createdDate)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- Seed permissions
INSERT IGNORE INTO lk_permission (permissionCode, permissionName, moduleName, menuLabel, menuIcon, menuRoute, sortOrder) VALUES
('contact.view',   'Lihat Pesan Kontak', 'contact', 'Pesan Kontak', 'messages', '/cms/contact-messages', 9),
('contact.delete', 'Hapus Pesan Kontak', 'contact', NULL, NULL, NULL, NULL);

-- Assign to Super Admin
INSERT IGNORE INTO map_role_permission (roleID, permissionID)
SELECT r.roleID, p.permissionID FROM ms_role r JOIN lk_permission p
WHERE r.roleName = 'Super Admin' AND p.permissionCode LIKE 'contact.%';

-- Assign to Editor
INSERT IGNORE INTO map_role_permission (roleID, permissionID)
SELECT r.roleID, p.permissionID FROM ms_role r JOIN lk_permission p
WHERE r.roleName = 'Editor' AND p.permissionCode IN ('contact.view', 'contact.delete');
