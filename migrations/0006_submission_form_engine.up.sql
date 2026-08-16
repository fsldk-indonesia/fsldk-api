-- ============================================================
-- FSLDK API — Reusable Submission Form Engine
-- Form template → version (DRAFT/PUBLISHED/ARCHIVED) → section → field → option.
-- Idempoten: aman dijalankan ulang (CREATE TABLE IF NOT EXISTS / INSERT IGNORE).
-- ============================================================

CREATE TABLE IF NOT EXISTS ms_submission_form (
    formID        BIGINT AUTO_INCREMENT PRIMARY KEY,
    formCode      VARCHAR(50) NOT NULL UNIQUE,
    formName      VARCHAR(200) NOT NULL,
    description   VARCHAR(500) NULL,
    isActive      TINYINT(1) NOT NULL DEFAULT 1,
    createdDate   DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    createdBy     BIGINT NULL,
    updatedDate   DATETIME NULL,
    updatedBy     BIGINT NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS ms_submission_form_version (
    versionID     BIGINT AUTO_INCREMENT PRIMARY KEY,
    formID        BIGINT NOT NULL,
    versionNumber INT NOT NULL,
    status        ENUM('DRAFT','PUBLISHED','ARCHIVED') NOT NULL DEFAULT 'DRAFT',
    publishedDate DATETIME NULL,
    publishedBy   BIGINT NULL,
    createdDate   DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    createdBy     BIGINT NULL,
    updatedDate   DATETIME NULL,
    updatedBy     BIGINT NULL,
    CONSTRAINT fk_form_version_form FOREIGN KEY (formID) REFERENCES ms_submission_form(formID),
    UNIQUE KEY uq_form_version_number (formID, versionNumber),
    INDEX idx_form_version_status (formID, status)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS ms_submission_form_section (
    sectionID     BIGINT AUTO_INCREMENT PRIMARY KEY,
    versionID     BIGINT NOT NULL,
    sectionCode   VARCHAR(50) NOT NULL,
    sectionLabel  VARCHAR(150) NOT NULL,
    sortOrder     INT NOT NULL DEFAULT 0,
    description   VARCHAR(500) NULL,
    CONSTRAINT fk_form_section_version FOREIGN KEY (versionID) REFERENCES ms_submission_form_version(versionID) ON DELETE CASCADE,
    UNIQUE KEY uq_form_section_code (versionID, sectionCode),
    INDEX idx_form_section_version (versionID, sortOrder)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS ms_submission_form_field (
    fieldID              BIGINT AUTO_INCREMENT PRIMARY KEY,
    sectionID            BIGINT NOT NULL,
    fieldCode            VARCHAR(50) NOT NULL,
    fieldLabel           VARCHAR(200) NOT NULL,
    fieldType            ENUM('TEXT','TEXTAREA','NUMBER','DATE','SELECT','MULTISELECT','RADIO','CHECKBOX','FILE_DOCUMENT','FILE_IMAGE') NOT NULL,
    isRequired           TINYINT(1) NOT NULL DEFAULT 0,
    sortOrder            INT NOT NULL DEFAULT 0,
    validationRuleJSON   TEXT NULL,
    conditionalOnFieldID BIGINT NULL,
    conditionalRuleJSON  TEXT NULL,
    helpText             VARCHAR(500) NULL,
    CONSTRAINT fk_form_field_section FOREIGN KEY (sectionID) REFERENCES ms_submission_form_section(sectionID) ON DELETE CASCADE,
    CONSTRAINT fk_form_field_conditional FOREIGN KEY (conditionalOnFieldID) REFERENCES ms_submission_form_field(fieldID),
    UNIQUE KEY uq_form_field_code (sectionID, fieldCode),
    INDEX idx_form_field_section (sectionID, sortOrder)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS ms_submission_form_field_option (
    optionID     BIGINT AUTO_INCREMENT PRIMARY KEY,
    fieldID      BIGINT NOT NULL,
    optionValue  VARCHAR(100) NOT NULL,
    optionLabel  VARCHAR(200) NOT NULL,
    sortOrder    INT NOT NULL DEFAULT 0,
    isActive     TINYINT(1) NOT NULL DEFAULT 1,
    CONSTRAINT fk_form_field_option_field FOREIGN KEY (fieldID) REFERENCES ms_submission_form_field(fieldID) ON DELETE CASCADE,
    UNIQUE KEY uq_form_field_option_value (fieldID, optionValue),
    INDEX idx_form_field_option_field (fieldID, sortOrder)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- Periode submission opsional — belum dipakai form manapun pada MVP (baik
-- Levelisasi maupun Sensus Kader bersifat open-ended), disiapkan sebagai
-- fitur opsional untuk form pendataan masa depan.
CREATE TABLE IF NOT EXISTS ms_submission_period (
    periodID      BIGINT AUTO_INCREMENT PRIMARY KEY,
    formID        BIGINT NOT NULL,
    formVersionID BIGINT NOT NULL,
    periodLabel   VARCHAR(100) NOT NULL,
    startDate     DATE NOT NULL,
    endDate       DATE NOT NULL,
    isActive      TINYINT(1) NOT NULL DEFAULT 1,
    CONSTRAINT fk_submission_period_form FOREIGN KEY (formID) REFERENCES ms_submission_form(formID),
    CONSTRAINT fk_submission_period_version FOREIGN KEY (formVersionID) REFERENCES ms_submission_form_version(versionID),
    INDEX idx_submission_period_form (formID, isActive)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

INSERT IGNORE INTO lk_permission (permissionCode, permissionName, moduleName, menuLabel, menuIcon, menuRoute, sortOrder) VALUES
('submission_form.view',   'Lihat Form Builder',   'submission_form', 'Form Pendataan', 'file-sliders', '/cms/submission-forms', 6),
('submission_form.manage', 'Kelola Form Builder',  'submission_form', NULL, NULL, NULL, NULL);

INSERT IGNORE INTO map_role_permission (roleID, permissionID)
SELECT r.roleID, p.permissionID FROM ms_role r JOIN lk_permission p
WHERE r.roleName = 'Super Admin' AND p.permissionCode IN ('submission_form.view','submission_form.manage');
