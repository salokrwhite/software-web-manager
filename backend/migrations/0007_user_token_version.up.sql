-- Per-user session epoch. Bumped on password change (and other credential
-- invalidations) to revoke all previously issued JWTs: the value is embedded in
-- each token (claim "tv") and compared on every authenticated request.
ALTER TABLE users ADD COLUMN token_version INT NOT NULL DEFAULT 0;
