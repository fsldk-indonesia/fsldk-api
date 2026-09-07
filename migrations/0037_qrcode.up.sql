-- ============================================================
-- FSLDK API — Fitur QR Code (pengganti "tautan pendek" jadi gambar QR)
-- Tabel ms_qrcode + permission qrcode.* + pemetaan role.
-- Gambar QR meng-encode destinationURL LANGSUNG (tidak ada kunci / redirect /
-- pelacakan pindaian — itu akan sama saja dengan Shortlink). Baris menyimpan
-- opsi kustomisasi gambar: warna, ikon tengah, teks caption. Ikon tengah bisa
-- berupa URL unggahan (/uploads/...) ATAU data URI PNG (ikon preset yang
-- dikomposisi di frontend) — makanya centerIconURL bertipe TEXT.
-- centerIconKey menandai preset yang dipilih (fsldk/instagram/dst) supaya form
-- CMS bisa memulihkan pilihan & meng-generate ulang saat warna berubah.
-- Idempoten: aman dijalankan ulang (CREATE TABLE IF NOT EXISTS / INSERT IGNORE).
-- ============================================================

CREATE TABLE IF NOT EXISTS ms_qrcode (
    qrCodeID        BIGINT AUTO_INCREMENT PRIMARY KEY,
    label           VARCHAR(255) NULL,
    destinationURL  VARCHAR(1000) NOT NULL,
    foregroundColor VARCHAR(9) NOT NULL DEFAULT '#16211C',
    backgroundColor VARCHAR(9) NOT NULL DEFAULT '#FFFFFF',
    centerIconURL   TEXT NULL,
    centerIconKey   VARCHAR(32) NULL,
    captionText     VARCHAR(120) NULL,
    createdDate     DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    createdBy       BIGINT NULL,
    updatedDate     DATETIME NULL,
    updatedBy       BIGINT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- Permission — qrcode.view jadi anak pertama grup sidebar "QR Code" (prefix
-- menuRoute /cms/qrcode/, dikelompokkan collapsible di cms-layout.component.ts
-- SIDEBAR_GROUPS, pola sama seperti "Shortlink" & "Kantong Amal"). sortOrder 7
-- menentukan posisi grup di sidebar.
INSERT IGNORE INTO lk_permission (permissionCode, permissionName, moduleName, menuLabel, menuIcon, menuRoute, sortOrder) VALUES
('qrcode.view',   'Lihat QR Code',  'qrcode', 'Daftar QR Code', 'qr-code', '/cms/qrcode/list', 7),
('qrcode.create', 'Tambah QR Code', 'qrcode', NULL, NULL, NULL, NULL),
('qrcode.update', 'Ubah QR Code',   'qrcode', NULL, NULL, NULL, NULL),
('qrcode.delete', 'Hapus QR Code',  'qrcode', NULL, NULL, NULL, NULL);

-- Super Admin: seluruh qrcode.* (wildcard seed 0002 tidak menjangkau baris
-- baru, perlu di-insert eksplisit — sama seperti preseden shortlink).
INSERT IGNORE INTO map_role_permission (roleID, permissionID)
SELECT r.roleID, p.permissionID FROM ms_role r JOIN lk_permission p
WHERE r.roleName = 'Super Admin' AND p.permissionCode LIKE 'qrcode.%';

-- Editor: seluruh qrcode.*
INSERT IGNORE INTO map_role_permission (roleID, permissionID)
SELECT r.roleID, p.permissionID FROM ms_role r JOIN lk_permission p
WHERE r.roleName = 'Editor' AND p.permissionCode IN (
  'qrcode.view','qrcode.create','qrcode.update','qrcode.delete'
);
