-- ============================================================
-- Kantong Amal — Withdrawal: simpan URL bukti transfer (receipt) yang
-- dikirim Bisabiller lewat callback disbursement (field "receipt" pada
-- payload webhook, lihat DisbursementCallbackRequest) — sebelumnya di-parse
-- tapi tidak pernah dipersist, jadi tombol "Lihat Bukti" tidak punya sumber
-- data.
-- ============================================================

ALTER TABLE tr_withdrawal
    ADD COLUMN receiptUrl VARCHAR(500) NULL AFTER completedDate;
