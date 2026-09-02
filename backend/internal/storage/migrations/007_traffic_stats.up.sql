-- Traffic statistics reported by node agents.
CREATE TABLE IF NOT EXISTS traffic_stats (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    device_id UUID REFERENCES devices(id) ON DELETE CASCADE,
    node_id UUID REFERENCES nodes(id) ON DELETE CASCADE,
    ts TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    uplink BIGINT NOT NULL DEFAULT 0,
    downlink BIGINT NOT NULL DEFAULT 0
);

CREATE INDEX IF NOT EXISTS idx_traffic_stats_device_id ON traffic_stats(device_id);
CREATE INDEX IF NOT EXISTS idx_traffic_stats_node_id ON traffic_stats(node_id);
CREATE INDEX IF NOT EXISTS idx_traffic_stats_ts ON traffic_stats(ts);
