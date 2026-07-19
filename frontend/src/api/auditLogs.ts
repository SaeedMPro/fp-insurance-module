import { client } from './client'
import type { AuditLog, Paginated } from './types'

export interface ListAuditLogsParams {
  entity_type?: string
  entity_id?: string
  actor_user_id?: string
  action?: string
  from?: string
  to?: string
  page?: number
  page_size?: number
}

export async function listAuditLogs(params: ListAuditLogsParams = {}): Promise<Paginated<AuditLog>> {
  const { data } = await client.get<Paginated<AuditLog>>('/audit-logs', { params })
  return data
}
