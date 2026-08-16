import { useCallback, useEffect, useRef, useState } from 'react'

import {
  ATTACHMENT_ACCEPT,
  MAX_ATTACHMENT_BYTES,
  downloadAttachment,
  listAttachments,
  uploadAttachment,
  type Attachment,
} from '../../api/attachments'
import { apiErrorMessage } from '../../api/client'
import { useToast } from '../../context/useToast'
import { Card } from '../../components/Card'
import { ErrorBanner } from '../../components/ErrorBanner'
import { formatDateTime } from '../../lib/format'

interface Props {
  claimId: string
  /** True when the API will accept uploads: owner (or admin) on a claim in
   *  draft / returned-for-docs. The server enforces this too; the UI just
   *  avoids offering an action that would be refused. */
  canUpload: boolean
  /** Called after a successful upload so the parent can refresh the history. */
  onUploaded?: () => void
}

export function ClaimAttachments({ claimId, canUpload, onUploaded }: Props) {
  const { showToast } = useToast()
  const inputRef = useRef<HTMLInputElement>(null)

  const [items, setItems] = useState<Attachment[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [uploading, setUploading] = useState(false)
  const [downloadingId, setDownloadingId] = useState<string | null>(null)

  const load = useCallback(async () => {
    setLoading(true)
    setError(null)
    try {
      setItems(await listAttachments(claimId))
    } catch (err) {
      setError(apiErrorMessage(err))
    } finally {
      setLoading(false)
    }
  }, [claimId])

  useEffect(() => {
    load()
  }, [load])

  async function handleFile(file: File) {
    // Check the size before sending: a 20 MB photo should fail instantly rather
    // than after a long upload the server will reject anyway.
    if (file.size > MAX_ATTACHMENT_BYTES) {
      showToast('حجم فایل بیش از ۵ مگابایت است.', 'error')
      return
    }
    if (file.size === 0) {
      showToast('فایل انتخاب‌شده خالی است.', 'error')
      return
    }
    setUploading(true)
    setError(null)
    try {
      await uploadAttachment(claimId, file)
      showToast('مدرک بارگذاری شد.', 'success')
      await load()
      onUploaded?.()
    } catch (err) {
      const msg = apiErrorMessage(err)
      setError(msg)
      showToast(msg, 'error')
    } finally {
      setUploading(false)
      if (inputRef.current) inputRef.current.value = '' // allow re-picking the same file
    }
  }

  async function handleDownload(att: Attachment) {
    setDownloadingId(att.id)
    try {
      await downloadAttachment(att)
    } catch (err) {
      showToast(apiErrorMessage(err), 'error')
    } finally {
      setDownloadingId(null)
    }
  }

  return (
    <Card>
      <div className="flex flex-wrap items-center justify-between gap-2">
        <h2 className="text-sm font-semibold text-slate-900 dark:text-slate-50">مدارک</h2>
        {canUpload && (
          <>
            <input
              ref={inputRef}
              type="file"
              accept={ATTACHMENT_ACCEPT}
              className="hidden"
              onChange={(e) => {
                const file = e.target.files?.[0]
                if (file) handleFile(file)
              }}
            />
            <button
              type="button"
              disabled={uploading}
              onClick={() => inputRef.current?.click()}
              className="rounded-lg bg-brand-600 px-3 py-1.5 text-sm font-semibold text-white hover:bg-brand-700 disabled:cursor-not-allowed disabled:opacity-60"
            >
              {uploading ? 'در حال بارگذاری…' : 'بارگذاری مدرک'}
            </button>
          </>
        )}
      </div>

      {error && (
        <div className="mt-3">
          <ErrorBanner message={error} />
        </div>
      )}

      {loading ? (
        <p className="mt-3 text-sm text-slate-400 dark:text-slate-500">در حال دریافت…</p>
      ) : items.length === 0 ? (
        <p className="mt-3 text-sm text-slate-400 dark:text-slate-500">
          {canUpload
            ? 'هنوز مدرکی بارگذاری نشده است. تصویر یا فایل فاکتور را اضافه کنید.'
            : 'مدرکی برای این درخواست ثبت نشده است.'}
        </p>
      ) : (
        <ul className="mt-3 divide-y divide-slate-100 text-sm dark:divide-slate-800">
          {items.map((att) => (
            <li key={att.id} className="flex flex-wrap items-center justify-between gap-2 py-2">
              <div className="min-w-0">
                <div className="truncate font-medium text-slate-800 dark:text-slate-100">{att.file_name}</div>
                <div className="text-xs text-slate-400 dark:text-slate-500">{formatDateTime(att.uploaded_at)}</div>
              </div>
              <button
                type="button"
                disabled={downloadingId === att.id}
                onClick={() => handleDownload(att)}
                className="rounded-md border border-slate-200 px-2.5 py-1 text-xs font-medium text-slate-600 hover:bg-slate-100 disabled:opacity-60 dark:border-slate-700 dark:text-slate-300 dark:hover:bg-slate-800"
              >
                {downloadingId === att.id ? 'در حال دریافت…' : 'دانلود'}
              </button>
            </li>
          ))}
        </ul>
      )}

      {canUpload && (
        <p className="mt-3 text-xs text-slate-400 dark:text-slate-500">
          فرمت‌های مجاز: PDF، JPG، PNG، WebP — حداکثر ۵ مگابایت.
        </p>
      )}
    </Card>
  )
}
