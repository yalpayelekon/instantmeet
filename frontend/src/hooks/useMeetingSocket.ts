import { useCallback, useEffect, useRef, useState } from 'react'

export type SocketEvent = { type: string; meetingId?: string; userId?: string; payload?: unknown }
export type SocketStatus = 'connecting' | 'open' | 'reconnecting' | 'closed'

const MAX_BACKOFF_MS = 15_000

export function useMeetingSocket(meetingId: string | undefined, onEvent: (event: SocketEvent) => void) {
  const [status, setStatus] = useState<SocketStatus>('closed')
  const onEventRef = useRef(onEvent)
  const retryRef = useRef(0)
  const manualCloseRef = useRef(false)
  const socketRef = useRef<WebSocket | null>(null)
  const timerRef = useRef<ReturnType<typeof setTimeout> | null>(null)
  const [retryTick, setRetryTick] = useState(0)

  useEffect(() => { onEventRef.current = onEvent }, [onEvent])

  const clearTimer = () => {
    if (timerRef.current) {
      clearTimeout(timerRef.current)
      timerRef.current = null
    }
  }

  const connect = useCallback(() => {
    if (!meetingId) return
    clearTimer()
    manualCloseRef.current = false
    const scheme = location.protocol === 'https:' ? 'wss' : 'ws'
    setStatus(retryRef.current > 0 ? 'reconnecting' : 'connecting')
    const socket = new WebSocket(`${scheme}://${location.host}/ws?meetingId=${encodeURIComponent(meetingId)}`)
    socketRef.current = socket
    socket.onopen = () => {
      retryRef.current = 0
      setStatus('open')
    }
    socket.onmessage = message => {
      try { onEventRef.current(JSON.parse(message.data)) } catch { /* ignore malformed events */ }
    }
    socket.onclose = () => {
      socketRef.current = null
      if (manualCloseRef.current || !meetingId) {
        setStatus('closed')
        return
      }
      setStatus('reconnecting')
      const delay = Math.min(MAX_BACKOFF_MS, 1000 * 2 ** Math.min(retryRef.current, 4))
      retryRef.current += 1
      timerRef.current = setTimeout(() => setRetryTick(t => t + 1), delay)
    }
    socket.onerror = () => {
      socket.close()
    }
  }, [meetingId])

  useEffect(() => {
    if (!meetingId) {
      manualCloseRef.current = true
      clearTimer()
      socketRef.current?.close()
      socketRef.current = null
      setStatus('closed')
      return
    }
    connect()
    return () => {
      manualCloseRef.current = true
      clearTimer()
      socketRef.current?.close()
      socketRef.current = null
    }
  }, [meetingId, connect, retryTick])

  const retryNow = useCallback(() => {
    if (!meetingId) return
    manualCloseRef.current = true
    clearTimer()
    socketRef.current?.close()
    socketRef.current = null
    retryRef.current = 0
    manualCloseRef.current = false
    setRetryTick(t => t + 1)
  }, [meetingId])

  return { status, retryNow }
}
