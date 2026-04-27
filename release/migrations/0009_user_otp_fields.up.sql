ALTER TABLE users
  ADD COLUMN otp_secret varchar(128) NULL,
  ADD COLUMN otp_enabled boolean NOT NULL DEFAULT FALSE;
