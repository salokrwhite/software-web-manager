-- SSO single sign-on: stable federated subject identifier on users.
-- sso_sub is the OIDC `sub` claim from the external IdP, used as the primary
-- binding key for SSO login (email is only a fallback on first link).
-- A UNIQUE index still allows multiple NULLs in MySQL, so password-only
-- accounts that never used SSO are unaffected.
ALTER TABLE users ADD COLUMN sso_sub VARCHAR(255) NULL;
CREATE UNIQUE INDEX uniq_users_sso_sub ON users (sso_sub);
