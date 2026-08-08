-- ============================================================
-- FSLDK API — Skema Awal Database
-- Konvensi: nama tabel snake_case (lowercase), nama kolom camelCase.
-- Engine InnoDB, charset utf8mb4.
-- ============================================================

CREATE TABLE IF NOT EXISTS ms_role (
    roleID           BIGINT AUTO_INCREMENT PRIMARY KEY,
    roleName         VARCHAR(100) NOT NULL UNIQUE,
    roleDescription  VARCHAR(255) NULL,
    isSystemRole     TINYINT(1) NOT NULL DEFAULT 0,
    isActive         TINYINT(1) NOT NULL DEFAULT 1,
    createdDate      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    createdBy        BIGINT NULL,
    updatedDate      DATETIME NULL,
    updatedBy        BIGINT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS lk_permission (
    permissionID     BIGINT AUTO_INCREMENT PRIMARY KEY,
    permissionCode   VARCHAR(100) NOT NULL UNIQUE,
    permissionName   VARCHAR(150) NOT NULL,
    moduleName       VARCHAR(100) NOT NULL,
    menuLabel        VARCHAR(100) NULL,
    menuIcon         VARCHAR(50) NULL,
    menuRoute        VARCHAR(150) NULL,
    sortOrder        INT NULL,
    isActive         TINYINT(1) NOT NULL DEFAULT 1
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS map_role_permission (
    rolePermissionID BIGINT AUTO_INCREMENT PRIMARY KEY,
    roleID           BIGINT NOT NULL,
    permissionID     BIGINT NOT NULL,
    createdDate      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    createdBy        BIGINT NULL,
    UNIQUE KEY uq_role_permission (roleID, permissionID),
    CONSTRAINT fk_mrp_role FOREIGN KEY (roleID) REFERENCES ms_role(roleID) ON DELETE CASCADE,
    CONSTRAINT fk_mrp_permission FOREIGN KEY (permissionID) REFERENCES lk_permission(permissionID) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS ms_user (
    userID             BIGINT AUTO_INCREMENT PRIMARY KEY,
    roleID             BIGINT NOT NULL,
    fullName           VARCHAR(150) NOT NULL,
    email              VARCHAR(150) NOT NULL UNIQUE,
    username           VARCHAR(100) NULL UNIQUE,
    password           VARCHAR(255) NULL,
    googleID           VARCHAR(100) NULL UNIQUE,
    emailVerifiedDate  DATETIME NULL,
    phoneNumber        VARCHAR(30) NULL,
    photoURL           VARCHAR(255) NULL,
    mustChangePassword TINYINT(1) NOT NULL DEFAULT 0,
    isActive           TINYINT(1) NOT NULL DEFAULT 1,
    createdDate        DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    createdBy          BIGINT NULL,
    updatedDate        DATETIME NULL,
    updatedBy          BIGINT NULL,
    CONSTRAINT fk_user_role FOREIGN KEY (roleID) REFERENCES ms_role(roleID),
    INDEX idx_user_email (email),
    INDEX idx_user_role (roleID)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS lk_news_category (
    categoryID   BIGINT AUTO_INCREMENT PRIMARY KEY,
    categoryName VARCHAR(100) NOT NULL,
    categorySlug VARCHAR(100) NOT NULL UNIQUE,
    isActive     TINYINT(1) NOT NULL DEFAULT 1
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS ms_news (
    newsID        BIGINT AUTO_INCREMENT PRIMARY KEY,
    newsTitle     VARCHAR(255) NOT NULL,
    newsSlug      VARCHAR(255) NOT NULL UNIQUE,
    newsExcerpt   VARCHAR(500) NULL,
    newsContent   LONGTEXT NOT NULL,
    newsImage     VARCHAR(255) NULL,
    newsPublisher VARCHAR(150) NULL,
    newsReporter  VARCHAR(150) NULL,
    newsEditor    VARCHAR(150) NULL,
    categoryID    BIGINT NOT NULL,
    isFeatured    TINYINT(1) NOT NULL DEFAULT 0,
    isPublished   TINYINT(1) NOT NULL DEFAULT 0,
    publishedDate DATETIME NULL,
    viewCount     BIGINT NOT NULL DEFAULT 0,
    authorID      BIGINT NOT NULL,
    createdDate   DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    createdBy     BIGINT NULL,
    updatedDate   DATETIME NULL,
    updatedBy     BIGINT NULL,
    CONSTRAINT fk_news_category FOREIGN KEY (categoryID) REFERENCES lk_news_category(categoryID),
    CONSTRAINT fk_news_author FOREIGN KEY (authorID) REFERENCES ms_user(userID),
    INDEX idx_news_published (isPublished),
    INDEX idx_news_category (categoryID),
    INDEX idx_news_slug (newsSlug)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS lk_article_category (
    categoryID   BIGINT AUTO_INCREMENT PRIMARY KEY,
    categoryName VARCHAR(100) NOT NULL,
    categorySlug VARCHAR(100) NOT NULL UNIQUE,
    isActive     TINYINT(1) NOT NULL DEFAULT 1
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS ms_article (
    articleID      BIGINT AUTO_INCREMENT PRIMARY KEY,
    articleTitle   VARCHAR(255) NOT NULL,
    articleSlug    VARCHAR(255) NOT NULL UNIQUE,
    articleIntro   LONGTEXT NOT NULL,
    articleImage   VARCHAR(255) NULL,
    articleWriter  VARCHAR(150) NULL,
    articleEditor  VARCHAR(150) NULL,
    articlePdf     VARCHAR(255) NULL,
    categoryID     BIGINT NOT NULL,
    isPublished    TINYINT(1) NOT NULL DEFAULT 0,
    publishedDate  DATETIME NULL,
    authorID       BIGINT NOT NULL,
    createdDate    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    createdBy      BIGINT NULL,
    updatedDate    DATETIME NULL,
    updatedBy      BIGINT NULL,
    CONSTRAINT fk_article_category FOREIGN KEY (categoryID) REFERENCES lk_article_category(categoryID),
    CONSTRAINT fk_article_author FOREIGN KEY (authorID) REFERENCES ms_user(userID),
    INDEX idx_article_published (isPublished),
    INDEX idx_article_slug (articleSlug)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS tr_user_login_log (
    loginLogID  BIGINT AUTO_INCREMENT PRIMARY KEY,
    userID      BIGINT NOT NULL,
    loginDate   DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    ipAddress   VARCHAR(45) NULL,
    userAgent   VARCHAR(255) NULL,
    loginStatus VARCHAR(20) NOT NULL,
    INDEX idx_login_user (userID)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS tr_email_token (
    tokenID     BIGINT AUTO_INCREMENT PRIMARY KEY,
    userID      BIGINT NOT NULL,
    token       VARCHAR(255) NOT NULL UNIQUE,
    tokenType   VARCHAR(30) NOT NULL,
    expiresAt   DATETIME NOT NULL,
    usedAt      DATETIME NULL,
    createdDate DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT fk_token_user FOREIGN KEY (userID) REFERENCES ms_user(userID) ON DELETE CASCADE,
    INDEX idx_token_lookup (token, tokenType)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
