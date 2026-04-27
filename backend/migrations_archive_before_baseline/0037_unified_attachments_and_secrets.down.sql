RENAME TABLE legacy_api_keys TO api_keys;
RENAME TABLE legacy_org_registration_materials TO org_registration_materials;
RENAME TABLE legacy_feedback_attachments TO feedback_attachments;
RENAME TABLE legacy_ticket_message_attachments TO ticket_message_attachments;
RENAME TABLE legacy_ticket_attachments TO ticket_attachments;

DROP TABLE IF EXISTS attachments;
