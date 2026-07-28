-- Money becomes whole rial (ADR-0003).
--
-- The rial has no fractional unit in everyday use, and NUMERIC(_,2) invited
-- sub-rial amounts that no payment could actually settle. Go now models money
-- as an integer (domain.Rial) with one half-up rounding step in the pricing
-- engine; the schema follows so the two cannot disagree.
--
-- Safety first: if any stored value is NOT already integral, abort with a clear
-- message instead of silently rounding production data. Operators can then
-- decide how to settle those fractions before re-running.
--
-- Recovery after that abort (golang-migrate marks the failed version dirty):
--   1. settle/round the offending rows (the RAISE below prints the query),
--   2. migrate -path migrations -database "$DATABASE_URL" force 2
--   3. migrate -path migrations -database "$DATABASE_URL" up
-- The schema is left untouched by the abort, so step 2 is safe.
DO $$
DECLARE
    offenders bigint;
BEGIN
    SELECT
        (SELECT count(*) FROM claims
          WHERE requested_amount <> trunc(requested_amount)
             OR (payable_amount IS NOT NULL AND payable_amount <> trunc(payable_amount)))
      + (SELECT count(*) FROM coverage_rules
          WHERE (per_claim_cap IS NOT NULL AND per_claim_cap <> trunc(per_claim_cap))
             OR (annual_cap    IS NOT NULL AND annual_cap    <> trunc(annual_cap)))
      + (SELECT count(*) FROM payments
          WHERE amount <> trunc(amount))
    INTO offenders;

    IF offenders > 0 THEN
        RAISE EXCEPTION
            'migration 000003 aborted: % row(s) hold fractional rial amounts. '
            'Settle or round them explicitly, then re-run. '
            'Inspect with: SELECT id, requested_amount, payable_amount FROM claims '
            'WHERE requested_amount <> trunc(requested_amount) OR payable_amount <> trunc(payable_amount);',
            offenders;
    END IF;
END $$;

ALTER TABLE claims
    ALTER COLUMN requested_amount TYPE NUMERIC(14,0),
    ALTER COLUMN payable_amount   TYPE NUMERIC(14,0);

ALTER TABLE coverage_rules
    ALTER COLUMN per_claim_cap TYPE NUMERIC(14,0),
    ALTER COLUMN annual_cap    TYPE NUMERIC(14,0);

ALTER TABLE payments
    ALTER COLUMN amount TYPE NUMERIC(14,0);

COMMENT ON COLUMN claims.requested_amount IS 'Whole rial (ADR-0003)';
COMMENT ON COLUMN claims.payable_amount   IS 'Whole rial, computed by the coverage engine (ADR-0003)';
COMMENT ON COLUMN coverage_rules.per_claim_cap IS 'Whole rial; NULL = no per-claim cap';
COMMENT ON COLUMN coverage_rules.annual_cap    IS 'Whole rial; NULL = no annual cap';
COMMENT ON COLUMN payments.amount IS 'Whole rial (ADR-0003)';

-- Indexes matching the query patterns the service layer actually issues:
-- claims are listed by owner (employees see only their own) and the annual-cap
-- sum filters (employee, service_type, plan, status, receipt_date).
CREATE INDEX IF NOT EXISTS idx_claims_created_by ON claims (created_by);
CREATE INDEX IF NOT EXISTS idx_claims_annual_cap_lookup
    ON claims (employee_id, service_type_id, plan_id, status, receipt_date);
