-- ============================================================
-- FSLDK API — Rapimnas 1 FSLDK Indonesia 2026 (ms_rapimnas_setting + 6 tabel anak)
-- Modul singleton + child-lists untuk microsite RAPIMNAS 1 FSLDK Indonesia
-- 2026 (/rapimnas) — satu baris pengaturan (id=1) + 6 koleksi ber-sortOrder
-- (galeri, home card, rundown hari+acara, arsip/resource, titik penjemputan,
-- kontak). CMS "Rapimnas Setup" (Super Admin only). Konten non-gambar diseed
-- dari repo referensi (rapimnas26, Next.js) agar halaman hasil porting identik
-- di hari pertama; field gambar diseed NULL (diunggah manual via CMS nanti).
-- Idempoten: aman dijalankan ulang (CREATE TABLE IF NOT EXISTS / INSERT IGNORE).
-- ============================================================

CREATE TABLE IF NOT EXISTS ms_rapimnas_setting (
    id                                  TINYINT UNSIGNED PRIMARY KEY,

    heroBadgeText                       VARCHAR(100)  NOT NULL DEFAULT '',
    heroTitle                           VARCHAR(255)  NOT NULL DEFAULT '',
    heroDateRangeText                   VARCHAR(100)  NOT NULL DEFAULT '',
    heroTaglineQuote                    VARCHAR(500)  NOT NULL DEFAULT '',
    heroImageUrl                        VARCHAR(500)  NULL,

    countdownTargetDate                 DATETIME      NULL,

    feature1IconKey                     VARCHAR(50)   NOT NULL DEFAULT '',
    feature1Title                       VARCHAR(255)  NOT NULL DEFAULT '',
    feature1Desc                        TEXT          NOT NULL,
    feature2IconKey                     VARCHAR(50)   NOT NULL DEFAULT '',
    feature2Title                       VARCHAR(255)  NOT NULL DEFAULT '',
    feature2Desc                        TEXT          NOT NULL,

    ctaTitle                            VARCHAR(255)  NOT NULL DEFAULT '',
    ctaDescription                      TEXT          NOT NULL,
    ctaButtonLabel                      VARCHAR(100)  NOT NULL DEFAULT '',
    ctaMascotImageUrl                   VARCHAR(500)  NULL,

    footerContactEmail                  VARCHAR(255)  NOT NULL DEFAULT '',
    footerCopyrightText                 VARCHAR(255)  NOT NULL DEFAULT '',
    footerIgHandle                      VARCHAR(100)  NOT NULL DEFAULT '',
    footerIgUrl                         VARCHAR(500)  NOT NULL DEFAULT '',
    footerTiktokHandle                  VARCHAR(100)  NOT NULL DEFAULT '',
    footerTiktokUrl                     VARCHAR(500)  NOT NULL DEFAULT '',

    jadwalHeaderSubtitle                VARCHAR(500)  NOT NULL DEFAULT '',

    tentangTaglineQuote                 VARCHAR(500)  NOT NULL DEFAULT '',
    tentangDescParagraph1               TEXT          NOT NULL,
    tentangDescParagraph2               TEXT          NOT NULL,
    tentangVisiText                     TEXT          NOT NULL,
    tentangMisiJSON                     LONGTEXT      NULL,
    tentangTujuanJSON                   LONGTEXT      NULL,
    tentangKegiatanJSON                 LONGTEXT      NULL,

    pesertaEarlyBirdDateRange           VARCHAR(100)  NOT NULL DEFAULT '',
    pesertaRegulerDateRange             VARCHAR(100)  NOT NULL DEFAULT '',
    pesertaHargaNonSemarangEarlyBird    INT           NOT NULL DEFAULT 0,
    pesertaHargaNonSemarangReguler      INT           NOT NULL DEFAULT 0,
    pesertaHargaSemarangEarlyBird       INT           NOT NULL DEFAULT 0,
    pesertaHargaSemarangReguler         INT           NOT NULL DEFAULT 0,

    pesertaBankName                     VARCHAR(100)  NOT NULL DEFAULT '',
    pesertaBankAccountNumber            VARCHAR(50)   NOT NULL DEFAULT '',
    pesertaBankAccountHolder            VARCHAR(100)  NOT NULL DEFAULT '',
    pesertaGuidebookUrl                 VARCHAR(500)  NULL,
    pesertaGoogleFormUrl                VARCHAR(500)  NOT NULL DEFAULT '',
    pesertaMapEmbedUrl                  VARCHAR(1000) NULL,

    panitiaIsOpen                       BOOLEAN       NOT NULL DEFAULT FALSE,
    panitiaClosedMessage                TEXT          NOT NULL,

    updatedDate                         DATETIME      NULL,
    updatedBy                           BIGINT        NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS ms_rapimnas_gallery_image (
    id        BIGINT AUTO_INCREMENT PRIMARY KEY,
    imageUrl  VARCHAR(500) NOT NULL,
    sortOrder INT          NOT NULL DEFAULT 0,
    INDEX idx_rapimnas_gallery_image_sort (sortOrder)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS ms_rapimnas_home_card (
    id          BIGINT AUTO_INCREMENT PRIMARY KEY,
    iconKey     VARCHAR(50)  NOT NULL DEFAULT '',
    title       VARCHAR(255) NOT NULL DEFAULT '',
    description TEXT         NOT NULL,
    sortOrder   INT          NOT NULL DEFAULT 0,
    INDEX idx_rapimnas_home_card_sort (sortOrder)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS ms_rapimnas_rundown_day (
    id        BIGINT AUTO_INCREMENT PRIMARY KEY,
    dayLabel  VARCHAR(100) NOT NULL DEFAULT '',
    dateText  VARCHAR(100) NOT NULL DEFAULT '',
    sortOrder INT          NOT NULL DEFAULT 0,
    INDEX idx_rapimnas_rundown_day_sort (sortOrder)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS ms_rapimnas_rundown_event (
    id          BIGINT AUTO_INCREMENT PRIMARY KEY,
    dayID       BIGINT       NOT NULL,
    `time`      VARCHAR(100) NOT NULL DEFAULT '',
    title       VARCHAR(255) NOT NULL DEFAULT '',
    description TEXT         NOT NULL,
    venue       VARCHAR(255) NOT NULL DEFAULT '',
    sortOrder   INT          NOT NULL DEFAULT 0,
    CONSTRAINT fk_rapimnas_rundown_event_day FOREIGN KEY (dayID)
        REFERENCES ms_rapimnas_rundown_day (id) ON DELETE CASCADE,
    INDEX idx_rapimnas_rundown_event_day (dayID),
    INDEX idx_rapimnas_rundown_event_sort (dayID, sortOrder)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS ms_rapimnas_resource (
    id          BIGINT AUTO_INCREMENT PRIMARY KEY,
    title       VARCHAR(255) NOT NULL DEFAULT '',
    description TEXT         NOT NULL,
    iconKey     VARCHAR(50)  NOT NULL DEFAULT '',
    url         VARCHAR(500) NOT NULL DEFAULT '',
    buttonLabel VARCHAR(100) NOT NULL DEFAULT '',
    isVisible   BOOLEAN      NOT NULL DEFAULT TRUE,
    sortOrder   INT          NOT NULL DEFAULT 0,
    INDEX idx_rapimnas_resource_sort (sortOrder)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS ms_rapimnas_pickup_location (
    id          BIGINT AUTO_INCREMENT PRIMARY KEY,
    name        VARCHAR(255) NOT NULL DEFAULT '',
    type        VARCHAR(50)  NOT NULL DEFAULT '',
    description TEXT         NOT NULL,
    mapLink     VARCHAR(500) NOT NULL DEFAULT '',
    sortOrder   INT          NOT NULL DEFAULT 0,
    INDEX idx_rapimnas_pickup_location_sort (sortOrder)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS ms_rapimnas_contact (
    id          BIGINT AUTO_INCREMENT PRIMARY KEY,
    contactType ENUM('peserta_cp', 'footer_wa') NOT NULL,
    name        VARCHAR(100) NOT NULL DEFAULT '',
    phoneNumber VARCHAR(30)  NOT NULL DEFAULT '',
    sortOrder   INT          NOT NULL DEFAULT 0,
    INDEX idx_rapimnas_contact_type_sort (contactType, sortOrder)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- ---------- Seed: baris singleton (konten asli dari repo referensi rapimnas26) ----------

INSERT IGNORE INTO ms_rapimnas_setting (
    id,
    heroBadgeText, heroTitle, heroDateRangeText, heroTaglineQuote, heroImageUrl,
    countdownTargetDate,
    feature1IconKey, feature1Title, feature1Desc,
    feature2IconKey, feature2Title, feature2Desc,
    ctaTitle, ctaDescription, ctaButtonLabel, ctaMascotImageUrl,
    footerContactEmail, footerCopyrightText, footerIgHandle, footerIgUrl, footerTiktokHandle, footerTiktokUrl,
    jadwalHeaderSubtitle,
    tentangTaglineQuote, tentangDescParagraph1, tentangDescParagraph2, tentangVisiText,
    tentangMisiJSON, tentangTujuanJSON, tentangKegiatanJSON,
    pesertaEarlyBirdDateRange, pesertaRegulerDateRange,
    pesertaHargaNonSemarangEarlyBird, pesertaHargaNonSemarangReguler,
    pesertaHargaSemarangEarlyBird, pesertaHargaSemarangReguler,
    pesertaBankName, pesertaBankAccountNumber, pesertaBankAccountHolder,
    pesertaGuidebookUrl, pesertaGoogleFormUrl, pesertaMapEmbedUrl,
    panitiaIsOpen, panitiaClosedMessage
) VALUES (
    1,
    'RAPIMNAS 1 FSLDK INDONESIA 2026',
    'Rapat Pimpinan Nasional FSLDK Indonesia 2026',
    'Universitas Diponegoro, Semarang • 12 - 15 November 2026',
    '"Diponegoro''s Spirit: Berdikarya dalam Gerak, Berdampak bagi Bangsa"',
    NULL,
    '2026-11-12 08:00:00',
    'user-group', 'Ukhuwah', 'Mempererat jejaring kolaborasi antarlembaga.',
    'file-sliders', 'Sistem', 'Merumuskan arah gerak dakwah nasional.',
    'Mari Berkontribusi!',
    'Segera daftarkan diri Anda dan ikuti rangkaian acara dari tanggal 12 hingga 15 November 2026.',
    'Daftar Peserta/Delegasi',
    NULL,
    'rapimnasone@gmail.com',
    '© 2026 Divisi Multimedia RAPIMNAS FSLDK. All rights reserved.',
    '@rapimnas.fsldk', 'https://instagram.com/rapimnas.fsldk',
    '@rapimnas.fsldk', 'https://tiktok.com/@rapimnas.fsldk',
    'Rangkaian acara diselenggarakan pada tanggal 12 - 15 November 2026 di Semarang.',
    '"Diponegoro''s Spirit: Berdikarya dalam Gerak, Berdampak bagi Bangsa"',
    'Rapimnas × FSLDK 2026 merupakan forum nasional yang mempertemukan pimpinan dan perwakilan Lembaga Dakwah Kampus (LDK) dari berbagai perguruan tinggi di Indonesia. Kegiatan ini menjadi ruang silaturahmi, konsolidasi, dan pertukaran gagasan dalam memperkuat peran serta sinergi LDK di tingkat nasional.',
    'Melalui rangkaian agenda persidangan, diskusi, dan forum silaturahmi, Rapimnas × FSLDK 2026 diharapkan mampu menghasilkan gagasan, rekomendasi, serta arah gerak bersama yang relevan dengan dinamika dakwah kampus. Kegiatan ini juga menjadi momentum untuk memperluas jejaring, mempererat ukhuwah, dan membangun kolaborasi antarlembaga demi memberikan kontribusi positif bagi kehidupan kampus dan masyarakat.',
    '"Mewujudkan Rapimnas 1 FSLDK Indonesia 2026 sebagai ruang konsolidasi dan kolaborasi untuk menguatkan arah gerak dakwah mahasiswa yang progresif dan berdampak bagi umat dan bangsa."',
    '["Menguatkan konsolidasi nasional antar-Puskomda dan Lembaga Dakwah Kampus sebagai bagian dari upaya menyatukan arah gerak dakwah mahasiswa.","Membangun ruang kolaborasi dan pertukaran gagasan antar-elemen FSLDK Indonesia dalam merespons isu dan tantangan strategis umat dan bangsa.","Merumuskan arah gerak dan rekomendasi strategis yang relevan dengan kebutuhan dakwah mahasiswa di tingkat nasional maupun daerah.","Mendorong lahirnya inisiatif dan kontribusi nyata yang dapat diimplementasikan oleh Puskomda dan LDK pasca-Rapimnas.","Menumbuhkan semangat kepemimpinan dan kebermanfaatan bagi peserta agar mampu menjadi bagian dari gerakan mahasiswa yang memberikan dampak nyata di lingkungan masing-masing."]',
    '["Mempererat ukhuwah dan silaturahmi antarpimpinan serta anggota Lembaga Dakwah Kampus di tingkat nasional.","Memperkuat koordinasi, komunikasi, dan sinergi antar-LDK dalam menjalankan peran dakwah di lingkungan perguruan tinggi.","Menjadi wadah diskusi dan pertukaran gagasan mengenai tantangan serta peluang pengembangan dakwah kampus.","Menyusun rekomendasi dan arah gerak bersama yang sesuai dengan kebutuhan serta dinamika LDK di Indonesia.","Membangun jejaring kolaborasi yang berkelanjutan antarlembaga untuk memberikan kontribusi positif bagi kampus dan masyarakat."]',
    '["Sidang Rapimnas","Malam Keakraban","Seminar Kepemudaan","Pelatihan Manajemen LDK (PMLDK)","Closing UMF","Gerakan Subuh Jamaah Nasional (GSJN)","Eco Movement x Semai Asa","Business Case Competition & Poster","Field Trip Semarang"]',
    '6 - 12 Oktober 2026', '13 - 26 Oktober 2026',
    450000, 500000,
    360000, 385000,
    'BSI', '7278224532', 'NAURA YAZMI',
    'https://canva.link/akvmw3lihntwuwo',
    'https://bit.ly/OpregPesertaRapimnas2026',
    'https://www.google.com/maps/d/embed?mid=1tneEBlxcaA6xdQSv-Y60v9_Cv9Iigm0&ehbc=2E312F',
    FALSE,
    'Pendaftaran Panitia belum dibuka. Nantikan informasi selanjutnya melalui media sosial resmi RAPIMNAS 1 FSLDK Indonesia 2026.'
);

-- ---------- Seed: ms_rapimnas_gallery_image ----------
-- SENGAJA tidak ada baris (0 rows) — tabel ini seluruhnya berisi URL gambar,
-- dan gambar tidak dimigrasikan (lihat scope decision #6 di spec); Super
-- Admin mengunggah foto galeri manual lewat CMS setelah rilis.

-- ---------- Seed: ms_rapimnas_home_card ("Rangkaian Kegiatan" teaser, Beranda) ----------

INSERT INTO ms_rapimnas_home_card (iconKey, title, description, sortOrder)
SELECT * FROM (
    SELECT 'clipboard-list' AS iconKey, 'Sidang Pleno' AS title, 'Menghimpun aspirasi dan merumuskan rekomendasi gerak FSLDK Indonesia.' AS description, 0 AS sortOrder
    UNION ALL SELECT 'user-group', 'Seminar Kepemudaan', 'Ruang diskusi generasi muda dalam menghadapi dinamika bangsa.', 1
    UNION ALL SELECT 'book-open', 'Pelatihan Manajemen (PMLDK)', 'Pengembangan kapasitas untuk mengelola LDK secara strategis.', 2
    UNION ALL SELECT 'map-pin', 'Field Trip Semarang', 'Mengeksplorasi budaya kota dan mempererat ukhuwah antardelegasi.', 3
) AS seed
WHERE NOT EXISTS (SELECT 1 FROM ms_rapimnas_home_card);

-- ---------- Seed: ms_rapimnas_rundown_day + ms_rapimnas_rundown_event ----------
-- Data persis dari src/components/Rundown.tsx (dokumen "Rancangan Guidebook
-- Rapimnas x FSLDK.docx"). Kolom `time`/`venue` dipecah dari field gabungan
-- "jam | lokasi" pada sumber aslinya; event tanpa lokasi eksplisit diberi
-- venue kosong.

INSERT IGNORE INTO ms_rapimnas_rundown_day (id, dayLabel, dateText, sortOrder) VALUES
(1, 'Hari Pertama', 'Kamis, 12 November 2026', 0),
(2, 'Hari Kedua', 'Jum''at, 13 November 2026', 1),
(3, 'Hari Ketiga', 'Sabtu, 14 November 2026', 2),
(4, 'Hari Keempat', 'Minggu, 15 November 2026', 3);

INSERT INTO ms_rapimnas_rundown_event (dayID, `time`, title, description, venue, sortOrder)
SELECT * FROM (
    SELECT 1 AS dayID, '' AS `time`, 'Kedatangan Peserta' AS title, 'Penyambutan akbar delegasi LDK dari seluruh Indonesia di Universitas Diponegoro.' AS description, '' AS venue, 0 AS sortOrder
    UNION ALL SELECT 1, '', 'Malam Keakraban Peserta', 'Momen untuk melepas penat, mempererat ukhuwah, dan membangun kedekatan antardelegasi.', '', 1
    UNION ALL SELECT 2, '', 'Tahajud Berjamaah', 'Memulai hari dengan ibadah dan munajat bersama.', '', 0
    UNION ALL SELECT 2, '08.00 - 11.30', 'Grand Opening RAPIMNAS', 'Pembukaan resmi rangkaian Rapimnas FSLDK Indonesia 2026.', 'Hall Gedung Kewirausahaan FEB Undip Lt 4', 1
    UNION ALL SELECT 2, '13.00 - 21.10', 'Sidang Komisi', 'Awal rangkaian sidang untuk mengevaluasi gerak bersama dan isu strategis.', 'Aula Gedung Art Center A', 2
    UNION ALL SELECT 2, '13.00 - 16.45', 'Seminar Kepemudaan', 'Ruang inspirasi bagi generasi muda untuk memperluas wawasan, mengasah perspektif, dan membangun semangat kepemimpinan dalam menghadapi tantangan zaman.', 'Hall Gedung Kewirausahaan FEB Undip Lt 4', 3
    UNION ALL SELECT 2, '13.00 - 15.00', 'Final Lomba', 'Awal rangkaian sidang untuk mengevaluasi gerak bersama dan isu strategis.', 'Aula FPP Undip', 4
    UNION ALL SELECT 2, '19.00 - 22.00', 'Closing UMF', 'Sesi diskusi inspiratif ''More Than What You See: Mengenal Palestina dari Sisi yang Jarang Kita Ceritakan''.', 'Masjid Kampus Undip', 5
    UNION ALL SELECT 3, '07.00 - 10.30', 'Eco Movement x Semai Asa', 'Aksi nyata kepedulian terhadap lingkungan sebagai bentuk tanggung jawab ekologis.', '', 0
    UNION ALL SELECT 3, '13.20 - 15.00', 'Sidang Komisi (Lanjutan)', 'Melanjutkan pembahasan agenda strategis nasional.', 'BBPMP Provinsi Jateng', 1
    UNION ALL SELECT 3, '15.30 - 17.45', 'Sidang Pemilihan Tuan Rumah RAPIMNAS 2', 'Sidang penentuan tuan rumah agenda selanjutnya.', 'BBPMP Provinsi Jateng', 2
    UNION ALL SELECT 3, '12.30 - 15.00', 'PMLDK', 'Sesi pengembangan kapasitas untuk membekali peserta dengan wawasan dan keterampilan dalam mengelola organisasi, membangun tim, serta merancang gerak LDK yang efektif.', 'BBPMP Provinsi Jateng', 3
    UNION ALL SELECT 3, '14.40 - 17.15', 'Bedah GD Kaderisasi & Sosialisasi Sensus Nasional', 'Ruang untuk memetakan kondisi serta arah kaderisasi FSLDK Indonesia secara menyeluruh.', 'BBPMP Provinsi Jateng', 4
    UNION ALL SELECT 3, '18.00 - 22.00', 'Grand Closing', 'Penutup rangkaian Rapimnas 1 FSLDK Indonesia 2026.', 'BBPMP Provinsi Jateng', 5
    UNION ALL SELECT 4, '04.00 - 06.30', 'Gerakan Subuh Jamaah Nasional (GSJN)', 'Momentum spiritual menyatukan langkah dalam ibadah salat Subuh berjamaah serentak.', 'Masjid Kampus Undip', 0
    UNION ALL SELECT 4, '07.00 - 17.00', 'Semarang Field Trip', 'Eksplorasi berbagai destinasi di Kota Semarang untuk mengenal kekayaan sejarah, budaya, dan suasana kota sekaligus mempererat kebersamaan antardelgasi.', '', 1
) AS seed
WHERE NOT EXISTS (SELECT 1 FROM ms_rapimnas_rundown_event);

-- ---------- Seed: ms_rapimnas_resource (Arsip & Dokumen) ----------
-- 2 entri terakhir diseed isVisible=FALSE — persis 2 entri yang dikomentari
-- (bukan dihapus) di src/app/arsip/page.tsx (Twibbon, Panduan Lomba Essai).

INSERT INTO ms_rapimnas_resource (title, description, iconKey, url, buttonLabel, isVisible, sortOrder)
SELECT * FROM (
    SELECT 'Logo & Maskot RAPIMNAS 1 2026' AS title, 'Unduh logo resmi dan Maskot RAPIMNAS 1 2026 Indonesia format PNG resolusi tinggi.' AS description, 'file-text' AS iconKey, 'https://drive.google.com/drive/folders/14r_q9l9CKuw-4fFvy64rzXUjjHait5r3?usp=sharing' AS url, 'Unduh Logo' AS buttonLabel, TRUE AS isVisible, 0 AS sortOrder
    UNION ALL SELECT 'Twibbon & Caption Publikasi', 'Mari meriahkan timeline media sosial dengan menggunakan Twibbon resmi RAPIMNAS 1. Sudah termasuk template caption untuk Instagram.', 'file-text', '#', 'Pasang Twibbon', FALSE, 1
    UNION ALL SELECT 'Panduan Lomba Essai Nasional', 'Buku panduan lengkap (syarat, ketentuan, dan timeline) Lomba Essai Nasional dalam rangka menyemarakkan RAPIMNAS 1 FSLDK Indonesia.', 'book-open', '#', 'Unduh Panduan', FALSE, 2
    UNION ALL SELECT 'Proposal Acara', 'Proposal lengkap untuk acara RAPIMNAS 1 FSLDK Indonesia 2026.', 'clipboard-list', 'https://docs.google.com/document/d/1x91GD0PbsdjJV3qMh0b5S1lhziZkqcHV/edit', 'Lihat Proposal', TRUE, 3
) AS seed
WHERE NOT EXISTS (SELECT 1 FROM ms_rapimnas_resource);

-- ---------- Seed: ms_rapimnas_pickup_location ----------

INSERT INTO ms_rapimnas_pickup_location (name, type, description, mapLink, sortOrder)
SELECT * FROM (
    SELECT 'Stasiun Tawang' AS name, 'Stasiun Kereta' AS type, '' AS description, 'https://maps.app.goo.gl/uhWsicESR7dhy6rA9' AS mapLink, 0 AS sortOrder
    UNION ALL SELECT 'Stasiun Poncol', 'Stasiun Kereta', '', 'https://maps.app.goo.gl/HGqB5AiWU6XBdVaM8', 1
    UNION ALL SELECT 'Bandara Ahmad Yani', 'Bandara', '', 'https://maps.app.goo.gl/YakbzkmQkKTNWucx9', 2
    UNION ALL SELECT 'Terminal Banyumanik', 'Terminal Bus', '', 'https://maps.app.goo.gl/a8YXt1uwqGtm9GkB9', 3
) AS seed
WHERE NOT EXISTS (SELECT 1 FROM ms_rapimnas_pickup_location);

-- ---------- Seed: ms_rapimnas_contact ----------

INSERT INTO ms_rapimnas_contact (contactType, name, phoneNumber, sortOrder)
SELECT * FROM (
    SELECT 'peserta_cp' AS contactType, 'Fakhri' AS name, '0895-3842-52700' AS phoneNumber, 0 AS sortOrder
    UNION ALL SELECT 'peserta_cp', 'Alya', '0823-2219-6244', 1
    UNION ALL SELECT 'footer_wa', 'Ghozi', '0813-9321-0245', 0
    UNION ALL SELECT 'footer_wa', 'Aina', '0813-2844-2500', 1
) AS seed
WHERE NOT EXISTS (SELECT 1 FROM ms_rapimnas_contact);

-- ---------- Seed: permission & menu CMS ----------

INSERT IGNORE INTO lk_permission (permissionCode, permissionName, moduleName, menuLabel, menuIcon, menuRoute, sortOrder)
VALUES ('rapimnas.view',   'Lihat Rapimnas Setup', 'rapimnas', 'Rapimnas Setup', 'calendar-days', '/cms/rapimnas-setup', 97),
       ('rapimnas.update', 'Ubah Rapimnas Setup',  'rapimnas', NULL, NULL, NULL, 97);

INSERT IGNORE INTO map_role_permission (roleID, permissionID)
SELECT r.roleID, p.permissionID FROM ms_role r, lk_permission p
WHERE r.roleName = 'Super Admin' AND p.permissionCode IN ('rapimnas.view', 'rapimnas.update');
-- HANYA Super Admin — konfigurasi event nasional, bukan untuk LDK/Puskomda/
-- Puskomnas admin (lihat spec scope decision #8). Route juga hanya dipasang
-- di tree /cms (FSLDK) sebagai lapis kedua, bukan bagian dari migration ini.
