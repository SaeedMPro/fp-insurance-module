import { useEffect, useId, useMemo, useRef, useState } from 'react'
import { createPortal } from 'react-dom'

import {
  formatDigits,
  formatJalaliYmd,
  gregorianYmdToJalali,
  JALAALI_MONTHS,
  jalaliMonthDayCount,
  jalaliToGregorianYmd,
  todayYmd,
} from '../lib/format'
import { inputClass } from './FormField'

const WEEKDAYS = ['ش', 'ی', 'د', 'س', 'چ', 'پ', 'ج'] as const

type Props = {
  /** Gregorian calendar date as `YYYY-MM-DD` (API / state wire format). */
  value: string
  onChange: (gregorianYmd: string) => void
  className?: string
  disabled?: boolean
  id?: string
}

type View = { jy: number; jm: number }

function shiftMonth(view: View, delta: number): View {
  let { jy, jm } = view
  jm += delta
  while (jm < 1) {
    jm += 12
    jy -= 1
  }
  while (jm > 12) {
    jm -= 12
    jy += 1
  }
  return { jy, jm }
}

/** Saturday-based weekday index for a Jalali date (0 = شنبه). */
function jalaliWeekday(jy: number, jm: number, jd: number): number {
  const ymd = jalaliToGregorianYmd(jy, jm, jd)
  const [gy, gm, gd] = ymd.split('-').map(Number)
  // JS: 0=Sun … 6=Sat → convert so Saturday=0
  const sunBased = new Date(gy, gm - 1, gd).getDay()
  return (sunBased + 1) % 7
}

/**
 * Single-field Jalali date picker with an in-app calendar popover.
 * Parent state stays Gregorian `YYYY-MM-DD` for API compatibility.
 */
