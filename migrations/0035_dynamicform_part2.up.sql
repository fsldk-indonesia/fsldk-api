-- Dynamic Form — Part 2 corrections (techspec-dynamic-form-part2.md).
--
-- K4: `archived` status is unreachable in the reference — the lifecycle is
--     draft -> published -> closed. Fold any stray archived rows into closed,
--     then narrow the enum.
-- K1: `ms_dynamic_form_section` was never populated — a "section" is the run of
--     fields between two `section_break` fields, resolved at render time. Drop
--     the table and the field.sectionID column.
-- K2: conditional show/hide-a-field logic does not exist in the reference; the
--     real branching mechanism is section routing, stored inside fieldConfigJSON.
--     Drop the dead conditionalLogicJSON column.

UPDATE ms_dynamic_form SET status = 'closed' WHERE status = 'archived';

ALTER TABLE ms_dynamic_form
    MODIFY status ENUM('draft', 'published', 'closed') NOT NULL DEFAULT 'draft';

ALTER TABLE ms_dynamic_form_field
    DROP FOREIGN KEY fk_dynamic_form_field_section,
    DROP COLUMN sectionID,
    DROP COLUMN conditionalLogicJSON;

DROP TABLE IF EXISTS ms_dynamic_form_section;
