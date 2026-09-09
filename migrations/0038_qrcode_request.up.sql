-- ============================================================
-- FSLDK API — QR Code Request (permintaan publik + persetujuan admin)
-- Tabel ms_qrcode_request + permission qrcode.approve + pemetaan role +
-- setting PIC. Pola identik dengan 0009_shortlink_request.up.sql, TANPA
-- kolom requestedKey — pemohon menyebut URL tujuan + memilih kustomisasi
-- gambar (warna/ikon/caption) yang ikut tersimpan; saat approve nilai-nilai
-- itu disalin ke baris ms_qrcode baru. Kolom reviewedVia disertakan sejak
-- awal (tidak ada baris legacy).
-- Idempoten: aman dijalankan ulang (CREATE TABLE IF NOT EXISTS / INSERT IGNORE).
-- ============================================================

CREATE TABLE IF NOT EXISTS ms_qrcode_request (
    qrCodeRequestID   BIGINT AUTO_INCREMENT PRIMARY KEY,
    requesterName     VARCHAR(255) NOT NULL,
    requesterEmail    VARCHAR(255) NOT NULL,
    requesterWhatsapp VARCHAR(50) NOT NULL,
    destinationURL    VARCHAR(1000) NOT NULL,
    foregroundColor   VARCHAR(9) NOT NULL DEFAULT '#16211C',
    backgroundColor   VARCHAR(9) NOT NULL DEFAULT '#FFFFFF',
    centerIconURL     TEXT NULL,
    centerIconKey     VARCHAR(32) NULL,
    captionText       VARCHAR(120) NULL,
    note              TEXT NULL,
    status            VARCHAR(20) NOT NULL DEFAULT 'pending',
    qrCodeID          BIGINT NULL,
    rejectionReason   VARCHAR(500) NULL,
    reviewedBy        BIGINT NULL,
    reviewedVia       VARCHAR(20) NOT NULL DEFAULT 'cms',
    reviewedDate      DATETIME NULL,
    createdDate       DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,

    INDEX idx_qrcode_request_status (status),
    CONSTRAINT fk_qrcode_request_qrcode   FOREIGN KEY (qrCodeID)   REFERENCES ms_qrcode(qrCodeID),
    CONSTRAINT fk_qrcode_request_reviewer FOREIGN KEY (reviewedBy) REFERENCES ms_user(userID)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- Permission approve — anak kedua grup sidebar "QR Code" (prefix
-- /cms/qrcode/), menu terpisah dari CRUD langsung. sortOrder 8.
INSERT IGNORE INTO lk_permission (permissionCode, permissionName, moduleName, menuLabel, menuIcon, menuRoute, sortOrder)
VALUES ('qrcode.approve', 'Setujui Permintaan QR Code', 'qrcode', 'Permintaan QR Code', 'clock', '/cms/qrcode/permintaan', 8);

-- Super Admin & Editor (mengikuti preseden qrcode.* lain di 0037).
INSERT IGNORE INTO map_role_permission (roleID, permissionID)
SELECT r.roleID, p.permissionID FROM ms_role r, lk_permission p
WHERE r.roleName IN ('Super Admin', 'Editor') AND p.permissionCode = 'qrcode.approve';

-- Kontak PIC (Penanggung Jawab) QR Code — dibaca qrcoderequest_service untuk
-- notifikasi WhatsApp + kartu "Konfirmasi via WhatsApp" di halaman pengajuan.
-- Group 'layanan' sama dengan shortlink_pic_*; isHide=0 (tampil di App
-- Settings CMS). Nilai kosong = fitur notifikasi WA dorman sampai diisi admin.
INSERT IGNORE INTO ms_setting (settingGroup, settingKey, settingLabel, settingValue, isHide) VALUES
  ('layanan', 'qrcode_pic_name',     'Nama PIC QR Code',           '', 0),
  ('layanan', 'qrcode_pic_whatsapp', 'Nomor WhatsApp PIC QR Code', '', 0);
