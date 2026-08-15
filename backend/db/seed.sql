-- =============================================================================
-- Reference + demo dataset for the Supplementary Insurance Module.
-- Apply on an empty DB: `make seed` or `psql … -f backend/db/seed.sql`
-- (Not applied automatically on API boot — only db/init.sql is.)
--
-- Amounts are whole rial (ADR-0003). Dates are Gregorian; contract year
-- ۱۴۰۴–۱۴۰۵ spans 2025-03-21 → 2027-03-20. Most demo claims sit in ۱۴۰۵ so the
-- Reports page default range (current Jalali year → today) is populated.
--
-- Demo logins (bcrypt hashes below):
--   admin           / Admin123!      (sole admin — seed / make create-admin only)
--   reviewer        / Reviewer123!
--   auditor         / Auditor123!
--   saeed.mazahery  / Employee123!   → P-1001 سعید مظاهری (طرح استاندارد)
--   farzin.hamzei   / Employee123!   → P-1002 فرزین حمزه‌ای (طرح ویژه)
--
-- Integration (parent HR):
--   Header  X-API-Key: dev-integration-key
--   Stored  SHA-256 hex of that plaintext (never the raw key).
--
-- Scenario map (claims):
--   draft / submitted / under_review / returned_for_docs /
--   approved / rejected / paid / closed — plus dependent & premium paths,
--   per-claim cap, config-rule versioning, terminated employee, no-plan employee.
-- =============================================================================

-- ---------------------------------------------------------------------------
-- 1. Contract + plans
-- ---------------------------------------------------------------------------
INSERT INTO insurance_contracts (name, start_date, end_date, is_active)
VALUES (
    'قرارداد بیمه تکمیلی شرکت نوآوران داده ۱۴۰۴–۱۴۰۵',
    DATE '2025-03-21',
    DATE '2027-03-20',
    true
);

INSERT INTO coverage_plans (contract_id, name, description)
SELECT id,
       'استاندارد',
       'طرح پایهٔ کارکنان؛ سقف و درصد متوسط، دورهٔ انتظار برای دندان‌پزشکی و عینک'
FROM insurance_contracts
WHERE name = 'قرارداد بیمه تکمیلی شرکت نوآوران داده ۱۴۰۴–۱۴۰۵';

INSERT INTO coverage_plans (contract_id, name, description)
SELECT id,
       'ویژه',
       'طرح مدیران و کارکنان کلیدی؛ سقف بالاتر، پوشش بیشتر، دورهٔ انتظار کوتاه‌تر'
FROM insurance_contracts
WHERE name = 'قرارداد بیمه تکمیلی شرکت نوآوران داده ۱۴۰۴–۱۴۰۵';

-- ---------------------------------------------------------------------------
-- 2. Service catalogue
-- ---------------------------------------------------------------------------
INSERT INTO service_types (code, name) VALUES
    ('outpatient_visit', 'ویزیت'),
    ('pharmacy',         'دارو'),
    ('dental',           'دندان‌پزشکی'),
    ('hospitalization',  'بستری'),
    ('optometry',        'عینک');

-- ---------------------------------------------------------------------------
-- 3. Coverage rules (versioned)
--    استاندارد dental: v1 closed → v2 current (config_change story).
--    Other rules: single open version from contract start.
-- ---------------------------------------------------------------------------

-- Standard — all services except dental (dental versions below)
INSERT INTO coverage_rules (
    plan_id, service_type_id, coverage_percent, per_claim_cap, annual_cap,
    waiting_period_days, eligible_relations, effective_from
)
SELECT p.id, s.id, v.coverage_percent, v.per_claim_cap, v.annual_cap,
       v.waiting_period_days, v.eligible_relations, DATE '2025-03-21'
FROM coverage_plans p
JOIN service_types s ON true
JOIN (VALUES
    ('outpatient_visit', 70.00, 500000,   5000000,   0,   ARRAY['self','spouse','child','parent']),
    ('pharmacy',         80.00, 1000000,  10000000,  0,   ARRAY['self','spouse','child','parent']),
    ('hospitalization',  90.00, 50000000, 100000000, 30,  ARRAY['self','spouse','child','parent']),
    ('optometry',        60.00, 2000000,  4000000,   180, ARRAY['self','spouse','child'])
) AS v(code, coverage_percent, per_claim_cap, annual_cap, waiting_period_days, eligible_relations)
  ON v.code = s.code
WHERE p.name = 'استاندارد';

-- Standard dental v1 (closed) — early ۱۴۰۴, 45%
INSERT INTO coverage_rules (
    plan_id, service_type_id, coverage_percent, per_claim_cap, annual_cap,
    waiting_period_days, eligible_relations, effective_from, effective_to
)
SELECT p.id, s.id, 45.00, 2500000, 12000000, 90,
       ARRAY['self','spouse','child'],
       DATE '2025-03-21', DATE '2025-09-22'
FROM coverage_plans p
JOIN service_types s ON s.code = 'dental'
WHERE p.name = 'استاندارد';

-- Standard dental v2 (current) — raised to 50% mid-year via policy change
INSERT INTO coverage_rules (
    plan_id, service_type_id, coverage_percent, per_claim_cap, annual_cap,
    waiting_period_days, eligible_relations, effective_from
)
SELECT p.id, s.id, 50.00, 3000000, 15000000, 90,
       ARRAY['self','spouse','child'],
       DATE '2025-09-23'
FROM coverage_plans p
JOIN service_types s ON s.code = 'dental'
WHERE p.name = 'استاندارد';

-- Premium — all five services
INSERT INTO coverage_rules (
    plan_id, service_type_id, coverage_percent, per_claim_cap, annual_cap,
    waiting_period_days, eligible_relations, effective_from
)
SELECT p.id, s.id, v.coverage_percent, v.per_claim_cap, v.annual_cap,
       v.waiting_period_days, v.eligible_relations, DATE '2025-03-21'
FROM coverage_plans p
JOIN service_types s ON true
JOIN (VALUES
    ('outpatient_visit', 80.00, 800000,   8000000,   0,  ARRAY['self','spouse','child','parent']),
    ('pharmacy',         90.00, 1500000,  15000000,  0,  ARRAY['self','spouse','child','parent']),
    ('dental',           70.00, 5000000,  25000000,  60, ARRAY['self','spouse','child']),
    ('hospitalization',  95.00, 80000000, 150000000, 0,  ARRAY['self','spouse','child','parent']),
    ('optometry',        75.00, 3000000,  6000000,   90, ARRAY['self','spouse','child'])
) AS v(code, coverage_percent, per_claim_cap, annual_cap, waiting_period_days, eligible_relations)
  ON v.code = s.code
