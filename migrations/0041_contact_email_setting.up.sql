-- ============================================================
-- Email kontak publik ("Hubungi Kami" beranda + /tentang/kontak) — dulu
-- hardcoded di frontend (fsldkindonesia29@gmail.com), dipindah ke App
-- Settings supaya bisa diubah tanpa deploy. isHide=0 (tampil di CMS App
-- Settings, group "Kontak").
-- ============================================================

INSERT IGNORE INTO ms_setting (settingGroup, settingKey, settingLabel, settingValue, isHide) VALUES
  ('kontak', 'contact_email', 'Email Kontak Publik (Beranda & Hubungi Kami)', 'fsldkindonesia29@gmail.com', 0);
