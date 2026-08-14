-- Reference/config data: contract, plans, service types, and initial coverage-rule
-- versions. Applied manually (make seed-db), not on API boot.
-- Amounts are whole rial (ADR-0003).

INSERT INTO insurance_contracts (name, start_date, end_date, is_active)
VALUES ('قرارداد سالانه بیمه تکمیلی ۱۴۰۴', '2025-03-21', '2026-03-20', true);

INSERT INTO coverage_plans (contract_id, name, description)
SELECT id, 'استاندارد', 'طرح پوشش بیمه تکمیلی استاندارد'
FROM insurance_contracts WHERE name = 'قرارداد سالانه بیمه تکمیلی ۱۴۰۴';

INSERT INTO coverage_plans (contract_id, name, description)
SELECT id, 'ویژه', 'طرح پوشش بیمه تکمیلی ویژه با سقف‌های بالاتر'
FROM insurance_contracts WHERE name = 'قرارداد سالانه بیمه تکمیلی ۱۴۰۴';

INSERT INTO service_types (code, name, name_fa) VALUES
    ('outpatient_visit', 'Outpatient Visit', 'ویزیت'),
    ('pharmacy',         'Pharmacy / Medicine', 'دارو'),
    ('dental',           'Dental', 'دندان‌پزشکی'),
    ('hospitalization',  'Hospitalization', 'بستری'),
    ('optometry',        'Optometry / Eyewear', 'عینک');

-- Standard plan rules
INSERT INTO coverage_rules (plan_id, service_type_id, coverage_percent, per_claim_cap, annual_cap, waiting_period_days, eligible_relations, effective_from)
SELECT p.id, s.id, v.coverage_percent, v.per_claim_cap, v.annual_cap, v.waiting_period_days, v.eligible_relations, DATE '2025-03-21'
FROM coverage_plans p
JOIN service_types s ON true
JOIN (VALUES
    ('outpatient_visit', 70.00, 500000,  5000000,  0,   ARRAY['self','spouse','child','parent']),
    ('pharmacy',         80.00, 1000000, 10000000, 0,   ARRAY['self','spouse','child','parent']),
    ('dental',           50.00, 3000000, 15000000, 90,  ARRAY['self','spouse','child']),
    ('hospitalization',  90.00, 50000000,100000000,30,  ARRAY['self','spouse','child','parent']),
    ('optometry',        60.00, 2000000, 4000000,  180, ARRAY['self','spouse','child'])
) AS v(code, coverage_percent, per_claim_cap, annual_cap, waiting_period_days, eligible_relations)
    ON v.code = s.code
WHERE p.name = 'استاندارد';

-- Premium plan rules
INSERT INTO coverage_rules (plan_id, service_type_id, coverage_percent, per_claim_cap, annual_cap, waiting_period_days, eligible_relations, effective_from)
SELECT p.id, s.id, v.coverage_percent, v.per_claim_cap, v.annual_cap, v.waiting_period_days, v.eligible_relations, DATE '2025-03-21'
FROM coverage_plans p
JOIN service_types s ON true
JOIN (VALUES
    ('outpatient_visit', 80.00, 800000,  8000000,  0,  ARRAY['self','spouse','child','parent']),
    ('pharmacy',         90.00, 1500000, 15000000, 0,  ARRAY['self','spouse','child','parent']),
    ('dental',           70.00, 5000000, 25000000, 60, ARRAY['self','spouse','child']),
    ('hospitalization',  95.00, 80000000,150000000,0,  ARRAY['self','spouse','child','parent']),
    ('optometry',        75.00, 3000000, 6000000,  90, ARRAY['self','spouse','child'])
) AS v(code, coverage_percent, per_claim_cap, annual_cap, waiting_period_days, eligible_relations)
    ON v.code = s.code
WHERE p.name = 'ویژه';