WHERE p.name = 'ویژه';

-- ---------------------------------------------------------------------------
-- 4. Employees (HR roster)
-- ---------------------------------------------------------------------------
INSERT INTO employees (personnel_no, full_name, national_id, employment_status, hire_date, department, plan_id)
SELECT 'P-1001', 'سعید مظاهری', '0012345678', 'active', DATE '2020-09-22', 'مهندسی نرم‌افزار', p.id
FROM coverage_plans p WHERE p.name = 'استاندارد';

INSERT INTO employees (personnel_no, full_name, national_id, employment_status, hire_date, department, plan_id)
SELECT 'P-1002', 'فرزین حمزه‌ای', '0013456789', 'active', DATE '2021-04-04', 'مالی و حسابداری', p.id
FROM coverage_plans p WHERE p.name = 'ویژه';

INSERT INTO employees (personnel_no, full_name, national_id, employment_status, hire_date, department, plan_id)
SELECT 'P-1003', 'مریم رضایی', '0014567890', 'active', DATE '2019-03-21', 'منابع انسانی', p.id
FROM coverage_plans p WHERE p.name = 'استاندارد';

-- Recent hire — dental/optometry waiting period still open on ویژه‌ish wait rules if assigned special; on ویژه dental wait=60d from 2026-06-01 → ready after ~2026-07-31
INSERT INTO employees (personnel_no, full_name, national_id, employment_status, hire_date, department, plan_id)
SELECT 'P-1004', 'حسین محمدی', '0015678901', 'active', DATE '2026-06-01', 'پشتیبانی فنی', p.id
FROM coverage_plans p WHERE p.name = 'ویژه';

INSERT INTO employees (personnel_no, full_name, national_id, employment_status, hire_date, department, plan_id)
SELECT 'P-1005', 'نازنین کاظمی', '0016789012', 'terminated', DATE '2018-03-21', 'فروش', p.id
FROM coverage_plans p WHERE p.name = 'استاندارد';

-- Active but no plan — create-claim must 422
INSERT INTO employees (personnel_no, full_name, national_id, employment_status, hire_date, department, plan_id)
VALUES ('P-1006', 'علی نوری', '0017890123', 'active', DATE '2023-11-11', 'انبار', NULL);

INSERT INTO employees (personnel_no, full_name, national_id, employment_status, hire_date, department, plan_id)
SELECT 'P-1007', 'زهرا اکبری', '0018901234', 'active', DATE '2022-07-15', 'مارکتینگ', p.id
FROM coverage_plans p WHERE p.name = 'استاندارد';

-- ---------------------------------------------------------------------------
-- 5. Dependents
-- ---------------------------------------------------------------------------
INSERT INTO dependents (employee_id, full_name, relation, national_id, birth_date)
SELECT e.id, 'سارا مظاهری', 'spouse', '0012345679', DATE '1992-11-03'
FROM employees e WHERE e.personnel_no = 'P-1001';

INSERT INTO dependents (employee_id, full_name, relation, national_id, birth_date)
SELECT e.id, 'آوا مظاهری', 'child', '0012345680', DATE '2015-05-12'
FROM employees e WHERE e.personnel_no = 'P-1001';

INSERT INTO dependents (employee_id, full_name, relation, national_id, birth_date)
SELECT e.id, 'اکبر مظاهری', 'parent', '0012345681', DATE '1965-02-18'
FROM employees e WHERE e.personnel_no = 'P-1001';

INSERT INTO dependents (employee_id, full_name, relation, national_id, birth_date)
SELECT e.id, 'لیلا حمزه‌ای', 'spouse', '0013456790', DATE '1990-08-27'
FROM employees e WHERE e.personnel_no = 'P-1002';

INSERT INTO dependents (employee_id, full_name, relation, national_id, birth_date)
SELECT e.id, 'پارسا حمزه‌ای', 'child', '0013456791', DATE '2018-01-09'
FROM employees e WHERE e.personnel_no = 'P-1002';

INSERT INTO dependents (employee_id, full_name, relation, national_id, birth_date)
SELECT e.id, 'فاطمه رضایی', 'parent', '0014567891', DATE '1960-12-01'
FROM employees e WHERE e.personnel_no = 'P-1003';

-- ---------------------------------------------------------------------------
-- 6. Users
-- ---------------------------------------------------------------------------
INSERT INTO users (username, password_hash, full_name, role, employee_id, is_active) VALUES
    ('admin',    '$2a$10$LMJy.iaiEr25zBdrpjeQeOGC2UNvFqXBMLrHAmdvwOQerxQi3h6Ky', 'مدیر سامانه',           'admin',    NULL, true),
    ('reviewer', '$2a$10$i0sBSNoZKNGY1PwsE.r0peVA9tLTq7eo6rK2Sj.YzyVQIIU3mPjPW', 'کارشناس بررسی خسارت',  'reviewer', NULL, true),
    ('auditor',  '$2a$10$ibuxYYYuT5mR5jFkFPGXq.K4LuQ5h8LuA0Zs60rM5iNv2rqCruyIm', 'بازرس سامانه',         'auditor',  NULL, true);

INSERT INTO users (username, password_hash, full_name, role, employee_id, is_active)
SELECT 'saeed.mazahery', '$2a$10$YRb6MN9a459kZTJSz4.My.mAKcchkJ9PEC7ODMcv64pBcMWWYiBHe',
       e.full_name, 'employee', e.id, true
FROM employees e WHERE e.personnel_no = 'P-1001';

INSERT INTO users (username, password_hash, full_name, role, employee_id, is_active)
SELECT 'farzin.hamzei', '$2a$10$YRb6MN9a459kZTJSz4.My.mAKcchkJ9PEC7ODMcv64pBcMWWYiBHe',
       e.full_name, 'employee', e.id, true
FROM employees e WHERE e.personnel_no = 'P-1002';

-- Inactive account (login must 401) — former employee
INSERT INTO users (username, password_hash, full_name, role, employee_id, is_active)
SELECT 'nazanin.kazemi', '$2a$10$YRb6MN9a459kZTJSz4.My.mAKcchkJ9PEC7ODMcv64pBcMWWYiBHe',
       e.full_name, 'employee', e.id, false
FROM employees e WHERE e.personnel_no = 'P-1005';