export function PersianDateInput({
  value,
  onChange,
  className = inputClass,
  disabled,
  id,
}: Props) {
  const today = useMemo(() => todayYmd(), [])
  const todayJ = useMemo(() => gregorianYmdToJalali(today)!, [today])
  const selected = value ? gregorianYmdToJalali(value) : null

  const [open, setOpen] = useState(false)
  const [view, setView] = useState<View>(() =>
    selected ? { jy: selected.jy, jm: selected.jm } : { jy: todayJ.jy, jm: todayJ.jm },
  )
  const rootRef = useRef<HTMLDivElement>(null)
  const panelRef = useRef<HTMLDivElement>(null)
  const [panelPos, setPanelPos] = useState<{ top: number; left: number; width: number } | null>(null)
  const listboxId = useId()

  const years = useMemo(() => {
    const list: number[] = []
    for (let y = todayJ.jy + 5; y >= todayJ.jy - 80; y--) list.push(y)
    return list
  }, [todayJ.jy])

  const cells = useMemo(() => {
    const daysInMonth = jalaliMonthDayCount(view.jy, view.jm)
    const startPad = jalaliWeekday(view.jy, view.jm, 1)
    const out: Array<{ jd: number; ymd: string } | null> = []
    for (let i = 0; i < startPad; i++) out.push(null)
    for (let jd = 1; jd <= daysInMonth; jd++) {
      out.push({ jd, ymd: jalaliToGregorianYmd(view.jy, view.jm, jd) })
    }
    while (out.length % 7 !== 0) out.push(null)
    return out
  }, [view.jy, view.jm])

  function placePanel() {
    const el = rootRef.current
    if (!el) return
    const r = el.getBoundingClientRect()
    const width = Math.max(r.width, 280)
    let left = r.left
    if (left + width > window.innerWidth - 8) left = Math.max(8, window.innerWidth - width - 8)
    const below = r.bottom + 6
    const approxHeight = 340
    const top =
      below + approxHeight > window.innerHeight - 8
        ? Math.max(8, r.top - approxHeight - 6)
        : below
    setPanelPos({ top, left, width })
  }

  useEffect(() => {
    if (!open) return
    placePanel()
    const onDoc = (e: MouseEvent) => {
      const t = e.target as Node
      if (rootRef.current?.contains(t) || panelRef.current?.contains(t)) return
      setOpen(false)
    }
    const onKey = (e: KeyboardEvent) => {
      if (e.key === 'Escape') setOpen(false)
    }
    const onReposition = () => placePanel()
    document.addEventListener('mousedown', onDoc)
    document.addEventListener('keydown', onKey)
    window.addEventListener('resize', onReposition)
    window.addEventListener('scroll', onReposition, true)
    return () => {
      document.removeEventListener('mousedown', onDoc)
      document.removeEventListener('keydown', onKey)
      window.removeEventListener('resize', onReposition)
      window.removeEventListener('scroll', onReposition, true)
    }
  }, [open])

  function openPicker() {
    if (disabled) return
    const next = selected ? { jy: selected.jy, jm: selected.jm } : { jy: todayJ.jy, jm: todayJ.jm }
    setView(next)
    setOpen(true)
  }

  function pick(ymd: string) {
    onChange(ymd)
    setOpen(false)
  }

  const label = value ? formatJalaliYmd(value) : 'انتخاب تاریخ'

  return (
    <div ref={rootRef} className="relative mt-1">
      <button
        type="button"
        id={id}
        disabled={disabled}
        aria-haspopup="dialog"
        aria-expanded={open}
        aria-controls={open ? listboxId : undefined}
        onClick={() => (open ? setOpen(false) : openPicker())}
        className={`${className} mt-0 flex w-full items-center justify-between gap-2 text-start ${
          value ? '' : 'text-slate-400 dark:text-slate-500'
        }`}
      >
        <span>{label}</span>
        <svg
          aria-hidden
          viewBox="0 0 20 20"
          className="h-4 w-4 shrink-0 text-slate-400"
          fill="none"
          stroke="currentColor"
          strokeWidth="1.8"
        >
          <rect x="3" y="4" width="14" height="13" rx="2" />
          <path d="M3 8h14M7 2v4M13 2v4" />
        </svg>
      </button>

      {open &&
        panelPos &&
        createPortal(
          <div
            ref={panelRef}
            id={listboxId}
            role="dialog"
            aria-label="تقویم شمسی"
            style={{
              position: 'fixed',
              top: panelPos.top,
              left: panelPos.left,
              width: panelPos.width,
              zIndex: 3000,
            }}
            className="rounded-xl border border-slate-200 bg-white p-3 shadow-xl dark:border-slate-700 dark:bg-slate-900"
          >
            <div className="mb-3 flex items-center gap-2">
              <button
                type="button"
                aria-label="ماه قبل"
                onClick={() => setView((v) => shiftMonth(v, -1))}
                className="rounded-lg p-1.5 text-slate-500 hover:bg-slate-100 dark:hover:bg-slate-800"
              >
                ‹
              </button>
              <select
                aria-label="ماه"
                value={view.jm}
                onChange={(e) => setView((v) => ({ ...v, jm: Number(e.target.value) }))}
                className="min-w-0 flex-1 rounded-md border border-slate-200 bg-transparent px-2 py-1 text-sm dark:border-slate-700"
              >
                {JALAALI_MONTHS.map((name, i) => (
                  <option key={name} value={i + 1}>
                    {name}
                  </option>
                ))}
              </select>
              <select
                aria-label="سال"
                value={view.jy}
                onChange={(e) => setView((v) => ({ ...v, jy: Number(e.target.value) }))}
                className="w-[5.5rem] rounded-md border border-slate-200 bg-transparent px-2 py-1 text-sm dark:border-slate-700"
              >
                {years.map((y) => (
                  <option key={y} value={y}>
                    {formatDigits(y)}
                  </option>
                ))}
              </select>
              <button
                type="button"
                aria-label="ماه بعد"
                onClick={() => setView((v) => shiftMonth(v, 1))}
                className="rounded-lg p-1.5 text-slate-500 hover:bg-slate-100 dark:hover:bg-slate-800"
              >
                ›
              </button>
            </div>

            <div className="mb-1 grid grid-cols-7 gap-1 text-center text-xs text-slate-400">
              {WEEKDAYS.map((d) => (
                <div key={d} className="py-1">
                  {d}
                </div>
              ))}
            </div>

            <div className="grid grid-cols-7 gap-1">
              {cells.map((cell, i) =>
                cell ? (
                  <button
                    key={cell.ymd}
                    type="button"
                    onClick={() => pick(cell.ymd)}
                    className={[
                      'rounded-lg py-1.5 text-sm transition-colors',
                      cell.ymd === value
                        ? 'bg-brand-600 font-semibold text-white'
                        : cell.ymd === today
                          ? 'bg-brand-50 text-brand-800 ring-1 ring-brand-200 dark:bg-brand-950 dark:text-brand-200 dark:ring-brand-800'
                          : 'text-slate-800 hover:bg-slate-100 dark:text-slate-100 dark:hover:bg-slate-800',
                    ].join(' ')}
                  >
                    {formatDigits(cell.jd)}
                  </button>
                ) : (
                  <div key={`e-${i}`} />
                ),
              )}
            </div>

            <div className="mt-3 flex items-center justify-between border-t border-slate-100 pt-2 dark:border-slate-800">
              <button
                type="button"
                onClick={() => {
                  onChange('')
                  setOpen(false)
                }}
                className="text-xs text-slate-500 hover:text-slate-700 dark:hover:text-slate-300"
              >
                پاک کردن
              </button>
              <button
                type="button"
                onClick={() => pick(today)}
                className="text-xs font-medium text-brand-600 hover:text-brand-700 dark:text-brand-400"
              >
                امروز
              </button>
            </div>
          </div>,
          document.body,
        )}
    </div>
  )
}
