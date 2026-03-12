-- Support email/password auth alongside OAuth.
-- auth_provider = 'email' and auth_subject = email for local accounts.
ALTER TABLE users ADD COLUMN password_hash TEXT;

-- Make auth_provider/auth_subject nullable for email/password users
-- (they use 'email' as provider and their email as subject)