-- Stamp created_by on the current dental rule version (admin published it)
UPDATE coverage_rules r
SET created_by = u.id
FROM users u, coverage_plans p, service_types s
WHERE u.username = 'admin'
  AND p.name = 'استاندارد'
  AND s.code = 'dental'
  AND r.plan_id = p.id
  AND r.service_type_id = s.id
  AND r.effective_from = DATE '2025-09-23';

-- ---------------------------------------------------------------------------
-- 7. Claims — full lifecycle mix (payable math matches active rules)
--    Standard outpatient 70%, pharmacy 80% (cap 1_000_000), dental 50%, …
--    Premium outpatient 80%, pharmacy 90% (cap 1_500_000), …
-- ---------------------------------------------------------------------------

-- C01 draft — employee can submit
INSERT INTO claims (
    employee_id, beneficiary_type, service_type_id, plan_id,
    requested_amount, receipt_date, description, status, created_by, created_at, updated_at
)
SELECT e.id, 'self', s.id, e.plan_id, 350000, DATE '2026-08-10',
       'ویزیت پزشک عمومی — درمانگاه طرف قرارداد؛ پیش‌نویس کارمند',
       'draft', u.id,
       TIMESTAMPTZ '2026-08-10 09:15:00+03:30',
       TIMESTAMPTZ '2026-08-10 09:15:00+03:30'
FROM employees e
JOIN users u ON u.employee_id = e.id
JOIN service_types s ON s.code = 'outpatient_visit'
WHERE e.personnel_no = 'P-1001';

-- C02 submitted — reviewer queue
INSERT INTO claims (
    employee_id, beneficiary_type, service_type_id, plan_id,
    requested_amount, receipt_date, description, status,
    submitted_at, created_by, created_at, updated_at
)
SELECT e.id, 'self', s.id, e.plan_id, 480000, DATE '2026-07-18',
       'ویزیت متخصص داخلی — فاکتور ۷ تیر ۱۴۰۵',
       'submitted',
       TIMESTAMPTZ '2026-07-19 11:02:00+03:30', u.id,
       TIMESTAMPTZ '2026-07-18 20:40:00+03:30',
       TIMESTAMPTZ '2026-07-19 11:02:00+03:30'
FROM employees e
JOIN users u ON u.employee_id = e.id
JOIN service_types s ON s.code = 'outpatient_visit'
WHERE e.personnel_no = 'P-1001';

-- C03 under_review — dental self (approve path)
INSERT INTO claims (
    employee_id, beneficiary_type, service_type_id, plan_id,
    requested_amount, receipt_date, description, status,
    submitted_at, reviewed_by, reviewed_at, created_by, created_at, updated_at
)
SELECT e.id, 'self', s.id, e.plan_id, 1800000, DATE '2026-06-15',
       'ترمیم دندان ۶ بالا — کلینیک دندان‌پزشکی آریا',
       'under_review',
       TIMESTAMPTZ '2026-06-16 10:00:00+03:30',
       rev.id, TIMESTAMPTZ '2026-06-17 09:30:00+03:30',
       u.id,
       TIMESTAMPTZ '2026-06-15 18:00:00+03:30',
       TIMESTAMPTZ '2026-06-17 09:30:00+03:30'
FROM employees e
JOIN users u ON u.employee_id = e.id
JOIN users rev ON rev.username = 'reviewer'
JOIN service_types s ON s.code = 'dental'
WHERE e.personnel_no = 'P-1001';

-- C04 returned_for_docs — pharmacy for child (missing Rx image)
INSERT INTO claims (
    employee_id, beneficiary_type, dependent_id, service_type_id, plan_id,
    requested_amount, receipt_date, description, status, rejection_reason,
    submitted_at, reviewed_by, reviewed_at, created_by, created_at, updated_at
)
SELECT e.id, 'dependent', d.id, s.id, e.plan_id, 820000, DATE '2026-05-20',
       'داروی آنتی‌بیوتیک کودک — داروخانه دکتر جباری',
       'returned_for_docs',
       'تصویر نسخهٔ پزشک ضمیمه نشده است؛ لطفاً پس از بارگذاری مجدداً ارسال کنید.',
       TIMESTAMPTZ '2026-05-21 14:10:00+03:30',
       rev.id, TIMESTAMPTZ '2026-05-22 11:45:00+03:30',
       u.id,
       TIMESTAMPTZ '2026-05-20 19:00:00+03:30',
       TIMESTAMPTZ '2026-05-22 11:45:00+03:30'
FROM employees e
JOIN users u ON u.employee_id = e.id
JOIN users rev ON rev.username = 'reviewer'
JOIN dependents d ON d.employee_id = e.id AND d.full_name = 'آوا مظاهری'
JOIN service_types s ON s.code = 'pharmacy'
WHERE e.personnel_no = 'P-1001';

-- C05 approved — outpatient 70% of 400_000 = 280_000 (awaiting mark-paid)
INSERT INTO claims (
    employee_id, beneficiary_type, service_type_id, plan_id,
    requested_amount, receipt_date, description, status,
    coverage_percent_applied, payable_amount,
    submitted_at, reviewed_by, reviewed_at, created_by, created_at, updated_at
)
SELECT e.id, 'self', s.id, e.plan_id, 400000, DATE '2026-04-12',
       'ویزیت اورژانس درمانگاه شبانه‌روزی',
       'approved', 70.00, 280000,
       TIMESTAMPTZ '2026-04-13 08:20:00+03:30',
       rev.id, TIMESTAMPTZ '2026-04-14 16:05:00+03:30',
       u.id,
       TIMESTAMPTZ '2026-04-12 22:10:00+03:30',
       TIMESTAMPTZ '2026-04-14 16:05:00+03:30'
FROM employees e
JOIN users u ON u.employee_id = e.id
JOIN users rev ON rev.username = 'reviewer'
JOIN service_types s ON s.code = 'outpatient_visit'
WHERE e.personnel_no = 'P-1001';

-- C06 paid — pharmacy 80% of 900_000 = 720_000 (+ payment row)
INSERT INTO claims (
    employee_id, beneficiary_type, service_type_id, plan_id,
    requested_amount, receipt_date, description, status,
    coverage_percent_applied, payable_amount,
    submitted_at, reviewed_by, reviewed_at, paid_at, created_by, created_at, updated_at
)
SELECT e.id, 'self', s.id, e.plan_id, 900000, DATE '2026-04-22',
       'داروهای فشار خون — داروخانه مرکزی',
       'paid', 80.00, 720000,
       TIMESTAMPTZ '2026-04-23 09:00:00+03:30',
       rev.id, TIMESTAMPTZ '2026-04-24 10:15:00+03:30',
       TIMESTAMPTZ '2026-04-28 13:00:00+03:30',
       u.id,
       TIMESTAMPTZ '2026-04-22 17:30:00+03:30',
       TIMESTAMPTZ '2026-04-28 13:00:00+03:30'
