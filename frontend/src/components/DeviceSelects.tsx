import type { MediaPrefs } from '../hooks/useMediaDevices'

export type DeviceSelectProps = {
  audioInputs: MediaDeviceInfo[]
  videoInputs: MediaDeviceInfo[]
  audioOutputs: MediaDeviceInfo[]
  prefs: MediaPrefs
  setPrefs: (patch: Partial<MediaPrefs>) => void
  supportsOutput: boolean
  compact?: boolean
}

export function DeviceSelects({
  audioInputs, videoInputs, audioOutputs, prefs, setPrefs, supportsOutput, compact,
}: DeviceSelectProps) {
  return (
    <div className={compact ? 'device-selects compact' : 'device-selects'}>
      <label>
        <span>Microphone</span>
        <select
          aria-label="Microphone"
          value={prefs.audioDeviceId}
          onChange={e => setPrefs({ audioDeviceId: e.target.value })}
        >
          {audioInputs.length === 0 && <option value="">No microphone found</option>}
          {audioInputs.map(d => <option key={d.deviceId} value={d.deviceId}>{d.label || 'Microphone'}</option>)}
        </select>
      </label>
      <label>
        <span>Camera</span>
        <select
          aria-label="Camera"
          value={prefs.videoDeviceId}
          onChange={e => setPrefs({ videoDeviceId: e.target.value })}
        >
          {videoInputs.length === 0 && <option value="">No camera found</option>}
          {videoInputs.map(d => <option key={d.deviceId} value={d.deviceId}>{d.label || 'Camera'}</option>)}
        </select>
      </label>
      {supportsOutput && (
        <label>
          <span>Speaker</span>
          <select
            aria-label="Speaker"
            value={prefs.outputDeviceId}
            onChange={e => setPrefs({ outputDeviceId: e.target.value })}
          >
            {audioOutputs.length === 0 && <option value="">Default speaker</option>}
            {audioOutputs.map(d => <option key={d.deviceId} value={d.deviceId}>{d.label || 'Speaker'}</option>)}
          </select>
        </label>
      )}
    </div>
  )
}
