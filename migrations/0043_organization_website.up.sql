-- ============================================================
-- Kolom website organisasi (LDK/Puskomda/Puskomnas) — ms_organization
-- sebelumnya tidak punya field untuk link website resmi organisasi, cuma
-- contactEmail/contactPhone. Diisi lewat form profil CMS organisasi,
-- ditampilkan di Direktori Jaringan publik (statistik-jaringan).
-- ============================================================

ALTER TABLE ms_organization ADD COLUMN websiteURL VARCHAR(255) NULL AFTER contactPhone;
