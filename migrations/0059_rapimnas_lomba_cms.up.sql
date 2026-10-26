
-- Permission untuk CMS Pendaftaran Lomba RAPIMNAS

INSERT IGNORE INTO lk_permission (
    permissionCode,
    permissionName,
    moduleName,
    menuLabel,
    menuIcon,
    menuRoute,
    sortOrder
)
VALUES
(
    'rapimnas.lomba.view',
    'Lihat Pendaftaran Lomba RAPIMNAS',
    'rapimnas',
    'Pendaftaran Lomba',
    'trophy',
    '/cms/rapimnas-lomba',
    98
),
(
    'rapimnas.lomba.manage',
    'Kelola Pendaftaran Lomba RAPIMNAS',
    'rapimnas',
    NULL,
    NULL,
    NULL,
    98
);

-- Super Admin mendapat akses melihat dan mengelola
INSERT IGNORE INTO map_role_permission (
    roleID,
    permissionID
)
SELECT
    r.roleID,
    p.permissionID
FROM ms_role r
CROSS JOIN lk_permission p
WHERE r.roleName = 'Super Admin'
AND p.permissionCode IN (
    'rapimnas.lomba.view',
    'rapimnas.lomba.manage'
);


