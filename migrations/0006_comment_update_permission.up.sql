-- ============================================================
-- FSLDK API — Comment Update Permission
-- Adds comment.update so non-owner edits require an explicit role
-- checklist entry (mirrors the existing comment.delete pattern), instead
-- of edit being owner-only with no moderator override.
-- Idempotent: safe to re-run (INSERT IGNORE).
-- ============================================================

-- Action-only permission, no menu entry (same as comment.delete).
INSERT IGNORE INTO lk_permission (permissionCode, permissionName, moduleName, menuLabel, menuIcon, menuRoute, sortOrder) VALUES
('comment.update', 'Ubah Komentar', 'comment', NULL, NULL, NULL, NULL);

-- Super Admin & Editor get it granted immediately (same roles already
-- granted comment.delete in 0005_comment.up.sql).
INSERT IGNORE INTO map_role_permission (roleID, permissionID)
SELECT r.roleID, p.permissionID FROM ms_role r JOIN lk_permission p
WHERE r.roleName IN ('Super Admin', 'Editor') AND p.permissionCode = 'comment.update';
