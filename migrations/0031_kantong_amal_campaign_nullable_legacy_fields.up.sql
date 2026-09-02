-- ============================================================
-- Kantong Amal — perbaikan campaign gagal dibuat (Error 1364:
-- Field 'ownerUserID' doesn't have a default value).
--
-- Revisi Round 1/2 (migrasi 0028/0029, lihat docs/DEPLOYMENT.md)
-- menghapus konsep kepemilikan (ownerUserID) dan rekening
-- penerima (beneficiary*) dari kode Go, tapi kolomnya sengaja
-- TIDAK dihapus dari skema (precedent tr_queue_job) — hanya
-- lupa dilonggarkan dari NOT NULL, jadi setiap INSERT campaign
-- baru (yang tidak lagi mengisi kolom-kolom ini) selalu gagal.
-- ============================================================

ALTER TABLE ms_campaign
    MODIFY COLUMN ownerUserID             BIGINT NULL,
    MODIFY COLUMN beneficiaryName          VARCHAR(150) NULL,
    MODIFY COLUMN beneficiaryBankCode      VARCHAR(20) NULL,
    MODIFY COLUMN beneficiaryAccountNumber VARCHAR(30) NULL,
    MODIFY COLUMN beneficiaryAccountHolder VARCHAR(150) NULL;
