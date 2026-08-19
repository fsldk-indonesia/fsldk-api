-- Enhancement: Flexible Scoring pada Form Builder & Consolidated Assessment
-- (design/development/enahnce-development-submission-dashboard/new-enhance-development.md)
--
-- Skala skor per field TIDAK dikunci ke 0-100/1-4 — dikonfigurasi bebas lewat
-- Form Builder (minScore/maxScore/weight per field, score per option). Skor
-- otomatis (Single Choice: SELECT/RADIO) tidak pernah disimpan terpisah,
-- selalu dihitung ulang dari jawaban tersimpan + score opsi yang dipilih.
-- tr_submission_field_score HANYA menyimpan skor manual (diberikan Puskomnas)
-- untuk field non-Single-Choice.
ALTER TABLE ms_submission_form_field
    ADD COLUMN useScoring    TINYINT(1) NOT NULL DEFAULT 0 AFTER helpText,
    ADD COLUMN scoringMethod VARCHAR(20) NULL AFTER useScoring,
    ADD COLUMN minScore      DECIMAL(10,2) NULL AFTER scoringMethod,
    ADD COLUMN maxScore      DECIMAL(10,2) NULL AFTER minScore,
    ADD COLUMN weight        DECIMAL(5,2) NULL AFTER maxScore;

ALTER TABLE ms_submission_form_field_option
    ADD COLUMN score DECIMAL(10,2) NULL AFTER isActive;

CREATE TABLE IF NOT EXISTS tr_submission_field_score (
    scoreID        BIGINT AUTO_INCREMENT PRIMARY KEY,
    submissionID   BIGINT NOT NULL,
    fieldID        BIGINT NOT NULL,
    rawScore       DECIMAL(10,2) NOT NULL,
    scoredByUserID BIGINT NOT NULL,
    createdDate    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updatedDate    DATETIME NULL,
    UNIQUE KEY uq_submission_field_score (submissionID, fieldID),
    CONSTRAINT fk_field_score_submission FOREIGN KEY (submissionID) REFERENCES tr_submission(submissionID) ON DELETE CASCADE,
    CONSTRAINT fk_field_score_field FOREIGN KEY (fieldID) REFERENCES ms_submission_form_field(fieldID)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
