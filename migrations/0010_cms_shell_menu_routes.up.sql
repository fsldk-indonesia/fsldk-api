-- Memisahkan menuRoute lk_permission ke 4 shell CMS terpisah (FSLDK Utama
-- tetap /cms/, LDK/Puskomda/Puskomnas pindah ke prefix masing-masing) sesuai
-- keputusan miss-development-clarification.md poin 1-4. Menu Form Builder
-- (submission_form.view) TETAP di /cms/ — itu admin config skema form
-- (CMS Utama), bukan "Pendataan" milik LDK (submission.create).
--
-- Sekaligus menyatukan label menu yang sebelumnya tampak dobel karena berada
-- dalam satu sidebar gabungan (poin 11/12) — sekarang aman disamakan karena
-- masing-masing sudah baris terpisah di shell-nya sendiri.

-- ---------- CMS LDK ----------
UPDATE lk_permission SET menuRoute = '/cms-ldk/organization/profile'
  WHERE permissionCode = 'organization.profile.manage';
UPDATE lk_permission SET menuRoute = '/cms-ldk/submissions/pendataan'
  WHERE permissionCode = 'submission.create';
UPDATE lk_permission SET menuRoute = '/cms-ldk/submissions/status'
  WHERE permissionCode = 'submission.view';
UPDATE lk_permission SET menuRoute = '/cms-ldk/kaders/persetujuan'
  WHERE permissionCode = 'submission.review.ldk';

-- ---------- CMS Puskomda ----------
UPDATE lk_permission SET menuRoute = '/cms-puskomda/organizations/ldk', menuLabel = 'LDK'
  WHERE permissionCode = 'organization.ldk.list';
UPDATE lk_permission SET menuRoute = '/cms-puskomda/submissions/verifikasi'
  WHERE permissionCode = 'submission.review.tier1';
UPDATE lk_permission SET menuRoute = '/cms-puskomda/submissions/persetujuan'
  WHERE permissionCode = 'submission.approve.tier1';
UPDATE lk_permission SET menuRoute = '/cms-puskomda/reports/wilayah', menuLabel = 'Laporan'
  WHERE permissionCode = 'report.region.view';

-- ---------- CMS Puskomnas ----------
UPDATE lk_permission SET menuRoute = '/cms-puskomnas/organizations/ldk-nasional', menuLabel = 'LDK'
  WHERE permissionCode = 'organization.ldk.list.national';
UPDATE lk_permission SET menuRoute = '/cms-puskomnas/organizations/puskomda'
  WHERE permissionCode = 'organization.puskomda.list';
UPDATE lk_permission SET menuRoute = '/cms-puskomnas/submissions/verifikasi-akhir'
  WHERE permissionCode = 'submission.review.tier2';
UPDATE lk_permission SET menuRoute = '/cms-puskomnas/submissions/penetapan-level'
  WHERE permissionCode = 'submission.level.establish';
UPDATE lk_permission SET menuRoute = '/cms-puskomnas/submissions/publikasi'
  WHERE permissionCode = 'submission.publish';
UPDATE lk_permission SET menuRoute = '/cms-puskomnas/reports/nasional', menuLabel = 'Laporan'
  WHERE permissionCode = 'report.national.view';
