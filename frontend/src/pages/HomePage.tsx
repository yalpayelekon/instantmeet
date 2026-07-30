import { useState } from 'react'
import { ArrowRight, Check, Copy, Github, LogOut, Plus, ShieldCheck, Sparkles, Video } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { useNavigate } from 'react-router-dom'
import { LanguageSwitcher } from '../components/LanguageSwitcher'
import { api } from '../services/api'
import type { User } from '../types'

export function HomePage({ user, onLogout }: { user: User|null; onLogout: () => Promise<void> }) {
  const navigate = useNavigate()
  const { t } = useTranslation()
  const [meetingCode, setMeetingCode] = useState('')
  const [creating, setCreating] = useState(false)
  const [copied, setCopied] = useState(false)
  const [createdUrl, setCreatedUrl] = useState('')

  const create = async () => {
    setCreating(true)
    try {
      const result = await api.createMeeting()
      setCreatedUrl(`${location.origin}${result.url}`)
    } finally {
      setCreating(false)
    }
  }
  const join = () => {
    const id = meetingCode.trim().split('/').filter(Boolean).at(-1)
    if (id) navigate(`/meet/${id}`)
  }
  const copy = async () => {
    await navigator.clipboard.writeText(createdUrl)
    setCopied(true)
    setTimeout(() => setCopied(false), 1500)
  }

  return <main className="home">
    <nav className="nav shell">
      <a className="brand" href="/"><span className="brand-mark">I</span><span>InstantMeet</span></a>
      <div className="nav-actions">
        <a className="github-link" href="https://github.com/yalpayelekon/instantmeet" target="_blank" rel="noreferrer"><Github size={18}/> {t('home.openSource')}</a>
        <LanguageSwitcher compact />
        {user && <button className="sign-out-button" onClick={onLogout} title={t('home.signOut')} aria-label={t('home.signOut')}>
          <LogOut size={17}/>
        </button>}
      </div>
    </nav>

    <section className="hero shell">
      <div className="eyebrow"><Sparkles size={14}/> {t('home.eyebrow')}</div>
      <h1>{t('home.headline')}<br/><span>{t('home.headlineAccent')}</span></h1>
      <p className="hero-copy">{t('home.description')}</p>

      {!user ? <div className="login-card">
        <a className="button primary google" href="/api/login/google">
          <span className="google-g">G</span> {t('home.continueGoogle')} <ArrowRight size={18}/>
        </a>
        {import.meta.env.DEV && <a className="dev-link" href="/api/login/dev">{t('home.localDemo')}</a>}
        <small>{t('home.accountNote')}</small>
      </div> : <div className="launch-panel">
        {!createdUrl ? <>
          <button className="button primary" onClick={create} disabled={creating}><Plus size={19}/>{creating ? t('home.openingRoom') : t('home.newMeeting')}</button>
          <div className="join-box">
            <Video size={19}/>
            <input aria-label={t('home.meetingCode')} placeholder={t('home.codePlaceholder')} value={meetingCode} onChange={e => setMeetingCode(e.target.value)} onKeyDown={e => e.key === 'Enter' && join()}/>
            <button onClick={join}>{t('home.join')}</button>
          </div>
        </> : <div className="created-box">
          <div><span>{t('home.roomReady')}</span><strong>{createdUrl.split('/').at(-1)}</strong></div>
          <button className="button primary" onClick={copy}>{copied ? <Check size={18}/> : <Copy size={18}/>} {copied ? t('home.copied') : t('home.copyLink')}</button>
          <button className="button ghost" onClick={() => navigate(new URL(createdUrl).pathname)}>{t('home.enterRoom')} <ArrowRight size={18}/></button>
        </div>}
      </div>}
    </section>

    <section className="promise shell">
      <div><ShieldCheck/><span><strong>{t('home.simpleTitle')}</strong>{t('home.simpleDescription')}</span></div>
      <div className="stats"><span><strong>100</strong> {t('home.people')}</span><span><strong>0</strong> {t('home.minuteLimit')}</span><span><strong>0</strong> {t('home.recordings')}</span></div>
    </section>
    <footer className="shell">{t('home.footer')}<span>WebRTC · LiveKit · Go</span></footer>
  </main>
}
