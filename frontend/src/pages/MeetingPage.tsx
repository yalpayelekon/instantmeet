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
import { useTranslation } from 'react-i18next'
import { LanguageSwitcher } from '../components/LanguageSwitcher'
import { PreJoin } from '../components/PreJoin'
import { useMeetingSocket, type SocketEvent, type SocketStatus } from '../hooks/useMeetingSocket'
import { DeviceSelects } from '../components/DeviceSelects'
import { loadMediaPrefs, useMediaDevices } from '../hooks/useMediaDevices'
import { api } from '../services/api'
import type { ChatMessage, JoinResponse, Meeting, User } from '../types'

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
  const { t } = useTranslation()
  if (wsStatus === 'open' && mediaStatus === 'connected') return null
  const wsBad = wsStatus === 'reconnecting' || wsStatus === 'connecting' || wsStatus === 'closed'
  const mediaBad = mediaStatus === 'reconnecting' || mediaStatus === 'disconnected'
  if (!wsBad && !mediaBad) return null

  let message = t('meeting.reconnecting')
  if (mediaStatus === 'disconnected' && (wsStatus === 'closed' || wsStatus === 'reconnecting')) {
    message = t('meeting.connectionLost')
  } else if (mediaStatus === 'reconnecting') {
    message = t('meeting.reconnectingMedia')
  } else if (wsStatus === 'reconnecting' || wsStatus === 'connecting') {
    message = t('meeting.reconnecting')
  } else if (wsStatus === 'closed' || mediaStatus === 'disconnected') {
    message = t('meeting.connectionLost')
  }

  return (
    <div className="connection-banner" role="status">
      <span>{message}</span>
      {(wsStatus === 'closed' || wsStatus === 'reconnecting' || mediaStatus === 'disconnected') && (
        <button type="button" onClick={onRetryWs}>{t('common.retry')}</button>
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

export default function MeetingPage({ user }: { user: User }) {
  const { id } = useParams()
  const navigate = useNavigate()
  const { t, i18n } = useTranslation()
  const [join, setJoin] = useState<JoinResponse | null>(null)
  const [error, setError] = useState('')
  const [panel, setPanel] = useState<'people' | 'chat' | 'settings' | null>(null)
  const [message, setMessage] = useState('')
  const [chatTarget, setChatTarget] = useState('everyone')
  const [copied, setCopied] = useState(false)
  const [ready, setReady] = useState(false)
  const [mediaStatus, setMediaStatus] = useState<'connected' | 'reconnecting' | 'disconnected'>('connected')
  const intentionalLeave = useRef(false)

  const refreshJoin = useCallback(async () => {
    if (!id) return
    try {
      setJoin(await api.join(id))
    } catch {
      setError('meeting.unableToJoin')
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
      setError('meeting.hostDeclined')
      return
    }
    if ((event.type === 'meeting.updated' || event.type.startsWith('participant.')) && event.payload) {
      const snapshot = event.payload as Meeting
      setJoin(current => {
        if (!current) return current
        // Chat is append-only for the meeting lifetime. Snapshots can arrive out of
        // order relative to chat.message, so keep every message we already know.
        const byId = new Map<string, ChatMessage>()
        for (const m of current.meeting.chat) byId.set(m.id, m)
        for (const m of snapshot.chat ?? []) byId.set(m.id, m)
        const chat = [...byId.values()].sort((a, b) => a.sentAt.localeCompare(b.sentAt))
        return {
          ...current,
          meeting: { ...snapshot, isHost: snapshot.hostId === user.id, chat },
        }
      })
    }
    if (event.type === 'chat.message' && event.payload) {
      const incoming = event.payload as ChatMessage
      setJoin(current => {
        if (!current) return current
        if (current.meeting.chat.some(m => m.id === incoming.id)) return current
        return {
          ...current,
          meeting: { ...current.meeting, chat: [...current.meeting.chat, incoming] },
        }
      })
    }
  }, [intentionalLeave, navigate, refreshJoin, user.id])

  const { status: wsStatus, retryNow } = useMeetingSocket(join ? id : undefined, onSocket)

  useEffect(() => {
    if (chatTarget !== 'everyone' && join && !join.meeting.participants[chatTarget]) {
      setChatTarget('everyone')
    }
  }, [chatTarget, join])

  const leave = async () => {
    intentionalLeave.current = true
    if (id) await api.leave(id).catch(() => {})
    navigate('/')
  }
  const end = async () => {
    if (id && confirm(t('meeting.endConfirm'))) {
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
    await api.chat(id, text, chatTarget === 'everyone' ? undefined : chatTarget)
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
        <h2>{t('meeting.enterFailed')}</h2>
        <p>{t(error)}</p>
        <button className="button primary" onClick={() => navigate('/')}>{t('meeting.backHome')}</button>
      </div>
    )
  }
  if (!join) {
    return (
      <div className="center-state">
        <div className="pulse-ring" />
        <h2>{t('meeting.findingRoom')}</h2>
      </div>
    )
  }

  if (join.status === 'waiting') {
    return (
      <div className="waiting-page with-prejoin">
        <div className="standalone-language"><LanguageSwitcher /></div>
        <div className="waiting-visual">
          <div className="avatar-large">{user.avatar ? <img src={user.avatar} alt="" /> : user.displayName[0]}</div>
          <span className="orbit one" />
          <span className="orbit two" />
        </div>
        <h1>{t('meeting.readyToJoin')}</h1>
        <p>{t('meeting.hostNotified')}</p>
        <div className="waiting-code">
          <span>{id}</span>
          <button type="button" onClick={copy}>{copied ? <Check /> : <Copy />}</button>
        </div>
        <PreJoin
          meetingId={id!}
          waiting
          title={t('meeting.setupDevices')}
          subtitle={t('meeting.setupDevicesHint')}
          onLeave={leave}
        />
        <ConnectionBanner wsStatus={wsStatus} mediaStatus="connected" onRetryWs={retryNow} />
      </div>
    )
  }

  if (!ready) {
    return (
      <div className="prejoin-page">
        <div className="standalone-language"><LanguageSwitcher /></div>
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
          <div className="meeting-title">
            <span className="live-dot" /> {t('meeting.live')} <span>·</span> {id}
          </div>
          <div className="meeting-header-actions">
            <LanguageSwitcher compact />
            <button className="icon-button" onClick={copy} title={t('meeting.copyMeetingLink')} type="button">
              {copied ? <Check /> : <Copy />}
            </button>
          </div>
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
                {panel === 'people' && t('meeting.peopleCount', { count: people.length })}
                {panel === 'chat' && t('meeting.messages')}
                {panel === 'settings' && t('common.settings')}
              </h2>
              <button type="button" onClick={() => setPanel(null)}><X /></button>
            </div>
            {panel === 'people' && (
              <div className="people-list">
                {join.meeting.isHost && Object.values(join.meeting.waitingRoom).length > 0 && (
                  <section>
                    <h3>{t('meeting.waitingToJoin')}</h3>
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
                  <h3>{t('meeting.inMeeting')}</h3>
                  {people.map(p => (
                    <div className="person" key={p.userId}>
                      <Avatar name={p.displayName} src={p.avatar} />
                      <span>
                        {p.displayName}{p.userId === user.id && ` (${t('meeting.you')})`}
                        <small>{p.isHost && <><Shield /> {t('meeting.host')}</>}</small>
                      </span>
                      {!p.micEnabled && <MicOff className="muted-icon" />}
                      {join.meeting.isHost && !p.isHost && (
                        <div className="person-actions">
                          <button type="button" onClick={() => api.action(id!, 'mute', p.userId)} title={t('common.mute')}><MicOff /></button>
                          <button type="button" onClick={() => api.action(id!, 'remove', p.userId)} title={t('meeting.remove')}><LogOut /></button>
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
                      <p>{t(chatTarget === 'everyone' ? 'meeting.emptyChat' : 'meeting.emptyPrivateChat')}</p>
                    </div>
                  ) : (
                    join.meeting.chat.map(m => (
                      <div className="message" key={m.id}>
                        {m.recipientId && (
                          <span className="message-private">
                            {m.userId === user.id
                              ? t('meeting.privatelyTo', { name: m.recipientName || m.recipientId })
                              : t('meeting.privatelyFrom', { name: m.displayName })}
                          </span>
                        )}
                        <strong>
                          {m.displayName}
                          <time>{new Date(m.sentAt).toLocaleTimeString(i18n.language, { hour: '2-digit', minute: '2-digit' })}</time>
                        </strong>
                        <p>{m.text}</p>
                      </div>
                    ))
                  )}
                </div>
                <div className="chat-compose">
                  <div className="chat-recipient">
                    <label htmlFor="chat-target">{t('meeting.chatTo')}</label>
                    <select
                      id="chat-target"
                      value={chatTarget}
                      onChange={e => setChatTarget(e.target.value)}
                    >
                      <option value="everyone">{t('meeting.chatEveryone')}</option>
                      {people.filter(p => p.userId !== user.id).map(p => (
                        <option key={p.userId} value={p.userId}>{p.displayName}</option>
                      ))}
                    </select>
                  </div>
                  <form className="chat-form" onSubmit={send}>
                    <input maxLength={1000} placeholder={t('meeting.sendMessage')} value={message} onChange={e => setMessage(e.target.value)} />
                    <button type="submit" disabled={!message.trim()}><Send /></button>
                  </form>
                </div>
              </>
            )}
            {panel === 'settings' && <SettingsPanel meetingId={id!} />}
          </aside>
        )}
        {mediaStatus === 'disconnected' && (
          <div className="media-recover">
            <h2>{t('meeting.mediaDisconnected')}</h2>
            <p>{t('meeting.mediaDisconnectedHint')}</p>
            <button
              type="button"
              className="button primary"
              onClick={() => {
                setMediaStatus('reconnecting')
                setReady(false)
                requestAnimationFrame(() => setReady(true))
              }}
            >
              {t('meeting.reconnectMedia')}
            </button>
            <button type="button" className="button ghost" onClick={leave}>{t('meeting.leaveMeeting')}</button>
          </div>
        )}
        <RoomAudioRenderer />
        <StartAudio label={t('meeting.enableAudio')} />
      </div>
    </LiveKitRoom>
  )
}

function SettingsPanel({ meetingId }: { meetingId: string }) {
  const { t } = useTranslation()
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
      <p className="settings-copy">{t('meeting.deviceChoices')}</p>
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
          <span>{media.prefs.micEnabled ? t('meeting.micOn') : t('meeting.micOff')}</span>
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
          <span>{media.prefs.cameraEnabled ? t('meeting.cameraOn') : t('meeting.cameraOff')}</span>
        </button>
      </div>
      {media.permissionError && <p className="prejoin-warn">{t(media.permissionError)}</p>}
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
  const { t, i18n } = useTranslation()
  const prefs = loadMediaPrefs(id)
  const [mic, setMic] = useState(prefs.micEnabled)
  const [camera, setCamera] = useState(prefs.cameraEnabled)
  const [screen, setScreen] = useState(false)
  const [now, setNow] = useState(() => new Date())
  const { localParticipant } = useLocalParticipant()

  useEffect(() => {
    const timer = window.setInterval(() => setNow(new Date()), 1000)
    return () => window.clearInterval(timer)
  }, [])

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
      <div className="meeting-clock">{now.toLocaleTimeString(i18n.language, { hour: '2-digit', minute: '2-digit' })}</div>
      <div className="control-center">
        <button type="button" className={!mic ? 'off' : ''} onClick={() => toggle('mic')} title={mic ? t('common.mute') : t('common.unmute')}>
          {mic ? <Mic /> : <MicOff />}
        </button>
        <button type="button" className={!camera ? 'off' : ''} onClick={() => toggle('camera')} title={camera ? t('common.cameraOff') : t('common.cameraOn')}>
          {camera ? <Video /> : <VideoOff />}
        </button>
        <button type="button" className={screen ? 'active' : ''} onClick={() => toggle('screen')} title={t('meeting.shareScreen')}>
          <MonitorUp />
        </button>
        <button type="button" className="end-call" onClick={meeting.isHost ? end : leave} title={meeting.isHost ? t('meeting.endForEveryone') : t('common.leave')}>
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
          title={t('common.settings')}
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
