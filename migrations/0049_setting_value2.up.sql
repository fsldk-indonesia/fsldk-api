-- ============================================================
-- Kolom kedua generik untuk App Settings (settingValue2) — dipakai pertama
-- kali oleh group "deadline" untuk menyimpan estimasi biaya (format digit
-- polos, mis. "1500000") yang ditampilkan sebagai "Rp 1.500.000" di banner
-- peringatan dashboard CMS Utama (lihat dashboard_dto.DeadlineAlert.Cost).
-- NULL untuk semua setting lain yang tidak membutuhkannya.
-- ============================================================

ALTER TABLE ms_setting ADD COLUMN settingValue2 TEXT NULL AFTER settingValue;
