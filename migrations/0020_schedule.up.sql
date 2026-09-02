-- Jadwal (Schedule) — schedule module.
--
-- Public monthly calendar of LDK activities (/jadwal) + CMS CRUD. Input is a
-- structured form (title, category, dates, times, location, ...), NOT a poster
-- image, so there is no imageURL column and no dependency on POST /uploads/image.
--
-- Dates and times are stored as separate DATE / TIME columns (plus isAllDay) so
-- the app can treat them as local wall-clock time with no timezone conversion.
-- A single-day activity always has endDate IS NULL (the service normalises
-- endDate == startDate to NULL).

CREATE TABLE IF NOT EXISTS ms_schedule (
    scheduleID    BIGINT AUTO_INCREMENT PRIMARY KEY,
    title         VARCHAR(150)  NOT NULL,
    description   TEXT          NULL,
    category      VARCHAR(30)   NOT NULL DEFAULT 'lainnya',   -- slug; validated against constants.ScheduleCategories
    startDate     DATE          NOT NULL,
    endDate       DATE          NULL,                          -- NULL => single-day activity
    isAllDay      TINYINT(1)    NOT NULL DEFAULT 0,
    startTime     TIME          NULL,                          -- required when isAllDay = 0
    endTime       TIME          NULL,
    location      VARCHAR(200)  NULL,
    organizer     VARCHAR(150)  NULL,
    contactPerson VARCHAR(100)  NULL,
    url           VARCHAR(300)  NULL,
    isActive      TINYINT(1)    NOT NULL DEFAULT 1,
    createdDate   DATETIME      NOT NULL DEFAULT CURRENT_TIMESTAMP,
    createdBy     BIGINT        NULL,
    updatedDate   DATETIME      NULL,
    updatedBy     BIGINT        NULL,

    CONSTRAINT fk_schedule_creator FOREIGN KEY (createdBy) REFERENCES ms_user(userID),
    CONSTRAINT fk_schedule_updater FOREIGN KEY (updatedBy) REFERENCES ms_user(userID),

    INDEX idx_schedule_active   (isActive),
    INDEX idx_schedule_range    (startDate, endDate),
    INDEX idx_schedule_category (category),
    INDEX idx_schedule_title    (title)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

INSERT IGNORE INTO lk_permission (permissionCode, permissionName, moduleName, menuLabel, menuIcon, menuRoute, sortOrder) VALUES
('schedule.view',    'Lihat Jadwal',               'schedule', 'Jadwal', 'calendar', '/cms/schedules', 7),
('schedule.create',  'Tambah Jadwal',              'schedule', NULL, NULL, NULL, NULL),
('schedule.update',  'Ubah Jadwal',                'schedule', NULL, NULL, NULL, NULL),
('schedule.delete',  'Hapus Jadwal',               'schedule', NULL, NULL, NULL, NULL),
('schedule.publish', 'Aktifkan/Nonaktifkan Jadwal', 'schedule', NULL, NULL, NULL, NULL);

-- Super Admin only got a blanket grant for permissions that existed at
-- 0002_seed.up.sql time — every module added since then grants its own
-- permissions to Super Admin explicitly (see the identical note in
-- 0018_catalogbook.up.sql), so schedule must do the same.
INSERT IGNORE INTO map_role_permission (roleID, permissionID)
SELECT r.roleID, p.permissionID FROM ms_role r JOIN lk_permission p
WHERE r.roleName = 'Super Admin' AND p.permissionCode LIKE 'schedule.%';

-- Editor: full parity (CRUD + active toggle) — article/news/catalogbook pattern.
INSERT IGNORE INTO map_role_permission (roleID, permissionID)
SELECT r.roleID, p.permissionID FROM ms_role r, lk_permission p
WHERE r.roleName = 'Editor' AND p.permissionCode IN
  ('schedule.view','schedule.create','schedule.update','schedule.delete','schedule.publish');

-- Kontributor: view/create/update only — no delete, no active toggle.
INSERT IGNORE INTO map_role_permission (roleID, permissionID)
SELECT r.roleID, p.permissionID FROM ms_role r, lk_permission p
WHERE r.roleName = 'Kontributor' AND p.permissionCode IN
  ('schedule.view','schedule.create','schedule.update');
