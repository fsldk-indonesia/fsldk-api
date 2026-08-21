-- ============================================================
-- FSLDK API — Submission Core (draft, submit, jawaban, riwayat status)
-- tr_submission, tr_submission_answer, tr_submission_status_history, ms_kader.
-- Idempoten: aman dijalankan ulang (CREATE TABLE IF NOT EXISTS / INSERT IGNORE).
-- ============================================================

CREATE TABLE IF NOT EXISTS tr_submission (
    submissionID       BIGINT AUTO_INCREMENT PRIMARY KEY,
    formID             BIGINT NOT NULL,
    formVersionID      BIGINT NOT NULL,
    periodID           BIGINT NULL,
    organizationID     BIGINT NOT NULL,
    subjectType        ENUM('ORGANIZATION','KADER') NOT NULL,
    subjectReferenceID BIGINT NULL,
    submittedByUserID  BIGINT NOT NULL,
    status             VARCHAR(40) NOT NULL DEFAULT 'DRAFT',
    version            INT NOT NULL DEFAULT 1,
    -- MySQL memperlakukan setiap NULL sebagai nilai berbeda pada unique index.
    -- periodKeyForUnique menormalkan periodID NULL -> 0 agar form open-ended
    -- (Levelisasi/Sensus Kader) tetap tercegah duplikat.
    periodKeyForUnique BIGINT GENERATED ALWAYS AS (COALESCE(periodID, 0)) STORED,
    -- Aturan keunikan berbeda per subjectType, dan MySQL 5.7 tidak mendukung
    -- unique index bersyarat — disiasati lewat 2 generated column terpisah:
    -- ownerOrgKey hanya terisi untuk submission ORGANIZATION (Levelisasi: 1
    -- submission per organisasi), ownerUserKey hanya terisi untuk submission
    -- KADER (Sensus Kader: 1 submission per pengguna, ke LDK manapun — bukan
    -- per organizationID, karena banyak Kader boleh menunjuk LDK yang sama).
    -- Baris yang tidak relevan bernilai NULL sehingga tidak saling bentrok di
    -- unique index (perilaku NULL-tidak-unik MySQL dimanfaatkan, bukan dihindari).
    ownerOrgKey        BIGINT GENERATED ALWAYS AS (IF(subjectType = 'ORGANIZATION', organizationID, NULL)) STORED,
    ownerUserKey       BIGINT GENERATED ALWAYS AS (IF(subjectType = 'KADER', submittedByUserID, NULL)) STORED,
    submittedDate      DATETIME NULL,
    lastUpdatedDate    DATETIME NULL,
    createdDate        DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    createdBy          BIGINT NULL,
    updatedDate        DATETIME NULL,
    updatedBy          BIGINT NULL,
    CONSTRAINT fk_submission_form FOREIGN KEY (formID) REFERENCES ms_submission_form(formID),
    CONSTRAINT fk_submission_form_version FOREIGN KEY (formVersionID) REFERENCES ms_submission_form_version(versionID),
    CONSTRAINT fk_submission_period FOREIGN KEY (periodID) REFERENCES ms_submission_period(periodID),
    CONSTRAINT fk_submission_organization FOREIGN KEY (organizationID) REFERENCES ms_organization(organizationID),
    CONSTRAINT fk_submission_submitter FOREIGN KEY (submittedByUserID) REFERENCES ms_user(userID),
    UNIQUE KEY uq_submission_owner_org (ownerOrgKey, formID, periodKeyForUnique),
    UNIQUE KEY uq_submission_owner_user (ownerUserKey, formID, periodKeyForUnique),
    INDEX idx_submission_status (status),
    INDEX idx_submission_org_status (organizationID, status),
    INDEX idx_submission_submitter (submittedByUserID)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS tr_submission_answer (
    answerID      BIGINT AUTO_INCREMENT PRIMARY KEY,
    submissionID  BIGINT NOT NULL,
    fieldID       BIGINT NOT NULL,
    valueText     TEXT NULL,
    valueNumber   DECIMAL(18,4) NULL,
    valueDate     DATE NULL,
    valueOptionID BIGINT NULL,
    valueFileURL  VARCHAR(500) NULL,
    valueFileName VARCHAR(255) NULL,
    CONSTRAINT fk_submission_answer_submission FOREIGN KEY (submissionID) REFERENCES tr_submission(submissionID) ON DELETE CASCADE,
    CONSTRAINT fk_submission_answer_field FOREIGN KEY (fieldID) REFERENCES ms_submission_form_field(fieldID),
    CONSTRAINT fk_submission_answer_option FOREIGN KEY (valueOptionID) REFERENCES ms_submission_form_field_option(optionID),
    UNIQUE KEY uq_submission_answer_field (submissionID, fieldID),
    INDEX idx_submission_answer_submission (submissionID)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS tr_submission_status_history (
    historyID           BIGINT AUTO_INCREMENT PRIMARY KEY,
    submissionID         BIGINT NOT NULL,
    fromStatus            VARCHAR(40) NULL,
    toStatus               VARCHAR(40) NOT NULL,
    actorUserID            BIGINT NOT NULL,
    actorOrganizationID    BIGINT NULL,
    note                   TEXT NULL,
    createdDate            DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT fk_submission_history_submission FOREIGN KEY (submissionID) REFERENCES tr_submission(submissionID) ON DELETE CASCADE,
    CONSTRAINT fk_submission_history_actor FOREIGN KEY (actorUserID) REFERENCES ms_user(userID),
    CONSTRAINT fk_submission_history_actor_org FOREIGN KEY (actorOrganizationID) REFERENCES ms_organization(organizationID),
    INDEX idx_submission_history_submission (submissionID, createdDate)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- Baris dibuat oleh service layer saat submission Sensus Kader mencapai
-- status SUBMITTED (bukan menunggu persetujuan LDK) sehingga kader dapat
-- memantau status pendaftarannya sejak awal.
CREATE TABLE IF NOT EXISTS ms_kader (
    kaderID        BIGINT AUTO_INCREMENT PRIMARY KEY,
    submissionID   BIGINT NOT NULL UNIQUE,
    organizationID BIGINT NOT NULL,
    userID         BIGINT NOT NULL,
    uniqueCode     VARCHAR(50) NULL UNIQUE,
    fullName       VARCHAR(150) NOT NULL,
    status         ENUM('PENDING','ACTIVE','REJECTED','INACTIVE') NOT NULL DEFAULT 'PENDING',
    issuedDate     DATETIME NULL,
    createdDate    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    createdBy      BIGINT NULL,
    updatedDate    DATETIME NULL,
    updatedBy      BIGINT NULL,
    CONSTRAINT fk_kader_submission FOREIGN KEY (submissionID) REFERENCES tr_submission(submissionID),
    CONSTRAINT fk_kader_organization FOREIGN KEY (organizationID) REFERENCES ms_organization(organizationID),
    CONSTRAINT fk_kader_user FOREIGN KEY (userID) REFERENCES ms_user(userID),
    INDEX idx_kader_org_status (organizationID, status)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

INSERT IGNORE INTO lk_permission (permissionCode, permissionName, moduleName, menuLabel, menuIcon, menuRoute, sortOrder) VALUES
('submission.create', 'Isi Pendataan',      'submission', 'Pendataan',       'clipboard-list', '/cms/submissions/pendataan', 7),
('submission.update', 'Ubah Jawaban',       'submission', NULL, NULL, NULL, NULL),
('submission.cancel', 'Batalkan Pendataan', 'submission', NULL, NULL, NULL, NULL),
('submission.view',   'Lihat Status Pendataan', 'submission', 'Status Pendataan', 'list-checks', '/cms/submissions/status', 8);

INSERT IGNORE INTO map_role_permission (roleID, permissionID)
SELECT r.roleID, p.permissionID FROM ms_role r JOIN lk_permission p
WHERE r.roleName = 'Super Admin' AND p.permissionCode IN ('submission.create','submission.update','submission.cancel','submission.view');

INSERT IGNORE INTO map_role_permission (roleID, permissionID)
SELECT r.roleID, p.permissionID FROM ms_role r JOIN lk_permission p
WHERE r.roleName IN ('LDK Admin','Kader') AND p.permissionCode IN ('submission.create','submission.update','submission.cancel','submission.view');

INSERT IGNORE INTO map_role_permission (roleID, permissionID)
SELECT r.roleID, p.permissionID FROM ms_role r JOIN lk_permission p
WHERE r.roleName IN ('Puskomda Verifikator','Puskomnas Verifikator') AND p.permissionCode = 'submission.view';
