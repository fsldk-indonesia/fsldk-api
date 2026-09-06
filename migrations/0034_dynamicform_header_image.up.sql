-- Dynamic Form — optional header image (Google-Forms-style banner shown above
-- the title on the public form and in the CMS builder preview). Stores just
-- the URL; the file itself goes through the existing shared POST
-- /uploads/image endpoint (pkg/upload), same as builder "image" fields.
ALTER TABLE ms_dynamic_form
    ADD COLUMN headerImageUrl VARCHAR(500) NULL AFTER description;