FROM employees e
JOIN users u ON u.employee_id = e.id
JOIN users rev ON rev.username = 'reviewer'
JOIN service_types s ON s.code = 'pharmacy'
WHERE e.personnel_no = 'P-1001';

INSERT INTO payments (claim_id, amount, payment_reference, status, paid_at)
SELECT c.id, 720000, 'SIM-a4f91c02', 'simulated', c.paid_at
FROM claims c
JOIN employees e ON e.id = c.employee_id
WHERE e.personnel_no = 'P-1001'
  AND c.status = 'paid'
  AND c.requested_amount = 900000;

-- C07 closed — dental spouse 50% of 2_000_000 = 1_000_000
INSERT INTO claims (
    employee_id, beneficiary_type, dependent_id, service_type_id, plan_id,
    requested_amount, receipt_date, description, status,
    coverage_percent_applied, payable_amount,
    submitted_at, reviewed_by, reviewed_at, paid_at, closed_at, created_by, created_at, updated_at
)
SELECT e.id, 'dependent', d.id, s.id, e.plan_id, 2000000, DATE '2026-03-28',
       'جرم‌گیری و جرم‌برداری — همسر بیمه‌شده',
       'closed', 50.00, 1000000,
       TIMESTAMPTZ '2026-03-29 12:00:00+03:30',
       rev.id, TIMESTAMPTZ '2026-03-30 15:40:00+03:30',
       TIMESTAMPTZ '2026-04-05 11:00:00+03:30',
       TIMESTAMPTZ '2026-04-06 09:00:00+03:30',
       u.id,
       TIMESTAMPTZ '2026-03-28 16:20:00+03:30',
       TIMESTAMPTZ '2026-04-06 09:00:00+03:30'
FROM employees e
JOIN users u ON u.employee_id = e.id
JOIN users rev ON rev.username = 'reviewer'
JOIN dependents d ON d.employee_id = e.id AND d.full_name = 'سارا مظاهری'
JOIN service_types s ON s.code = 'dental'
WHERE e.personnel_no = 'P-1001';

INSERT INTO payments (claim_id, amount, payment_reference, status, paid_at)
SELECT c.id, 1000000, 'SIM-b7e2201d', 'simulated', c.paid_at
FROM claims c
JOIN employees e ON e.id = c.employee_id
WHERE e.personnel_no = 'P-1001'
  AND c.status = 'closed'
  AND c.requested_amount = 2000000;

-- C08 rejected → closed — invalid invoice
INSERT INTO claims (
    employee_id, beneficiary_type, service_type_id, plan_id,
    requested_amount, receipt_date, description, status, rejection_reason,
    submitted_at, reviewed_by, reviewed_at, closed_at, created_by, created_at, updated_at
)
SELECT e.id, 'self', s.id, e.plan_id, 5500000, DATE '2026-05-05',
       'عینک آفتابی ورزشی — فاکتور فروشگاه عینک آسمان',
       'closed',
       'خدمت درخواستی مشمول پوشش بیمه تکمیلی نیست؛ فاکتور مربوط به عینک طبی اصلاحی نیست.',
       TIMESTAMPTZ '2026-05-06 10:00:00+03:30',
       rev.id, TIMESTAMPTZ '2026-05-07 14:20:00+03:30',
       TIMESTAMPTZ '2026-05-08 09:10:00+03:30',
       u.id,
       TIMESTAMPTZ '2026-05-05 21:00:00+03:30',
       TIMESTAMPTZ '2026-05-08 09:10:00+03:30'
FROM employees e
JOIN users u ON u.employee_id = e.id
JOIN users rev ON rev.username = 'reviewer'
JOIN service_types s ON s.code = 'optometry'
WHERE e.personnel_no = 'P-1001';

-- C09 closed — child outpatient 70% of 300_000 = 210_000
INSERT INTO claims (
    employee_id, beneficiary_type, dependent_id, service_type_id, plan_id,
    requested_amount, receipt_date, description, status,
    coverage_percent_applied, payable_amount,
    submitted_at, reviewed_by, reviewed_at, paid_at, closed_at, created_by, created_at, updated_at
)
SELECT e.id, 'dependent', d.id, s.id, e.plan_id, 300000, DATE '2026-05-18',
       'ویزیت اطفال — درمانگاه کودکان مهر',
       'closed', 70.00, 210000,
       TIMESTAMPTZ '2026-05-18 19:30:00+03:30',
       rev.id, TIMESTAMPTZ '2026-05-19 11:00:00+03:30',
       TIMESTAMPTZ '2026-05-22 10:00:00+03:30',
       TIMESTAMPTZ '2026-05-23 08:30:00+03:30',
       u.id,
       TIMESTAMPTZ '2026-05-18 18:00:00+03:30',
       TIMESTAMPTZ '2026-05-23 08:30:00+03:30'
FROM employees e
JOIN users u ON u.employee_id = e.id
JOIN users rev ON rev.username = 'reviewer'
JOIN dependents d ON d.employee_id = e.id AND d.full_name = 'آوا مظاهری'
JOIN service_types s ON s.code = 'outpatient_visit'
WHERE e.personnel_no = 'P-1001';

INSERT INTO payments (claim_id, amount, payment_reference, status, paid_at)
SELECT c.id, 210000, 'SIM-c91aa0ef', 'simulated', c.paid_at
FROM claims c
JOIN employees e ON e.id = c.employee_id
JOIN dependents d ON d.id = c.dependent_id
WHERE e.personnel_no = 'P-1001'
  AND d.full_name = 'آوا مظاهری'
  AND c.requested_amount = 300000;

