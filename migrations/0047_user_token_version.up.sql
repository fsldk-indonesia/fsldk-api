-- ============================================================
-- tokenVersion — penanda "generasi" access token milik satu akun. Dibaca
-- sebagai klaim di access token saat terbit (lihat base/token.Claims) dan
-- dibandingkan ke nilai live di DB oleh middlewares.Auth() pada SETIAP
-- request; selisih (token lama vs DB baru) ditolak 401 sehingga pengguna
-- harus login ulang untuk menerbitkan token dengan versi terkini.
--
-- Dinaikkan (+1) oleh:
--   - user_service_impl.go Update()      — saat roleID akun diganti
--   - role_service_impl.go SetPermissions() — saat permission suatu role
--     diedit (menaikkan versi SEMUA akun yang sedang memegang role itu)
-- ============================================================

ALTER TABLE ms_user ADD COLUMN tokenVersion INT NOT NULL DEFAULT 1 AFTER roleID;
