-- ============================================================
-- Hapus opsi ketersediaan "Available at Mahar" dari katalog buku — product
-- decision: hanya 2 status ketersediaan yang relevan (Full PDF / Preview
-- Only PDF). Buku yang sebelumnya memakai "Available at Mahar" dipindah ke
-- "Available Full PDF" DULU sebelum baris lookup-nya dihapus — FK
-- fk_catalogbook_availability (ms_catalog_book.availabilityTypeID ->
-- lk_book_availability_type.availabilityTypeID) akan menolak DELETE selama
-- masih ada baris yang mereferensikannya.
-- ============================================================

UPDATE ms_catalog_book
SET availabilityTypeID = (
    SELECT availabilityTypeID FROM lk_book_availability_type WHERE availabilityTypeName = 'Available Full PDF'
)
WHERE availabilityTypeID = (
    SELECT availabilityTypeID FROM lk_book_availability_type WHERE availabilityTypeName = 'Available at Mahar'
);

DELETE FROM lk_book_availability_type WHERE availabilityTypeName = 'Available at Mahar';
