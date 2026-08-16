// End-to-end test for the Supplementary Insurance Module.
//
// Drives the real Persian UI in a headless Chrome through the full claim
// lifecycle (create -> submit -> review -> approve -> pay -> close, plus the
// reject path), publishes a coverage-rule change through the admin UI and
// proves it reprices the next claim with no code change, and checks RBAC and
// the audit trail. Numeric/state assertions go through the REST API; the UI
// assertions check the Persian screens actually drive those transitions.
//
// Requirements: the stack must be running (make up), seeded (make seed), and
// a Chrome/Chromium binary available.
//
// Env:
//   E2E_BASE_URL  frontend  (default http://localhost:5173)
//   E2E_API_URL   API base  (default http://localhost:8080/api/v1)
//   CHROME_PATH   Chrome binary (default /usr/bin/google-chrome-stable)
//
// Usage: cd e2e && npm install && npm test
import puppeteer from 'puppeteer-core'
import { mkdirSync, writeFileSync } from 'node:fs'

const BASE = process.env.E2E_BASE_URL ?? 'http://localhost:5173'
const API = process.env.E2E_API_URL ?? 'http://localhost:8080/api/v1'
const CHROME = process.env.CHROME_PATH ?? '/usr/bin/google-chrome-stable'
const ART = new URL('./artifacts/', import.meta.url).pathname
mkdirSync(ART, { recursive: true })

// A minimal but genuinely valid PDF. It has to be real bytes: the upload
// endpoint sniffs the content rather than trusting the name or the declared
// type, so a text file called .pdf is rejected — which is the point.
const SAMPLE_PDF = `${ART}فاکتور-داروخانه.pdf`
writeFileSync(
  SAMPLE_PDF,
  '%PDF-1.4\n1 0 obj<</Type/Catalog>>endobj\ntrailer<</Root 1 0 R>>\n%%EOF\n',
)

const wait = (ms) => new Promise((r) => setTimeout(r, ms))

// ---------- tiny assertion / reporting harness ----------
let passed = 0
let failed = 0
let currentPage = null

function assert(cond, msg) {
  if (!cond) throw new Error(`assertion failed: ${msg}`)
}
function assertEq(got, want, msg) {
  if (got !== want) throw new Error(`${msg}: got ${JSON.stringify(got)}, want ${JSON.stringify(want)}`)
}
function assertClose(got, want, msg, eps = 0.01) {
  if (Math.abs(got - want) > eps) throw new Error(`${msg}: got ${got}, want ${want}`)
}

async function step(name, fn) {
  try {
    await fn()
    passed++
    console.log(`  PASS  ${name}`)
  } catch (err) {
    failed++
    console.error(`  FAIL  ${name}\n        ${err.message}`)
    if (currentPage) {
      const shot = `${ART}FAIL-${name.replace(/[^a-z0-9]+/gi, '-')}.png`
      try {
        await currentPage.screenshot({ path: shot })
        console.error(`        screenshot: ${shot}`)
      } catch {
        /* page may be closed */
      }
    }
  }
}

// ---------- API helpers ----------
async function api(path, { method = 'GET', token, body, headers = {} } = {}) {
  const res = await fetch(`${API}${path}`, {
    method,
    headers: {
      'Content-Type': 'application/json',
      ...(token ? { Authorization: `Bearer ${token}` } : {}),
      ...headers,
    },
    body: body !== undefined ? JSON.stringify(body) : undefined,
  })
  let data = null
  try {
    data = await res.json()
  } catch {
    /* non-JSON */
  }
  return { status: res.status, data }
}

async function apiLogin(username, password) {
  const { status, data } = await api('/auth/login', { method: 'POST', body: { username, password } })
  assertEq(status, 200, `login ${username}`)
  return data
}

// ---------- browser helpers ----------
let browser

async function newSession() {
  const ctx = await browser.createBrowserContext()
  const page = await ctx.newPage()
  await page.setViewport({ width: 1440, height: 1400 })
  await page.emulateMediaFeatures([{ name: 'prefers-color-scheme', value: 'light' }])
  currentPage = page
  return page
}

