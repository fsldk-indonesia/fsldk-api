-- Dynamic Form — Revision R1: remove in-app collaborators.
--
-- The LDK Syahid reference has no in-app collaborator UI; access to a form's CMS
-- pages is owner-or-`dynamicform.manage.all` only. The comma-separated email list
-- (ms_dynamic_form.notifyEmailsJSON) still drives notifications and Google
-- Drive/Sheet sharing. Drop the unused map table.

DROP TABLE IF EXISTS map_dynamic_form_collaborator;
