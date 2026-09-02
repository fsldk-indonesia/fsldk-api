-- Finance Format (Format Keuangan) — financeformat module.
--
-- Public repository of blank Excel (.xlsx) templates for financial reports,
-- grouped by a fixed 9-row format-type taxonomy. CRUD + active toggle from
-- the CMS; files are uploaded through the shared modules/upload endpoint.
--
-- The lookup table is seed-only (no CMS CRUD) — same precedent as
-- lk_book_category / lk_article_category. A 10th category is a future
-- migration, not an admin form field.

CREATE TABLE IF NOT EXISTS lk_finance_format_type (
    formatTypeID   BIGINT AUTO_INCREMENT PRIMARY KEY,
    formatTypeName VARCHAR(100) NOT NULL UNIQUE,
    sortOrder      TINYINT NOT NULL DEFAULT 0
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- 9 fixed categories, copied verbatim from the tech spec.
INSERT IGNORE INTO lk_finance_format_type (formatTypeName, sortOrder) VALUES
  ('Format Arus Kas',                     1),
  ('Format RAB',                          2),
  ('Format Laporan Keuangan Proker',      3),
  ('Format Laporan Keuangan Open Donasi', 4),
  ('Format Laporan Uang Kas',             5),
  ('Format Laporan Penjualan',            6),
  ('Format Laporan Keuangan Fundraising', 7),
  ('Format Laporan Dana Sosial',          8),
  ('Format Laporan Keuangan Dakwah',      9);

CREATE TABLE IF NOT EXISTS ms_finance_format (
    financeFormatID BIGINT AUTO_INCREMENT PRIMARY KEY,
    fileName        VARCHAR(255) NOT NULL,
    fileURL         VARCHAR(500) NOT NULL,
    formatTypeID    BIGINT NOT NULL,
    isActive        TINYINT(1) NOT NULL DEFAULT 1,
    createdDate     DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    createdBy       BIGINT NULL,
    updatedDate     DATETIME NULL,
    updatedBy       BIGINT NULL,

    CONSTRAINT fk_financeformat_type    FOREIGN KEY (formatTypeID) REFERENCES lk_finance_format_type(formatTypeID),
    CONSTRAINT fk_financeformat_creator FOREIGN KEY (createdBy)    REFERENCES ms_user(userID),
    CONSTRAINT fk_financeformat_updater FOREIGN KEY (updatedBy)    REFERENCES ms_user(userID),

    INDEX idx_financeformat_active (isActive),
    INDEX idx_financeformat_type (formatTypeID),
    INDEX idx_financeformat_filename (fileName)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- Optional contact-person card on the public page — reuses ms_setting, edited
-- only by Super Admin via App Settings. Empty WhatsApp value hides the card.
INSERT IGNORE INTO ms_setting (settingGroup, settingKey, settingLabel, settingValue) VALUES
  ('format_keuangan', 'finance_format_cp_name',     'Nama Contact Person Format Keuangan',              'Kestari LDK'),
  ('format_keuangan', 'finance_format_cp_whatsapp', 'No. WhatsApp Contact Person (format 62xxxxxxxxxx)', '');

INSERT IGNORE INTO lk_permission (permissionCode, permissionName, moduleName, menuLabel, menuIcon, menuRoute, sortOrder) VALUES
  ('financeformat.view',    'Lihat Format Keuangan',      'financeformat', 'Format Keuangan', 'file-spreadsheet', '/cms/finance-formats', 12),
  ('financeformat.create',  'Tambah Format Keuangan',     'financeformat', NULL, NULL, NULL, NULL),
  ('financeformat.update',  'Ubah Format Keuangan',       'financeformat', NULL, NULL, NULL, NULL),
  ('financeformat.delete',  'Hapus Format Keuangan',      'financeformat', NULL, NULL, NULL, NULL),
  ('financeformat.publish', 'Aktifkan/Nonaktifkan Format', 'financeformat', NULL, NULL, NULL, NULL);

-- Super Admin's blanket grant in 0002_seed.up.sql only covered permissions
-- that existed then — every later module grants its own, so this must too.
INSERT IGNORE INTO map_role_permission (roleID, permissionID)
SELECT r.roleID, p.permissionID FROM ms_role r JOIN lk_permission p
WHERE r.roleName = 'Super Admin' AND p.permissionCode LIKE 'financeformat.%';

-- Editor only (not Kontributor) — official finance documents, same reasoning
-- as shortlink.approve.
INSERT IGNORE INTO map_role_permission (roleID, permissionID)
SELECT r.roleID, p.permissionID FROM ms_role r, lk_permission p
WHERE r.roleName = 'Editor' AND p.permissionCode IN
  ('financeformat.view','financeformat.create','financeformat.update','financeformat.delete','financeformat.publish');
