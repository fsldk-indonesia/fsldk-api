-- ============================================================
-- Kantong Amal — Withdrawal (Phase 1: skema saja; modul Go dibangun
-- Phase 7/8). tr_withdrawal.
-- ============================================================

CREATE TABLE IF NOT EXISTS tr_withdrawal (
    withdrawalID              BIGINT AUTO_INCREMENT PRIMARY KEY,
    withdrawalRef               VARCHAR(60) NOT NULL,
    campaignID                  BIGINT NOT NULL,
    requestedByUserID            BIGINT NOT NULL,
    amount                       DECIMAL(18,2) NOT NULL,
    fee                          DECIMAL(18,2) NOT NULL DEFAULT 0,
    netAmount                    DECIMAL(18,2) NOT NULL,
    beneficiaryBankCode           VARCHAR(20) NOT NULL,
    beneficiaryAccountNumber      VARCHAR(30) NOT NULL,
    beneficiaryAccountHolder      VARCHAR(150) NOT NULL,
    status                        ENUM('REQUESTED','SECURITY_CHECK','PENDING_APPROVAL','APPROVED','PROCESSING','SUCCESS','FAILED','REJECTED','CANCELLED','REVERSED') NOT NULL DEFAULT 'REQUESTED',
    securityVerifiedDate          DATETIME NULL,
    securityVerifiedMethod        VARCHAR(30) NULL,
    approvedByUserID               BIGINT NULL,
    approvedDate                  DATETIME NULL,
    rejectionReason                VARCHAR(255) NULL,
    idempotencyKey                 VARCHAR(100) NOT NULL,
    gatewayStatusID                 INT NULL,
    gatewayResponseJSON             TEXT NULL,
    executedDate                   DATETIME NULL,
    completedDate                  DATETIME NULL,
    createdDate                    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updatedDate                    DATETIME NULL,
    CONSTRAINT fk_withdrawal_campaign FOREIGN KEY (campaignID) REFERENCES ms_campaign(campaignID),
    CONSTRAINT fk_withdrawal_requester FOREIGN KEY (requestedByUserID) REFERENCES ms_user(userID),
    CONSTRAINT fk_withdrawal_approver FOREIGN KEY (approvedByUserID) REFERENCES ms_user(userID),
    UNIQUE KEY uq_withdrawal_ref (withdrawalRef),
    UNIQUE KEY uq_withdrawal_idem (idempotencyKey),
    INDEX idx_withdrawal_campaign_status (campaignID, status)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
