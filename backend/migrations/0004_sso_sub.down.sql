DROP INDEX uniq_users_sso_sub ON users;
ALTER TABLE users DROP COLUMN sso_sub;
