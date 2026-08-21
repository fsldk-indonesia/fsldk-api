-- ============================================================
-- FSLDK API — Review, Approval & Levelisasi Workflow
-- tr_submission_review, lk_level, tr_levelisasi_result.
-- Idempoten: aman dijalankan ulang (CREATE TABLE IF NOT EXISTS / INSERT IGNORE).
-- ============================================================

CREATE TABLE IF NOT EXISTS tr_submission_review (
    reviewID       BIGINT AUTO_INCREMENT PRIMARY KEY,
    submissionID   BIGINT NOT NULL,
    tierLevel      ENUM('LDK','PUSKOMDA','PUSKOMNAS') NOT NULL,
    reviewerUserID BIGINT NOT NULL,
    decision       ENUM('APPROVED','REVISION_REQUESTED','REJECTED') NOT NULL,
    checklistJSON  TEXT NULL,
    note           TEXT NULL,
    reviewedDate   DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT fk_submission_review_submission FOREIGN KEY (submissionID) REFERENCES tr_submission(submissionID),
    CONSTRAINT fk_submission_review_reviewer FOREIGN KEY (reviewerUserID) REFERENCES ms_user(userID),
    INDEX idx_submission_review_submission (submissionID, reviewedDate)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS lk_level (
    levelCode  VARCHAR(20) PRIMARY KEY,
    levelLabel VARCHAR(100) NOT NULL,
    sortOrder  TINYINT NOT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

INSERT IGNORE INTO lk_level (levelCode, levelLabel, sortOrder) VALUES
('PRA_MULA', 'Pra Mula', 1),
('MULA',     'Mula',     2),
('MADYA',    'Madya',    3),
('MANDIRI',  'Mandiri',  4);

CREATE TABLE IF NOT EXISTS tr_levelisasi_result (
    resultID            BIGINT AUTO_INCREMENT PRIMARY KEY,
    submissionID        BIGINT NOT NULL,
    organizationID       BIGINT NOT NULL,
    periodID             BIGINT NULL,
    levelCode             VARCHAR(20) NOT NULL,
    justificationNote     TEXT NULL,
    establishedByUserID   BIGINT NOT NULL,
    establishedDate       DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    isPublished           TINYINT(1) NOT NULL DEFAULT 0,
    publishedDate         DATETIME NULL,
    isCurrent             TINYINT(1) NOT NULL DEFAULT 1,
    -- Menjamin hanya satu baris isCurrent=1 per organisasi di waktu manapun —
    -- baris historis (isCurrent=0) bernilai NULL di kolom ini sehingga tidak
    -- saling bentrok di unique index (pola sama seperti periodKeyForUnique, DL-14).
    isCurrentKey          BIGINT GENERATED ALWAYS AS (IF(isCurrent = 1, organizationID, NULL)) STORED,
    CONSTRAINT fk_levelisasi_result_submission FOREIGN KEY (submissionID) REFERENCES tr_submission(submissionID),
    CONSTRAINT fk_levelisasi_result_organization FOREIGN KEY (organizationID) REFERENCES ms_organization(organizationID),
    CONSTRAINT fk_levelisasi_result_period FOREIGN KEY (periodID) REFERENCES ms_submission_period(periodID),
    CONSTRAINT fk_levelisasi_result_level FOREIGN KEY (levelCode) REFERENCES lk_level(levelCode),
    CONSTRAINT fk_levelisasi_result_actor FOREIGN KEY (establishedByUserID) REFERENCES ms_user(userID),
    UNIQUE KEY uq_levelisasi_result_current (isCurrentKey),
    INDEX idx_levelisasi_result_submission (submissionID)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

INSERT IGNORE INTO lk_permission (permissionCode, permissionName, moduleName, menuLabel, menuIcon, menuRoute, sortOrder) VALUES
('submission.review.ldk',      'Persetujuan Kader',           'submission', 'Persetujuan Kader', 'user-check', '/cms/kaders/persetujuan', 9),
('submission.review.tier1',    'Verifikasi Wilayah',          'submission', 'Verifikasi', 'clipboard-check', '/cms/submissions/verifikasi', 12),
('submission.approve.tier1',   'Persetujuan Wilayah',         'submission', 'Persetujuan', 'check-circle', '/cms/submissions/persetujuan', 13),
('submission.review.tier2',    'Verifikasi Akhir Nasional',   'submission', 'Verifikasi Akhir', 'clipboard-check', '/cms/submissions/verifikasi-akhir', 14),
('submission.level.establish', 'Penetapan Levelisasi',        'submission', 'Penetapan Levelisasi', 'award', '/cms/submissions/penetapan-level', 15),
('submission.publish',         'Publikasi Hasil Levelisasi',  'submission', 'Publikasi Hasil', 'megaphone', '/cms/submissions/publikasi', 16),
('submission.reopen',          'Buka Kembali untuk Koreksi',  'submission', NULL, NULL, NULL, NULL),
('submission.reassess',        'Ajukan Reassessment',         'submission', NULL, NULL, NULL, NULL),
('kader.deactivate',           'Nonaktifkan Kader',           'submission', NULL, NULL, NULL, NULL);

INSERT IGNORE INTO map_role_permission (roleID, permissionID)
SELECT r.roleID, p.permissionID FROM ms_role r JOIN lk_permission p
WHERE r.roleName = 'Super Admin' AND p.permissionCode IN (
  'submission.review.ldk','submission.review.tier1','submission.approve.tier1','submission.review.tier2',
  'submission.level.establish','submission.publish','submission.reopen','submission.reassess','kader.deactivate'
);

INSERT IGNORE INTO map_role_permission (roleID, permissionID)
SELECT r.roleID, p.permissionID FROM ms_role r JOIN lk_permission p
WHERE r.roleName = 'LDK Admin' AND p.permissionCode IN ('submission.review.ldk','submission.reassess','kader.deactivate');

INSERT IGNORE INTO map_role_permission (roleID, permissionID)
SELECT r.roleID, p.permissionID FROM ms_role r JOIN lk_permission p
WHERE r.roleName = 'Puskomda Verifikator' AND p.permissionCode IN ('submission.review.tier1','submission.approve.tier1');

INSERT IGNORE INTO map_role_permission (roleID, permissionID)
SELECT r.roleID, p.permissionID FROM ms_role r JOIN lk_permission p
WHERE r.roleName = 'Puskomnas Verifikator' AND p.permissionCode IN (
  'submission.review.tier2','submission.level.establish','submission.publish','submission.reopen','submission.reassess'
);
