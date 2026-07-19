import { useEffect, useState } from 'react'
import { listServiceTypes } from '../api/reference'
import type { ServiceType } from '../api/types'

let cache: ServiceType[] | null = null

/** Service types rarely change within a session; fetch once and cache in memory. */
export function useServiceTypes() {
  const [serviceTypes, setServiceTypes] = useState<ServiceType[]>(cache ?? [])
  const [loading, setLoading] = useState(!cache)

  useEffect(() => {
    if (cache) return
    let cancelled = false
    listServiceTypes()
      .then((data) => {
        if (cancelled) return
        cache = data
        setServiceTypes(data)
      })
      .finally(() => {
        if (!cancelled) setLoading(false)
      })
    return () => {
      cancelled = true
    }
  }, [])

  const byId = new Map(serviceTypes.map((s) => [s.id, s]))
  return { serviceTypes, byId, loading }
}
