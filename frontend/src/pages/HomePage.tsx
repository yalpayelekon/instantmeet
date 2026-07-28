import { useState } from 'react'
import { ArrowRight, Check, Copy, Github, LogOut, Plus, ShieldCheck, Sparkles, Video } from 'lucide-react'
import { useNavigate } from 'react-router-dom'
import { api } from '../services/api'
import type { User } from '../types'

export function HomePage({ user, onLogout }: { user: User|null; onLogout: () => Promise<void> }) {
  const navigate = useNavigate()
  const [meetingCode, setMeetingCode] = useState('')
  const [creating, setCreating] = useState(false)
  const [copied, setCopied] = useState(false)
  const [createdUrl, setCreatedUrl] = useState('')

  const create = async () => {
    setCreating(true)
    try {
      const result = await api.createMeeting()
      const url = `${location.origin}${result.url}`
      setCreatedUrl(url)
    } finally { setCreating(false) }
  }
  const join = () => {
    const id = meetingCode.trim().split('/').filter(Boolean).at(-1)
    if (id) navigate(`/meet/${id}`)
  }
  const copy = async () => { await navigator.clipboard.writeText(createdUrl); setCopied(true); setTimeout(() => setCopied(false), 1500) }

  return <main className="home">
    <nav className="nav shell">
      <a className="brand" href="/"><span className="brand-mark">I</span><span>InstantMeet</span></a>
      <div className="nav-actions">
        <a className="github-link" href="https://github.com" target="_blank" rel="noreferrer"><Github size={18}/> Open source</a>
        {user && <button className="avatar-button" onClick={onLogout} title="Sign out">
          {user.avatar ? <img src={user.avatar} alt="" /> : user.displayName.slice(0,1)}
          <LogOut size={15}/>
        </button>}
      </div>
    </nav>

    <section className="hero shell">
      <div className="eyebrow"><Sparkles size={14}/> Meetings without meters</div>
      <h1>Meet now.<br/><span>Stay as long as you like.</span></h1>
      <p className="hero-copy">A fast, private video room that disappears when you’re done. No schedules, no recordings, no surprise paywalls.</p>

      {!user ? <div className="login-card">
        <a className="button primary google" href="/api/login/google">
          <span className="google-g">G</span> Continue with Google <ArrowRight size={18}/>
        </a>
        {import.meta.env.DEV && <a className="dev-link" href="/api/login/dev">Use local demo account</a>}
        <small>One account. Nothing else stored.</small>
      </div> : <div className="launch-panel">
        {!createdUrl ? <>
          <button className="button primary" onClick={create} disabled={creating}><Plus size={19}/>{creating ? 'Opening room…' : 'New meeting'}</button>
          <div className="join-box"><Video size={19}/><input aria-label="Meeting code" placeholder="Enter a code or link" value={meetingCode} onChange={e => setMeetingCode(e.target.value)} onKeyDown={e => e.key === 'Enter' && join()}/><button onClick={join}>Join</button></div>
        </> : <div className="created-box">
          <div><span>Room ready</span><strong>{createdUrl.split('/').at(-1)}</strong></div>
          <button className="button primary" onClick={copy}>{copied ? <Check size={18}/> : <Copy size={18}/>} {copied ? 'Copied' : 'Copy link'}</button>
          <button className="button ghost" onClick={() => navigate(new URL(createdUrl).pathname)}>Enter room <ArrowRight size={18}/></button>
        </div>}
      </div>}
    </section>

    <section className="promise shell">
      <div><ShieldCheck/><span><strong>Ephemeral by design</strong>Room, chat, and participant state vanish when the meeting ends.</span></div>
      <div className="stats"><span><strong>100</strong> people</span><span><strong>0</strong> minute limit</span><span><strong>0</strong> recordings</span></div>
    </section>
    <footer className="shell">Built in the open for conversations that belong to you.<span>WebRTC · LiveKit · Go</span></footer>
  </main>
}
