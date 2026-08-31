ALTER TABLE devices DROP COLUMN IF EXISTS node_id;

ALTER TABLE nodes
    DROP COLUMN IF EXISTS name,
    DROP COLUMN IF EXISTS api_token,
    DROP COLUMN IF EXISTS reality_dest,
    DROP COLUMN IF EXISTS reality_private_key,
    DROP COLUMN IF EXISTS reality_public_key,
    DROP COLUMN IF EXISTS reality_short_ids,
    DROP COLUMN IF EXISTS decoy_host;

ALTER TABLE users
    DROP COLUMN IF EXISTS google_id,
    DROP COLUMN IF EXISTS telegram_id;
