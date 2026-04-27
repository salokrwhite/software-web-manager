ALTER TABLE feedbacks
  DROP KEY idx_feedbacks_status,
  DROP KEY idx_feedbacks_updated_at,
  DROP COLUMN updated_at,
  DROP COLUMN handled_at,
  DROP COLUMN handled_by,
  DROP COLUMN internal_note,
  DROP COLUMN status;
