import { act, renderHook, waitFor } from '@testing-library/react'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { api } from '../services/api'
import { useAuth } from './useAuth'

vi.mock('../services/api', () => ({
  api: { me: vi.fn(), logout: vi.fn() },
}))

const user = { id: 'u1', email: 'one@example.test', displayName: 'One', avatar: '' }

describe('useAuth', () => {
  beforeEach(() => vi.clearAllMocks())

  it('loads the current session and exposes its loading transition', async () => {
    vi.mocked(api.me).mockResolvedValue(user)
    const { result } = renderHook(() => useAuth())

    expect(result.current.loading).toBe(true)
    await waitFor(() => expect(result.current.loading).toBe(false))
    expect(result.current.user).toEqual(user)
  })

  it('treats a failed session lookup as signed out', async () => {
    vi.mocked(api.me).mockRejectedValue(new Error('unauthorized'))
    const { result } = renderHook(() => useAuth())

    await waitFor(() => expect(result.current.loading).toBe(false))
    expect(result.current.user).toBeNull()
  })

  it('clears the user after a successful logout', async () => {
    vi.mocked(api.me).mockResolvedValue(user)
    vi.mocked(api.logout).mockResolvedValue(undefined)
    const { result } = renderHook(() => useAuth())
    await waitFor(() => expect(result.current.user).toEqual(user))

    await act(() => result.current.logout())

    expect(api.logout).toHaveBeenCalledOnce()
    expect(result.current.user).toBeNull()
  })
})
