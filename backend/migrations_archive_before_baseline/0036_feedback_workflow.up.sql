ALTER TABLE feedbacks
  ADD COLUMN status varchar(32) NOT NULL DEFAULT 'open',
  ADD COLUMN internal_note text,
  ADD COLUMN handled_by char(36),
  ADD COLUMN handled_at datetime,
  ADD COLUMN updated_at datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  ADD KEY idx_feedbacks_status (status),
  ADD KEY idx_feedbacks_updated_at (updated_at);

UPDATE feedbacks SET status = 'open' WHERE status IS NULL OR status = '';
