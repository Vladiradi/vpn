-- Offerings / location+tariff table (pricing is a stub for now)
CREATE TABLE IF NOT EXISTS offerings (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    duration_months INT NOT NULL,
    price_rubles INT NOT NULL,
    speed_mbps INT NOT NULL DEFAULT 100,
    country_code VARCHAR(10) NOT NULL,
    country_name VARCHAR(255) NOT NULL,
    region VARCHAR(255),
    is_active BOOLEAN NOT NULL DEFAULT true,
    is_available BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Subscriptions linked to offerings
DROP TABLE IF EXISTS subscriptions;

CREATE TABLE IF NOT EXISTS subscriptions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    offering_id UUID NOT NULL REFERENCES offerings(id) ON DELETE RESTRICT,
    status VARCHAR(20) NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'active', 'expired', 'cancelled')),
    started_at TIMESTAMPTZ,
    expires_at TIMESTAMPTZ,
    total_traffic_bytes BIGINT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Link devices to subscriptions
ALTER TABLE devices
    ADD COLUMN IF NOT EXISTS subscription_id UUID REFERENCES subscriptions(id) ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS idx_offerings_active ON offerings(is_active);
CREATE INDEX IF NOT EXISTS idx_offerings_available ON offerings(is_available);
CREATE INDEX IF NOT EXISTS idx_offerings_country ON offerings(country_code);
CREATE INDEX IF NOT EXISTS idx_subscriptions_user_id ON subscriptions(user_id);
CREATE INDEX IF NOT EXISTS idx_subscriptions_status ON subscriptions(status);
CREATE INDEX IF NOT EXISTS idx_devices_subscription_id ON devices(subscription_id);

-- Seed default offerings
INSERT INTO offerings (name, duration_months, price_rubles, speed_mbps, country_code, country_name, region, is_available)
VALUES
    ('Германия / Франкфурт / 100 Мбит/с', 6, 500, 100, 'DE', 'Германия', 'Франкфурт', true),
    ('Германия / Франкфурт / 100 Мбит/с', 12, 1000, 100, 'DE', 'Германия', 'Франкфурт', true),
    ('Финляндия / Хельсинки / 100 Мбит/с', 6, 500, 100, 'FI', 'Финляндия', 'Хельсинки', false),
    ('Финляндия / Хельсинки / 100 Мбит/с', 12, 1000, 100, 'FI', 'Финляндия', 'Хельсинки', false)
ON CONFLICT DO NOTHING;