async function uiLogin(page, username, password) {
  await page.goto(`${BASE}/login`, { waitUntil: 'networkidle2' })
  await page.waitForSelector('input#username')
  await page.type('input#username', username)
  await page.type('input#password', password)
  await Promise.all([
    page.waitForNavigation({ waitUntil: 'networkidle2' }).catch(() => {}),
    clickButton(page, 'ورود'),
  ])
  await wait(600)
}

/** Click the first visible <button> whose trimmed text equals `text`. */
async function clickButton(page, text, { last = false } = {}) {
  const ok = await page.evaluate(
    (t, pickLast) => {
      const btns = [...document.querySelectorAll('button')].filter(
        (b) => b.textContent.trim() === t && !b.disabled && b.offsetParent !== null,
      )
      const btn = pickLast ? btns[btns.length - 1] : btns[0]
      if (!btn) return false
      btn.click()
      return true
    },
    text,
    last,
  )
  assert(ok, `button «${text}» found and clicked`)
}

/** Set a React-controlled <select> found inside the <label> containing labelText. */
async function selectByLabel(page, labelText, optionText) {
  const ok = await page.evaluate(
    (lt, ot) => {
      const label = [...document.querySelectorAll('label')].find((l) => l.textContent.includes(lt))
      const sel = label?.querySelector('select')
      if (!sel) return false
      const opt = [...sel.options].find((o) => o.textContent.includes(ot))
      if (!opt) return false
      const setter = Object.getOwnPropertyDescriptor(window.HTMLSelectElement.prototype, 'value').set
      setter.call(sel, opt.value)
      sel.dispatchEvent(new Event('change', { bubbles: true }))
      return true
    },
    labelText,
    optionText,
  )
  assert(ok, `select «${labelText}» -> option «${optionText}»`)
}

/** Set a React-controlled <input>/<textarea> inside the <label> containing labelText. */
async function fillByLabel(page, labelText, value) {
  const ok = await page.evaluate(
    (lt, v) => {
      const label = [...document.querySelectorAll('label')].find((l) => l.textContent.includes(lt))
      const el = label?.querySelector('input, textarea')
      if (!el) return false
      const proto = el.tagName === 'TEXTAREA' ? window.HTMLTextAreaElement.prototype : window.HTMLInputElement.prototype
      const setter = Object.getOwnPropertyDescriptor(proto, 'value').set
      setter.call(el, v)
      el.dispatchEvent(new Event('input', { bubbles: true }))
      el.dispatchEvent(new Event('change', { bubbles: true }))
      return true
    },
    labelText,
    value,
  )
  assert(ok, `fill «${labelText}»`)
}

/**
 * Set a date field to today. The date inputs are Jalali pickers
 * (components/PersianDateInput), not plain <input type=date>: the trigger is a
 * button inside the label and the calendar renders in a portal on document.body,
 * so it cannot be reached from within the label subtree.
 */
async function pickTodayByLabel(page, labelText) {
  const opened = await page.evaluate((lt) => {
    const label = [...document.querySelectorAll('label')].find((l) => l.textContent.includes(lt))
    const btn = label?.querySelector('button[aria-haspopup="dialog"]')
    if (!btn) return false
    btn.click()
    return true
  }, labelText)
  assert(opened, `open date picker «${labelText}»`)

  // The panel is portalled, so wait for it rather than querying inside the label.
  await page.waitForSelector('div[role="dialog"][aria-label="تقویم شمسی"]', { timeout: 5000 })
  const picked = await page.evaluate(() => {
    const panel = document.querySelector('div[role="dialog"][aria-label="تقویم شمسی"]')
    const today = [...panel.querySelectorAll('button')].find((b) => b.textContent.trim() === 'امروز')
    if (!today) return false
    today.click()
    return true
  })
  assert(picked, `click «امروز» in the «${labelText}» picker`)
  await wait(200)
}

