-- ============================================================
-- FSLDK API — Comment System (article & news)
-- Tables: ms_comment (threaded comments, generic contentType/contentID),
--         tr_comment_reaction (emoji reactions)
-- Role `Member` (public commenter, no CMS access) + permission comment.*
-- Idempotent: safe to re-run (CREATE TABLE IF NOT EXISTS / INSERT IGNORE).
-- ============================================================

CREATE TABLE IF NOT EXISTS ms_comment (
    commentID     BIGINT AUTO_INCREMENT PRIMARY KEY,
    contentType   VARCHAR(50) NOT NULL,
    contentID     BIGINT NOT NULL,
    parentID      BIGINT NULL,
    commentText   TEXT NULL,
    mediaURL      VARCHAR(500) NULL,
    mediaType     VARCHAR(20) NULL,
    createdDate   DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    createdBy     BIGINT NOT NULL,
    updatedDate   DATETIME NULL,
    updatedBy     BIGINT NULL,
    INDEX idx_comment_content (contentType, contentID),
    INDEX idx_comment_parent (parentID),
    CONSTRAINT fk_comment_parent FOREIGN KEY (parentID) REFERENCES ms_comment(commentID) ON DELETE CASCADE,
    CONSTRAINT fk_comment_author FOREIGN KEY (createdBy) REFERENCES ms_user(userID)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS tr_comment_reaction (
    reactionID    BIGINT AUTO_INCREMENT PRIMARY KEY,
    commentID     BIGINT NOT NULL,
    userID        BIGINT NOT NULL,
    reactionType  VARCHAR(30) NOT NULL,
    createdDate   DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE KEY uq_comment_user_reaction (commentID, userID, reactionType),
    CONSTRAINT fk_reaction_comment FOREIGN KEY (commentID) REFERENCES ms_comment(commentID) ON DELETE CASCADE,
    CONSTRAINT fk_reaction_user    FOREIGN KEY (userID)    REFERENCES ms_user(userID)       ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- Role for public commenters (no CMS access at all). Point REGISTER_DEFAULT_ROLE
-- at this role in app.env once this migration lands, so local self-registration
-- stops defaulting new accounts to a staff role.
INSERT IGNORE INTO ms_role (roleName) VALUES ('Member');

-- Comment moderation permission (menu in CMS sidebar, sortOrder 6 — next free slot).
INSERT IGNORE INTO lk_permission (permissionCode, permissionName, moduleName, menuLabel, menuIcon, menuRoute, sortOrder) VALUES
('comment.view',   'Lihat Komentar', 'comment', 'Komentar', 'message-circle', '/cms/comments', 6),
('comment.delete', 'Hapus Komentar', 'comment', NULL, NULL, NULL, NULL);

-- Super Admin & Editor can moderate comments; Kontributor & Member cannot.
INSERT IGNORE INTO map_role_permission (roleID, permissionID)
SELECT r.roleID, p.permissionID FROM ms_role r JOIN lk_permission p
WHERE r.roleName IN ('Super Admin', 'Editor') AND p.permissionCode IN ('comment.view', 'comment.delete');
