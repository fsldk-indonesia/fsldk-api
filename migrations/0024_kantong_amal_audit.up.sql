-- ============================================================
-- Kantong Amal — Finance Audit Log (Phase 1: skema + extend
-- pkg/auditlog dengan LogFinance()). tr_finance_audit_log, skema
-- identik tr_form_audit_log/tr_user_audit_log/tr_export_log existing.
-- ============================================================

CREATE TABLE IF NOT EXISTS tr_finance_audit_log (
    logID                BIGINT AUTO_INCREMENT PRIMARY KEY,
    actorUserID           BIGINT NOT NULL,
    actorOrganizationID    BIGINT NULL,
    action                 VARCHAR(50) NOT NULL,
    entity                  VARCHAR(50) NOT NULL,
    entityID                 BIGINT NOT NULL,
    beforeJSON               TEXT NULL,
    afterJSON                TEXT NULL,
    ipAddress                VARCHAR(45) NULL,
    metadata                 TEXT NULL,
    createdDate              DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_finance_audit_entity (entity, entityID)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