/** Wait until the claim-detail status badge shows `label`. */
async function waitForBadge(page, label, timeoutMs = 8000) {
  const deadline = Date.now() + timeoutMs
  for (;;) {
    const found = await page.evaluate(
      (l) => [...document.querySelectorAll('span')].some((s) => s.textContent.trim() === l),
      label,
    )
    if (found) return
    if (Date.now() > deadline) throw new Error(`status badge «${label}» did not appear`)
    await wait(300)
  }
}

function claimIdFromUrl(url) {
  const m = url.match(/\/claims\/([0-9a-f-]{36})/)
  assert(m, `claim id present in url (${url})`)
  return m[1]
}

const todayInput = () => new Date().toISOString().slice(0, 10)

// ---------- the suite ----------
console.log(`E2E against UI=${BASE} API=${API}`)
browser = await puppeteer.launch({
  executablePath: CHROME,
  headless: 'new',
  args: ['--no-sandbox', '--disable-setuid-sandbox', '--window-size=1440,1400'],
})

let adminToken, employeeToken
let claim1Id, claim2Id, claim3Id, claim4Id
let pharmacyId
let activeOptometryPctAtStart, newPct

await step('API: health check', async () => {
  const res = await fetch(`${API.replace(/\/api\/v1$/, '')}/healthz`)
  assertEq(res.status, 200, 'healthz')
})

await step('API: admin + employee tokens', async () => {
  adminToken = (await apiLogin('admin', 'Admin123!')).token
  employeeToken = (await apiLogin('saeed.mazahery', 'Employee123!')).token
})

// Demo accounts can drift while people click around (e.g. the Users page role
// dropdown); pin the roles this suite depends on back to their seeded values.
await step('API: reset demo-account roles (self-healing precondition)', async () => {
  const users = await api('/admin/users', { token: adminToken })
  assertEq(users.status, 200, 'list users')
  const want = { 'saeed.mazahery': 'employee', 'farzin.hamzei': 'employee', reviewer: 'reviewer', auditor: 'auditor' }
  for (const [username, role] of Object.entries(want)) {
    const u = users.data.find((x) => x.username === username)
    assert(u, `demo user ${username} exists (run \`make seed\`)`)
    if (u.role !== role) {
      const res = await api(`/admin/users/${u.id}`, { method: 'PATCH', token: adminToken, body: { role } })
      assertEq(res.status, 200, `reset ${username} -> ${role}`)
    }
  }
  // re-login so the employee token carries the (possibly fixed) role claim
  employeeToken = (await apiLogin('saeed.mazahery', 'Employee123!')).token
})

await step('API: read active استاندارد/عینک rule percent', async () => {
  const { status, data } = await api('/coverage-rules', { token: adminToken })
  assertEq(status, 200, 'list rules')
  const svc = await api('/service-types', { token: adminToken })
  const optometry = svc.data.find((s) => s.code === 'optometry')
  const plans = await api('/plans', { token: adminToken })
  const standard = plans.data.find((p) => p.name === 'استاندارد')
  const active = data.find(
    (r) => r.plan_id === standard.id && r.service_type_id === optometry.id && r.effective_to === null,
  )
  assert(active, 'an active optometry rule exists')
  activeOptometryPctAtStart = active.coverage_percent

  pharmacyId = svc.data.find((x) => x.code === 'pharmacy')?.id
  assert(pharmacyId, 'pharmacy service type exists')
})

// --- T1: employee creates + submits a claim through the UI ---
await step('UI: employee creates & submits an عینک claim', async () => {
  const page = await newSession()
  await uiLogin(page, 'saeed.mazahery', 'Employee123!')
  await page.goto(`${BASE}/claims/new`, { waitUntil: 'networkidle2' })
  await selectByLabel(page, 'نوع خدمت', 'عینک')
  await fillByLabel(page, 'مبلغ درخواستی', '100000')
  await pickTodayByLabel(page, 'تاریخ فاکتور')
  await fillByLabel(page, 'توضیحات', 'آزمون یکپارچه سرتاسری - عینک')
  await Promise.all([
    page.waitForNavigation({ waitUntil: 'networkidle2' }).catch(() => {}),
    clickButton(page, 'ایجاد پیش‌نویس درخواست'),
  ])
  await wait(600)
  claim1Id = claimIdFromUrl(page.url())
  await waitForBadge(page, 'پیش‌نویس')
  await clickButton(page, 'ثبت')
  await waitForBadge(page, 'ثبت‌شده')
  await page.screenshot({ path: `${ART}01-submitted.png` })
  await page.browserContext().close()
})

