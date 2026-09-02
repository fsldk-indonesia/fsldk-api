-- ============================================================
-- Kantong Amal — Lookup & Permission (Phase 1).
-- lk_campaign_category + seluruh permission kantong_amal.* (didaftarkan
-- sekaligus di sini walau modul yang memakainya — campaign/donation/
-- wallet/withdrawal/report/queue/audit — baru dibangun di fase-fase
-- berikutnya; ini mengikuti pola migration existing yang pre-declare
-- permission sebelum CRUD lengkap, mis. comment.view/.delete di
-- 0005_comment.up.sql).
-- ============================================================

CREATE TABLE IF NOT EXISTS lk_campaign_category (
    campaignCategoryID INT AUTO_INCREMENT PRIMARY KEY,
    categoryCode        VARCHAR(30) NOT NULL UNIQUE,
    categoryName        VARCHAR(100) NOT NULL,
    sortOrder            TINYINT NOT NULL DEFAULT 0
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

INSERT IGNORE INTO lk_campaign_category (categoryCode, categoryName, sortOrder) VALUES
('KEMANUSIAAN',   'Kemanusiaan',        1),
('PENDIDIKAN',    'Pendidikan',         2),
('KESEHATAN',     'Kesehatan',          3),
('BENCANA_ALAM',  'Bencana Alam',       4),
('INFAK_MASJID',  'Infak & Masjid',     5),
('LAINNYA',       'Lainnya',            6);

-- Permission kantong_amal.* — moduleName dikelompokkan sebagai satu
-- 'kantong_amal' (bukan per-entitas) karena fitur ini menaungi banyak
-- sub-resource (campaign/donation/wallet/withdrawal/report/queue/audit)
-- yang dikelola sebagai satu produk.
INSERT IGNORE INTO lk_permission (permissionCode, permissionName, moduleName, menuLabel, menuIcon, menuRoute, sortOrder) VALUES
('kantong_amal.campaign.create',   'Ajukan Campaign',            'kantong_amal', NULL, NULL, NULL, NULL),
('kantong_amal.campaign.view',     'Lihat Campaign',             'kantong_amal', 'Kantong Amal', 'hand-heart', '/cms/kantong-amal/campaigns', 19),
('kantong_amal.campaign.update',   'Ubah Campaign',              'kantong_amal', NULL, NULL, NULL, NULL),
('kantong_amal.campaign.review',   'Review Campaign',            'kantong_amal', 'Moderasi Campaign', 'shield-check', '/cms/kantong-amal/moderasi', 20),
('kantong_amal.campaign.publish',  'Publish Campaign',           'kantong_amal', NULL, NULL, NULL, NULL),
('kantong_amal.campaign.moderate', 'Moderasi Campaign (pause/archive)', 'kantong_amal', NULL, NULL, NULL, NULL),
('kantong_amal.donation.view',     'Lihat Donasi',                'kantong_amal', 'Donasi', 'hand-coins', '/cms/kantong-amal/donasi', 21),
('kantong_amal.wallet.view',       'Lihat Saldo',                 'kantong_amal', NULL, NULL, NULL, NULL),
('kantong_amal.wallet.adjust',     'Adjustment Saldo Manual',     'kantong_amal', NULL, NULL, NULL, NULL),
('kantong_amal.withdrawal.request','Ajukan Penarikan Saldo',      'kantong_amal', NULL, NULL, NULL, NULL),
('kantong_amal.withdrawal.approve','Setujui Penarikan Saldo',     'kantong_amal', 'Persetujuan Penarikan', 'banknote', '/cms/kantong-amal/persetujuan-penarikan', 22),
('kantong_amal.withdrawal.reject', 'Tolak Penarikan Saldo',       'kantong_amal', NULL, NULL, NULL, NULL),
('kantong_amal.withdrawal.process','Proses Pencairan Saldo',      'kantong_amal', NULL, NULL, NULL, NULL),
('kantong_amal.report.view',       'Lihat Laporan Kantong Amal',  'kantong_amal', 'Laporan Kantong Amal', 'file-bar-chart', '/cms/kantong-amal/laporan', 23),
('kantong_amal.report.export',     'Ekspor Laporan Kantong Amal', 'kantong_amal', NULL, NULL, NULL, NULL),
('kantong_amal.queue.view',        'Lihat Queue Kantong Amal',    'kantong_amal', 'Queue Kantong Amal', 'list-todo', '/cms/kantong-amal/queue', 24),
('kantong_amal.queue.retry',       'Retry Job Queue',             'kantong_amal', NULL, NULL, NULL, NULL),
('kantong_amal.audit.view',        'Lihat Audit Log Kantong Amal','kantong_amal', 'Audit Log Kantong Amal', 'history', '/cms/kantong-amal/audit-log', 25);

-- Super Admin: akses penuh ke seluruh permission kantong_amal.*.
INSERT IGNORE INTO map_role_permission (roleID, permissionID)
SELECT r.roleID, p.permissionID FROM ms_role r JOIN lk_permission p
WHERE r.roleName = 'Super Admin' AND p.permissionCode LIKE 'kantong_amal.%';

-- Kader & LDK Admin: self-service terbatas (OQ-05) — boleh mengajukan
-- campaign sendiri dan mengajukan penarikan saldo dari campaign miliknya.
INSERT IGNORE INTO map_role_permission (roleID, permissionID)
SELECT r.roleID, p.permissionID FROM ms_role r JOIN lk_permission p
WHERE r.roleName IN ('Kader', 'LDK Admin')
  AND p.permissionCode IN ('kantong_amal.campaign.create', 'kantong_amal.campaign.view', 'kantong_amal.withdrawal.request');

-- Puskomda/Puskomnas Verifikator: moderasi campaign & monitoring donasi
-- (OQ-06) — TIDAK termasuk approval withdrawal/wallet adjustment, yang
-- untuk fase ini dibatasi Super Admin saja (keputusan konservatif,
-- ditinjau ulang setelah kebutuhan operasional nyata diketahui).
INSERT IGNORE INTO map_role_permission (roleID, permissionID)
SELECT r.roleID, p.permissionID FROM ms_role r JOIN lk_permission p
WHERE r.roleName IN ('Puskomda Verifikator', 'Puskomnas Verifikator')
  AND p.permissionCode IN ('kantong_amal.campaign.review', 'kantong_amal.donation.view');
