SET @missing_legacy_attachment_rows := (
  SELECT
    (SELECT COUNT(*)
     FROM legacy_ticket_attachments l
     LEFT JOIN attachments a ON a.id = l.id AND a.owner_type = 'ticket' AND a.owner_id = l.ticket_id
     WHERE a.id IS NULL) +
    (SELECT COUNT(*)
     FROM legacy_ticket_message_attachments l
     LEFT JOIN attachments a ON a.id = l.id AND a.owner_type = 'ticket_message' AND a.owner_id = l.message_id
     WHERE a.id IS NULL) +
    (SELECT COUNT(*)
     FROM legacy_feedback_attachments l
     LEFT JOIN attachments a ON a.id = l.id AND a.owner_type = 'feedback' AND a.owner_id = l.feedback_id
     WHERE a.id IS NULL) +
    (SELECT COUNT(*)
     FROM legacy_org_registration_materials l
     LEFT JOIN attachments a ON a.id = l.id AND a.owner_type = 'org_registration_material' AND a.owner_id = l.org_id
     WHERE a.id IS NULL)
);

SET @validation_sql := IF(
  @missing_legacy_attachment_rows = 0,
  'SELECT 1',
  'SIGNAL SQLSTATE ''45000'' SET MESSAGE_TEXT = ''legacy attachment rows are missing from attachments'''
);
PREPARE validation_stmt FROM @validation_sql;
EXECUTE validation_stmt;
DEALLOCATE PREPARE validation_stmt;

DROP TABLE IF EXISTS legacy_ticket_attachments;
DROP TABLE IF EXISTS legacy_ticket_message_attachments;
DROP TABLE IF EXISTS legacy_feedback_attachments;
DROP TABLE IF EXISTS legacy_org_registration_materials;
DROP TABLE IF EXISTS legacy_api_keys;