// --- T2: reviewer approves, pays, closes through the UI ---
await step('UI: reviewer approve -> pay -> close', async () => {
  const page = await newSession()
  await uiLogin(page, 'reviewer', 'Reviewer123!')
  await page.goto(`${BASE}/claims/${claim1Id}`, { waitUntil: 'networkidle2' })
  await waitForBadge(page, 'ثبت‌شده')
  await clickButton(page, 'شروع بررسی')
  await waitForBadge(page, 'در حال بررسی')
  await clickButton(page, 'تأیید')
  await waitForBadge(page, 'تأییدشده')
  await clickButton(page, 'ثبت پرداخت')
  await waitForBadge(page, 'پرداخت‌شده')
  await clickButton(page, 'بستن')
  await waitForBadge(page, 'بسته‌شده')
  await page.screenshot({ path: `${ART}02-closed.png` })
  await page.browserContext().close()
})

await step('API: closed claim priced by the active rule', async () => {
  const { status, data } = await api(`/claims/${claim1Id}`, { token: adminToken })
  assertEq(status, 200, 'get claim')
  assertEq(data.status, 'closed', 'final status')
  assertEq(data.coverage_percent_applied, activeOptometryPctAtStart, 'coverage % = active rule')
  assertClose(data.payable_amount, (100000 * activeOptometryPctAtStart) / 100, 'payable = amount × %')
  assert(data.paid_at && data.closed_at, 'paid_at/closed_at set')
})

// --- T3: reject path with mandatory reason ---
await step('UI: reject path with mandatory Persian reason', async () => {
  let page = await newSession()
  await uiLogin(page, 'saeed.mazahery', 'Employee123!')
  await page.goto(`${BASE}/claims/new`, { waitUntil: 'networkidle2' })
  await selectByLabel(page, 'نوع خدمت', 'دارو')
  await fillByLabel(page, 'مبلغ درخواستی', '50000')
  await pickTodayByLabel(page, 'تاریخ فاکتور')
  await Promise.all([
    page.waitForNavigation({ waitUntil: 'networkidle2' }).catch(() => {}),
    clickButton(page, 'ایجاد پیش‌نویس درخواست'),
  ])
  await wait(600)
  claim2Id = claimIdFromUrl(page.url())
  await clickButton(page, 'ثبت')
  await waitForBadge(page, 'ثبت‌شده')
  await page.browserContext().close()

  page = await newSession()
  await uiLogin(page, 'reviewer', 'Reviewer123!')
  await page.goto(`${BASE}/claims/${claim2Id}`, { waitUntil: 'networkidle2' })
  await clickButton(page, 'شروع بررسی')
  await waitForBadge(page, 'در حال بررسی')
  await clickButton(page, 'رد کردن')
  await wait(300)
  // the reason panel opens with a textarea; its confirm button is also «تأیید»,
  // rendered after the approve button — pick the last match.
  await page.evaluate((v) => {
    const ta = document.querySelector('textarea')
    const setter = Object.getOwnPropertyDescriptor(window.HTMLTextAreaElement.prototype, 'value').set
    setter.call(ta, v)
    ta.dispatchEvent(new Event('input', { bubbles: true }))
  }, 'نسخه پزشک پیوست نشده است')
  await clickButton(page, 'تأیید', { last: true })
  await waitForBadge(page, 'رد شده')
  await clickButton(page, 'بستن')
  await waitForBadge(page, 'بسته‌شده')
  await page.screenshot({ path: `${ART}03-rejected.png` })
  await page.browserContext().close()

  const { data } = await api(`/claims/${claim2Id}`, { token: adminToken })
  assertEq(data.status, 'closed', 'rejected claim closed')
  assertEq(data.rejection_reason, 'نسخه پزشک پیوست نشده است', 'reason persisted')
  assert(data.payable_amount === null, 'no payable on rejected claim')
})

