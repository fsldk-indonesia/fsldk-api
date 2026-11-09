-- Opsi "Lainnya" pada field pilihan (SELECT/RADIO/MULTISELECT/CHECKBOX).
--
-- Opsi bertanda isOther=1 membuat pengisi form dapat mengetik jawaban manual
-- (disimpan di tr_submission_answer.valueOtherText). Opsi "Lainnya" tetap opsi
-- biasa untuk scoring otomatis: skornya diatur lewat kolom score yang sama.
ALTER TABLE ms_submission_form_field_option
    ADD COLUMN isOther TINYINT(1) NOT NULL DEFAULT 0 AFTER score;

ALTER TABLE tr_submission_answer
    ADD COLUMN valueOtherText VARCHAR(500) NULL AFTER valueFileName;
