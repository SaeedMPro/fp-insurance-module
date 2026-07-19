import { useState } from 'react'

/** A simple collapsible key/value renderer for audit log before/after JSON blobs. */
export function JsonViewer({ data, label }: { data: Record<string, unknown> | null; label: string }) {
  const [open, setOpen] = useState(false)

  if (!data || Object.keys(data).length === 0) {
    return <span className="text-xs text-slate-400 dark:text-slate-500">{label}: موردی نیست</span>
  }

  return (
    <div className="rounded-lg border border-slate-200 dark:border-slate-800">
      <button
        type="button"
        onClick={() => setOpen((o) => !o)}
        className="flex w-full items-center justify-between px-3 py-2 text-start text-xs font-semibold text-slate-600 hover:bg-slate-50 dark:text-slate-300 dark:hover:bg-slate-800"
      >
        <span>{label}</span>
        <span aria-hidden>{open ? '▾' : '◂'}</span>
      </button>
      {open && (
        <dl className="divide-y divide-slate-100 border-t border-slate-100 text-xs dark:divide-slate-800 dark:border-slate-800">
          {Object.entries(data).map(([key, value]) => (
            <div key={key} className="flex gap-3 px-3 py-1.5">
              <dt className="dir-ltr w-1/3 shrink-0 truncate text-start font-mono text-slate-400 dark:text-slate-500">{key}</dt>
              <dd className="dir-ltr flex-1 break-all text-start font-mono text-slate-700 dark:text-slate-200">
                {formatValue(value)}
              </dd>
            </div>
          ))}
        </dl>
      )}
    </div>
  )
}

function formatValue(value: unknown): string {
  if (value === null || value === undefined) return '—'
  if (typeof value === 'object') return JSON.stringify(value)
  return String(value)
}
