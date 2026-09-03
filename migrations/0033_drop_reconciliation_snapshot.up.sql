-- ============================================================
-- Balance Report jadi real-time (revision-prompt-4.md, item tambahan) —
-- rekonsiliasi ledger vs wallet gateway sekarang dihitung LIVE tiap dibuka
-- (GET /reports/kantong-amal/reconciliation), tidak lagi disimpan sebagai
-- histori. tr_finance_reconciliation_snapshot murni tabel turunan/pelaporan
-- (tidak ada tabel lain yang FK ke sini) — aman dihapus bersama fiturnya.
-- ============================================================

DROP TABLE IF EXISTS tr_finance_reconciliation_snapshot;
