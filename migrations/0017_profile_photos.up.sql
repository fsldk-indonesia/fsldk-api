-- Foto profil untuk Profil Saya (Kader/CMS) & Profil LDK.
--
-- ms_user.photoURL sudah ada sejak 0001_init, tapi HANYA pernah diisi oleh
-- auto-sync login Google (lihat auth_service_impl.go LoginGoogle) — akun
-- tersebut ditimpa ulang setiap login Google berikutnya kalau foto Google-nya
-- berubah. customPhotoURL adalah kolom TERPISAH untuk foto yang diunggah
-- sendiri lewat Profil Saya, supaya tidak pernah tertimpa oleh sinkronisasi
-- Google — prioritas tampil: customPhotoURL (kalau ada) > photoURL (Google) >
-- inisial huruf (fallback di frontend bila keduanya kosong).
ALTER TABLE ms_user ADD COLUMN customPhotoURL VARCHAR(255) NULL AFTER photoURL;

-- ms_organization belum punya kolom foto/logo sama sekali — dipakai Profil
-- LDK, fallback ke avatar inisial huruf di frontend bila kosong.
ALTER TABLE ms_organization ADD COLUMN photoURL VARCHAR(255) NULL AFTER isActive;
