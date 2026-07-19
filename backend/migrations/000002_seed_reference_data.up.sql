-- Reference/config data: contract, plans, service types, and the initial coverage-rule
-- versions. This is exactly the data an admin edits later through the API to change
-- benefits without any code change (acceptance criterion: policy changes via config only).

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
    ('outpatient_visit', 70.00, 500000.00,  5000000.00,  0,   ARRAY['self','spouse','child','parent']),
    ('pharmacy',         80.00, 1000000.00, 10000000.00, 0,   ARRAY['self','spouse','child','parent']),
    ('dental',           50.00, 3000000.00, 15000000.00, 90,  ARRAY['self','spouse','child']),
    ('hospitalization',  90.00, 50000000.00,100000000.00,30,  ARRAY['self','spouse','child','parent']),
    ('optometry',        60.00, 2000000.00, 4000000.00,  180, ARRAY['self','spouse','child'])
) AS v(code, coverage_percent, per_claim_cap, annual_cap, waiting_period_days, eligible_relations)
    ON v.code = s.code
WHERE p.name = 'استاندارد';

-- Premium plan rules (more generous)
INSERT INTO coverage_rules (plan_id, service_type_id, coverage_percent, per_claim_cap, annual_cap, waiting_period_days, eligible_relations, effective_from)
SELECT p.id, s.id, v.coverage_percent, v.per_claim_cap, v.annual_cap, v.waiting_period_days, v.eligible_relations, DATE '2025-03-21'
FROM coverage_plans p
JOIN service_types s ON true
JOIN (VALUES
    ('outpatient_visit', 80.00, 800000.00,  8000000.00,  0,  ARRAY['self','spouse','child','parent']),
    ('pharmacy',         90.00, 1500000.00, 15000000.00, 0,  ARRAY['self','spouse','child','parent']),
    ('dental',           70.00, 5000000.00, 25000000.00, 60, ARRAY['self','spouse','child']),
    ('hospitalization',  95.00, 80000000.00,150000000.00,0,  ARRAY['self','spouse','child','parent']),
    ('optometry',        75.00, 3000000.00, 6000000.00,  90, ARRAY['self','spouse','child'])
) AS v(code, coverage_percent, per_claim_cap, annual_cap, waiting_period_days, eligible_relations)
    ON v.code = s.code
WHERE p.name = 'ویژه';
