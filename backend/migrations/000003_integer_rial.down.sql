-- Revert to two-decimal money columns. Values already stored are integral, so
-- widening the scale is lossless; the application code, however, expects whole
-- rial (ADR-0003) and should be rolled back alongside this migration.
DROP INDEX IF EXISTS idx_claims_annual_cap_lookup;
DROP INDEX IF EXISTS idx_claims_created_by;

ALTER TABLE payments
    ALTER COLUMN amount TYPE NUMERIC(14,2);

ALTER TABLE coverage_rules
    ALTER COLUMN per_claim_cap TYPE NUMERIC(14,2),
    ALTER COLUMN annual_cap    TYPE NUMERIC(14,2);

ALTER TABLE claims
    ALTER COLUMN requested_amount TYPE NUMERIC(14,2),
    ALTER COLUMN payable_amount   TYPE NUMERIC(14,2);
