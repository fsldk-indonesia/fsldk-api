-- ============================================================
-- Kantong Amal — Withdrawal: simpan pesan asli dari BisaTopup/Bisabiller
-- saat gateway menolak permintaan disbursement secara langsung (sinkron,
-- lihat withdrawal_service.Process -> extractGatewayRejectionMessage) —
-- sebelumnya dibuang begitu toast error lewat, jadi admin CMS tidak pernah
-- tahu alasan konkret (saldo gateway kurang, rekening invalid, dll).
-- Beda dari donation_service yang sengaja menyembunyikan alasan ini dari
-- donatur publik — withdrawal murni halaman CMS/admin, bukan donor-facing,
-- sehingga pesan asli gateway aman ditampilkan.
-- Tidak terisi untuk kegagalan yang datang lewat callback async (field
-- "status_id" webhook Bisabiller tidak pernah membawa pesan, lihat
-- DisbursementCallbackRequest).
-- ============================================================

ALTER TABLE tr_withdrawal
    ADD COLUMN gatewayMessage VARCHAR(255) NULL AFTER gatewayResponseJSON;
