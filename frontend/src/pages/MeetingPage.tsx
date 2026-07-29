import { FormEvent, useCallback, useEffect, useMemo, useRef, useState } from 'react'
import {
  LiveKitRoom,
  ParticipantTile,
  RoomAudioRenderer,
  StartAudio,
  useLocalParticipant,
  useRoomContext,
  useTracks,
} from '@livekit/components-react'
import { RoomEvent, Track } from 'livekit-client'
import {
  Check, Copy, LogOut, MessageSquare, Mic, MicOff, MonitorUp, PhoneOff, Send, Settings, Shield, Users, Video, VideoOff, X,
} from 'lucide-react'
import { useNavigate, useParams } from 'react-router-dom'
import { PreJoin } from '../components/PreJoin'
import { useMeetingSocket, type SocketEvent, type SocketStatus } from '../hooks/useMeetingSocket'
import { DeviceSelects } from '../components/DeviceSelects'
import { loadMediaPrefs, useMediaDevices } from '../hooks/useMediaDevices'
import { api } from '../services/api'
import type { JoinResponse, Meeting, User } from '../types'

function VideoGrid() {
  const tracks = useTracks(
    [{ source: Track.Source.Camera, withPlaceholder: true }, { source: Track.Source.ScreenShare, withPlaceholder: false }],
    { onlySubscribed: false },
  )
  return (
    <div className={`video-grid tiles-${Math.min(tracks.length, 9)}`}>
      {tracks.map(track => (
        <ParticipantTile key={`${track.participant.identity}-${track.source}`} trackRef={track} />
      ))}
    </div>
  )
}

function ConnectionBanner({
  wsStatus,
  mediaStatus,
  onRetryWs,
}: {
  wsStatus: SocketStatus
  mediaStatus: 'connected' | 'reconnecting' | 'disconnected'
  onRetryWs: () => void
}) {
  if (wsStatus === 'open' && mediaStatus === 'connected') return null
  const wsBad = wsStatus === 'reconnecting' || wsStatus === 'connecting' || wsStatus === 'closed'
  const mediaBad = mediaStatus === 'reconnecting' || mediaStatus === 'disconnected'
  if (!wsBad && !mediaBad) return null

  let message = 'Reconnecting…'
  if (mediaStatus === 'disconnected' && (wsStatus === 'closed' || wsStatus === 'reconnecting')) {
    message = 'Connection lost'
  } else if (mediaStatus === 'reconnecting') {
    message = 'Reconnecting to media…'
  } else if (wsStatus === 'reconnecting' || wsStatus === 'connecting') {
    message = 'Reconnecting…'
  } else if (wsStatus === 'closed' || mediaStatus === 'disconnected') {
    message = 'Connection lost'
  }

  return (
    <div className="connection-banner" role="status">
      <span>{message}</span>
      {(wsStatus === 'closed' || wsStatus === 'reconnecting' || mediaStatus === 'disconnected') && (
        <button type="button" onClick={onRetryWs}>Retry</button>
      )}
    </div>
  )
}

function MediaReconnectWatcher({ onStatus }: { onStatus: (s: 'connected' | 'reconnecting' | 'disconnected') => void }) {
  const room = useRoomContext()
  useEffect(() => {
    const connected = () => onStatus('connected')
    const reconnecting = () => onStatus('reconnecting')
    const disconnected = () => onStatus('disconnected')
    onStatus('connected')
    room.on(RoomEvent.Reconnecting, reconnecting)
    room.on(RoomEvent.Reconnected, connected)
    room.on(RoomEvent.Connected, connected)
    room.on(RoomEvent.Disconnected, disconnected)
    return () => {
      room.off(RoomEvent.Reconnecting, reconnecting)
      room.off(RoomEvent.Reconnected, connected)
      room.off(RoomEvent.Connected, connected)
      room.off(RoomEvent.Disconnected, disconnected)
    }
  }, [room, onStatus])
  return null
}

