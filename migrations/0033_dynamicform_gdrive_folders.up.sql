-- Dynamic Form — Google Drive folder tree per form (ported from ldksyahid-app
-- DynamicFormGDriveService). When a form has the Google Sheets mirror enabled,
-- it now also gets its own Drive folder named after the form title, containing:
--   <Form Title>/
--     ├── <Form Title> — Responses   (the spreadsheet)
--     ├── attachments/  └── <file field label>/   (respondent uploads)
--     └── assets/       └── <image field label>/   (builder images)

ALTER TABLE ms_dynamic_form
    ADD COLUMN gdriveFormFolderID        VARCHAR(255) NULL AFTER gsheetTabName,
    ADD COLUMN gdriveAttachmentsFolderID VARCHAR(255) NULL AFTER gdriveFormFolderID,
    ADD COLUMN gdriveAssetsFolderID      VARCHAR(255) NULL AFTER gdriveAttachmentsFolderID;

-- Per file/image field: the Drive subfolder that holds this field's uploads.
ALTER TABLE ms_dynamic_form_field
    ADD COLUMN gdriveFolderID VARCHAR(255) NULL AFTER fieldConfigJSON;

-- Per uploaded file: the Drive file id (used for cleanup); fileURL now holds the
-- Drive webViewLink when the form uses the Drive tree, otherwise the local URL.
ALTER TABLE tr_dynamic_form_file
    ADD COLUMN gdriveFileID VARCHAR(255) NULL AFTER fileURL;
