-- 0057_report_revamp_menu.up.sql
-- Laporan Puskomda/Puskomnas sekarang punya halaman hub (2 card: Levelisasi
-- & Pendataan) di /reports, bukan langsung ke tabel di /reports/wilayah
-- atau /reports/nasional (TechSpec-Report-Revamp.md §6).
UPDATE lk_permission SET menuRoute = '/cms-puskomda/reports'
  WHERE permissionCode = 'report.region.view';

UPDATE lk_permission SET menuRoute = '/cms-puskomnas/reports'
  WHERE permissionCode = 'report.national.view';
