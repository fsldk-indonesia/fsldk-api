-- Perbaikan lanjutan hasil review kedua (miss-development-prompt-2.md).

-- Poin 1: Form Pendataan (Form Builder Levelisasi) pindah dari CMS Utama ke
-- CMS Puskomnas — Puskomnas yang mengelola struktur form penilaian nasional,
-- bukan admin sistem FSLDK Utama.
UPDATE lk_permission SET menuRoute = '/cms-puskomnas/submission-forms'
  WHERE permissionCode = 'submission_form.view';

-- Poin 8: "Verifikasi Wilayah" dan "Persetujuan Wilayah" digabung jadi satu
-- menu (antreannya identik, DL-06/OQ-04 sudah menegaskan keduanya satu
-- keputusan oleh satu role) — submission.approve.tier1 tidak lagi punya
-- entri menu sendiri (route-nya juga sudah dihapus di frontend), submission
-- .review.tier1 yang bertahan menampung kedua kemampuan (relabel).
UPDATE lk_permission SET menuRoute = NULL, menuLabel = NULL
  WHERE permissionCode = 'submission.approve.tier1';
UPDATE lk_permission SET menuLabel = 'Verifikasi & Persetujuan'
  WHERE permissionCode = 'submission.review.tier1';
