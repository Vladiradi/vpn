-- Add OAuth/Telegram fields to users
ALTER TABLE users
    ADD COLUMN IF NOT EXISTS google_id VARCHAR(255) UNIQUE,
    ADD COLUMN IF NOT EXISTS telegram_id BIGINT UNIQUE;

-- Extend nodes table for agent registration and XRay REALITY settings
ALTER TABLE nodes
    ADD COLUMN IF NOT EXISTS name VARCHAR(255),
    ADD COLUMN IF NOT EXISTS api_token VARCHAR(255) UNIQUE,
    ADD COLUMN IF NOT EXISTS reality_dest VARCHAR(255),
    ADD COLUMN IF NOT EXISTS reality_private_key VARCHAR(255),
    ADD COLUMN IF NOT EXISTS reality_public_key VARCHAR(255),
    ADD COLUMN IF NOT EXISTS reality_short_ids TEXT[],
    ADD COLUMN IF NOT EXISTS decoy_host VARCHAR(255);

-- Link devices to a specific node
ALTER TABLE devices
    ADD COLUMN IF NOT EXISTS node_id UUID REFERENCES nodes(id) ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS idx_devices_node_id ON devices(node_id);
