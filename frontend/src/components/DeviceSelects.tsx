import type { MediaPrefs } from '../hooks/useMediaDevices'
import { useTranslation } from 'react-i18next'

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
  const { t } = useTranslation()
  return (
    <div className={compact ? 'device-selects compact' : 'device-selects'}>
      <label>
        <span>{t('devices.microphone')}</span>
        <select
          aria-label={t('devices.microphone')}
          value={prefs.audioDeviceId}
          onChange={e => setPrefs({ audioDeviceId: e.target.value })}
        >
          {audioInputs.length === 0 && <option value="">{t('devices.noMicrophone')}</option>}
          {audioInputs.map(d => <option key={d.deviceId} value={d.deviceId}>{d.label || t('devices.microphone')}</option>)}
        </select>
      </label>
      <label>
        <span>{t('devices.camera')}</span>
        <select
          aria-label={t('devices.camera')}
          value={prefs.videoDeviceId}
          onChange={e => setPrefs({ videoDeviceId: e.target.value })}
        >
          {videoInputs.length === 0 && <option value="">{t('devices.noCamera')}</option>}
          {videoInputs.map(d => <option key={d.deviceId} value={d.deviceId}>{d.label || t('devices.camera')}</option>)}
        </select>
      </label>
      {supportsOutput && (
        <label>
          <span>{t('devices.speaker')}</span>
          <select
            aria-label={t('devices.speaker')}
            value={prefs.outputDeviceId}
            onChange={e => setPrefs({ outputDeviceId: e.target.value })}
          >
            {audioOutputs.length === 0 && <option value="">{t('devices.defaultSpeaker')}</option>}
            {audioOutputs.map(d => <option key={d.deviceId} value={d.deviceId}>{d.label || t('devices.speaker')}</option>)}
          </select>
        </label>
      )}
    </div>
  )
}
