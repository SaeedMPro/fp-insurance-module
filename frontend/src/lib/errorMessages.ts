/**
 * Persian text for the API's error messages.
 *
 * The API deliberately answers in English: it is a machine-facing contract that
 * the parent system also consumes (see backend/api/openapi.yaml), and its
 * messages double as log/debug text. The *user interface*, however, is entirely
 * Persian — so the translation belongs here, at the presentation boundary.
 *
 * Anything not in this map falls through unchanged, so a new backend message
 * degrades to English rather than to a useless generic string.
 */

const EXACT: Record<string, string> = {
  // --- authentication / authorization -------------------------------------
  'invalid username or password': 'نام کاربری یا گذرواژه نامعتبر است.',
  'missing bearer token': 'برای این عملیات باید وارد حساب شوید.',
  'invalid or expired token': 'نشست شما منقضی شده است؛ دوباره وارد شوید.',
  'authentication required': 'برای این عملیات باید وارد حساب شوید.',
  'insufficient role for this action': 'سطح دسترسی شما برای این اقدام کافی نیست.',
  'actor is not permitted to perform this action': 'شما اجازهٔ انجام این اقدام را ندارید.',
  'not permitted to access this claim': 'اجازهٔ دسترسی به این درخواست را ندارید.',
  'not permitted to view this employee': 'اجازهٔ مشاهدهٔ این پروندهٔ کارمندی را ندارید.',
  'missing X-API-Key header': 'کلید دسترسی ارسال نشده است.',
  'invalid API key': 'کلید دسترسی نامعتبر است.',

  // --- claim workflow ----------------------------------------------------
  "transition not allowed from the claim's current status":
    'این اقدام در وضعیت فعلی درخواست مجاز نیست.',
  'a reason is required for this action': 'برای این اقدام ذکر دلیل الزامی است.',
  'only employees or admins may submit claims':
    'ثبت درخواست فقط برای کارمند یا مدیر سامانه امکان‌پذیر است.',
  'claim has no payable amount to disburse':
    'برای این درخواست مبلغ قابل پرداختی محاسبه نشده است.',

  // --- pricing / eligibility (422) ---------------------------------------
  'no active coverage rule for this plan/service type on the receipt date':
    'برای این طرح و نوع خدمت، در تاریخ فاکتور قانون پوشش فعالی وجود ندارد.',
  'beneficiary is not eligible for this service under the current rule':
    'ذی‌نفع بر اساس قانون فعلی مشمول این خدمت نیست.',
  'employee has not completed the required waiting period':
    'دورهٔ انتظار لازم برای این خدمت سپری نشده است.',
  'employee is not active': 'این کارمند فعال نیست.',
  'dependent does not belong to this employee':
    'عضو تحت تکفل انتخاب‌شده به این کارمند تعلق ندارد.',
  'employee has no coverage plan assigned':
    'برای این کارمند طرح پوششی تعیین نشده است.',

  // --- validation --------------------------------------------------------
  'invalid request body': 'داده‌های ارسالی نامعتبر است.',
  'invalid id': 'شناسهٔ نامعتبر.',
  'requested amount must be positive': 'مبلغ درخواستی باید بزرگ‌تر از صفر باشد.',
  'dependent_id is required for a dependent claim':
    'برای درخواست عضو تحت تکفل، انتخاب عضو الزامی است.',
  'employee_id is required': 'انتخاب کارمند الزامی است.',
  'could not resolve caller\'s employee record':
    'پروندهٔ کارمندی حساب شما یافت نشد.',
  'this account is not linked to an employee record':
    'این حساب به هیچ پروندهٔ کارمندی متصل نیست.',
  'at least one eligible relation is required':
    'انتخاب حداقل یک نسبت مجاز الزامی است.',
  'coverage percent must be between 0 and 100':
    'درصد پوشش باید بین ۰ و ۱۰۰ باشد.',
  'code and name are required': 'کد و نام الزامی است.',
  'code or name exceeds maximum length': 'طول کد یا نام از حد مجاز بیشتر است.',
  'code must be lowercase letters, digits, and underscores':
    'کد فقط می‌تواند حروف کوچک انگلیسی، عدد و زیرخط باشد.',
  'service type code already exists': 'این کد نوع خدمت قبلاً ثبت شده است.',
  'username, password and full name are required':
    'نام کاربری، گذرواژه و نام و نام خانوادگی الزامی است.',
  'employee role requires a linked employee record':
    'نقش کارمند باید به یک پروندهٔ کارمندی متصل باشد.',
  'admin accounts cannot be created via the API; use seed or make create-admin':
    'نمی‌توانید حساب مدیر سامانه بسازید.',
  'cannot assign the admin role via the API':
    'نمی‌توانید نقش مدیر سامانه را اختصاص دهید.',
  'the admin role cannot be changed via the API':
    'نمی‌توانید نقش مدیر سامانه را تغییر دهید.',
  'session expired; please log in again': 'نشست منقضی شده؛ دوباره وارد شوید.',
  'account is inactive': 'حساب کاربری غیرفعال است.',

  // --- server ------------------------------------------------------------
  'internal error': 'خطای داخلی سامانه. اگر تکرار شد با پشتیبانی تماس بگیرید.',
}
const ENTITIES: Record<string, string> = {
  claim: 'درخواست',
  employee: 'کارمند',
  dependent: 'عضو تحت تکفل',
  user: 'کاربر',
  plan: 'طرح پوشش',
  'service type': 'نوع خدمت',
  'active coverage rule': 'قانون پوشش فعال',
}

/** Request-body field names used in the API's "invalid <field>" messages. */
const FIELDS: Record<string, string> = {
  plan_id: 'طرح',
  contract_id: 'قرارداد',
  service_type_id: 'نوع خدمت',
  dependent_id: 'عضو تحت تکفل',
  employee_id: 'کارمند',
  actor_user_id: 'کاربر',
}

/**
 * Translate one API error message to Persian, or return it unchanged when
 * there is no translation for it.
 */
export function translateApiError(message: string): string {
  const key = message.trim()
  if (EXACT[key]) return EXACT[key]

  const notFound = /^(.+) not found$/.exec(key)
  if (notFound) {
    const entity = ENTITIES[notFound[1]]
    return entity ? `${entity} یافت نشد.` : 'مورد درخواستی یافت نشد.'
  }

  const invalidField = /^invalid (.+)$/.exec(key)
  if (invalidField) {
    const field = FIELDS[invalidField[1]]
    return field ? `مقدار «${field}» نامعتبر است.` : 'مقدار ارسالی نامعتبر است.'
  }

  return message
}
