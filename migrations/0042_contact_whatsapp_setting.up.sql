-- ============================================================
-- Nomor WhatsApp kontak publik (floating button + section "Hubungi Kami" di
-- beranda) — disimpan format tampilan biasa ("+62 851-1133-2861", bukan
-- digit polos) supaya gampang diedit admin CMS apa adanya; link wa.me
-- dibentuk di frontend dengan membuang semua karakter non-digit dari nilai
-- ini. isHide=0 (tampil di CMS App Settings, group "Kontak").
-- ============================================================

INSERT IGNORE INTO ms_setting (settingGroup, settingKey, settingLabel, settingValue, isHide) VALUES
  ('kontak', 'contact_whatsapp', 'Nomor WhatsApp Publik (Beranda & Floating Button)', '+62 851-1133-2861', 0);
