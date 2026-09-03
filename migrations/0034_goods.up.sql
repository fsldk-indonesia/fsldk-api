-- FSLDK Goods — marketplace/product catalog module (no cart/checkout/payment,
-- purchase button redirects to an external purchaseUrl configured per product).

CREATE TABLE IF NOT EXISTS lk_goods_category (
    goodsCategoryID BIGINT AUTO_INCREMENT PRIMARY KEY,
    categoryName    VARCHAR(100) NOT NULL,
    categorySlug    VARCHAR(100) NOT NULL UNIQUE,
    isActive        TINYINT(1) NOT NULL DEFAULT 1,
    sortOrder       INT NOT NULL DEFAULT 0,
    createdDate     DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    createdBy       BIGINT NULL,
    updatedDate     DATETIME NULL,
    updatedBy       BIGINT NULL,

    CONSTRAINT fk_goods_category_creator FOREIGN KEY (createdBy) REFERENCES ms_user(userID),
    CONSTRAINT fk_goods_category_updater FOREIGN KEY (updatedBy) REFERENCES ms_user(userID),

    INDEX idx_goods_category_active (isActive, sortOrder)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS ms_goods (
    goodsID             BIGINT AUTO_INCREMENT PRIMARY KEY,
    goodsName           VARCHAR(255) NOT NULL,
    goodsSlug           VARCHAR(255) NOT NULL UNIQUE,
    skuCode             VARCHAR(50) NULL UNIQUE,
    goodsCategoryID     BIGINT NOT NULL,
    shortDescription    VARCHAR(500) NULL,
    fullDescription     LONGTEXT NULL,
    price               DECIMAL(14,2) NOT NULL DEFAULT 0,
    mainImageUrl        VARCHAR(500) NULL,
    availabilityStatus  ENUM('available','out_of_stock','coming_soon') NOT NULL DEFAULT 'available',
    isFeatured          TINYINT(1) NOT NULL DEFAULT 0,
    isPublished         TINYINT(1) NOT NULL DEFAULT 0,
    publishedDate       DATETIME NULL,
    sortOrder           INT NOT NULL DEFAULT 0,
    purchaseUrl         VARCHAR(500) NOT NULL,
    purchaseButtonLabel VARCHAR(100) NOT NULL DEFAULT 'Beli Sekarang',
    createdDate         DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    createdBy           BIGINT NULL,
    updatedDate         DATETIME NULL,
    updatedBy           BIGINT NULL,

    CONSTRAINT fk_goods_category FOREIGN KEY (goodsCategoryID) REFERENCES lk_goods_category(goodsCategoryID),
    CONSTRAINT fk_goods_creator  FOREIGN KEY (createdBy)       REFERENCES ms_user(userID),
    CONSTRAINT fk_goods_updater  FOREIGN KEY (updatedBy)       REFERENCES ms_user(userID),

    INDEX idx_goods_published (isPublished, sortOrder),
    INDEX idx_goods_slug (goodsSlug),
    INDEX idx_goods_featured (isFeatured),
    INDEX idx_goods_category (goodsCategoryID)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS ms_goods_image (
    goodsImageID BIGINT AUTO_INCREMENT PRIMARY KEY,
    goodsID      BIGINT NOT NULL,
    imageUrl     VARCHAR(500) NOT NULL,
    sortOrder    TINYINT NOT NULL DEFAULT 0,

    CONSTRAINT fk_goods_image_goods FOREIGN KEY (goodsID) REFERENCES ms_goods(goodsID) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

INSERT IGNORE INTO lk_permission (permissionCode, permissionName, moduleName, menuLabel, menuIcon, menuRoute, sortOrder) VALUES
('goods.view',            'Lihat Produk FSLDK Goods',  'goods',         'FSLDK Goods',    'shopping-bag', '/cms/goods',            26),
('goods.create',          'Tambah Produk',              'goods',         NULL, NULL, NULL, NULL),
('goods.update',          'Ubah Produk',                'goods',         NULL, NULL, NULL, NULL),
('goods.delete',          'Hapus Produk',               'goods',         NULL, NULL, NULL, NULL),
('goods.publish',         'Publish/Unpublish Produk',   'goods',         NULL, NULL, NULL, NULL),
('goodscategory.view',    'Lihat Kategori Produk',      'goodscategory', 'Kategori Goods', 'tags',         '/cms/goods-categories', 27),
('goodscategory.create',  'Tambah Kategori Produk',     'goodscategory', NULL, NULL, NULL, NULL),
('goodscategory.update',  'Ubah Kategori Produk',       'goodscategory', NULL, NULL, NULL, NULL),
('goodscategory.delete',  'Hapus Kategori Produk',      'goodscategory', NULL, NULL, NULL, NULL);

-- Super Admin only got a blanket grant for permissions that existed at
-- 0002_seed.up.sql time — every module added since then grants its own.
INSERT IGNORE INTO map_role_permission (roleID, permissionID)
SELECT r.roleID, p.permissionID FROM ms_role r JOIN lk_permission p
WHERE r.roleName = 'Super Admin' AND (p.permissionCode LIKE 'goods.%' OR p.permissionCode LIKE 'goodscategory.%');

-- Editor: full parity with catalogbook/article/news (CRUD + publish).
INSERT IGNORE INTO map_role_permission (roleID, permissionID)
SELECT r.roleID, p.permissionID FROM ms_role r, lk_permission p
WHERE r.roleName = 'Editor' AND p.permissionCode IN
  ('goods.view','goods.create','goods.update','goods.delete','goods.publish',
   'goodscategory.view','goodscategory.create','goodscategory.update','goodscategory.delete');

-- Kontributor: view/create/update only — no delete, no publish (same
-- pattern as catalogbook/article/news for this role).
INSERT IGNORE INTO map_role_permission (roleID, permissionID)
SELECT r.roleID, p.permissionID FROM ms_role r, lk_permission p
WHERE r.roleName = 'Kontributor' AND p.permissionCode IN
  ('goods.view','goods.create','goods.update','goodscategory.view');