export function MeetingPage({ user }: { user: User }) {
  const { id } = useParams()
  const navigate = useNavigate()
  const [join, setJoin] = useState<JoinResponse | null>(null)
  const [error, setError] = useState('')
  const [panel, setPanel] = useState<'people' | 'chat' | 'settings' | null>(null)
  const [message, setMessage] = useState('')
  const [copied, setCopied] = useState(false)
  const [ready, setReady] = useState(false)
  const [mediaStatus, setMediaStatus] = useState<'connected' | 'reconnecting' | 'disconnected'>('connected')
  const intentionalLeave = useRef(false)

  const refreshJoin = useCallback(async () => {
    if (!id) return
    try {
      setJoin(await api.join(id))
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Unable to join')
    }
  }, [id])

  useEffect(() => { void refreshJoin() }, [refreshJoin])

  // Reset pre-join when leaving the admitted state or switching meetings.
  useEffect(() => {
    if (join?.status !== 'admitted') setReady(false)
  }, [join?.status, id])

  const onSocket = useCallback((event: SocketEvent) => {
    if (event.type === 'meeting.ended' || (event.type === 'participant.removed' && event.userId === user.id)) {
      intentionalLeave.current = true
      navigate('/', { replace: true })
      return
    }
    if (event.type === 'participant.muted' && event.userId === user.id) {
      window.dispatchEvent(new Event('instantmeet:mute'))
    }
    if (event.type === 'participant.admitted' && event.userId === user.id) {
      void refreshJoin()
      return
    }
    if (event.type === 'participant.rejected' && event.userId === user.id) {
      setError('The host declined your request.')
      return
    }
    if ((event.type === 'meeting.updated' || event.type.startsWith('participant.')) && event.payload) {
      const meeting = event.payload as Meeting
      setJoin(current => (current ? { ...current, meeting } : current))
    }
    if (event.type === 'chat.message' && event.payload) {
      setJoin(current =>
        current
          ? { ...current, meeting: { ...current.meeting, chat: [...current.meeting.chat, event.payload as Meeting['chat'][number]] } }
          : current,
      )
    }
  }, [intentionalLeave, navigate, refreshJoin, user.id])

  const { status: wsStatus, retryNow } = useMeetingSocket(join ? id : undefined, onSocket)

  const leave = async () => {
    intentionalLeave.current = true
    if (id) await api.leave(id).catch(() => {})
    navigate('/')
  }
  const end = async () => {
    if (id && confirm('End this meeting for everyone?')) {
      intentionalLeave.current = true
      await api.end(id)
      navigate('/')
    }
  }
  const send = async (e: FormEvent) => {
    e.preventDefault()
    if (!id || !message.trim()) return
    const text = message
    setMessage('')
    await api.chat(id, text)
  }
  const copy = async () => {
    await navigator.clipboard.writeText(location.href)
    setCopied(true)
    setTimeout(() => setCopied(false), 1500)
  }
  const people = useMemo(() => (join ? Object.values(join.meeting.participants) : []), [join])

  const onLiveKitDisconnected = useCallback(() => {
    if (intentionalLeave.current) {
      navigate('/', { replace: true })
      return
    }
    setMediaStatus('disconnected')
  }, [intentionalLeave, navigate])

  if (error) {
    return (
      <div className="center-state">
        <span className="brand-mark">I</span>
        <h2>Couldn’t enter this room</h2>
        <p>{error}</p>
        <button className="button primary" onClick={() => navigate('/')}>Back home</button>
      </div>
    )
  }
  if (!join) {
    return (
      <div className="center-state">
        <div className="pulse-ring" />
        <h2>Finding your room…</h2>
      </div>
    )
  }

  if (join.status === 'waiting') {
    return (
      <div className="waiting-page with-prejoin">
        <div className="waiting-visual">
          <div className="avatar-large">{user.avatar ? <img src={user.avatar} alt="" /> : user.displayName[0]}</div>
          <span className="orbit one" />
          <span className="orbit two" />
        </div>
        <h1>You’re ready to join</h1>
        <p>The host knows you’re here. We’ll bring you in as soon as they approve your request.</p>
        <div className="waiting-code">
          <span>{id}</span>
          <button type="button" onClick={copy}>{copied ? <Check /> : <Copy />}</button>
        </div>
        <PreJoin
          meetingId={id!}
          waiting
          title="Set up your devices"
          subtitle="These preferences carry into the call when you’re admitted."
          onLeave={leave}
        />
        <ConnectionBanner wsStatus={wsStatus} mediaStatus="connected" onRetryWs={retryNow} />
      </div>
    )
  }

  if (!ready) {
    return (
      <div className="prejoin-page">
        <PreJoin
          meetingId={id!}
          onJoin={() => setReady(true)}
          onLeave={leave}
        />
        <ConnectionBanner wsStatus={wsStatus} mediaStatus="connected" onRetryWs={retryNow} />
      </div>
    )
  }

  const prefs = loadMediaPrefs(id!)

  return (
    <LiveKitRoom
      token={join.livekitToken}
      serverUrl={join.livekitUrl}
      connect
      audio={prefs.micEnabled ? (prefs.audioDeviceId ? { deviceId: prefs.audioDeviceId } : true) : false}
      video={prefs.cameraEnabled ? (prefs.videoDeviceId ? { deviceId: prefs.videoDeviceId } : true) : false}
      options={{
        audioOutput: prefs.outputDeviceId ? { deviceId: prefs.outputDeviceId } : undefined,
      }}
      data-lk-theme="default"
      onDisconnected={onLiveKitDisconnected}
    >
      <MediaReconnectWatcher onStatus={setMediaStatus} />
      <div className="meeting-shell">
        <ConnectionBanner wsStatus={wsStatus} mediaStatus={mediaStatus} onRetryWs={retryNow} />
        <header className="meeting-header">
          <a className="brand compact" href="/">
            <span className="brand-mark">I</span>
            <span>InstantMeet</span>
          </a>
          <div className="meeting-title">
            <span className="live-dot" /> Live <span>·</span> {id}
          </div>
          <button className="icon-button" onClick={copy} title="Copy meeting link" type="button">
            {copied ? <Check /> : <Copy />}
          </button>
        </header>
        <VideoGrid />
        <MeetingControls
          meeting={join.meeting}
          id={id!}
          panel={panel}
          setPanel={setPanel}
          leave={leave}
          end={end}
        />
        {panel && (
          <aside className="side-panel">
            <div className="panel-head">
              <h2>
                {panel === 'people' && `People (${people.length})`}
                {panel === 'chat' && 'In-call messages'}
                {panel === 'settings' && 'Settings'}
              </h2>
              <button type="button" onClick={() => setPanel(null)}><X /></button>
            </div>
            {panel === 'people' && (
              <div className="people-list">
                {join.meeting.isHost && Object.values(join.meeting.waitingRoom).length > 0 && (
                  <section>
                    <h3>Waiting to join</h3>
                    {Object.values(join.meeting.waitingRoom).map(p => (
                      <div className="person" key={p.userId}>
                        <Avatar name={p.displayName} src={p.avatar} />
                        <span>{p.displayName}</span>
                        <button className="accept" type="button" onClick={() => api.action(id!, 'admit', p.userId)}><Check /></button>
                        <button className="reject" type="button" onClick={() => api.action(id!, 'reject', p.userId)}><X /></button>
                      </div>
                    ))}
                  </section>
                )}
                <section>
                  <h3>In this meeting</h3>
                  {people.map(p => (
                    <div className="person" key={p.userId}>
                      <Avatar name={p.displayName} src={p.avatar} />
                      <span>
                        {p.displayName}{p.userId === user.id && ' (you)'}
                        <small>{p.isHost && <><Shield /> Host</>}</small>
                      </span>
                      {!p.micEnabled && <MicOff className="muted-icon" />}
                      {join.meeting.isHost && !p.isHost && (
                        <div className="person-actions">
                          <button type="button" onClick={() => api.action(id!, 'mute', p.userId)} title="Mute"><MicOff /></button>
                          <button type="button" onClick={() => api.action(id!, 'remove', p.userId)} title="Remove"><LogOut /></button>
                        </div>
                      )}
                    </div>
                  ))}
                </section>
              </div>
            )}
            {panel === 'chat' && (
              <>
                <div className="messages">
                  {join.meeting.chat.length === 0 ? (
                    <div className="empty-chat">
                      <MessageSquare />
                      <p>Messages are visible to everyone here and disappear when the meeting ends.</p>
                    </div>
                  ) : (
                    join.meeting.chat.map(m => (
                      <div className="message" key={m.id}>
                        <strong>
                          {m.displayName}
                          <time>{new Date(m.sentAt).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })}</time>
                        </strong>
                        <p>{m.text}</p>
                      </div>
                    ))
                  )}
                </div>
                <form className="chat-form" onSubmit={send}>
                  <input maxLength={1000} placeholder="Send a message" value={message} onChange={e => setMessage(e.target.value)} />
                  <button type="submit" disabled={!message.trim()}><Send /></button>
                </form>
              </>
            )}
            {panel === 'settings' && <SettingsPanel meetingId={id!} />}
          </aside>
        )}
        {mediaStatus === 'disconnected' && (
          <div className="media-recover">
            <h2>Media disconnected</h2>
            <p>Your connection to the room dropped. You can retry without leaving the meeting.</p>
            <button
              type="button"
              className="button primary"
              onClick={() => {
                setMediaStatus('reconnecting')
                setReady(false)
                requestAnimationFrame(() => setReady(true))
              }}
            >
              Reconnect media
            </button>
            <button type="button" className="button ghost" onClick={leave}>Leave meeting</button>
          </div>
        )}
        <RoomAudioRenderer />
        <StartAudio label="Enable meeting audio" />
      </div>
    </LiveKitRoom>
  )
}