-- C10 pharmacy hit per-claim cap: 80% of 2_000_000 = 1_600_000 → capped to 1_000_000
INSERT INTO claims (
    employee_id, beneficiary_type, service_type_id, plan_id,
    requested_amount, receipt_date, description, status,
    coverage_percent_applied, payable_amount,
    submitted_at, reviewed_by, reviewed_at, paid_at, closed_at, created_by, created_at, updated_at
)
SELECT e.id, 'self', s.id, e.plan_id, 2000000, DATE '2026-06-02',
       'داروهای تخصصی بعد از جراحی سرپایی — سقف هر خسارت اعمال شد',
       'closed', 80.00, 1000000,
       TIMESTAMPTZ '2026-06-03 09:40:00+03:30',
       rev.id, TIMESTAMPTZ '2026-06-04 12:00:00+03:30',
       TIMESTAMPTZ '2026-06-08 14:00:00+03:30',
       TIMESTAMPTZ '2026-06-09 10:00:00+03:30',
       u.id,
       TIMESTAMPTZ '2026-06-02 16:00:00+03:30',
       TIMESTAMPTZ '2026-06-09 10:00:00+03:30'
FROM employees e
JOIN users u ON u.employee_id = e.id
JOIN users rev ON rev.username = 'reviewer'
JOIN service_types s ON s.code = 'pharmacy'
WHERE e.personnel_no = 'P-1001';

INSERT INTO payments (claim_id, amount, payment_reference, status, paid_at)
SELECT c.id, 1000000, 'SIM-d02bb1f0', 'simulated', c.paid_at
FROM claims c
JOIN employees e ON e.id = c.employee_id
WHERE e.personnel_no = 'P-1001'
  AND c.requested_amount = 2000000
  AND c.status = 'closed'
  AND c.service_type_id = (SELECT id FROM service_types WHERE code = 'pharmacy');

-- C11 parent hospitalization eligible on standard — paid/closed
INSERT INTO claims (
    employee_id, beneficiary_type, dependent_id, service_type_id, plan_id,
    requested_amount, receipt_date, description, status,
    coverage_percent_applied, payable_amount,
    submitted_at, reviewed_by, reviewed_at, paid_at, closed_at, created_by, created_at, updated_at
)
SELECT e.id, 'dependent', d.id, s.id, e.plan_id, 12000000, DATE '2026-07-01',
       'بستری دو روزهٔ پدر بیمه‌شده — بیمارستان پارسیان',
       'closed', 90.00, 10800000,
       TIMESTAMPTZ '2026-07-05 10:00:00+03:30',
       rev.id, TIMESTAMPTZ '2026-07-07 16:30:00+03:30',
       TIMESTAMPTZ '2026-07-15 11:00:00+03:30',
       TIMESTAMPTZ '2026-07-16 09:00:00+03:30',
       u.id,
       TIMESTAMPTZ '2026-07-04 20:00:00+03:30',
       TIMESTAMPTZ '2026-07-16 09:00:00+03:30'
FROM employees e
JOIN users u ON u.employee_id = e.id
JOIN users rev ON rev.username = 'reviewer'
JOIN dependents d ON d.employee_id = e.id AND d.full_name = 'اکبر مظاهری'
JOIN service_types s ON s.code = 'hospitalization'
WHERE e.personnel_no = 'P-1001';

INSERT INTO payments (claim_id, amount, payment_reference, status, paid_at)
SELECT c.id, 10800000, 'SIM-e13cc201', 'simulated', c.paid_at
FROM claims c
JOIN employees e ON e.id = c.employee_id
JOIN dependents d ON d.id = c.dependent_id
WHERE e.personnel_no = 'P-1001'
  AND d.full_name = 'اکبر مظاهری'
  AND c.requested_amount = 12000000;

-- ---- Premium employee (P-1002) ----

-- C12 submitted optometry
INSERT INTO claims (
    employee_id, beneficiary_type, service_type_id, plan_id,
    requested_amount, receipt_date, description, status,
    submitted_at, created_by, created_at, updated_at
)
SELECT e.id, 'self', s.id, e.plan_id, 2800000, DATE '2026-07-25',
       'عینک طبی پروگرسیو — بینایی‌سنجی نور',
       'submitted',
       TIMESTAMPTZ '2026-07-26 08:50:00+03:30', u.id,
       TIMESTAMPTZ '2026-07-25 21:15:00+03:30',
       TIMESTAMPTZ '2026-07-26 08:50:00+03:30'
FROM employees e
JOIN users u ON u.employee_id = e.id
JOIN service_types s ON s.code = 'optometry'
WHERE e.personnel_no = 'P-1002';

-- C13 under_review outpatient premium
INSERT INTO claims (
    employee_id, beneficiary_type, service_type_id, plan_id,
    requested_amount, receipt_date, description, status,
    submitted_at, reviewed_by, reviewed_at, created_by, created_at, updated_at
)
SELECT e.id, 'self', s.id, e.plan_id, 600000, DATE '2026-08-02',
       'ویزیت متخصص قلب — کلینیک حکیم',
       'under_review',
       TIMESTAMPTZ '2026-08-03 09:00:00+03:30',
       rev.id, TIMESTAMPTZ '2026-08-04 10:20:00+03:30',
       u.id,
       TIMESTAMPTZ '2026-08-02 18:40:00+03:30',
       TIMESTAMPTZ '2026-08-04 10:20:00+03:30'
FROM employees e
JOIN users u ON u.employee_id = e.id
JOIN users rev ON rev.username = 'reviewer'
JOIN service_types s ON s.code = 'outpatient_visit'
WHERE e.personnel_no = 'P-1002';

-- C14 closed — premium pharmacy 90% of 1_200_000 = 1_080_000
INSERT INTO claims (
    employee_id, beneficiary_type, service_type_id, plan_id,
    requested_amount, receipt_date, description, status,
    coverage_percent_applied, payable_amount,
    submitted_at, reviewed_by, reviewed_at, paid_at, closed_at, created_by, created_at, updated_at
)
SELECT e.id, 'self', s.id, e.plan_id, 1200000, DATE '2026-04-08',
       'داروهای دیابت — داروخانه ۱۳ آبان',
       'closed', 90.00, 1080000,
       TIMESTAMPTZ '2026-04-09 11:00:00+03:30',
       rev.id, TIMESTAMPTZ '2026-04-10 09:45:00+03:30',
       TIMESTAMPTZ '2026-04-14 12:00:00+03:30',
       TIMESTAMPTZ '2026-04-15 08:00:00+03:30',
       u.id,
       TIMESTAMPTZ '2026-04-08 15:00:00+03:30',
       TIMESTAMPTZ '2026-04-15 08:00:00+03:30'
FROM employees e
JOIN users u ON u.employee_id = e.id
JOIN users rev ON rev.username = 'reviewer'
JOIN service_types s ON s.code = 'pharmacy'
WHERE e.personnel_no = 'P-1002';

