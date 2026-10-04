-- ============================================================
-- Kolom slug untuk ms_gallery — URL publik Galeri sebelumnya pakai ID
-- numerik (/galeri/1), diganti slug ramah-URL turunan eventName (pola sama
-- dengan newsSlug/eventSlug, lihat base/slug.Make() + uniqueSlug() di
-- gallery_service_impl.go). Backfill slug data existing dari eventName,
-- dedup kalau ada tabrakan, baru dikunci NOT NULL UNIQUE.
--
-- Ditulis kompatibel MySQL 5.7 (server dev aktual, BUKAN 8.0+ seperti
-- didokumentasikan di docs/INSTALLATION.md) — sengaja TIDAK memakai
-- REGEXP_REPLACE() atau ROW_NUMBER() OVER(...) (keduanya fitur 8.0+):
-- slugify pakai rantai REPLACE(), dedup pakai user-variable running-count
-- (idiom standar pre-8.0 pengganti ROW_NUMBER, masih berfungsi di 5.7).
-- ============================================================

ALTER TABLE ms_gallery ADD COLUMN gallerySlug VARCHAR(255) NULL AFTER eventTheme;

UPDATE ms_gallery
SET gallerySlug = TRIM(BOTH '-' FROM
    REPLACE(REPLACE(REPLACE(REPLACE(REPLACE(REPLACE(REPLACE(REPLACE(REPLACE(REPLACE(REPLACE(REPLACE(
      LOWER(eventName)
      , ' ', '-')
      , '_', '-')
      , '.', '')
      , ',', '')
      , ':', '')
      , ';', '')
      , '/', '-')
      , '\\', '-')
      , '(', '')
      , ')', '')
      , '\'', '')
      , '"', '')
  )
WHERE gallerySlug IS NULL OR gallerySlug = '';

-- Kolaps hyphen beruntun (mis. dari "A - B" -> "a--b") yang bisa muncul dari
-- rantai REPLACE di atas — beberapa pass berjaga-jaga untuk tanda baca
-- berurutan panjang.
UPDATE ms_gallery SET gallerySlug = TRIM(BOTH '-' FROM REPLACE(REPLACE(REPLACE(gallerySlug, '----', '-'), '--', '-'), '--', '-'))
WHERE gallerySlug LIKE '%--%';

UPDATE ms_gallery SET gallerySlug = 'galeri' WHERE gallerySlug IS NULL OR gallerySlug = '';

-- Dedup tabrakan slug dengan menambahkan suffix -{galleryID} pada baris
-- kedua dst per nilai slug, urut dari ID terkecil — running-count lewat
-- user variable (ROW_NUMBER() OVER() belum ada di MySQL 5.7).
SET @rn := 0, @prev_slug := NULL;

UPDATE ms_gallery g
JOIN (
    SELECT galleryID, gallerySlug,
           @rn := IF(@prev_slug <=> gallerySlug, @rn + 1, 1) AS rn,
           @prev_slug := gallerySlug AS _prev
    FROM ms_gallery
    ORDER BY gallerySlug, galleryID
) ranked ON g.galleryID = ranked.galleryID
SET g.gallerySlug = CONCAT(g.gallerySlug, '-', g.galleryID)
WHERE ranked.rn > 1;

ALTER TABLE ms_gallery MODIFY COLUMN gallerySlug VARCHAR(255) NOT NULL;
ALTER TABLE ms_gallery ADD UNIQUE INDEX idx_gallery_slug (gallerySlug);
