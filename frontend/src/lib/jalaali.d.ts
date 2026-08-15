//#region src/index.d.ts
/**
 * **jalaali-js** — Conversions between the Jalaali (Persian) and Gregorian
 * calendar systems.
 *
 * The conversion algorithm is the one published by Kazimierz M. Borkowski
 * in *The Persian calendar for 3000 years* (1996); it is exact within the
 * Jalaali year range `-61 … 3177` inclusive. Outside that range the
 * functions in this module throw {@link RangeError}.
 *
 * The Borkowski algorithm diverges from the leap-year rule used by the
 * ECMAScript `Intl` API (`en-US-u-ca-persian`) after Gregorian year 2256
 * (Jalaali 1634). Inside the range `1800 … 2256` the two agree exactly.
 *
 * @see https://www.astro.uni.torun.pl/~kb/Papers/EMP/PersianC-EMP.htm
 * @packageDocumentation
 */
/** A date in the Jalaali (Persian) calendar. */
interface JalaaliDate {
  /** Jalaali year (`-61 … 3177`). */
  jy: number;
  /** Jalaali month (`1 … 12`; 1 = Farvardin). */
  jm: number;
  /** Jalaali day of month (`1 … 31`). */
  jd: number;
}
/** A date in the Gregorian (proleptic) calendar. */
interface GregorianDate {
  /** Gregorian year. BC years are numbered `0, -1, -2, …`. */
  gy: number;
  /** Gregorian month (`1 … 12`). */
  gm: number;
  /** Gregorian day of month (`1 … 31`). */
  gd: number;
}
/** Full result of {@link jalCal}. */
interface JalCalResult {
  /**
   * Number of years since the last leap year (`0 … 4`).
   * `0` means the year *is* a leap year.
   */
  leap: number;
  /** Gregorian year of the beginning of the Jalaali year. */
  gy: number;
  /** Day of March (Gregorian) on which Farvardin 1 falls. */
  march: number;
}
/** Slim result of {@link jalCalShort} — omits the leap-cycle computation. */
interface JalCalShortResult {
  /** Gregorian year of the beginning of the Jalaali year. */
  gy: number;
  /** Day of March (Gregorian) on which Farvardin 1 falls. */
  march: number;
}
/** Bounds of a Jalaali week, returned by {@link jalaaliWeek}. */
interface JalaaliWeek {
  /** Saturday of the containing Jalaali week. */
  saturday: JalaaliDate;
  /** Friday of the containing Jalaali week. */
  friday: JalaaliDate;
}
/** Minimum supported Jalaali year (inclusive). */
declare const MIN_JALAALI_YEAR: -61;
/** Maximum supported Jalaali year (inclusive). */
declare const MAX_JALAALI_YEAR: number;
/**
 * Converts a Gregorian date to Jalaali.
 *
 * Two call forms are supported: individual Gregorian components, or a
 * `Date` object whose **local-time** year/month/day are read. If you need
 * UTC semantics, build the Gregorian components from `Date.prototype.getUTC*`
 * yourself and pass them in.
 *
 * @example
 * toJalaali(2016, 4, 11)            // { jy: 1395, jm: 1, jd: 23 }
 * toJalaali(new Date(2016, 3, 11))  // { jy: 1395, jm: 1, jd: 23 }
 *
 * @throws {RangeError} if the resulting Jalaali year falls outside
 *   `[-61, 3177]`. In practice this means a Gregorian year roughly outside
 *   `[560, 3798]`.
 */
declare function toJalaali(date: Date): JalaaliDate;
declare function toJalaali(gy: number, gm: number, gd: number): JalaaliDate;
/**
 * Converts a Jalaali date to Gregorian.
 *
 * @example
 * toGregorian(1395, 1, 23) // { gy: 2016, gm: 4, gd: 11 }
 *
 * @throws {RangeError} if `jy` falls outside `[-61, 3177]`.
 */
declare function toGregorian(jy: number, jm: number, jd: number): GregorianDate;
/**
 * Returns `true` if `(jy, jm, jd)` is a real date in the Jalaali calendar.
 *
 * The check is **lenient** — non-integer inputs that happen to fall inside
 * the allowed numeric ranges are accepted. Pass integers in production code.
 *
 * @example
 * isValidJalaaliDate(1394, 12, 30) // false (1394 is a common year)
 * isValidJalaaliDate(1395, 12, 30) // true  (1395 is leap)
 */
declare function isValidJalaaliDate(jy: number, jm: number, jd: number): boolean;
/**
 * Returns `true` if the given Jalaali year is leap (366 days).
 *
 * @example
 * isLeapJalaaliYear(1394) // false
 * isLeapJalaaliYear(1395) // true
 *
 * @throws {RangeError} if `jy` falls outside `[-61, 3177]`.
 */
