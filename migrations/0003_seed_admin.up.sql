-- ============================================================
-- FSLDK API — Seed Akun Super Admin Awal
-- Email    : noreply-fsldk@gmail.com
-- Password : abc123 (sudah di-hash bcrypt — segera ganti setelah login pertama)
-- Idempoten: hanya insert bila email belum terdaftar.
-- ============================================================

INSERT INTO ms_user (roleID, fullName, email, password, emailVerifiedDate, mustChangePassword, isActive, createdDate)
SELECT r.roleID, 'Admin FSLDK', 'noreply-fsldk@gmail.com',
       '$2a$10$npFVVDfn9mAl8TObGTrMJOSJkT43xrya8M/etWI28tsd1mFljaGJ6',
       NOW(), 0, 1, NOW()
FROM ms_role r
WHERE r.roleName = 'Super Admin'
  AND NOT EXISTS (
      SELECT 1 FROM ms_user u WHERE u.email = 'noreply-fsldk@gmail.com'
  );
