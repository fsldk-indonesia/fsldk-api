
-- Sembunyikan menu Pendaftaran Lomba dari sidebar CMS.
-- Permission tetap aktif untuk akses API dan tab Rapimnas Setup.

UPDATE lk_permission
SET
    menuLabel = NULL,
    menuIcon = NULL,
    menuRoute = NULL
WHERE permissionCode = 'rapimnas.lomba.view';