function SettingsPanel({ meetingId }: { meetingId: string }) {
  const media = useMediaDevices(meetingId)
  const room = useRoomContext()
  const { localParticipant } = useLocalParticipant()

  const applyDevice = async (kind: 'audioinput' | 'videoinput' | 'audiooutput', deviceId: string) => {
    if (!deviceId) return
    try {
      await room.switchActiveDevice(kind, deviceId)
    } catch {
      /* some browsers reject output switches */
    }
  }

  return (
    <div className="settings-panel">
      <p className="settings-copy">Device choices apply to this meeting only and stay until you close the tab.</p>
      <DeviceSelects
        audioInputs={media.audioInputs}
        videoInputs={media.videoInputs}
        audioOutputs={media.audioOutputs}
        prefs={media.prefs}
        supportsOutput={media.supportsOutput}
        setPrefs={patch => {
          media.setPrefs(patch)
          if (patch.audioDeviceId) void applyDevice('audioinput', patch.audioDeviceId)
          if (patch.videoDeviceId) void applyDevice('videoinput', patch.videoDeviceId)
          if (patch.outputDeviceId) void applyDevice('audiooutput', patch.outputDeviceId)
        }}
      />
      <div className="settings-toggles">
        <button
          type="button"
          className={!media.prefs.micEnabled ? 'off' : ''}
          onClick={async () => {
            const next = !media.prefs.micEnabled
            media.setPrefs({ micEnabled: next })
            await localParticipant.setMicrophoneEnabled(next)
            await api.media(meetingId, { mic: next })
            window.dispatchEvent(new CustomEvent('instantmeet:media', { detail: { mic: next } }))
          }}
        >
          {media.prefs.micEnabled ? <Mic /> : <MicOff />}
          <span>{media.prefs.micEnabled ? 'Mic on' : 'Mic off'}</span>
        </button>
        <button
          type="button"
          className={!media.prefs.cameraEnabled ? 'off' : ''}
          onClick={async () => {
            const next = !media.prefs.cameraEnabled
            media.setPrefs({ cameraEnabled: next })
            await localParticipant.setCameraEnabled(next)
            await api.media(meetingId, { camera: next })
            window.dispatchEvent(new CustomEvent('instantmeet:media', { detail: { camera: next } }))
          }}
        >
          {media.prefs.cameraEnabled ? <Video /> : <VideoOff />}
          <span>{media.prefs.cameraEnabled ? 'Camera on' : 'Camera off'}</span>
        </button>
      </div>
      {media.permissionError && <p className="prejoin-warn">{media.permissionError}</p>}
    </div>
  )
}

