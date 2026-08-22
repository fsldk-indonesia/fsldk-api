-- ============================================================
-- Kantong Amal — Donation (Phase 1: skema saja; modul Go dibangun
-- Phase 3/4). tr_donation.
-- ============================================================

CREATE TABLE IF NOT EXISTS tr_donation (
    donationID              BIGINT AUTO_INCREMENT PRIMARY KEY,
    publicRef                CHAR(36) NOT NULL,
    campaignID                BIGINT NOT NULL,
    donorUserID                BIGINT NULL,
    donorName                  VARCHAR(150) NOT NULL,
    donorEmail                 VARCHAR(150) NOT NULL,
    donorPhone                 VARCHAR(30) NOT NULL,
    donorAge                   VARCHAR(10) NULL,
    donorDomicile               VARCHAR(150) NULL,
    donorOccupation             VARCHAR(150) NULL,
    isAnonymous                 TINYINT(1) NOT NULL DEFAULT 0,
    message                     TEXT NULL,
    amount                      DECIMAL(18,2) NOT NULL,
    adminFee                    DECIMAL(18,2) NOT NULL,
    totalAmount                 DECIMAL(18,2) NOT NULL,
    paymentStatus               ENUM('PENDING','PAID','EXPIRED','FAILED','CANCELLED','REFUNDED') NOT NULL DEFAULT 'PENDING',
    gateway                     VARCHAR(20) NOT NULL,
    externalTransactionID       VARCHAR(100) NULL,
    idempotencyKey              VARCHAR(100) NOT NULL,
    qrPayload                   TEXT NULL,
    paymentCode                 VARCHAR(500) NULL,
    paymentLink                 VARCHAR(500) NULL,
    gatewayStatusID              INT NULL,
    expiredDate                 DATETIME NULL,
    createdDate                 DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updatedDate                 DATETIME NULL,
    CONSTRAINT fk_donation_campaign FOREIGN KEY (campaignID) REFERENCES ms_campaign(campaignID),
    CONSTRAINT fk_donation_donor FOREIGN KEY (donorUserID) REFERENCES ms_user(userID),
    UNIQUE KEY uq_donation_public_ref (publicRef),
    UNIQUE KEY uq_donation_ext_txn (externalTransactionID),
    UNIQUE KEY uq_donation_idem (idempotencyKey),
    INDEX idx_donation_campaign_status (campaignID, paymentStatus),
    INDEX idx_donation_expiry (paymentStatus, expiredDate)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
