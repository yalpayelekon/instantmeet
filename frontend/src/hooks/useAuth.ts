import { useCallback, useEffect, useState } from 'react'
import { api } from '../services/api'
import type { User } from '../types'

export function useAuth() {
  const [user, setUser] = useState<User | null>(null)
  const [loading, setLoading] = useState(true)
  const refresh = useCallback(() => api.me().then(setUser).catch(() => setUser(null)).finally(() => setLoading(false)), [])
  useEffect(() => { void refresh() }, [refresh])
  const logout = async () => { await api.logout(); setUser(null) }
  return { user, loading, logout }
}

