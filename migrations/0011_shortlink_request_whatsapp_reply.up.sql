-- ============================================================
-- FSLDK API — Shortlink Request: jalur approval kedua via balasan WhatsApp
-- (§1a.5 techspec). Menambah kolom reviewedVia untuk membedakan penyelesaian
-- lewat CMS vs balasan WhatsApp PIC.
-- ============================================================

ALTER TABLE ms_shortlink_request
    ADD COLUMN reviewedVia VARCHAR(20) NOT NULL DEFAULT 'cms' AFTER reviewedBy;
