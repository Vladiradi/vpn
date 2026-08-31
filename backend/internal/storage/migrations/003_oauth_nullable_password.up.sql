-- OAuth/Telegram users may not have a local password.
ALTER TABLE users ALTER COLUMN password_hash DROP NOT NULL;
