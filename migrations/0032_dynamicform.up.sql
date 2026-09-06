-- Dynamic Form — public Google-Forms-style form builder (dynamicform module).
--
-- This is NOT the submission_form engine (Levelisasi/Sensus Kader, review
-- workflow). It is a standalone public form builder shared via /{slug}, fillable
-- anonymously, with per-field analytics and CSV export. Deliberately distinct
-- naming: ms_dynamic_form* vs ms_submission_form*, dynamicform.* vs
-- submission_form.* permissions, "Formulir Dinamis" menu.
--
-- DB is the source of truth (tr_dynamic_form_answer); Google Sheets is an
-- opt-in best-effort mirror (columns gsheet* below + tr_job_queue jobs).

CREATE TABLE IF NOT EXISTS ms_dynamic_form (
    formID                 BIGINT AUTO_INCREMENT PRIMARY KEY,
    title                  VARCHAR(255) NOT NULL,
    slug                   VARCHAR(255) NOT NULL UNIQUE,
    description            TEXT NULL,
    status                 ENUM('draft','published','closed','archived') NOT NULL DEFAULT 'draft',
    version                INT UNSIGNED NOT NULL DEFAULT 1,

    -- Submission rules
    maxSubmission          INT UNSIGNED NULL,
    isMultipleSubmit       TINYINT(1) NOT NULL DEFAULT 0,
    requireLogin           TINYINT(1) NOT NULL DEFAULT 0,

    -- Active period (local WIB, no tz conversion)
    startDate              DATETIME NULL,
    endDate                DATETIME NULL,

    -- Post-submit UX
    confirmationMessage    TEXT NULL,
    redirectUrl            VARCHAR(500) NULL,

    -- Notification & anti-spam (settings are plain columns, not a KV table)
    notifyEmailsJSON       TEXT NULL,
    sendConfirmationEmail  TINYINT(1) NOT NULL DEFAULT 1,
    rateLimitPerIP         INT UNSIGNED NOT NULL DEFAULT 5,
    rateLimitWindowMinutes INT UNSIGNED NOT NULL DEFAULT 10,

    -- Google Sheets mirror — all NULL while gsheetEnabled = 0
    gsheetEnabled          TINYINT(1) NOT NULL DEFAULT 0,
    gsheetSpreadsheetID    VARCHAR(255) NULL,
    gsheetSpreadsheetURL   VARCHAR(500) NULL,
    gsheetTabName          VARCHAR(100) NOT NULL DEFAULT 'Responses',
    gsheetLastSyncDate     DATETIME NULL,
    gsheetLastSyncError    VARCHAR(500) NULL,

    -- Fast denormalized counter, always bumped inside the submit transaction
    totalSubmission        INT UNSIGNED NOT NULL DEFAULT 0,

    isActive               TINYINT(1) NOT NULL DEFAULT 1,
    createdDate            DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    createdBy              BIGINT NULL,
    updatedDate            DATETIME NULL,
    updatedBy             BIGINT NULL,

    CONSTRAINT fk_dynamic_form_creator FOREIGN KEY (createdBy) REFERENCES ms_user(userID),
    CONSTRAINT fk_dynamic_form_updater FOREIGN KEY (updatedBy) REFERENCES ms_user(userID),
    INDEX idx_dynamic_form_status  (status),
    INDEX idx_dynamic_form_active  (isActive),
    INDEX idx_dynamic_form_creator (createdBy),
    INDEX idx_dynamic_form_title   (title)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS ms_dynamic_form_section (
    sectionID    BIGINT AUTO_INCREMENT PRIMARY KEY,
    formID       BIGINT NOT NULL,
    title        VARCHAR(255) NOT NULL,
    description  TEXT NULL,
    sortOrder    INT UNSIGNED NOT NULL DEFAULT 0,
    isActive     TINYINT(1) NOT NULL DEFAULT 1,
    createdDate  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updatedDate  DATETIME NULL,
    CONSTRAINT fk_dynamic_form_section_form FOREIGN KEY (formID) REFERENCES ms_dynamic_form(formID) ON DELETE CASCADE,
    INDEX idx_dynamic_form_section_form (formID, sortOrder)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS ms_dynamic_form_field (
    fieldID              BIGINT AUTO_INCREMENT PRIMARY KEY,
    formID               BIGINT NOT NULL,
    sectionID            BIGINT NULL,
    fieldType            VARCHAR(30) NOT NULL,
    label                VARCHAR(500) NOT NULL,
    placeholder          VARCHAR(255) NULL,
    helpText             VARCHAR(2000) NULL,
    isRequired           TINYINT(1) NOT NULL DEFAULT 0,
    isSystemField        TINYINT(1) NOT NULL DEFAULT 0,
    sortOrder            INT UNSIGNED NOT NULL DEFAULT 0,
    optionsJSON          TEXT NULL,
    validationJSON       TEXT NULL,
    defaultValue         TEXT NULL,
    conditionalLogicJSON TEXT NULL,
    fieldConfigJSON      TEXT NULL,
    isActive             TINYINT(1) NOT NULL DEFAULT 1,
    createdDate          DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updatedDate          DATETIME NULL,
    CONSTRAINT fk_dynamic_form_field_form    FOREIGN KEY (formID)    REFERENCES ms_dynamic_form(formID) ON DELETE CASCADE,
    CONSTRAINT fk_dynamic_form_field_section FOREIGN KEY (sectionID) REFERENCES ms_dynamic_form_section(sectionID) ON DELETE SET NULL,
    INDEX idx_dynamic_form_field_form (formID, sortOrder),
    INDEX idx_dynamic_form_field_type (fieldType)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS tr_dynamic_form_submission (
    submissionID     BIGINT AUTO_INCREMENT PRIMARY KEY,
    formID           BIGINT NOT NULL,
    respondentEmail  VARCHAR(255) NOT NULL,
    respondentName   VARCHAR(255) NULL,
    respondentUserID BIGINT NULL,
    ipAddress        VARCHAR(45) NULL,
    userAgent        TEXT NULL,
    isValid          TINYINT(1) NOT NULL DEFAULT 1,
    formVersion      INT UNSIGNED NOT NULL DEFAULT 1,
    gsheetRowIndex   INT UNSIGNED NULL,
    submittedDate    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT fk_dynamic_form_submission_form FOREIGN KEY (formID) REFERENCES ms_dynamic_form(formID) ON DELETE CASCADE,
    CONSTRAINT fk_dynamic_form_submission_user FOREIGN KEY (respondentUserID) REFERENCES ms_user(userID),
    INDEX idx_dynamic_form_submission_form  (formID, submittedDate),
    INDEX idx_dynamic_form_submission_email (formID, respondentEmail),
    INDEX idx_dynamic_form_submission_ip    (formID, ipAddress, submittedDate)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS tr_dynamic_form_answer (
    answerID       BIGINT AUTO_INCREMENT PRIMARY KEY,
    submissionID   BIGINT NOT NULL,
    fieldID        BIGINT NOT NULL,
    answerValue    TEXT NULL,
    CONSTRAINT fk_dynamic_form_answer_submission FOREIGN KEY (submissionID) REFERENCES tr_dynamic_form_submission(submissionID) ON DELETE CASCADE,
    CONSTRAINT fk_dynamic_form_answer_field      FOREIGN KEY (fieldID)      REFERENCES ms_dynamic_form_field(fieldID),
    INDEX idx_dynamic_form_answer_submission (submissionID),
    INDEX idx_dynamic_form_answer_field      (fieldID)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS tr_dynamic_form_file (
    fileID           BIGINT AUTO_INCREMENT PRIMARY KEY,
    submissionID     BIGINT NOT NULL,
    fieldID          BIGINT NOT NULL,
    fileURL          VARCHAR(500) NOT NULL,
    originalFileName VARCHAR(255) NOT NULL,
    mimeType         VARCHAR(100) NULL,
    fileSizeKB       INT UNSIGNED NULL,
    createdDate      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT fk_dynamic_form_file_submission FOREIGN KEY (submissionID) REFERENCES tr_dynamic_form_submission(submissionID) ON DELETE CASCADE,
    CONSTRAINT fk_dynamic_form_file_field      FOREIGN KEY (fieldID)      REFERENCES ms_dynamic_form_field(fieldID),
    INDEX idx_dynamic_form_file_submission (submissionID)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS tr_dynamic_form_draft (
    draftID      BIGINT AUTO_INCREMENT PRIMARY KEY,
    formID       BIGINT NOT NULL,
    userID       BIGINT NOT NULL,
    answersJSON  TEXT NOT NULL,
    createdDate  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updatedDate  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE KEY uq_dynamic_form_draft (formID, userID),
    CONSTRAINT fk_dynamic_form_draft_form FOREIGN KEY (formID) REFERENCES ms_dynamic_form(formID) ON DELETE CASCADE,
    CONSTRAINT fk_dynamic_form_draft_user FOREIGN KEY (userID) REFERENCES ms_user(userID) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS map_dynamic_form_collaborator (
    collaboratorID BIGINT AUTO_INCREMENT PRIMARY KEY,
    formID         BIGINT NOT NULL,
    userID         BIGINT NOT NULL,
    role           ENUM('editor','manager') NOT NULL DEFAULT 'editor',
    addedDate      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE KEY uq_dynamic_form_collaborator (formID, userID),
    CONSTRAINT fk_dynamic_form_collaborator_form FOREIGN KEY (formID) REFERENCES ms_dynamic_form(formID) ON DELETE CASCADE,
    CONSTRAINT fk_dynamic_form_collaborator_user FOREIGN KEY (userID) REFERENCES ms_user(userID) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- Permission seed (menu row on dynamicform.view, same pattern as 0018_catalogbook).
INSERT IGNORE INTO lk_permission (permissionCode, permissionName, moduleName, menuLabel, menuIcon, menuRoute, sortOrder) VALUES
('dynamicform.view',       'Lihat Formulir Dinamis',        'dynamicform', 'Formulir Dinamis', 'clipboard-list', '/cms/dynamic-forms', 8),
('dynamicform.create',     'Buat Formulir Dinamis',         'dynamicform', NULL, NULL, NULL, NULL),
('dynamicform.update',     'Ubah Formulir Dinamis',         'dynamicform', NULL, NULL, NULL, NULL),
('dynamicform.delete',     'Hapus Formulir Dinamis',        'dynamicform', NULL, NULL, NULL, NULL),
('dynamicform.publish',    'Publish/Tutup Formulir',        'dynamicform', NULL, NULL, NULL, NULL),
('dynamicform.manage.all', 'Kelola Semua Formulir Dinamis', 'dynamicform', NULL, NULL, NULL, NULL);

-- Super Admin: explicit grant (the 0002_seed wildcard only covered permissions
-- that existed then — same note as 0018_catalogbook).
INSERT IGNORE INTO map_role_permission (roleID, permissionID)
SELECT r.roleID, p.permissionID FROM ms_role r JOIN lk_permission p
WHERE r.roleName = 'Super Admin' AND p.permissionCode LIKE 'dynamicform.%';

-- Editor: full parity including manage.all (can manage any form).
INSERT IGNORE INTO map_role_permission (roleID, permissionID)
SELECT r.roleID, p.permissionID FROM ms_role r, lk_permission p
WHERE r.roleName = 'Editor' AND p.permissionCode IN
  ('dynamicform.view','dynamicform.create','dynamicform.update','dynamicform.delete','dynamicform.publish','dynamicform.manage.all');

-- Kontributor: view/create/update/publish but own forms only (ownership guard) — no delete, no manage.all.
INSERT IGNORE INTO map_role_permission (roleID, permissionID)
SELECT r.roleID, p.permissionID FROM ms_role r, lk_permission p
WHERE r.roleName = 'Kontributor' AND p.permissionCode IN
  ('dynamicform.view','dynamicform.create','dynamicform.update','dynamicform.publish');
