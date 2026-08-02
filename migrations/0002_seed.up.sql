-- ============================================================
-- FSLDK API — Seed Data Awal
-- Role bawaan, permission (+atribut menu), pemetaan role-permission,
-- dan kategori berita/artikel.
-- Idempoten: aman dijalankan ulang (INSERT IGNORE / UNIQUE key).
-- ============================================================

-- Role bawaan
INSERT IGNORE INTO ms_role (roleName, roleDescription, isSystemRole, isActive) VALUES
('Super Admin', 'Akses penuh ke seluruh fitur, pengguna, dan konten sistem.', 1, 1),
('Editor', 'Dapat membuat, mengedit, dan mempublikasikan berita/artikel.', 1, 1),
('Kontributor', 'Dapat membuat dan mengedit berita/artikel, namun tidak dapat mempublikasikan.', 1, 1);

-- Permission (kolom menu hanya diisi untuk aksi .view per modul)
INSERT IGNORE INTO lk_permission (permissionCode, permissionName, moduleName, menuLabel, menuIcon, menuRoute, sortOrder) VALUES
('news.view',    'Lihat Berita',     'news',    'Data Berita',   'newspaper',    '/cms/news',     1),
('news.create',  'Tambah Berita',    'news',    NULL, NULL, NULL, NULL),
('news.update',  'Ubah Berita',      'news',    NULL, NULL, NULL, NULL),
('news.delete',  'Hapus Berita',     'news',    NULL, NULL, NULL, NULL),
('news.publish', 'Publikasi Berita', 'news',    NULL, NULL, NULL, NULL),
('article.view',    'Lihat Artikel',     'article', 'Artikel',    'file-text', '/cms/articles', 2),
('article.create',  'Tambah Artikel',    'article', NULL, NULL, NULL, NULL),
('article.update',  'Ubah Artikel',      'article', NULL, NULL, NULL, NULL),
('article.delete',  'Hapus Artikel',     'article', NULL, NULL, NULL, NULL),
('article.publish', 'Publikasi Artikel', 'article', NULL, NULL, NULL, NULL),
('user.view',   'Lihat Pengguna',  'user', 'Pengguna', 'users', '/cms/users', 3),
('user.create', 'Tambah Pengguna', 'user', NULL, NULL, NULL, NULL),
('user.update', 'Ubah Pengguna',   'user', NULL, NULL, NULL, NULL),
('user.delete', 'Hapus Pengguna',  'user', NULL, NULL, NULL, NULL),
('role.view',   'Lihat Role',  'role', 'Role Pengguna', 'shield-check', '/cms/roles', 4),
('role.create', 'Tambah Role', 'role', NULL, NULL, NULL, NULL),
('role.update', 'Ubah Role',   'role', NULL, NULL, NULL, NULL),
('role.delete', 'Hapus Role',  'role', NULL, NULL, NULL, NULL);

-- Super Admin: seluruh permission
INSERT IGNORE INTO map_role_permission (roleID, permissionID)
SELECT r.roleID, p.permissionID FROM ms_role r JOIN lk_permission p
WHERE r.roleName = 'Super Admin';

-- Editor: seluruh news.* & article.*
INSERT IGNORE INTO map_role_permission (roleID, permissionID)
SELECT r.roleID, p.permissionID FROM ms_role r JOIN lk_permission p
WHERE r.roleName = 'Editor' AND p.permissionCode IN (
  'news.view','news.create','news.update','news.delete','news.publish',
  'article.view','article.create','article.update','article.delete','article.publish'
);

-- Kontributor: view/create/update berita & artikel (tanpa publish/delete/konten)
INSERT IGNORE INTO map_role_permission (roleID, permissionID)
SELECT r.roleID, p.permissionID FROM ms_role r JOIN lk_permission p
WHERE r.roleName = 'Kontributor' AND p.permissionCode IN (
  'news.view','news.create','news.update',
  'article.view','article.create','article.update'
);

-- Kategori berita
INSERT IGNORE INTO lk_news_category (categoryName, categorySlug, isActive) VALUES
('Nasional', 'nasional', 1),
('Regional', 'regional', 1),
('Kaderisasi', 'kaderisasi', 1),
('Kemuslimahan', 'kemuslimahan', 1);

-- Kategori artikel
INSERT IGNORE INTO lk_article_category (categoryName, categorySlug, isActive) VALUES
('Opini', 'opini', 1),
('Kajian', 'kajian', 1),
('Dakwah Kampus', 'dakwah-kampus', 1);

-- Konten Landing Page (visi/misi/struktur organisasi/kontak) sengaja tidak
-- disimpan di database — dikelola sebagai teks tetap (hardcoded) langsung di
-- frontend fsldk-web (lihat modules/home).