INSERT INTO payments (claim_id, amount, payment_reference, status, paid_at)
SELECT c.id, 1080000, 'SIM-f24dd312', 'simulated', c.paid_at
FROM claims c
JOIN employees e ON e.id = c.employee_id
WHERE e.personnel_no = 'P-1002'
  AND c.requested_amount = 1200000;

-- C15 closed — spouse dental premium 70% of 3_000_000 = 2_100_000
INSERT INTO claims (
    employee_id, beneficiary_type, dependent_id, service_type_id, plan_id,
    requested_amount, receipt_date, description, status,
    coverage_percent_applied, payable_amount,
    submitted_at, reviewed_by, reviewed_at, paid_at, closed_at, created_by, created_at, updated_at
)
SELECT e.id, 'dependent', d.id, s.id, e.plan_id, 3000000, DATE '2026-05-28',
       'ایمپلنت تک‌واحدی — همسر؛ طرح ویژه',
       'closed', 70.00, 2100000,
       TIMESTAMPTZ '2026-05-29 13:00:00+03:30',
       rev.id, TIMESTAMPTZ '2026-06-01 17:00:00+03:30',
       TIMESTAMPTZ '2026-06-10 10:30:00+03:30',
       TIMESTAMPTZ '2026-06-11 09:00:00+03:30',
       u.id,
       TIMESTAMPTZ '2026-05-28 19:00:00+03:30',
       TIMESTAMPTZ '2026-06-11 09:00:00+03:30'
FROM employees e
JOIN users u ON u.employee_id = e.id
JOIN users rev ON rev.username = 'reviewer'
JOIN dependents d ON d.employee_id = e.id AND d.full_name = 'لیلا حمزه‌ای'
JOIN service_types s ON s.code = 'dental'
WHERE e.personnel_no = 'P-1002';

INSERT INTO payments (claim_id, amount, payment_reference, status, paid_at)
SELECT c.id, 2100000, 'SIM-g35ee423', 'simulated', c.paid_at
FROM claims c
JOIN employees e ON e.id = c.employee_id
JOIN dependents d ON d.id = c.dependent_id
WHERE e.personnel_no = 'P-1002'
  AND d.full_name = 'لیلا حمزه‌ای'
  AND c.requested_amount = 3000000;

-- ---- Other staff (admin-created claims / report diversity) ----

-- C16 Maryam — submitted
INSERT INTO claims (
    employee_id, beneficiary_type, service_type_id, plan_id,
    requested_amount, receipt_date, description, status,
    submitted_at, created_by, created_at, updated_at
)
SELECT e.id, 'self', s.id, e.plan_id, 420000, DATE '2026-08-05',
       'ویزیت متخصص زنان — توسط ادمین به نمایندگی ثبت شده',
       'submitted',
       TIMESTAMPTZ '2026-08-05 16:00:00+03:30', adm.id,
       TIMESTAMPTZ '2026-08-05 15:50:00+03:30',
       TIMESTAMPTZ '2026-08-05 16:00:00+03:30'
FROM employees e
JOIN users adm ON adm.username = 'admin'
JOIN service_types s ON s.code = 'outpatient_visit'
WHERE e.personnel_no = 'P-1003';

-- C17 Maryam — closed pharmacy for spend-by-employee
INSERT INTO claims (
    employee_id, beneficiary_type, service_type_id, plan_id,
    requested_amount, receipt_date, description, status,
    coverage_percent_applied, payable_amount,
    submitted_at, reviewed_by, reviewed_at, paid_at, closed_at, created_by, created_at, updated_at
)
SELECT e.id, 'self', s.id, e.plan_id, 650000, DATE '2026-04-30',
       'داروهای فصلی — داروخانه دکتر حسینی',
       'closed', 80.00, 520000,
       TIMESTAMPTZ '2026-05-01 10:00:00+03:30',
       rev.id, TIMESTAMPTZ '2026-05-02 11:00:00+03:30',
       TIMESTAMPTZ '2026-05-06 12:00:00+03:30',
       TIMESTAMPTZ '2026-05-07 09:00:00+03:30',
       adm.id,
       TIMESTAMPTZ '2026-04-30 14:00:00+03:30',
       TIMESTAMPTZ '2026-05-07 09:00:00+03:30'
FROM employees e
JOIN users adm ON adm.username = 'admin'
JOIN users rev ON rev.username = 'reviewer'
JOIN service_types s ON s.code = 'pharmacy'
WHERE e.personnel_no = 'P-1003';

INSERT INTO payments (claim_id, amount, payment_reference, status, paid_at)
SELECT c.id, 520000, 'SIM-h46ff534', 'simulated', c.paid_at
FROM claims c
JOIN employees e ON e.id = c.employee_id
WHERE e.personnel_no = 'P-1003'
  AND c.requested_amount = 650000;

-- C18 Zahra — closed outpatient
INSERT INTO claims (
    employee_id, beneficiary_type, service_type_id, plan_id,
    requested_amount, receipt_date, description, status,
    coverage_percent_applied, payable_amount,
    submitted_at, reviewed_by, reviewed_at, paid_at, closed_at, created_by, created_at, updated_at
)
SELECT e.id, 'self', s.id, e.plan_id, 380000, DATE '2026-06-20',
       'ویزیت پوست — کلینیک زیبایی سلامت (بخش درمانی)',
       'closed', 70.00, 266000,
       TIMESTAMPTZ '2026-06-21 09:00:00+03:30',
       rev.id, TIMESTAMPTZ '2026-06-22 10:00:00+03:30',
       TIMESTAMPTZ '2026-06-25 11:00:00+03:30',
       TIMESTAMPTZ '2026-06-26 08:00:00+03:30',
       adm.id,
       TIMESTAMPTZ '2026-06-20 17:00:00+03:30',
       TIMESTAMPTZ '2026-06-26 08:00:00+03:30'
FROM employees e
JOIN users adm ON adm.username = 'admin'
JOIN users rev ON rev.username = 'reviewer'
JOIN service_types s ON s.code = 'outpatient_visit'
WHERE e.personnel_no = 'P-1007';

INSERT INTO payments (claim_id, amount, payment_reference, status, paid_at)
SELECT c.id, 266000, 'SIM-i57aa645', 'simulated', c.paid_at
FROM claims c
JOIN employees e ON e.id = c.employee_id
WHERE e.personnel_no = 'P-1007'
  AND c.requested_amount = 380000;

