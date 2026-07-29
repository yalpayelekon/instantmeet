import { useEffect, useRef } from 'react'
import { Mic, MicOff, Video, VideoOff } from 'lucide-react'
import { DeviceSelects } from './DeviceSelects'
import { type MediaPrefs, useMediaDevices } from '../hooks/useMediaDevices'

type Props = {
  meetingId: string
  title?: string
  subtitle?: string
  primaryLabel?: string
  onJoin?: () => void
  onLeave: () => void
  /** When true, hide Join CTA (waiting room strip). */
  waiting?: boolean
}

export function PreJoin({
  meetingId,
  title = 'Ready to join?',
  subtitle = 'Check your camera and mic before you enter the room.',
  primaryLabel = 'Join now',
  onJoin,
  onLeave,
  waiting,
}: Props) {
  const media = useMediaDevices(meetingId)
  const videoRef = useRef<HTMLVideoElement>(null)
  const streamRef = useRef<MediaStream | null>(null)

  useEffect(() => {
    let cancelled = false
    const start = async () => {
      streamRef.current?.getTracks().forEach(t => t.stop())
      streamRef.current = null
      if (!media.ready) return
      const constraints: MediaStreamConstraints = {
        audio: media.prefs.micEnabled
          ? (media.prefs.audioDeviceId ? { deviceId: { exact: media.prefs.audioDeviceId } } : true)
          : false,
        video: media.prefs.cameraEnabled
          ? (media.prefs.videoDeviceId ? { deviceId: { exact: media.prefs.videoDeviceId } } : true)
          : false,
      }
      if (!constraints.audio && !constraints.video) {
        if (videoRef.current) videoRef.current.srcObject = null
        return
      }
      try {
        const stream = await navigator.mediaDevices.getUserMedia(constraints)
        if (cancelled) {
          stream.getTracks().forEach(t => t.stop())
          return
        }
        streamRef.current = stream
        if (videoRef.current) {
          videoRef.current.srcObject = stream
          void videoRef.current.play().catch(() => {})
        }
      } catch {
        /* permission / device errors surfaced via media.permissionError */
      }
    }
    void start()
    return () => {
      cancelled = true
      streamRef.current?.getTracks().forEach(t => t.stop())
      streamRef.current = null
    }
  }, [
    media.ready,
    media.prefs.micEnabled,
    media.prefs.cameraEnabled,
    media.prefs.audioDeviceId,
    media.prefs.videoDeviceId,
  ])

  const toggle = (key: 'micEnabled' | 'cameraEnabled') => {
    media.setPrefs({ [key]: !media.prefs[key] } as Partial<MediaPrefs>)
  }

  return (
    <div className={`prejoin${waiting ? ' waiting-mode' : ''}`}>
      <div className="prejoin-preview">
        <video ref={videoRef} muted playsInline autoPlay className={media.prefs.cameraEnabled ? '' : 'hidden-video'} />
        {!media.prefs.cameraEnabled && (
          <div className="prejoin-placeholder">
            <VideoOff />
            <span>Camera is off</span>
          </div>
        )}
        <div className="prejoin-toggles">
          <button
            type="button"
            className={!media.prefs.micEnabled ? 'off' : ''}
            onClick={() => toggle('micEnabled')}
            title={media.prefs.micEnabled ? 'Mute' : 'Unmute'}
          >
            {media.prefs.micEnabled ? <Mic /> : <MicOff />}
          </button>
          <button
            type="button"
            className={!media.prefs.cameraEnabled ? 'off' : ''}
            onClick={() => toggle('cameraEnabled')}
            title={media.prefs.cameraEnabled ? 'Turn camera off' : 'Turn camera on'}
          >
            {media.prefs.cameraEnabled ? <Video /> : <VideoOff />}
          </button>
        </div>
        {media.prefs.micEnabled && <span className="mic-live">Mic on</span>}
      </div>
      <div className="prejoin-copy">
        <h1>{title}</h1>
        <p>{subtitle}</p>
        {media.permissionError && <p className="prejoin-warn">{media.permissionError}</p>}
        <DeviceSelects
          audioInputs={media.audioInputs}
          videoInputs={media.videoInputs}
          audioOutputs={media.audioOutputs}
          prefs={media.prefs}
          setPrefs={media.setPrefs}
          supportsOutput={media.supportsOutput}
          compact={waiting}
        />
        <div className="prejoin-actions">
          {!waiting && onJoin && (
            <button type="button" className="button primary" onClick={onJoin}>{primaryLabel}</button>
          )}
          <button type="button" className={waiting ? 'text-button' : 'button ghost'} onClick={onLeave}>
            {waiting ? 'Cancel request' : 'Leave'}
          </button>
        </div>
      </div>
    </div>
  )
}