// --- T4: config-driven policy change through the admin UI ---
await step('UI: admin publishes a new عینک rule version (no code change)', async () => {
  newPct = activeOptometryPctAtStart === 61 ? 62 : 61
  const page = await newSession()
  await uiLogin(page, 'admin', 'Admin123!')
  await page.goto(`${BASE}/coverage-rules`, { waitUntil: 'networkidle2' })
  await selectByLabel(page, 'طرح', 'استاندارد')
  await selectByLabel(page, 'نوع خدمت', 'عینک')
  await fillByLabel(page, 'درصد پوشش', String(newPct))
  await fillByLabel(page, 'دورهٔ انتظار', '0')
  await fillByLabel(page, 'سقف هر دفعه', '2000000')
  await fillByLabel(page, 'سقف سالانه', '4000000')
  await pickTodayByLabel(page, 'تاریخ اعمال از')
  await clickButton(page, 'انتشار نسخهٔ قانون')
  await wait(1200)
  await page.screenshot({ path: `${ART}04-rule-published.png` })
  await page.browserContext().close()

  const { data } = await api('/coverage-rules', { token: adminToken })
  const svc = await api('/service-types', { token: adminToken })
  const optometry = svc.data.find((s) => s.code === 'optometry')
  const plans = await api('/plans', { token: adminToken })
  const standard = plans.data.find((p) => p.name === 'استاندارد')
  const actives = data.filter(
    (r) => r.plan_id === standard.id && r.service_type_id === optometry.id && r.effective_to === null,
  )
  assertEq(actives.length, 1, 'exactly one active optometry rule after publish')
  assertEq(actives[0].coverage_percent, newPct, 'active rule has the new percent')
})

await step('UI+API: next claim is priced by the NEW rule version', async () => {
  const page = await newSession()
  await uiLogin(page, 'saeed.mazahery', 'Employee123!')
  await page.goto(`${BASE}/claims/new`, { waitUntil: 'networkidle2' })
  await selectByLabel(page, 'نوع خدمت', 'عینک')
  await fillByLabel(page, 'مبلغ درخواستی', '80000')
  await pickTodayByLabel(page, 'تاریخ فاکتور')
  await Promise.all([
    page.waitForNavigation({ waitUntil: 'networkidle2' }).catch(() => {}),
    clickButton(page, 'ایجاد پیش‌نویس درخواست'),
  ])
  await wait(600)
  claim3Id = claimIdFromUrl(page.url())
  await clickButton(page, 'ثبت')
  await waitForBadge(page, 'ثبت‌شده')
  await page.browserContext().close()

  const rev = await newSession()
  await uiLogin(rev, 'reviewer', 'Reviewer123!')
  await rev.goto(`${BASE}/claims/${claim3Id}`, { waitUntil: 'networkidle2' })
  await clickButton(rev, 'شروع بررسی')
  await waitForBadge(rev, 'در حال بررسی')
  await clickButton(rev, 'تأیید')
  await waitForBadge(rev, 'تأییدشده')
  await rev.screenshot({ path: `${ART}05-repriced.png` })
  await rev.browserContext().close()

  const { data } = await api(`/claims/${claim3Id}`, { token: adminToken })
  assertEq(data.coverage_percent_applied, newPct, 'claim priced with NEW percent')
  assertClose(data.payable_amount, (80000 * newPct) / 100, 'payable reflects new percent')
})

