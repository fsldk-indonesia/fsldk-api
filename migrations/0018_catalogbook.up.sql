-- Digital Book Catalog (Perpustakaan) — catalogbook module.
--
-- Lookup tables are prefixed "lk_book_" (not plain "lk_category"/"lk_language")
-- to avoid name clashes with other modules' lookup tables.

CREATE TABLE IF NOT EXISTS lk_book_category (
    bookCategoryID   BIGINT AUTO_INCREMENT PRIMARY KEY,
    bookCategoryName VARCHAR(100) NOT NULL UNIQUE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS lk_book_language (
    languageID   BIGINT AUTO_INCREMENT PRIMARY KEY,
    languageName VARCHAR(50) NOT NULL UNIQUE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS lk_book_author_type (
    authorTypeID   BIGINT AUTO_INCREMENT PRIMARY KEY,
    authorTypeName VARCHAR(50) NOT NULL UNIQUE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS lk_book_availability_type (
    availabilityTypeID   BIGINT AUTO_INCREMENT PRIMARY KEY,
    availabilityTypeName VARCHAR(50) NOT NULL UNIQUE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

INSERT IGNORE INTO lk_book_language (languageName) VALUES ('Indonesia'), ('English'), ('Arabic');
INSERT IGNORE INTO lk_book_author_type (authorTypeName) VALUES ('Kader'), ('Non Kader');
INSERT IGNORE INTO lk_book_availability_type (availabilityTypeName) VALUES
  ('Available Full PDF'), ('Available Preview Only PDF'), ('Available at Mahar');

-- 50 categories, copied verbatim from the ldksyahid-app reference
-- (database/migrations/2025_10_19_173916_create_lk_tables.php).
INSERT IGNORE INTO lk_book_category (bookCategoryName) VALUES
  ('Agama'), ('Pendidikan'), ('Sains'), ('Sejarah'), ('Sastra'), ('Teknologi'),
  ('Kesehatan'), ('Psikologi'), ('Ekonomi'), ('Bisnis & Manajemen'),
  ('Komputer & Informatika'), ('Hukum'), ('Politik & Pemerintahan'), ('Filsafat'),
  ('Seni & Desain'), ('Bahasa & Linguistik'), ('Komunikasi & Media'), ('Pertanian'),
  ('Peternakan'), ('Teknik Sipil'), ('Teknik Mesin'), ('Teknik Elektro'),
  ('Teknik Industri'), ('Teknik Kimia'), ('Teknik Lingkungan'), ('Matematika'),
  ('Fisika'), ('Kimia'), ('Biologi'), ('Astronomi'), ('Geografi'), ('Arsitektur'),
  ('Sosial & Budaya'), ('Keluarga & Parenting'), ('Motivasi & Pengembangan Diri'),
  ('Travel & Pariwisata'), ('Kuliner'), ('Fiksi'), ('Non-Fiksi'), ('Puisi'),
  ('Cerpen'), ('Novel'), ('Komik & Manga'), ('Ensiklopedia'), ('Majalah & Jurnal'),
  ('Biografi'), ('Autobiografi'), ('Anak-anak'), ('Remaja'), ('Umum');

CREATE TABLE IF NOT EXISTS ms_catalog_book (
    bookID             BIGINT AUTO_INCREMENT PRIMARY KEY,
    bookSlug           VARCHAR(255) NOT NULL UNIQUE,
    isbn               VARCHAR(50) NULL UNIQUE,
    bookTitle          VARCHAR(255) NOT NULL,
    authorName         VARCHAR(255) NOT NULL,
    authorTypeID       BIGINT NOT NULL,
    publisherName      VARCHAR(255) NOT NULL,
    bookCategoryID     BIGINT NOT NULL,
    languageID         BIGINT NOT NULL,
    availabilityTypeID BIGINT NOT NULL,
    bookPdf            VARCHAR(500) NULL,
    year               CHAR(4) NOT NULL,
    pages              INT NOT NULL,
    description        TEXT NOT NULL,
    synopsis           TEXT NULL,
    edition            VARCHAR(100) NULL,
    coverImage         VARCHAR(500) NULL,
    favoriteCount       INT NOT NULL DEFAULT 0,
    tags                 TEXT NULL,
    metaKeywords          TEXT NULL,
    metaDescription        TEXT NULL,
    isActive                TINYINT(1) NOT NULL DEFAULT 1,
    createdDate              DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    createdBy                BIGINT NULL,
    updatedDate                DATETIME NULL,
    updatedBy                  BIGINT NULL,

    CONSTRAINT fk_catalogbook_category     FOREIGN KEY (bookCategoryID)     REFERENCES lk_book_category(bookCategoryID),
    CONSTRAINT fk_catalogbook_language     FOREIGN KEY (languageID)         REFERENCES lk_book_language(languageID),
    CONSTRAINT fk_catalogbook_authortype   FOREIGN KEY (authorTypeID)       REFERENCES lk_book_author_type(authorTypeID),
    CONSTRAINT fk_catalogbook_availability FOREIGN KEY (availabilityTypeID) REFERENCES lk_book_availability_type(availabilityTypeID),
    CONSTRAINT fk_catalogbook_creator      FOREIGN KEY (createdBy)          REFERENCES ms_user(userID),
    CONSTRAINT fk_catalogbook_updater      FOREIGN KEY (updatedBy)          REFERENCES ms_user(userID),

    INDEX idx_catalogbook_active (isActive),
    INDEX idx_catalogbook_category (bookCategoryID),
    INDEX idx_catalogbook_title (bookTitle),
    INDEX idx_catalogbook_author (authorName),
    INDEX idx_catalogbook_publisher (publisherName)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

INSERT IGNORE INTO lk_permission (permissionCode, permissionName, moduleName, menuLabel, menuIcon, menuRoute, sortOrder) VALUES
('catalogbook.view',    'Lihat Katalog Buku',        'catalogbook', 'Perpustakaan', 'book-open', '/cms/catalog-books', 6),
('catalogbook.create',  'Tambah Buku',                'catalogbook', NULL, NULL, NULL, NULL),
('catalogbook.update',  'Ubah Buku',                  'catalogbook', NULL, NULL, NULL, NULL),
('catalogbook.delete',  'Hapus Buku',                 'catalogbook', NULL, NULL, NULL, NULL),
('catalogbook.publish', 'Aktifkan/Nonaktifkan Buku',  'catalogbook', NULL, NULL, NULL, NULL);

-- Super Admin only got a blanket grant for permissions that existed at
-- 0002_seed.up.sql time — every module added since then (event, comment,
-- organization, etc.) explicitly grants its own permissions to Super Admin
-- too, so catalogbook must do the same or it won't show up for anyone.
INSERT IGNORE INTO map_role_permission (roleID, permissionID)
SELECT r.roleID, p.permissionID FROM ms_role r JOIN lk_permission p
WHERE r.roleName = 'Super Admin' AND p.permissionCode LIKE 'catalogbook.%';

-- Editor: full parity with article/news (CRUD + active toggle).
INSERT IGNORE INTO map_role_permission (roleID, permissionID)
SELECT r.roleID, p.permissionID FROM ms_role r, lk_permission p
WHERE r.roleName = 'Editor' AND p.permissionCode IN
  ('catalogbook.view','catalogbook.create','catalogbook.update','catalogbook.delete','catalogbook.publish');

-- Kontributor: view/create/update only — no delete, no active toggle
-- (exact parity with the article/news pattern for this role).
INSERT IGNORE INTO map_role_permission (roleID, permissionID)
SELECT r.roleID, p.permissionID FROM ms_role r, lk_permission p
WHERE r.roleName = 'Kontributor' AND p.permissionCode IN
  ('catalogbook.view','catalogbook.create','catalogbook.update');
