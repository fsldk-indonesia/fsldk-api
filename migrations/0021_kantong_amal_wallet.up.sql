-- ============================================================
-- Kantong Amal — Wallet Ledger (Phase 1: skema + modul Go wallet_service
-- sebagai satu-satunya write path). tr_wallet_ledger, append-only.
-- UNIQUE(referenceType, referenceID, entryType) adalah guard idempotency
-- di level DB — mencegah ledger entry duplikat untuk event yang sama
-- meski ada bug/race condition di application layer.
-- ============================================================

CREATE TABLE IF NOT EXISTS tr_wallet_ledger (
    ledgerID          BIGINT AUTO_INCREMENT PRIMARY KEY,
    campaignID         BIGINT NOT NULL,
    entryType          ENUM('DONATION_CREDIT','WITHDRAWAL_RESERVE','WITHDRAWAL_RELEASE','REFUND_DEBIT','ADJUSTMENT_CREDIT','ADJUSTMENT_DEBIT','FEE_DEBIT') NOT NULL,
    direction          ENUM('CREDIT','DEBIT') NOT NULL,
    amount             DECIMAL(18,2) NOT NULL,
    balanceAfter        DECIMAL(18,2) NOT NULL,
    referenceType       VARCHAR(30) NOT NULL,
    referenceID          BIGINT NOT NULL,
    note                VARCHAR(255) NULL,
    createdByUserID      BIGINT NULL,
    createdDate         DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT fk_ledger_campaign FOREIGN KEY (campaignID) REFERENCES ms_campaign(campaignID),
    CONSTRAINT fk_ledger_actor FOREIGN KEY (createdByUserID) REFERENCES ms_user(userID),
    UNIQUE KEY uq_ledger_reference (referenceType, referenceID, entryType),
    INDEX idx_ledger_campaign_date (campaignID, createdDate)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
