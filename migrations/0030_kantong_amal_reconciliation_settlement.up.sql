-- ============================================================
-- Kantong Amal — Settlement window pada Balance Report/Rekonsiliasi.
-- Ditambahkan agar tr_finance_reconciliation_snapshot menyimpan berapa
-- besar "recentlyPaid" (donasi PAID dalam BISATOPUP_SETTLEMENT_MINUTES_
-- CROWDFUNDING menit terakhir, belum tentu sudah settle di wallet gateway)
-- yang dipakai melonggarkan ambang anomali saat snapshot dijalankan —
-- sebelumnya nilai ini dihitung di report_service_impl.go tapi tidak
-- pernah disimpan/ditampilkan, jadi admin tidak bisa tahu KENAPA sebuah
-- selisih dianggap normal. Setara "Settlement Pending" banner di
-- ldksyahid-app (balance-report.blade.php pendingSettlementTotal).
-- ============================================================

ALTER TABLE tr_finance_reconciliation_snapshot
    ADD COLUMN settlementPendingAmount DECIMAL(18,2) NOT NULL DEFAULT 0 AFTER discrepancyAmount,
    ADD COLUMN settlementMinutes       INT NOT NULL DEFAULT 0 AFTER settlementPendingAmount;
