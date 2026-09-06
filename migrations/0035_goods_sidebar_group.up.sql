-- Kelompokkan menu CMS "FSLDK Goods" & "Kategori Goods" jadi satu dropdown
-- collapsible di sidebar (pola sama seperti "Kantong Amal"). menuRoute
-- goods.view/goodscategory.view di-UPDATE ulang secara eksplisit di sini
-- (bukan hanya diedit di 0034) karena 0034 kemungkinan sudah pernah
-- dijalankan di database sebelum menuRoute-nya direvisi jadi nested —
-- migration di-track berdasarkan nama file, jadi mengedit isi 0034 saja
-- tidak akan diterapkan ulang ke database yang sudah menjalankannya.
-- Pola sama seperti 0012_cms_shell_fixups.up.sql.

UPDATE lk_permission SET menuRoute = '/cms/goods/products', menuLabel = 'FSLDK Goods'
WHERE permissionCode = 'goods.view';

UPDATE lk_permission SET menuRoute = '/cms/goods/categories', menuLabel = 'Kategori Goods'
WHERE permissionCode = 'goodscategory.view';
