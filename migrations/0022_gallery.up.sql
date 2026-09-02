-- Migration: 0022_gallery.up.sql
-- Description: Create ms_gallery and map_gallery_photo tables, seed gallery permissions.

CREATE TABLE IF NOT EXISTS ms_gallery (
    galleryID          BIGINT AUTO_INCREMENT PRIMARY KEY,
    eventName          VARCHAR(255)  NOT NULL   COMMENT 'Name of the event or activity',
    eventTheme         VARCHAR(255)  NOT NULL   COMMENT 'Theme or tagline of the event',
    eventDate          DATETIME      NULL       COMMENT 'Tanggal pelaksanaan kegiatan',
    eventDescription   LONGTEXT      NOT NULL   COMMENT 'Rich text description of the gallery event',
    coverImage         VARCHAR(255)  NOT NULL   COMMENT 'Cover / hero image path',
    youtubeVideoID     VARCHAR(50)   NULL       COMMENT 'Extracted YouTube video ID (not full URL)',
    documentLink       VARCHAR(500)  NULL       COMMENT 'External documentation drive folder URL',
    totalPhotos        INT           NOT NULL DEFAULT 0 COMMENT 'Denormalized count of photos in map_gallery_photo',
    createdDate        DATETIME      NOT NULL DEFAULT CURRENT_TIMESTAMP,
    createdBy          BIGINT        NULL,
    updatedDate        DATETIME      NULL,
    updatedBy          BIGINT        NULL,
    INDEX idx_gallery_event (eventName)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS map_gallery_photo (
    photoID        BIGINT AUTO_INCREMENT PRIMARY KEY,
    galleryID      BIGINT        NOT NULL   COMMENT 'Foreign key reference to ms_gallery',
    imagePath      VARCHAR(255)  NOT NULL   COMMENT 'Image file path',
    sortOrder      INT           NOT NULL DEFAULT 0 COMMENT 'Display order sequence (0 is first)',
    caption        VARCHAR(255)  NULL       COMMENT 'Optional image caption',
    uploadedDate   DATETIME      NOT NULL DEFAULT CURRENT_TIMESTAMP,
    uploadedBy     BIGINT        NULL,
    CONSTRAINT fk_gallery_photo FOREIGN KEY (galleryID)
        REFERENCES ms_gallery (galleryID) ON DELETE CASCADE,
    INDEX idx_gallery_photo_gallery (galleryID),
    INDEX idx_gallery_photo_sort (galleryID, sortOrder)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- Seed Permissions
INSERT IGNORE INTO lk_permission (permissionCode, permissionName, moduleName, menuLabel, menuIcon, menuRoute, sortOrder) VALUES
('gallery.view',   'Lihat Galeri',   'gallery', 'Galeri', 'images', '/cms/galleries', 8),
('gallery.create', 'Tambah Galeri',  'gallery', NULL, NULL, NULL, NULL),
('gallery.update', 'Ubah Galeri',    'gallery', NULL, NULL, NULL, NULL),
('gallery.delete', 'Hapus Galeri',   'gallery', NULL, NULL, NULL, NULL);

-- Assign to Super Admin
INSERT IGNORE INTO map_role_permission (roleID, permissionID)
SELECT r.roleID, p.permissionID FROM ms_role r JOIN lk_permission p
WHERE r.roleName = 'Super Admin' AND p.permissionCode LIKE 'gallery.%';
