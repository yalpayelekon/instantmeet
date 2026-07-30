import { act, renderHook, waitFor } from '@testing-library/react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import i18n from '../i18n'
import { useMediaDevices } from './useMediaDevices'

describe('useMediaDevices localization', () => {
  const getUserMedia = vi.fn()

  beforeEach(() => {
    const stream = { getTracks: () => [{ stop: vi.fn() }] }
    getUserMedia.mockResolvedValue(stream)
    Object.defineProperty(navigator, 'mediaDevices', {
      configurable: true,
      value: {
        getUserMedia,
        enumerateDevices: vi.fn().mockResolvedValue([]),
        addEventListener: vi.fn(),
        removeEventListener: vi.fn(),
      },
    })
  })

  afterEach(async () => {
    getUserMedia.mockReset()
    sessionStorage.clear()
    await i18n.changeLanguage('en')
  })

  it('does not request device permissions again when the language changes', async () => {
    const { result } = renderHook(() => useMediaDevices('room-language-test'))
    await waitFor(() => expect(result.current.ready).toBe(true))
    expect(getUserMedia).toHaveBeenCalledTimes(1)

    await act(async () => {
      await i18n.changeLanguage('tr')
    })

    expect(getUserMedia).toHaveBeenCalledTimes(1)
  })
})