-- C19 recent hire draft — dental; waiting period still relevant if rushed to approve before ~2026-07-31
INSERT INTO claims (
    employee_id, beneficiary_type, service_type_id, plan_id,
    requested_amount, receipt_date, description, status, created_by, created_at, updated_at
)
SELECT e.id, 'self', s.id, e.plan_id, 1500000, DATE '2026-07-10',
       'کاشت کامپوزیت — استخدام جدید؛ در صورت تأیید زودهنگام ممکن است دورهٔ انتظار رد شود',
       'draft', adm.id,
       TIMESTAMPTZ '2026-07-10 12:00:00+03:30',
       TIMESTAMPTZ '2026-07-10 12:00:00+03:30'
FROM employees e
JOIN users adm ON adm.username = 'admin'
JOIN service_types s ON s.code = 'dental'
WHERE e.personnel_no = 'P-1004';

-- C20 ۱۴۰۴ historical closed (still useful for claim history / older filters)
INSERT INTO claims (
    employee_id, beneficiary_type, service_type_id, plan_id,
    requested_amount, receipt_date, description, status,
    coverage_percent_applied, payable_amount,
    submitted_at, reviewed_by, reviewed_at, paid_at, closed_at, created_by, created_at, updated_at
)
SELECT e.id, 'self', s.id, e.plan_id, 450000, DATE '2025-11-10',
       'ویزیت عمومی — نمونهٔ سال ۱۴۰۴ (بسته)',
       'closed', 70.00, 315000,
       TIMESTAMPTZ '2025-11-11 10:00:00+03:30',
       rev.id, TIMESTAMPTZ '2025-11-12 11:00:00+03:30',
       TIMESTAMPTZ '2025-11-18 12:00:00+03:30',
       TIMESTAMPTZ '2025-11-19 09:00:00+03:30',
       u.id,
       TIMESTAMPTZ '2025-11-10 16:00:00+03:30',
       TIMESTAMPTZ '2025-11-19 09:00:00+03:30'
FROM employees e
JOIN users u ON u.employee_id = e.id
JOIN users rev ON rev.username = 'reviewer'
JOIN service_types s ON s.code = 'outpatient_visit'
WHERE e.personnel_no = 'P-1001';

INSERT INTO payments (claim_id, amount, payment_reference, status, paid_at)
SELECT c.id, 315000, 'SIM-j68bb756', 'simulated', c.paid_at
FROM claims c
JOIN employees e ON e.id = c.employee_id
WHERE e.personnel_no = 'P-1001'
  AND c.receipt_date = DATE '2025-11-10';

-- ---------------------------------------------------------------------------
-- 8. Claim attachments (schema present; metadata only — no real files on disk)
-- ---------------------------------------------------------------------------
INSERT INTO claim_attachments (claim_id, file_name, file_path, uploaded_at)
SELECT c.id, 'invoice-pharmacy-jabari.pdf',
       '/demo/attachments/P-1001/invoice-pharmacy-jabari.pdf',
       TIMESTAMPTZ '2026-05-20 19:05:00+03:30'
FROM claims c
JOIN employees e ON e.id = c.employee_id
WHERE e.personnel_no = 'P-1001' AND c.status = 'returned_for_docs';

INSERT INTO claim_attachments (claim_id, file_name, file_path, uploaded_at)
SELECT c.id, 'dental-receipt-arya.jpg',
       '/demo/attachments/P-1001/dental-receipt-arya.jpg',
       TIMESTAMPTZ '2026-06-15 18:10:00+03:30'
FROM claims c
JOIN employees e ON e.id = c.employee_id
WHERE e.personnel_no = 'P-1001' AND c.status = 'under_review' AND c.requested_amount = 1800000;

-- ---------------------------------------------------------------------------
-- 9. Audit trail (claim transitions + one config_change)
-- ---------------------------------------------------------------------------

-- Config change: استاندارد dental 45% → 50%
INSERT INTO audit_logs (
    entity_type, entity_id, action, actor_user_id, actor_username,
    before_data, after_data, occurred_at
)
SELECT 'coverage_rule', r_new.id::text, 'config_change', u.id, u.username,
       jsonb_build_object(
           'previous_rule', jsonb_build_object(
               'coverage_percent', 45,
               'per_claim_cap', 2500000,
               'annual_cap', 12000000,
               'effective_from', '2025-03-21',
               'effective_to', '2025-09-22'
           )
       ),
       jsonb_build_object(
           'new_rule', jsonb_build_object(
               'coverage_percent', 50,
               'per_claim_cap', 3000000,
               'annual_cap', 15000000,
               'effective_from', '2025-09-23',
               'waiting_period_days', 90
           )
       ),
       TIMESTAMPTZ '2025-09-22 17:00:00+03:30'
FROM coverage_rules r_new
JOIN coverage_plans p ON p.id = r_new.plan_id AND p.name = 'استاندارد'
JOIN service_types s ON s.id = r_new.service_type_id AND s.code = 'dental'
JOIN users u ON u.username = 'admin'
WHERE r_new.effective_from = DATE '2025-09-23';

-- Helper: audit chain for C05 approved claim (outpatient 400k)
INSERT INTO audit_logs (entity_type, entity_id, action, actor_user_id, actor_username, before_data, after_data, occurred_at)
SELECT 'claim', c.id::text, a.action, a.actor_id, a.actor_name, a.before_data, a.after_data, a.occurred_at
FROM claims c
JOIN employees e ON e.id = c.employee_id
JOIN users emp ON emp.employee_id = e.id
JOIN users rev ON rev.username = 'reviewer'
CROSS JOIN LATERAL (VALUES
    ('submit',       emp.id, emp.username,
        '{"status":"draft"}'::jsonb,
        '{"status":"submitted"}'::jsonb,
        TIMESTAMPTZ '2026-04-13 08:20:00+03:30'),
    ('start_review', rev.id, rev.username,
        '{"status":"submitted"}'::jsonb,
        '{"status":"under_review"}'::jsonb,
        TIMESTAMPTZ '2026-04-14 10:00:00+03:30'),
    ('approve',      rev.id, rev.username,
        '{"status":"under_review"}'::jsonb,
        '{"status":"approved","coverage_percent_applied":70,"payable_amount":280000}'::jsonb,
        TIMESTAMPTZ '2026-04-14 16:05:00+03:30')
) AS a(action, actor_id, actor_name, before_data, after_data, occurred_at)
WHERE e.personnel_no = 'P-1001' AND c.status = 'approved' AND c.requested_amount = 400000;

