import { client } from './client'

/** One document attached to a claim (mirrors the Attachment schema). */
export interface Attachment {
  id: string
  claim_id: string
  file_name: string
  uploaded_at: string
  download_url: string
}

/** Claim statuses in which the API accepts new documents. */
export const ATTACHMENT_UPLOADABLE_STATUSES = ['draft', 'returned_for_docs'] as const

/** Accepted types, matching the server's content sniffing. */
export const ATTACHMENT_ACCEPT = 'application/pdf,image/jpeg,image/png,image/webp'

/** Server-side size limit, mirrored here so the UI can reject early. */
export const MAX_ATTACHMENT_BYTES = 5 * 1024 * 1024

export async function listAttachments(claimId: string): Promise<Attachment[]> {
  const { data } = await client.get<Attachment[]>(`/claims/${claimId}/attachments`)
  return data
}

export async function uploadAttachment(claimId: string, file: File): Promise<Attachment> {
  const form = new FormData()
  form.append('file', file)
  // Content-Type is left unset on purpose: the browser must add the multipart
  // boundary itself.
  const { data } = await client.post<Attachment>(`/claims/${claimId}/attachments`, form)
  return data
}

/**
 * Fetch an attachment through the axios client (so the Authorization header is
 * attached) and hand it to the browser as a download. A plain <a href> cannot
 * be used because the endpoint requires a bearer token.
 */
export async function downloadAttachment(att: Attachment): Promise<void> {
  const { data } = await client.get<Blob>(
    `/claims/${att.claim_id}/attachments/${att.id}/download`,
    { responseType: 'blob' },
  )
  const url = URL.createObjectURL(data)
  try {
    const link = document.createElement('a')
    link.href = url
    link.download = att.file_name
    document.body.appendChild(link)
    link.click()
    link.remove()
  } finally {
    // Revoke on the next tick; revoking immediately can cancel the download.
    setTimeout(() => URL.revokeObjectURL(url), 10_000)
  }
}
