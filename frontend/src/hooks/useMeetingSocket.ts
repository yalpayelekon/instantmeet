import { useEffect } from 'react'

export type SocketEvent = { type: string; meetingId?: string; userId?: string; payload?: unknown }

export function useMeetingSocket(meetingId: string | undefined, onEvent: (event: SocketEvent) => void) {
  useEffect(() => {
    if (!meetingId) return
    const scheme = location.protocol === 'https:' ? 'wss' : 'ws'
    const socket = new WebSocket(`${scheme}://${location.host}/ws?meetingId=${encodeURIComponent(meetingId)}`)
    socket.onmessage = message => {
      try { onEvent(JSON.parse(message.data)) } catch { /* ignore malformed events */ }
    }
    return () => socket.close()
  }, [meetingId, onEvent])
}