function MeetingControls({
  meeting, id, panel, setPanel, leave, end,
}: {
  meeting: Meeting
  id: string
  panel: 'people' | 'chat' | 'settings' | null
  setPanel: (p: 'people' | 'chat' | 'settings' | null) => void
  leave: () => void
  end: () => void
}) {
  const prefs = loadMediaPrefs(id)
  const [mic, setMic] = useState(prefs.micEnabled)
  const [camera, setCamera] = useState(prefs.cameraEnabled)
  const [screen, setScreen] = useState(false)
  const { localParticipant } = useLocalParticipant()

  useEffect(() => {
    const mute = () => {
      void localParticipant.setMicrophoneEnabled(false)
      setMic(false)
    }
    const sync = (e: Event) => {
      const detail = (e as CustomEvent<{ mic?: boolean; camera?: boolean }>).detail
      if (typeof detail?.mic === 'boolean') setMic(detail.mic)
      if (typeof detail?.camera === 'boolean') setCamera(detail.camera)
    }
    window.addEventListener('instantmeet:mute', mute)
    window.addEventListener('instantmeet:media', sync)
    return () => {
      window.removeEventListener('instantmeet:mute', mute)
      window.removeEventListener('instantmeet:media', sync)
    }
  }, [localParticipant])

  const toggle = async (kind: 'mic' | 'camera' | 'screen') => {
    if (kind === 'mic') {
      const next = !mic
      await localParticipant.setMicrophoneEnabled(next)
      setMic(next)
      await api.media(id, { mic: next })
    }
    if (kind === 'camera') {
      const next = !camera
      await localParticipant.setCameraEnabled(next)
      setCamera(next)
      await api.media(id, { camera: next })
    }
    if (kind === 'screen') {
      const next = !screen
      await localParticipant.setScreenShareEnabled(next)
      setScreen(next)
      await api.media(id, { screen: next })
    }
  }

  return (
    <div className="controls">
      <div className="meeting-clock">{new Date().toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })}</div>
      <div className="control-center">
        <button type="button" className={!mic ? 'off' : ''} onClick={() => toggle('mic')} title={mic ? 'Mute' : 'Unmute'}>
          {mic ? <Mic /> : <MicOff />}
        </button>
        <button type="button" className={!camera ? 'off' : ''} onClick={() => toggle('camera')} title={camera ? 'Turn camera off' : 'Turn camera on'}>
          {camera ? <Video /> : <VideoOff />}
        </button>
        <button type="button" className={screen ? 'active' : ''} onClick={() => toggle('screen')} title="Share screen">
          <MonitorUp />
        </button>
        <button type="button" className="end-call" onClick={meeting.isHost ? end : leave} title={meeting.isHost ? 'End for everyone' : 'Leave'}>
          <PhoneOff />
        </button>
      </div>
      <div className="control-right">
        <button
          type="button"
          onClick={() => setPanel(panel === 'people' ? null : 'people')}
          className={panel === 'people' ? 'active' : ''}
        >
          <Users />
          <span>{Object.keys(meeting.participants).length}</span>
          {Object.keys(meeting.waitingRoom).length > 0 && <b>{Object.keys(meeting.waitingRoom).length}</b>}
        </button>
        <button
          type="button"
          onClick={() => setPanel(panel === 'chat' ? null : 'chat')}
          className={panel === 'chat' ? 'active' : ''}
        >
          <MessageSquare />
        </button>
        <button
          type="button"
          onClick={() => setPanel(panel === 'settings' ? null : 'settings')}
          className={panel === 'settings' ? 'active' : ''}
          title="Settings"
        >
          <Settings />
        </button>
      </div>
    </div>
  )
}

function Avatar({ name, src }: { name: string; src: string }) {
  return <span className="avatar">{src ? <img src={src} alt="" /> : name[0]}</span>
}
