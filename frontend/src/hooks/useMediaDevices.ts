import { useCallback, useEffect, useState } from 'react'

export type MediaPrefs = {
  audioDeviceId: string
  videoDeviceId: string
  outputDeviceId: string
  micEnabled: boolean
  cameraEnabled: boolean
}

const defaultPrefs = (): MediaPrefs => ({
  audioDeviceId: '',
  videoDeviceId: '',
  outputDeviceId: '',
  micEnabled: true,
  cameraEnabled: true,
})

function storageKey(meetingId: string) {
  return `instantmeet:media:${meetingId}`
}

export function loadMediaPrefs(meetingId: string): MediaPrefs {
  try {
    const raw = sessionStorage.getItem(storageKey(meetingId))
    if (!raw) return defaultPrefs()
    return { ...defaultPrefs(), ...JSON.parse(raw) as Partial<MediaPrefs> }
  } catch {
    return defaultPrefs()
  }
}

export function saveMediaPrefs(meetingId: string, prefs: MediaPrefs) {
  try {
    sessionStorage.setItem(storageKey(meetingId), JSON.stringify(prefs))
  } catch {
    /* ignore quota / private mode */
  }
}

export function supportsAudioOutputSelect() {
  return typeof HTMLMediaElement !== 'undefined' && 'setSinkId' in HTMLMediaElement.prototype
}

export function useMediaDevices(meetingId: string) {
  const [prefs, setPrefsState] = useState<MediaPrefs>(() => loadMediaPrefs(meetingId))
  const [audioInputs, setAudioInputs] = useState<MediaDeviceInfo[]>([])
  const [videoInputs, setVideoInputs] = useState<MediaDeviceInfo[]>([])
  const [audioOutputs, setAudioOutputs] = useState<MediaDeviceInfo[]>([])
  const [permissionError, setPermissionError] = useState('')
  const [ready, setReady] = useState(false)

  const setPrefs = useCallback((patch: Partial<MediaPrefs> | ((prev: MediaPrefs) => MediaPrefs)) => {
    setPrefsState(prev => {
      const next = typeof patch === 'function' ? patch(prev) : { ...prev, ...patch }
      saveMediaPrefs(meetingId, next)
      return next
    })
  }, [meetingId])

  const refresh = useCallback(async () => {
    if (!navigator.mediaDevices?.enumerateDevices) {
      setPermissionError('devices.unavailable')
      return
    }
    try {
      const stream = await navigator.mediaDevices.getUserMedia({ audio: true, video: true })
      stream.getTracks().forEach(t => t.stop())
      setPermissionError('')
    } catch {
      try {
        const stream = await navigator.mediaDevices.getUserMedia({ audio: true })
        stream.getTracks().forEach(t => t.stop())
        setPermissionError('devices.cameraDenied')
      } catch {
        setPermissionError('devices.permissionRequired')
      }
    }
    const devices = await navigator.mediaDevices.enumerateDevices()
    const mics = devices.filter(d => d.kind === 'audioinput')
    const cams = devices.filter(d => d.kind === 'videoinput')
    const outs = devices.filter(d => d.kind === 'audiooutput')
    setAudioInputs(mics)
    setVideoInputs(cams)
    setAudioOutputs(outs)
    setPrefs(prev => ({
      ...prev,
      audioDeviceId: prev.audioDeviceId && mics.some(d => d.deviceId === prev.audioDeviceId) ? prev.audioDeviceId : (mics[0]?.deviceId ?? ''),
      videoDeviceId: prev.videoDeviceId && cams.some(d => d.deviceId === prev.videoDeviceId) ? prev.videoDeviceId : (cams[0]?.deviceId ?? ''),
      outputDeviceId: prev.outputDeviceId && outs.some(d => d.deviceId === prev.outputDeviceId) ? prev.outputDeviceId : (outs[0]?.deviceId ?? ''),
    }))
    setReady(true)
  }, [setPrefs])

  useEffect(() => {
    setPrefsState(loadMediaPrefs(meetingId))
    void refresh()
    const onChange = () => { void refresh() }
    navigator.mediaDevices?.addEventListener?.('devicechange', onChange)
    return () => navigator.mediaDevices?.removeEventListener?.('devicechange', onChange)
  }, [meetingId, refresh])

  return {
    prefs,
    setPrefs,
    audioInputs,
    videoInputs,
    audioOutputs,
    permissionError,
    ready,
    refresh,
    supportsOutput: supportsAudioOutputSelect(),
  }
}
