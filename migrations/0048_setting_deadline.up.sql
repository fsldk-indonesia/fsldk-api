-- ============================================================
-- Tanggal deadline operasional (WhatsApp Integration, Hosting Server, Rent
-- Domain, Certificate Website) — dibaca dashboard CMS Utama untuk banner
-- peringatan closeable saat sisa waktu <=30 hari atau sudah lewat (lihat
-- dashboard_service_impl.go). settingValue sengaja kosong di sini, diisi
-- admin lewat App Settings CMS (group "deadline") dalam format YYYY-MM-DD.
-- isHide=0 supaya tampil di App Settings.
-- ============================================================

INSERT IGNORE INTO ms_setting (settingGroup, settingKey, settingLabel, settingValue, isHide) VALUES
  ('deadline', 'whatsapp_integration', 'Deadline WhatsApp Integration', '', 0),
  ('deadline', 'hosting_server', 'Deadline Hosting Server', '', 0),
  ('deadline', 'rent_domain', 'Deadline Rent Domain', '', 0),
  ('deadline', 'certificate_website', 'Deadline Certificate Website', '', 0);
