import type { ClaimStatus } from '../api/types'
import { CLAIM_STATUS_COLORS, CLAIM_STATUS_LABELS } from '../lib/format'

export function StatusBadge({ status }: { status: ClaimStatus }) {
  return (
    <span
      className={`inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-medium ${CLAIM_STATUS_COLORS[status]}`}
    >
      {CLAIM_STATUS_LABELS[status]}
    </span>
  )
}
