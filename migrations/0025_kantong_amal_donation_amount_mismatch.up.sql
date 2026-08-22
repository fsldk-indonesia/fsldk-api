-- ============================================================
-- Kantong Amal — Donation (Phase 4: tambah status AMOUNT_MISMATCH
-- untuk payload callback yang lolos signature namun totalnya menyimpang
-- di luar toleransi pembulatan CEIL, lihat donation_service.ProcessCallback).
-- ============================================================

ALTER TABLE tr_donation
    MODIFY COLUMN paymentStatus ENUM('PENDING','PAID','EXPIRED','FAILED','CANCELLED','REFUNDED','AMOUNT_MISMATCH') NOT NULL DEFAULT 'PENDING';
