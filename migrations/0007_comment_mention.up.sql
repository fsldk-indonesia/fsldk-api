-- ============================================================
-- FSLDK API — Comment Mentions
-- Table tr_comment_mention: which users were @mentioned in a comment.
-- Structured (not parsed from commentText) so mention pills render reliably
-- regardless of what the composer typed — see comment_service.Create/Update.
-- Idempotent: safe to re-run (CREATE TABLE IF NOT EXISTS).
-- ============================================================

CREATE TABLE IF NOT EXISTS tr_comment_mention (
    mentionID    BIGINT AUTO_INCREMENT PRIMARY KEY,
    commentID    BIGINT NOT NULL,
    userID       BIGINT NOT NULL,
    createdDate  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE KEY uq_comment_mention (commentID, userID),
    CONSTRAINT fk_mention_comment FOREIGN KEY (commentID) REFERENCES ms_comment(commentID) ON DELETE CASCADE,
    CONSTRAINT fk_mention_user    FOREIGN KEY (userID)    REFERENCES ms_user(userID)       ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
