-- ============================================================
-- Kantong Amal — Revisi Round 2 (2026-09-01): campaign & donation jadi
-- CRUD murni CMS (tanpa kepemilikan), field form disamakan persis dengan
-- Campaign/Donation celengan syahid (ldksyahid-app), donasi manual/offline,
-- OTP withdrawal via email, dan ms_setting.isHide. Murni additive — tidak
-- menyentuh data transaksi (donasi/withdrawal/ledger) yang sudah ada.
-- ============================================================

-- ---------- ms_campaign: field baru menggantikan rekening penerima ----------
-- Rekening penerima (beneficiaryName/BankCode/AccountNumber/AccountHolder/
-- LockedUntil) TIDAK dihapus kolomnya (precedent tr_queue_job) — sekadar
-- tidak dipakai lagi, digantikan input rekening per pengajuan withdrawal.
ALTER TABLE ms_campaign
    ADD COLUMN provinceName             VARCHAR(100) NULL AFTER organizationID,
    ADD COLUMN cityName                 VARCHAR(100) NULL AFTER provinceName,
    ADD COLUMN goals                    VARCHAR(2000) NOT NULL DEFAULT '' AFTER story,
    ADD COLUMN picName                  VARCHAR(150) NOT NULL DEFAULT '' AFTER targetAmount,
    ADD COLUMN picPhone                 VARCHAR(30) NOT NULL DEFAULT '' AFTER picName,
    ADD COLUMN organizationNameOverride VARCHAR(150) NULL AFTER picPhone,
    ADD COLUMN organizationLogoUrl      VARCHAR(500) NULL AFTER organizationNameOverride,
    ADD COLUMN organizationLinkUrl      VARCHAR(500) NULL AFTER organizationLogoUrl;

-- ---------- tr_donation: dukungan donasi manual/offline (admin CRUD) ----------
ALTER TABLE tr_donation
    ADD COLUMN paymentMethod VARCHAR(20) NULL AFTER gateway;

UPDATE tr_donation SET paymentMethod = 'QRIS' WHERE gateway = 'bisatopup' AND paymentMethod IS NULL;

-- ---------- ms_setting: kolom isHide (item 8 — OTP recipient tersembunyi) ----------
ALTER TABLE ms_setting
    ADD COLUMN isHide TINYINT(1) NOT NULL DEFAULT 0 AFTER settingValue;

INSERT IGNORE INTO ms_setting (settingGroup, settingKey, settingLabel, settingValue, isHide) VALUES
  ('kantong_amal', 'withdrawal_otp_email', 'Email Penerima OTP Penarikan Kantong Amal', 'yusufwijaya3@gmail.com', 1);

-- ---------- Permission baru: campaign.delete, donation CRUD ----------
INSERT IGNORE INTO lk_permission (permissionCode, permissionName, moduleName, menuLabel, menuIcon, menuRoute, sortOrder) VALUES
('kantong_amal.campaign.delete', 'Hapus Campaign', 'kantong_amal', NULL, NULL, NULL, NULL),
('kantong_amal.donation.create', 'Tambah Donasi Manual', 'kantong_amal', NULL, NULL, NULL, NULL),
('kantong_amal.donation.update', 'Ubah Donasi Manual',   'kantong_amal', NULL, NULL, NULL, NULL),
('kantong_amal.donation.delete', 'Hapus Donasi Manual',  'kantong_amal', NULL, NULL, NULL, NULL);

-- campaign.create sekarang murni aksi CMS ("siapapun boleh create/update/
-- delete campaign asal ada hak akses" — revision-prompt-2.md item 1), bukan
-- lagi "pengajuan" milik sendiri.
UPDATE lk_permission SET permissionName = 'Buat Campaign' WHERE permissionCode = 'kantong_amal.campaign.create';

-- Super Admin: akses penuh (termasuk permission baru di atas, wildcard).
INSERT IGNORE INTO map_role_permission (roleID, permissionID)
SELECT r.roleID, p.permissionID FROM ms_role r JOIN lk_permission p
WHERE r.roleName = 'Super Admin' AND p.permissionCode LIKE 'kantong_amal.%';

-- Kader & LDK Admin sudah punya campaign.create/withdrawal.request sejak
-- Phase 1 (self-service) — kini permission yang sama berarti akses CMS
-- penuh ke campaign manapun, jadi turut diberi campaign.update supaya bisa
-- menindaklanjuti campaign yang mereka buat.
INSERT IGNORE INTO map_role_permission (roleID, permissionID)
SELECT r.roleID, p.permissionID FROM ms_role r JOIN lk_permission p
WHERE r.roleName IN ('Kader', 'LDK Admin') AND p.permissionCode = 'kantong_amal.campaign.update';
