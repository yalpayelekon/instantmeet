import { act, renderHook } from '@testing-library/react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { useMeetingSocket } from './useMeetingSocket'

class MockWebSocket {
  static instances: MockWebSocket[] = []
  url: string
  onopen: (() => void) | null = null
  onmessage: ((event: { data: string }) => void) | null = null
  onclose: (() => void) | null = null
  onerror: (() => void) | null = null
  close = vi.fn(() => this.onclose?.())

  constructor(url: string) {
    this.url = url
    MockWebSocket.instances.push(this)
  }
}

describe('useMeetingSocket', () => {
  beforeEach(() => {
    vi.useFakeTimers()
    MockWebSocket.instances = []
    vi.stubGlobal('WebSocket', MockWebSocket)
  })

  afterEach(() => {
    vi.useRealTimers()
    vi.unstubAllGlobals()
  })

  it('connects with an encoded id, opens, and dispatches valid events', () => {
    const onEvent = vi.fn()
    const { result } = renderHook(() => useMeetingSocket('room / one', onEvent))
    const socket = MockWebSocket.instances[0]

    expect(result.current.status).toBe('connecting')
    expect(socket.url).toBe('ws://localhost:3000/ws?meetingId=room%20%2F%20one')
    act(() => socket.onopen?.())
    expect(result.current.status).toBe('open')
    act(() => socket.onmessage?.({ data: '{"type":"meeting.updated"}' }))
    expect(onEvent).toHaveBeenCalledWith({ type: 'meeting.updated' })
  })

  it('ignores malformed events', () => {
    const onEvent = vi.fn()
    renderHook(() => useMeetingSocket('room', onEvent))
    act(() => MockWebSocket.instances[0].onmessage?.({ data: '{bad' }))
    expect(onEvent).not.toHaveBeenCalled()
  })

  it('reconnects with exponential backoff and resets after opening', () => {
    const { result } = renderHook(() => useMeetingSocket('room', vi.fn()))
    act(() => MockWebSocket.instances[0].onclose?.())
    expect(result.current.status).toBe('reconnecting')

    act(() => vi.advanceTimersByTime(999))
    expect(MockWebSocket.instances).toHaveLength(1)
    act(() => vi.advanceTimersByTime(1))
    expect(MockWebSocket.instances).toHaveLength(2)

    act(() => MockWebSocket.instances[1].onclose?.())
    act(() => vi.advanceTimersByTime(1999))
    expect(MockWebSocket.instances).toHaveLength(2)
    act(() => vi.advanceTimersByTime(1))
    expect(MockWebSocket.instances).toHaveLength(3)
    act(() => MockWebSocket.instances[2].onopen?.())
    expect(result.current.status).toBe('open')
  })

  it('retries immediately and closes on cleanup without reconnecting', () => {
    const { result, unmount } = renderHook(() => useMeetingSocket('room', vi.fn()))
    const first = MockWebSocket.instances[0]

    act(() => result.current.retryNow())
    expect(first.close).toHaveBeenCalledOnce()
    expect(MockWebSocket.instances).toHaveLength(2)

    const second = MockWebSocket.instances[1]
    unmount()
    expect(second.close).toHaveBeenCalledOnce()
    act(() => vi.runAllTimers())
    expect(MockWebSocket.instances).toHaveLength(2)
  })

  it('stays closed when no meeting id is supplied', () => {
    const { result } = renderHook(() => useMeetingSocket(undefined, vi.fn()))
    expect(result.current.status).toBe('closed')
    expect(MockWebSocket.instances).toHaveLength(0)
  })
})
