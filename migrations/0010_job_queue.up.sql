-- ============================================================
-- FSLDK API — Job Queue (modul platform generik, §1b techspec)
-- Tabel tr_job_queue (antrian job async) + tr_whatsapp_message_log
-- (dipakai resolusi balasan WhatsApp jalur approval kedua, §1a.5).
-- Idempoten: aman dijalankan ulang (CREATE TABLE IF NOT EXISTS / INSERT IGNORE).
-- ============================================================

CREATE TABLE IF NOT EXISTS tr_job_queue (
    jobID           BIGINT AUTO_INCREMENT PRIMARY KEY,
    queue           VARCHAR(50)  NOT NULL,          -- 'whatsapp' | 'email'
    jobType         VARCHAR(100) NOT NULL,          -- 'whatsapp_template' | 'email_shortlink_approved' | 'email_shortlink_rejected'
    payload         TEXT NOT NULL,                  -- JSON, marshal/unmarshal di level aplikasi
    status          VARCHAR(20)  NOT NULL DEFAULT 'pending', -- pending | processing | completed | failed
    attempts        INT NOT NULL DEFAULT 0,
    maxAttempts     INT NOT NULL DEFAULT 5,
    availableDate   DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    reservedDate    DATETIME NULL,
    lastError       TEXT NULL,
    correlationType VARCHAR(50) NULL,
    correlationID   BIGINT NULL,
    createdDate     DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    completedDate   DATETIME NULL,
    failedDate      DATETIME NULL,

    INDEX idx_job_queue_dequeue (status, queue, availableDate),
    INDEX idx_job_queue_correlation (correlationType, correlationID)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS tr_whatsapp_message_log (
    messageLogID    BIGINT AUTO_INCREMENT PRIMARY KEY,
    jobID           BIGINT NULL,
    waMessageID     VARCHAR(150) NOT NULL,
    toPhone         VARCHAR(50) NOT NULL,
    templateName    VARCHAR(100) NOT NULL,
    correlationType VARCHAR(50) NULL,
    correlationID   BIGINT NULL,
    createdDate     DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,

    UNIQUE KEY uq_wa_message_id (waMessageID),
    INDEX idx_wa_message_correlation (correlationType, correlationID),
    INDEX idx_wa_message_phone (toPhone, correlationType),
    CONSTRAINT fk_wa_message_job FOREIGN KEY (jobID) REFERENCES tr_job_queue(jobID)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- jobqueue.* — Super Admin only (internal operasional, bukan konten editorial)
INSERT IGNORE INTO lk_permission (permissionCode, permissionName, moduleName, menuLabel, menuIcon, menuRoute, sortOrder)
VALUES ('jobqueue.view',   'Lihat Job Queue',  'jobqueue', 'Job Queue', 'list-checks', '/cms/job-queue', 98),
       ('jobqueue.retry',  'Retry Job',        'jobqueue', NULL, NULL, NULL, NULL),
       ('jobqueue.delete', 'Hapus Job',        'jobqueue', NULL, NULL, NULL, NULL);

INSERT IGNORE INTO map_role_permission (roleID, permissionID)
SELECT r.roleID, p.permissionID FROM ms_role r, lk_permission p
WHERE r.roleName = 'Super Admin' AND p.permissionCode LIKE 'jobqueue.%';
