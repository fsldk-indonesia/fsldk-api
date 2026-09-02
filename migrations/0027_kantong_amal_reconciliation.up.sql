-- ============================================================
-- Kantong Amal — Finance Reconciliation Snapshot (Phase 9).
-- tr_finance_reconciliation_snapshot menyimpan histori hasil job terjadwal
-- finance.daily_reconciliation (§15.5/§13 techspec) — snapshot, bukan
-- dihitung ulang setiap halaman dibuka, supaya ada tren discrepancy dari
-- waktu ke waktu (beda dari ldksyahid-app yang cache 5 menit lalu hilang).
-- Skema tidak dicantumkan eksplisit di 06-database-design.md (gap
-- ditemukan saat implementasi, sama polanya seperti endpoint beneficiary
-- Phase 7) — dirancang di sini berdasarkan lima sumber pembanding §15.5.
-- ============================================================

CREATE TABLE IF NOT EXISTS tr_finance_reconciliation_snapshot (
    snapshotID                  BIGINT AUTO_INCREMENT PRIMARY KEY,
    snapshotDate                 DATETIME NOT NULL,
    donationPaidCount             INT NOT NULL,
    donationPaidAmount            DECIMAL(18,2) NOT NULL,
    ledgerDonationCreditAmount     DECIMAL(18,2) NOT NULL,
    withdrawalSuccessCount         INT NOT NULL,
    withdrawalSuccessAmount        DECIMAL(18,2) NOT NULL,
    expectedBalance                 DECIMAL(18,2) NOT NULL,
    gatewayWalletBalance             DECIMAL(18,2) NOT NULL,
    discrepancyAmount                DECIMAL(18,2) NOT NULL,
    hasAnomaly                       TINYINT(1) NOT NULL DEFAULT 0,
    gatewayError                      VARCHAR(255) NULL,
    createdDate                       DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_reconciliation_snapshot_date (snapshotDate)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