-- Audit chain for C06 paid pharmacy
INSERT INTO audit_logs (entity_type, entity_id, action, actor_user_id, actor_username, before_data, after_data, occurred_at)
SELECT 'claim', c.id::text, a.action, a.actor_id, a.actor_name, a.before_data, a.after_data, a.occurred_at
FROM claims c
JOIN employees e ON e.id = c.employee_id
JOIN users emp ON emp.employee_id = e.id
JOIN users rev ON rev.username = 'reviewer'
CROSS JOIN LATERAL (VALUES
    ('submit',       emp.id, emp.username,
        '{"status":"draft"}'::jsonb, '{"status":"submitted"}'::jsonb,
        TIMESTAMPTZ '2026-04-23 09:00:00+03:30'),
    ('start_review', rev.id, rev.username,
        '{"status":"submitted"}'::jsonb, '{"status":"under_review"}'::jsonb,
        TIMESTAMPTZ '2026-04-24 09:00:00+03:30'),
    ('approve',      rev.id, rev.username,
        '{"status":"under_review"}'::jsonb,
        '{"status":"approved","coverage_percent_applied":80,"payable_amount":720000}'::jsonb,
        TIMESTAMPTZ '2026-04-24 10:15:00+03:30'),
    ('mark_paid',    rev.id, rev.username,
        '{"status":"approved"}'::jsonb,
        '{"status":"paid","payable_amount":720000}'::jsonb,
        TIMESTAMPTZ '2026-04-28 13:00:00+03:30')
) AS a(action, actor_id, actor_name, before_data, after_data, occurred_at)
WHERE e.personnel_no = 'P-1001' AND c.status = 'paid' AND c.requested_amount = 900000;

-- Audit chain for C04 returned_for_docs
INSERT INTO audit_logs (entity_type, entity_id, action, actor_user_id, actor_username, before_data, after_data, occurred_at)
SELECT 'claim', c.id::text, a.action, a.actor_id, a.actor_name, a.before_data, a.after_data, a.occurred_at
FROM claims c
JOIN employees e ON e.id = c.employee_id
JOIN users emp ON emp.employee_id = e.id
JOIN users rev ON rev.username = 'reviewer'
CROSS JOIN LATERAL (VALUES
    ('submit',           emp.id, emp.username,
        '{"status":"draft"}'::jsonb, '{"status":"submitted"}'::jsonb,
        TIMESTAMPTZ '2026-05-21 14:10:00+03:30'),
    ('start_review',     rev.id, rev.username,
        '{"status":"submitted"}'::jsonb, '{"status":"under_review"}'::jsonb,
        TIMESTAMPTZ '2026-05-22 10:00:00+03:30'),
    ('return_for_docs',  rev.id, rev.username,
        '{"status":"under_review"}'::jsonb,
        jsonb_build_object('status','returned_for_docs','reason', c.rejection_reason),
        TIMESTAMPTZ '2026-05-22 11:45:00+03:30')
) AS a(action, actor_id, actor_name, before_data, after_data, occurred_at)
WHERE e.personnel_no = 'P-1001' AND c.status = 'returned_for_docs';

-- Audit for rejected optometry path
INSERT INTO audit_logs (entity_type, entity_id, action, actor_user_id, actor_username, before_data, after_data, occurred_at)
SELECT 'claim', c.id::text, a.action, a.actor_id, a.actor_name, a.before_data, a.after_data, a.occurred_at
FROM claims c
JOIN employees e ON e.id = c.employee_id
JOIN users emp ON emp.employee_id = e.id
JOIN users rev ON rev.username = 'reviewer'
CROSS JOIN LATERAL (VALUES
    ('submit',       emp.id, emp.username,
        '{"status":"draft"}'::jsonb, '{"status":"submitted"}'::jsonb,
        TIMESTAMPTZ '2026-05-06 10:00:00+03:30'),
    ('start_review', rev.id, rev.username,
        '{"status":"submitted"}'::jsonb, '{"status":"under_review"}'::jsonb,
        TIMESTAMPTZ '2026-05-07 09:00:00+03:30'),
    ('reject',       rev.id, rev.username,
        '{"status":"under_review"}'::jsonb,
        jsonb_build_object('status','rejected','reason', c.rejection_reason),
        TIMESTAMPTZ '2026-05-07 14:20:00+03:30'),
    ('close',        rev.id, rev.username,
        '{"status":"rejected"}'::jsonb, '{"status":"closed"}'::jsonb,
        TIMESTAMPTZ '2026-05-08 09:10:00+03:30')
) AS a(action, actor_id, actor_name, before_data, after_data, occurred_at)
WHERE e.personnel_no = 'P-1001'
  AND c.rejection_reason IS NOT NULL
  AND c.service_type_id = (SELECT id FROM service_types WHERE code = 'optometry');

-- Sample logins (auditor filters)
INSERT INTO audit_logs (entity_type, entity_id, action, actor_user_id, actor_username, after_data, occurred_at)
SELECT 'user', u.id::text, 'login', u.id, u.username,
       jsonb_build_object('role', u.role),
       TIMESTAMPTZ '2026-08-15 08:30:00+03:30'
FROM users u WHERE u.username = 'admin';

INSERT INTO audit_logs (entity_type, entity_id, action, actor_user_id, actor_username, after_data, occurred_at)
SELECT 'user', u.id::text, 'login', u.id, u.username,
       jsonb_build_object('role', u.role),
       TIMESTAMPTZ '2026-08-15 09:05:00+03:30'
FROM users u WHERE u.username = 'reviewer';

INSERT INTO audit_logs (entity_type, entity_id, action, actor_user_id, actor_username, after_data, occurred_at)
SELECT 'user', u.id::text, 'login', u.id, u.username,
       jsonb_build_object('role', u.role),
       TIMESTAMPTZ '2026-08-14 14:20:00+03:30'
FROM users u WHERE u.username = 'saeed.mazahery';

-- ---------------------------------------------------------------------------
-- 10. Parent-system integration key
--     Raw key (dev only):  dev-integration-key
--     Hash: SHA-256 hex
-- ---------------------------------------------------------------------------
INSERT INTO integration_api_keys (name, api_key_hash, is_active)
VALUES (
    'سیستم منابع انسانی مادر — محیط توسعه',
    '076747a1afc12a721cf565002b57a7b07cd241b88c5de6ac040a963c1f08421f',
    true
);
