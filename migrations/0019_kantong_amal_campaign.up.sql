-- ============================================================
-- Kantong Amal — Campaign (Phase 1: skema saja; modul Go dibangun
-- Phase 2). ms_campaign, ms_campaign_image, tr_campaign_review.
-- ============================================================

CREATE TABLE IF NOT EXISTS ms_campaign (
    campaignID                BIGINT AUTO_INCREMENT PRIMARY KEY,
    publicRef                  CHAR(36) NOT NULL,
    slug                        VARCHAR(160) NOT NULL,
    title                       VARCHAR(200) NOT NULL,
    categoryID                  INT NOT NULL,
    ownerUserID                 BIGINT NOT NULL,
    organizationID              BIGINT NULL,
    story                       LONGTEXT NOT NULL,
    latestUpdate                LONGTEXT NULL,
    coverImageUrl               VARCHAR(500) NOT NULL,
    targetAmount                 DECIMAL(18,2) NOT NULL,
    collectedAmountCache         DECIMAL(18,2) NOT NULL DEFAULT 0,
    beneficiaryName              VARCHAR(150) NOT NULL,
    beneficiaryBankCode          VARCHAR(20) NOT NULL,
    beneficiaryAccountNumber     VARCHAR(30) NOT NULL,
    beneficiaryAccountHolder     VARCHAR(150) NOT NULL,
    beneficiaryLockedUntil       DATETIME NULL,
    startDate                   DATETIME NULL,
    endDate                     DATETIME NULL,
    status                       ENUM('DRAFT','SUBMITTED','REVISION_REQUESTED','APPROVED','PUBLISHED','PAUSED','COMPLETED','REJECTED','ARCHIVED','EXPIRED') NOT NULL DEFAULT 'DRAFT',
    moderationNote               TEXT NULL,
    isFeatured                   TINYINT(1) NOT NULL DEFAULT 0,
    isAnonymousAllowed           TINYINT(1) NOT NULL DEFAULT 1,
    createdDate                  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    createdBy                    BIGINT NULL,
    updatedDate                  DATETIME NULL,
    updatedBy                    BIGINT NULL,
    CONSTRAINT fk_campaign_category FOREIGN KEY (categoryID) REFERENCES lk_campaign_category(campaignCategoryID),
    CONSTRAINT fk_campaign_owner FOREIGN KEY (ownerUserID) REFERENCES ms_user(userID),
    CONSTRAINT fk_campaign_organization FOREIGN KEY (organizationID) REFERENCES ms_organization(organizationID),
    UNIQUE KEY uq_campaign_slug (slug),
    UNIQUE KEY uq_campaign_public_ref (publicRef),
    INDEX idx_campaign_status (status, endDate),
    INDEX idx_campaign_owner (ownerUserID)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS ms_campaign_image (
    campaignImageID BIGINT AUTO_INCREMENT PRIMARY KEY,
    campaignID       BIGINT NOT NULL,
    imageUrl         VARCHAR(500) NOT NULL,
    sortOrder        TINYINT NOT NULL DEFAULT 0,
    CONSTRAINT fk_campaign_image_campaign FOREIGN KEY (campaignID) REFERENCES ms_campaign(campaignID) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS tr_campaign_review (
    reviewID       BIGINT AUTO_INCREMENT PRIMARY KEY,
    campaignID      BIGINT NOT NULL,
    reviewerUserID  BIGINT NOT NULL,
    decision        ENUM('APPROVED','REVISION_REQUESTED','REJECTED') NOT NULL,
    note            TEXT NULL,
    reviewedDate    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT fk_campaign_review_campaign FOREIGN KEY (campaignID) REFERENCES ms_campaign(campaignID),
    CONSTRAINT fk_campaign_review_reviewer FOREIGN KEY (reviewerUserID) REFERENCES ms_user(userID),
    INDEX idx_campaign_review (campaignID, reviewedDate)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