declare function isLeapJalaaliYear(jy: number): boolean;
/**
 * Returns the number of days in the given Jalaali month.
 *
 * The function trusts its inputs: `jm` is not range-checked, and an
 * out-of-range `jm` will return whichever branch happens to match.
 * For input validation, use {@link isValidJalaaliDate} instead.
 *
 * @example
 * jalaaliMonthLength(1394, 12) // 29 (common year)
 * jalaaliMonthLength(1395, 12) // 30 (leap year)
 */
declare function jalaaliMonthLength(jy: number, jm: number): number;
/**
 * Computes the leap-cycle state of a Jalaali year and the Gregorian date
 * of Farvardin 1 in that year.
 *
 * Use this directly when you need the leap field; if you only need
 * `{ gy, march }`, prefer {@link jalCalShort}.
 *
 * @example
 * jalCal(1391) // { leap: 0, gy: 2012, march: 20 }
 * jalCal(1395) // { leap: 0, gy: 2016, march: 20 }
 *
 * @throws {RangeError} if `jy` falls outside `[-61, 3177]`.
 */
declare function jalCal(jy: number): JalCalResult;
/**
 * Like {@link jalCal} but omits the leap-cycle computation.
 *
 * @remarks
 * Equivalent to v1's `jalCal(jy, true)` — extracted to its own export
 * in v2 to give it a proper return type. Prefer this when the leap field
 * isn't needed; it avoids a small amount of work per call.
 *
 * @example
 * jalCalShort(1391) // { gy: 2012, march: 20 }
 *
 * @throws {RangeError} if `jy` falls outside `[-61, 3177]`.
 */
declare function jalCalShort(jy: number): JalCalShortResult;
/**
 * Converts a Jalaali date to a Julian Day number.
 *
 * The result is an integer corresponding to noon UT on the given calendar
 * day. Inputs are not range-checked — pass valid dates.
 *
 * @example
 * j2d(1395, 1, 23) // 2457490
 */
declare function j2d(jy: number, jm: number, jd: number): number;
/**
 * Converts a Julian Day number to a Jalaali date.
 *
 * @example
 * d2j(2457490) // { jy: 1395, jm: 1, jd: 23 }
 *
 * @throws {RangeError} if `jdn` falls outside the supported Jalaali range
 *   `[-61, 3177]`.
 */
declare function d2j(jdn: number): JalaaliDate;
/**
 * Converts a Gregorian date to a Julian Day number.
 *
 * Tested correct from 1 March, -100100 (of both calendars) through several
 * million years into the future.
 *
 * @example
 * g2d(2016, 4, 11) // 2457490
 */
declare function g2d(gy: number, gm: number, gd: number): number;
/**
 * Converts a Julian Day number to a Gregorian date.
 *
 * Valid for `jdn ≥ -34839655` (year -100100 of both calendars).
 *
 * @example
 * d2g(2457490) // { gy: 2016, gm: 4, gd: 11 }
 */
declare function d2g(jdn: number): GregorianDate;
/**
 * Converts a Jalaali date (optionally with a time of day) to a JavaScript
 * `Date` constructed in the local time zone.
 *
 * Time components default to `0`. Out-of-range time components are *not*
 * clamped — they overflow the way the `Date` constructor handles them
 * (e.g. `h = 25` rolls into the next day).
 *
 * @example
 * jalaaliToDateObject(1400, 4, 30)
 *   // → new Date(2021, 6, 21, 0, 0, 0, 0)
 *
 * jalaaliToDateObject(1400, 4, 30, 14, 30)
 *   // → new Date(2021, 6, 21, 14, 30)
 */
declare function jalaaliToDateObject(jy: number, jm: number, jd: number, h?: number, m?: number, s?: number, ms?: number): Date;
/**
 * Returns the Saturday and Friday bounding the Jalaali week that contains
 * the given date. The Jalaali week starts on **Saturday**.
 *
 * @example
 * jalaaliWeek(1400, 4, 30)
 *   // → { saturday: { jy: 1400, jm: 4, jd: 26 },
 *   //     friday:   { jy: 1400, jm: 5, jd:  1 } }
 *
 * @remarks
 * Uses {@link jalaaliToDateObject} to read the day-of-week, which reflects
 * the host's local time zone. For dates near a DST boundary the underlying
 * `Date.getDay()` is still correct because the conversion is done in local
 * time on both sides.
 */
declare function jalaaliWeek(jy: number, jm: number, jd: number): JalaaliWeek;
//#endregion
export { GregorianDate, JalCalResult, JalCalShortResult, JalaaliDate, JalaaliWeek, MAX_JALAALI_YEAR, MIN_JALAALI_YEAR, d2g, d2j, g2d, isLeapJalaaliYear, isValidJalaaliDate, j2d, jalCal, jalCalShort, jalaaliMonthLength, jalaaliToDateObject, jalaaliWeek, toGregorian, toJalaali };
//# sourceMappingURL=index.d.ts.map