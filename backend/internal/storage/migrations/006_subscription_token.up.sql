-- Public subscription token for /sub/{token}
ALTER TABLE subscriptions
    ADD COLUMN IF NOT EXISTS subscription_token VARCHAR(64) UNIQUE;

CREATE UNIQUE INDEX IF NOT EXISTS idx_subscriptions_token ON subscriptions(subscription_token);
