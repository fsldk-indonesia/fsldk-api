-- Migration 0036_subscription: Newsletter subscriber table and permissions
CREATE TABLE IF NOT EXISTS tr_subscriber (
    subscriberID     BIGINT AUTO_INCREMENT PRIMARY KEY,
    email            VARCHAR(255) NOT NULL,
    isActive         TINYINT(1) NOT NULL DEFAULT 1,
    unsubscribeToken VARCHAR(64) NULL COMMENT 'Random token embedded in the unsubscribe link sent by email',
    subscribedDate   DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    unsubscribedDate DATETIME NULL,
    createdDate      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE KEY uq_subscriber_email (email),
    INDEX idx_subscriber_active (isActive)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- Seed permissions
INSERT IGNORE INTO lk_permission (permissionCode, permissionName, moduleName, menuLabel, menuIcon, menuRoute, sortOrder) VALUES
('subscription.view',   'Lihat Subscriber Newsletter', 'subscription', 'Subscription', 'mail', '/cms/subscribers', 10),
('subscription.create', 'Tambah Subscriber Newsletter', 'subscription', NULL, NULL, NULL, NULL),
('subscription.delete', 'Hapus Subscriber Newsletter', 'subscription', NULL, NULL, NULL, NULL);

-- Assign view+create to Super Admin and Editor
INSERT IGNORE INTO map_role_permission (roleID, permissionID)
SELECT r.roleID, p.permissionID FROM ms_role r JOIN lk_permission p
WHERE r.roleName IN ('Super Admin', 'Editor') AND p.permissionCode IN ('subscription.view', 'subscription.create');

-- Delete restricted to Super Admin only
INSERT IGNORE INTO map_role_permission (roleID, permissionID)
SELECT r.roleID, p.permissionID FROM ms_role r JOIN lk_permission p
WHERE r.roleName = 'Super Admin' AND p.permissionCode = 'subscription.delete';
