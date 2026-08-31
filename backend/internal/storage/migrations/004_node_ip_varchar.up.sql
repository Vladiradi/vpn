-- Store node IP as varchar to avoid pgx inet scanning issues.
ALTER TABLE nodes ALTER COLUMN ip TYPE VARCHAR(255);
