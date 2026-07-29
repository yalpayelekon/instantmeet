import { afterEach, describe, expect, it, vi } from 'vitest'
import { api } from './api'

const jsonResponse = (body: unknown, init: ResponseInit = {}) =>
  new Response(JSON.stringify(body), {
    status: 200,
    headers: { 'Content-Type': 'application/json' },
    ...init,
  })

describe('api client', () => {
  afterEach(() => vi.unstubAllGlobals())

  it('includes credentials and JSON headers', async () => {
    const fetchMock = vi.fn().mockResolvedValue(jsonResponse({ id: 'u1' }))
    vi.stubGlobal('fetch', fetchMock)

    await api.me()

    expect(fetchMock).toHaveBeenCalledWith('/api/me', expect.objectContaining({
      credentials: 'include',
      headers: expect.objectContaining({ 'Content-Type': 'application/json' }),
    }))
  })

  it('constructs action requests with a JSON body', async () => {
    const fetchMock = vi.fn().mockResolvedValue(jsonResponse({}))
    vi.stubGlobal('fetch', fetchMock)

    await api.action('room/a', 'admit', 'guest')

    expect(fetchMock).toHaveBeenCalledWith('/api/meetings/room/a/admit', expect.objectContaining({
      method: 'POST',
      body: '{"userId":"guest"}',
    }))
  })

  it('returns undefined for a successful 204 response', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(new Response(null, { status: 204 })))
    await expect(api.logout()).resolves.toBeUndefined()
  })

  it('propagates a JSON backend error', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(jsonResponse(
      { error: 'meeting not found' },
      { status: 404 },
    )))
    await expect(api.getMeeting('missing')).rejects.toThrow('meeting not found')
  })

  it('falls back to the HTTP status text for a non-JSON error', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(new Response('bad gateway', {
      status: 502,
      statusText: 'Bad Gateway',
    })))
    await expect(api.me()).rejects.toThrow('Bad Gateway')
  })
})
