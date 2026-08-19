-- Portal Kader: Profil Saya menambahkan field Alamat swadaya (No Whatsapp
-- memakai kolom phoneNumber yang sudah ada di ms_user sejak 0001_init, belum
-- pernah diekspos di UI manapun sebelum ini). Tidak ada perubahan skema lain
-- untuk fitur "Isi Ulang Pendataan" (reassessment Kader tanpa re-generate
-- kode) maupun "Putihkan Kembali" (reinstate kader REJECTED) — keduanya murni
-- pemakaian ulang kolom status/formVersionID yang sudah ada di tr_submission
-- & ms_kader.
ALTER TABLE ms_user ADD COLUMN address VARCHAR(255) NULL AFTER phoneNumber;