// --- T5: RBAC via API ---
await step('API: RBAC denials (employee/admin-only/anonymous)', async () => {
  const a = await api('/admin/users', { token: employeeToken })
  assertEq(a.status, 403, 'employee cannot list users')
  const auditor = await apiLogin('auditor', 'Auditor123!')
  const b = await api('/coverage-rules', {
    method: 'POST',
    token: auditor.token,
    body: {},
  })
  assertEq(b.status, 403, 'auditor cannot publish rules')
  const c = await api('/claims')
  assertEq(c.status, 401, 'anonymous rejected')
  const d = await api(`/employees/${(await api('/auth/me', { token: employeeToken })).data.employee_id}/remaining-caps`, {
    token: employeeToken,
  })
  assertEq(d.status, 200, 'employee sees own caps')
})

// --- T6: audit trail recorded everything ---
await step('API+UI: audit trail contains lifecycle + config_change', async () => {
  const trail = await api(`/claims/${claim1Id}/history`, { token: adminToken })
  const actions = trail.data.map((e) => e.action)
  for (const a of ['submit', 'start_review', 'approve', 'mark_paid', 'close']) {
    assert(actions.includes(a), `history has ${a}`)
  }
  const cfg = await api('/audit-logs?entity_type=coverage_rule&page_size=5', { token: adminToken })
  assert(cfg.data.total >= 1, 'config_change entries exist')

  const page = await newSession()
  await uiLogin(page, 'auditor', 'Auditor123!')
  await page.goto(`${BASE}/audit-logs`, { waitUntil: 'networkidle2' })
  await wait(800)
  const hasPersianAction = await page.evaluate(() =>
    [...document.querySelectorAll('span')].some((s) => ['تأیید', 'تغییر پیکربندی', 'ثبت'].includes(s.textContent.trim())),
  )
  assert(hasPersianAction, 'audit page renders Persian action labels')
  const h1 = await page.evaluate(() => document.querySelector('h1')?.textContent.trim())
  assertEq(h1, 'تاریخچه اقدامات', 'page title is the friendly name')
  await page.screenshot({ path: `${ART}06-audit.png` })
  await page.browserContext().close()
})

