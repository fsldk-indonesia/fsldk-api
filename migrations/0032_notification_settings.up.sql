-- ============================================================
-- Saklar global notifikasi WhatsApp (revision-prompt-4.md item 4) —
-- mengaktifkan/menonaktifkan SEMUA pengiriman WhatsApp platform (donasi,
-- withdrawal OTP, balasan shortlink request, dst), digerbang satu titik di
-- jobqueue_service.executeWhatsAppTemplate. isHide=0 (sengaja TAMPIL di
-- App Settings CMS, beda dari kantong_amal.withdrawal_otp_email).
-- ============================================================

INSERT IGNORE INTO ms_setting (settingGroup, settingKey, settingLabel, settingValue, isHide) VALUES
  ('notifikasi', 'whatsapp_enabled', 'Aktifkan Notifikasi WhatsApp', 'true', 0);
