-- ============================================================
-- Kantong Amal — Job Queue (Phase 1: skema saja; pkg/queueworker &
-- modul queue dibangun Phase 9). tr_queue_job, tr_queue_job_log.
-- ============================================================

CREATE TABLE IF NOT EXISTS tr_queue_job (
    queueJobID       BIGINT AUTO_INCREMENT PRIMARY KEY,
    jobType           VARCHAR(50) NOT NULL,
    queue             VARCHAR(30) NOT NULL DEFAULT 'default',
    payloadJSON       TEXT NOT NULL,
    payloadHash       CHAR(64) NOT NULL,
    referenceType     VARCHAR(30) NULL,
    referenceID       BIGINT NULL,
    status            ENUM('PENDING','PROCESSING','SUCCESS','FAILED','RETRYING','CANCELLED') NOT NULL DEFAULT 'PENDING',
    attempt           INT NOT NULL DEFAULT 0,
    maxAttempt        INT NOT NULL DEFAULT 5,
    scheduledDate     DATETIME NOT NULL,
    startedDate       DATETIME NULL,
    completedDate     DATETIME NULL,
    failedDate        DATETIME NULL,
    errorCode         VARCHAR(50) NULL,
    errorMessage      TEXT NULL,
    lockedByWorker    VARCHAR(100) NULL,
    lockedUntil       DATETIME NULL,
    createdDate       DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updatedDate       DATETIME NULL,
    UNIQUE KEY uq_queue_payload_hash (payloadHash),
    INDEX idx_queue_pickup (status, scheduledDate),
    INDEX idx_queue_type (jobType)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS tr_queue_job_log (
    queueJobLogID    BIGINT AUTO_INCREMENT PRIMARY KEY,
    queueJobID        BIGINT NOT NULL,
    attempt            INT NOT NULL,
    status             ENUM('SUCCESS','FAILED','RETRYING') NOT NULL,
    durationMs          INT NOT NULL,
    errorMessage        TEXT NULL,
    workerName          VARCHAR(100) NOT NULL,
    createdDate         DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT fk_queue_job_log_job FOREIGN KEY (queueJobID) REFERENCES tr_queue_job(queueJobID),
    INDEX idx_job_log_job (queueJobID, createdDate)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