// --- T7: reports reflect the paid spend ---
// --- T5: the returned-for-docs loop, including the document upload ---
// This is the flow the whole "return for documents" branch exists to serve, so
// it is driven entirely through the UI: without an upload the employee has no
// way to answer the reviewer, and the loop is decorative.
await step('UI: return-for-docs → employee uploads a document → freeze on resubmit', async () => {
  // A claim in the review queue, created the fast way — the UI create path is
  // already covered by T1 and repeating it here only adds wall-clock.
  const created = await api('/claims', {
    method: 'POST',
    token: employeeToken,
    body: {
      service_type_id: pharmacyId,
      beneficiary_type: 'self',
      requested_amount: 250000,
      receipt_date: `${todayInput()}T00:00:00Z`,
      description: 'آزمون سرتاسری — بارگذاری مدارک',
    },
  })
  assertEq(created.status, 201, 'create claim for the docs loop')
  claim4Id = created.data.id
  assertEq((await api(`/claims/${claim4Id}/submit`, { method: 'POST', token: employeeToken })).status, 200, 'submit')

  // Reviewer sends it back for documents, through the UI.
  let page = await newSession()
  await uiLogin(page, 'reviewer', 'Reviewer123!')
  await page.goto(`${BASE}/claims/${claim4Id}`, { waitUntil: 'networkidle2' })
  await waitForBadge(page, 'ثبت‌شده')
  await clickButton(page, 'شروع بررسی')
  await waitForBadge(page, 'در حال بررسی')
  await clickButton(page, 'بازگرداندن برای مدارک')
  await wait(300)
  // Same trap as the reject path: the reason panel's confirm button is also
  // «تأیید» and is rendered after the approve button, so pick the last match.
  await page.evaluate((v) => {
    const ta = document.querySelector('textarea')
    const setter = Object.getOwnPropertyDescriptor(window.HTMLTextAreaElement.prototype, 'value').set
    setter.call(ta, v)
    ta.dispatchEvent(new Event('input', { bubbles: true }))
  }, 'لطفاً تصویر فاکتور داروخانه را پیوست کنید.')
  await clickButton(page, 'تأیید', { last: true })
  await waitForBadge(page, 'بازگشت برای تکمیل مدارک')
  await page.browserContext().close()

  // Employee attaches the invoice through the «مدارک» section.
  page = await newSession()
  await uiLogin(page, 'saeed.mazahery', 'Employee123!')
  await page.goto(`${BASE}/claims/${claim4Id}`, { waitUntil: 'networkidle2' })
  await waitForBadge(page, 'بازگشت برای تکمیل مدارک')

  const before = await api(`/claims/${claim4Id}/attachments`, { token: employeeToken })
  assertEq(before.data.length, 0, 'no documents before the upload')

  // The file input is deliberately hidden behind a styled button; puppeteer can
  // still set files on it directly, which is what clicking the button leads to.
  const input = await page.waitForSelector('input[type=file]', { timeout: 5000 })
  await input.uploadFile(SAMPLE_PDF)
  await wait(1500)

  const after = await api(`/claims/${claim4Id}/attachments`, { token: employeeToken })
  assertEq(after.status, 200, 'list attachments')
  assertEq(after.data.length, 1, 'exactly one document after the upload')
  assertEq(after.data[0].file_name, 'فاکتور-داروخانه.pdf', 'original Persian filename round-trips')

  const shown = await page.evaluate(() =>
    [...document.querySelectorAll('div')].some((d) => d.textContent.trim() === 'فاکتور-داروخانه.pdf'),
  )
  assert(shown, 'the document is listed in the «مدارک» section')
  await page.screenshot({ path: `${ART}07-attachment-uploaded.png` })

  // The upload is auditable like every other change.
  const trail = await api(`/claims/${claim4Id}/history`, { token: adminToken })
  assert(trail.data.some((e) => e.action === 'attachment_upload'), 'history records attachment_upload')

  await clickButton(page, 'ارسال مجدد')
  await waitForBadge(page, 'ثبت‌شده')
  // Back in the queue, the evidence is frozen: no upload control any more.
  const stillOffered = await page.evaluate(() =>
    [...document.querySelectorAll('button')].some((b) => b.textContent.trim() === 'بارگذاری مدرک'),
  )
  assert(!stillOffered, 'upload control disappears once the claim is resubmitted')
  await page.browserContext().close()

  // And the server refuses even if the UI is bypassed.
  const refused = await api(`/claims/${claim4Id}/attachments`, { method: 'POST', token: employeeToken })
  assert(refused.status === 409 || refused.status === 400, `frozen upload rejected (got ${refused.status})`)

  // A reviewer can read the documents but never add one.
  page = await newSession()
  await uiLogin(page, 'reviewer', 'Reviewer123!')
  await page.goto(`${BASE}/claims/${claim4Id}`, { waitUntil: 'networkidle2' })
  await wait(900)
  const reviewerSees = await page.evaluate(() => ({
    doc: [...document.querySelectorAll('div')].some((d) => d.textContent.trim() === 'فاکتور-داروخانه.pdf'),
    upload: [...document.querySelectorAll('button')].some((b) => b.textContent.trim() === 'بارگذاری مدرک'),
  }))
  assert(reviewerSees.doc, 'reviewer sees the uploaded document')
  assert(!reviewerSees.upload, 'reviewer is never offered the upload control')
  await page.browserContext().close()
})

await step('API: reports summary/spend sane after the run', async () => {
  const s = await api('/reports/summary', { token: adminToken })
  assertEq(s.status, 200, 'summary')
  assert(s.data.total_claims >= 3, 'claims counted')
  assert(s.data.total_paid_amount > 0, 'paid amount > 0')
  const byType = await api('/reports/spend-by-service-type', { token: adminToken })
  assert(byType.data.some((r) => r.service_type_name === 'عینک'), 'per-type report uses Persian name')
})

await browser.close()

console.log(`\n${passed} passed, ${failed} failed`)
process.exit(failed === 0 ? 0 : 1)
