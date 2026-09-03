-- ============================================================
-- Kantong Amal — OTP Challenge (Phase 7: security verification
-- withdrawal, step-up default risk-based, lihat 12-security.md §12.1).
-- tr_otp_challenge. codeHash disimpan hash (sha256), bukan plaintext.
-- ============================================================

CREATE TABLE IF NOT EXISTS tr_otp_challenge (
    challengeID     BIGINT AUTO_INCREMENT PRIMARY KEY,
    withdrawalID    BIGINT NOT NULL,
    userID          BIGINT NOT NULL,
    codeHash        VARCHAR(64) NOT NULL,
    channel         VARCHAR(20) NOT NULL DEFAULT 'OTP_WA',
    attemptCount    INT NOT NULL DEFAULT 0,
    expiredDate     DATETIME NOT NULL,
    verifiedDate    DATETIME NULL,
    createdDate     DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT fk_otp_withdrawal FOREIGN KEY (withdrawalID) REFERENCES tr_withdrawal(withdrawalID),
    CONSTRAINT fk_otp_user FOREIGN KEY (userID) REFERENCES ms_user(userID),
    INDEX idx_otp_withdrawal (withdrawalID)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
