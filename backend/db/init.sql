-- Supplementary Insurance Module: schema
-- Applied once on API boot (see internal/platform/database).

CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE insurance_contracts (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name          VARCHAR(150) NOT NULL,
    start_date    DATE NOT NULL,
    end_date      DATE NOT NULL,
    is_active     BOOLEAN NOT NULL DEFAULT true,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    CHECK (end_date > start_date)
);

CREATE TABLE coverage_plans (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    contract_id   UUID NOT NULL REFERENCES insurance_contracts(id) ON DELETE CASCADE,
    name          VARCHAR(100) NOT NULL,
    description   TEXT,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE employees (
    id                 UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    personnel_no       VARCHAR(30) UNIQUE NOT NULL,
    full_name          VARCHAR(200) NOT NULL,
    national_id        VARCHAR(20) UNIQUE,
    employment_status  VARCHAR(20) NOT NULL DEFAULT 'active' CHECK (employment_status IN ('active','terminated')),
    hire_date          DATE NOT NULL,
    department         VARCHAR(100),
    plan_id            UUID REFERENCES coverage_plans(id),
    created_at         TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at         TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE users (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    username       VARCHAR(64) UNIQUE NOT NULL,
    password_hash  VARCHAR(255) NOT NULL,
    full_name      VARCHAR(200) NOT NULL,
    role           VARCHAR(20) NOT NULL CHECK (role IN ('admin','reviewer','employee','auditor')),
    employee_id    UUID REFERENCES employees(id),
    is_active      BOOLEAN NOT NULL DEFAULT true,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE service_types (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    code       VARCHAR(30) UNIQUE NOT NULL,
    name       VARCHAR(100) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Versioned, config-driven coverage rules. A new row = a new version.
CREATE TABLE coverage_rules (
    id                   UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    plan_id              UUID NOT NULL REFERENCES coverage_plans(id) ON DELETE CASCADE,
    service_type_id      UUID NOT NULL REFERENCES service_types(id) ON DELETE CASCADE,
    coverage_percent     NUMERIC(5,2) NOT NULL CHECK (coverage_percent >= 0 AND coverage_percent <= 100),
    per_claim_cap        NUMERIC(14,0) CHECK (per_claim_cap IS NULL OR per_claim_cap >= 0),
    annual_cap           NUMERIC(14,0) CHECK (annual_cap IS NULL OR annual_cap >= 0),
    waiting_period_days  INTEGER NOT NULL DEFAULT 0,
    eligible_relations   TEXT[] NOT NULL DEFAULT '{self}',
    effective_from       DATE NOT NULL,
    effective_to         DATE,
    created_by           UUID REFERENCES users(id),
    created_at           TIMESTAMPTZ NOT NULL DEFAULT now(),
    CHECK (effective_to IS NULL OR effective_to >= effective_from)
);
CREATE INDEX idx_coverage_rules_lookup ON coverage_rules (plan_id, service_type_id, effective_from);

CREATE TABLE dependents (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    employee_id  UUID NOT NULL REFERENCES employees(id) ON DELETE CASCADE,
    full_name    VARCHAR(200) NOT NULL,
    relation     VARCHAR(20) NOT NULL CHECK (relation IN ('spouse','child','parent')),
    national_id  VARCHAR(20),
    birth_date   DATE,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE claims (
    id                        UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    employee_id               UUID NOT NULL REFERENCES employees(id),
    beneficiary_type          VARCHAR(10) NOT NULL CHECK (beneficiary_type IN ('self','dependent')),
    dependent_id              UUID REFERENCES dependents(id),
    service_type_id           UUID NOT NULL REFERENCES service_types(id),
    plan_id                   UUID NOT NULL REFERENCES coverage_plans(id),
    requested_amount          NUMERIC(14,0) NOT NULL CHECK (requested_amount > 0),
    receipt_date              DATE NOT NULL,
    description               TEXT,
    status                    VARCHAR(30) NOT NULL DEFAULT 'draft' CHECK (status IN (
                                  'draft','submitted','under_review','returned_for_docs',
                                  'approved','rejected','payment_calculated','paid','closed')),
    coverage_percent_applied  NUMERIC(5,2),
    payable_amount            NUMERIC(14,0),
    rejection_reason          TEXT,
    submitted_at              TIMESTAMPTZ,
    reviewed_by               UUID REFERENCES users(id),
    reviewed_at               TIMESTAMPTZ,
    paid_at                   TIMESTAMPTZ,
    closed_at                 TIMESTAMPTZ,
    created_by                UUID NOT NULL REFERENCES users(id),
    created_at                TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at                TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_claims_employee ON claims (employee_id);
CREATE INDEX idx_claims_status ON claims (status);
CREATE INDEX idx_claims_service_type ON claims (service_type_id);
CREATE INDEX idx_claims_created_by ON claims (created_by);
CREATE INDEX idx_claims_annual_cap_lookup
    ON claims (employee_id, service_type_id, plan_id, status, receipt_date);

COMMENT ON COLUMN claims.requested_amount IS 'Whole rial (ADR-0003)';
COMMENT ON COLUMN claims.payable_amount   IS 'Whole rial, computed by the coverage engine (ADR-0003)';
COMMENT ON COLUMN coverage_rules.per_claim_cap IS 'Whole rial; NULL = no per-claim cap';
COMMENT ON COLUMN coverage_rules.annual_cap    IS 'Whole rial; NULL = no annual cap';

CREATE TABLE claim_attachments (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    claim_id     UUID NOT NULL REFERENCES claims(id) ON DELETE CASCADE,
    file_name    VARCHAR(255) NOT NULL,
    file_path    VARCHAR(500) NOT NULL,
    uploaded_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE payments (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    claim_id            UUID NOT NULL UNIQUE REFERENCES claims(id),
    amount              NUMERIC(14,0) NOT NULL,
    payment_reference   VARCHAR(60) NOT NULL,
    status              VARCHAR(20) NOT NULL DEFAULT 'simulated' CHECK (status IN ('simulated','completed')),
    paid_at             TIMESTAMPTZ NOT NULL DEFAULT now()
);
COMMENT ON COLUMN payments.amount IS 'Whole rial (ADR-0003)';

CREATE TABLE audit_logs (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    entity_type     VARCHAR(50) NOT NULL,
    entity_id       VARCHAR(100) NOT NULL,
    action          VARCHAR(50) NOT NULL,
    actor_user_id   UUID REFERENCES users(id),
    actor_username  VARCHAR(64),
    before_data     JSONB,
    after_data      JSONB,
    metadata        JSONB,
    occurred_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_audit_entity ON audit_logs (entity_type, entity_id);
CREATE INDEX idx_audit_occurred_at ON audit_logs (occurred_at);
CREATE INDEX idx_audit_actor ON audit_logs (actor_user_id);

CREATE TABLE integration_api_keys (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name          VARCHAR(100) NOT NULL,
    api_key_hash  VARCHAR(255) NOT NULL,
    is_active     BOOLEAN NOT NULL DEFAULT true,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);
