-- +goose Up

-- Add new columns to users before data migration
ALTER TABLE users ADD COLUMN IF NOT EXISTS tenant_id VARCHAR(36) NOT NULL DEFAULT '';
ALTER TABLE users ADD COLUMN IF NOT EXISTS email_verified_at TIMESTAMPTZ;

-- user_credentials
CREATE TABLE IF NOT EXISTS  user_credentials (
    user_id       UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    password_hash VARCHAR(255) NOT NULL DEFAULT '',
    last_login_at TIMESTAMPTZ
);

-- user_security
CREATE TABLE IF NOT EXISTS  user_security (
    user_id       UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    login_attempts INTEGER NOT NULL DEFAULT 0,
    locked_until   TIMESTAMPTZ,
    mfa_enabled    BOOLEAN NOT NULL DEFAULT false,
    mfa_secret     VARCHAR(255)
);

-- user_tokens
CREATE TABLE IF NOT EXISTS  user_tokens (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id      UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_type   VARCHAR(50) NOT NULL,
    token_hash   VARCHAR(255),
    expires_at   TIMESTAMPTZ,
    consumed_at  TIMESTAMPTZ,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_user_tokens_user_id ON user_tokens(user_id);
CREATE INDEX IF NOT EXISTS idx_user_tokens_token_hash ON user_tokens(token_hash);

-- user_profiles
CREATE TABLE IF NOT EXISTS  user_profiles (
    user_id    UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    first_name VARCHAR(255) NOT NULL DEFAULT '',
    last_name  VARCHAR(255) NOT NULL DEFAULT '',
    phone      VARCHAR(50),
    avatar_url TEXT,
    timezone   VARCHAR(50) NOT NULL DEFAULT 'UTC',
    locale     VARCHAR(10) NOT NULL DEFAULT 'en',
    bio        TEXT
);

-- user_addresses
CREATE TABLE IF NOT EXISTS  user_addresses (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    label       VARCHAR(100) NOT NULL DEFAULT '',
    is_default  BOOLEAN NOT NULL DEFAULT false,
    street      VARCHAR(255) NOT NULL DEFAULT '',
    city        VARCHAR(100) NOT NULL DEFAULT '',
    state       VARCHAR(100) NOT NULL DEFAULT '',
    postal_code VARCHAR(20) NOT NULL DEFAULT '',
    country     VARCHAR(100) NOT NULL DEFAULT ''
);

CREATE INDEX IF NOT EXISTS idx_user_addresses_user_id ON user_addresses(user_id);

-- user_sessions
CREATE TABLE IF NOT EXISTS  user_sessions (
    id                 UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id            UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    refresh_token_hash VARCHAR(255),
    device_info        VARCHAR(500) NOT NULL DEFAULT '',
    ip_address         VARCHAR(45) NOT NULL DEFAULT '',
    user_agent         TEXT NOT NULL DEFAULT '',
    expires_at         TIMESTAMPTZ,
    last_used_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    revoked_at         TIMESTAMPTZ,
    created_at         TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_user_sessions_user_id ON user_sessions(user_id);

-- user_social_links
CREATE TABLE IF NOT EXISTS  user_social_links (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id        UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    provider       VARCHAR(50) NOT NULL,
    provider_id    VARCHAR(255) NOT NULL,
    provider_email VARCHAR(255),
    avatar_url     TEXT,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(provider, provider_id)
);

CREATE INDEX IF NOT EXISTS idx_user_social_links_user_id ON user_social_links(user_id);

-- user_preferences
CREATE TABLE IF NOT EXISTS  user_preferences (
    user_id     UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    preferences JSONB NOT NULL DEFAULT '{}',
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Drop old columns from users
ALTER TABLE users DROP COLUMN IF EXISTS password_hash;
ALTER TABLE users DROP COLUMN IF EXISTS login_attempts;
ALTER TABLE users DROP COLUMN IF EXISTS locked_until;
ALTER TABLE users DROP COLUMN IF EXISTS verification_token;
ALTER TABLE users DROP COLUMN IF EXISTS verification_token_expires_at;
ALTER TABLE users DROP COLUMN IF EXISTS reset_token;
ALTER TABLE users DROP COLUMN IF EXISTS reset_token_expires_at;
ALTER TABLE users DROP COLUMN IF EXISTS last_login_at;

-- +goose Down

-- Add back old columns
ALTER TABLE users ADD COLUMN IF NOT EXISTS password_hash VARCHAR(255);
ALTER TABLE users ADD COLUMN IF NOT EXISTS login_attempts INTEGER NOT NULL DEFAULT 0;
ALTER TABLE users ADD COLUMN IF NOT EXISTS locked_until TIMESTAMPTZ;
ALTER TABLE users ADD COLUMN IF NOT EXISTS verification_token VARCHAR(255);
ALTER TABLE users ADD COLUMN IF NOT EXISTS verification_token_expires_at TIMESTAMPTZ;
ALTER TABLE users ADD COLUMN IF NOT EXISTS reset_token VARCHAR(255);
ALTER TABLE users ADD COLUMN IF NOT EXISTS reset_token_expires_at TIMESTAMPTZ;
ALTER TABLE users ADD COLUMN IF NOT EXISTS last_login_at TIMESTAMPTZ;

-- Restore data from new tables
UPDATE users SET
    password_hash = uc.password_hash,
    last_login_at = uc.last_login_at
FROM user_credentials uc
WHERE users.id = uc.user_id;

UPDATE users SET
    login_attempts = us.login_attempts,
    locked_until = us.locked_until
FROM user_security us
WHERE users.id = us.user_id;

-- Drop new tables
DROP TABLE IF EXISTS user_social_links;
DROP TABLE IF EXISTS user_sessions;
DROP TABLE IF EXISTS user_addresses;
DROP TABLE IF EXISTS user_preferences;
DROP TABLE IF EXISTS user_profiles;
DROP TABLE IF EXISTS user_tokens;
DROP TABLE IF EXISTS user_security;
DROP TABLE IF EXISTS user_credentials;

-- Drop columns added by this migration
ALTER TABLE users DROP COLUMN IF EXISTS email_verified_at;
ALTER TABLE users DROP COLUMN IF EXISTS tenant_id;
