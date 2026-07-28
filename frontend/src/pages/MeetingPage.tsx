import { FormEvent, useCallback, useEffect, useMemo, useState } from 'react'
import { LiveKitRoom, ParticipantTile, RoomAudioRenderer, StartAudio, useLocalParticipant, useTracks } from '@livekit/components-react'
import { Track } from 'livekit-client'
import { Check, Copy, LogOut, MessageSquare, Mic, MicOff, MonitorUp, PhoneOff, Send, Settings, Shield, Users, Video, VideoOff, X } from 'lucide-react'
import { useNavigate, useParams } from 'react-router-dom'
import { useMeetingSocket, type SocketEvent } from '../hooks/useMeetingSocket'
import { api } from '../services/api'
import type { JoinResponse, Meeting, User } from '../types'

function VideoGrid() {
  const tracks = useTracks([{ source: Track.Source.Camera, withPlaceholder: true }, { source: Track.Source.ScreenShare, withPlaceholder: false }], { onlySubscribed: false })
  return <div className={`video-grid tiles-${Math.min(tracks.length, 9)}`}>{tracks.map(track => <ParticipantTile key={`${track.participant.identity}-${track.source}`} trackRef={track}/>)}</div>
}

export function MeetingPage({ user }: { user: User }) {
  const { id } = useParams(); const navigate = useNavigate()
  const [join, setJoin] = useState<JoinResponse|null>(null)
  const [error, setError] = useState('')
  const [panel, setPanel] = useState<'people'|'chat'|null>(null)
  const [message, setMessage] = useState('')
  const [copied, setCopied] = useState(false)

  const refreshJoin = useCallback(async () => {
    if (!id) return
    try { setJoin(await api.join(id)) } catch (e) { setError(e instanceof Error ? e.message : 'Unable to join') }
  }, [id])
  useEffect(() => { void refreshJoin() }, [refreshJoin])

  const onSocket = useCallback((event: SocketEvent) => {
    if (event.type === 'meeting.ended' || (event.type === 'participant.removed' && event.userId === user.id)) { navigate('/', { replace:true }); return }
    if (event.type === 'participant.muted' && event.userId === user.id) window.dispatchEvent(new Event('instantmeet:mute'))
    if (event.type === 'participant.admitted' && event.userId === user.id) { void refreshJoin(); return }
    if (event.type === 'participant.rejected' && event.userId === user.id) { setError('The host declined your request.'); return }
    if ((event.type === 'meeting.updated' || event.type.startsWith('participant.')) && event.payload) {
      const meeting = event.payload as Meeting
      setJoin(current => current ? { ...current, meeting } : current)
    }
    if (event.type === 'chat.message' && event.payload) setJoin(current => current ? {...current, meeting:{...current.meeting,chat:[...current.meeting.chat,event.payload as Meeting['chat'][number]]}} : current)
  }, [navigate, refreshJoin, user.id])
  useMeetingSocket(id, onSocket)

  const send = async (e: FormEvent) => { e.preventDefault(); if (!id || !message.trim()) return; const text=message; setMessage(''); await api.chat(id,text) }
  const leave = async () => { if (id) await api.leave(id).catch(()=>{}); navigate('/') }
  const end = async () => { if (id && confirm('End this meeting for everyone?')) { await api.end(id); navigate('/') } }
  const copy = async () => { await navigator.clipboard.writeText(location.href); setCopied(true); setTimeout(()=>setCopied(false),1500) }
  const people = useMemo(() => join ? Object.values(join.meeting.participants) : [], [join])

  if (error) return <div className="center-state"><span className="brand-mark">I</span><h2>Couldn’t enter this room</h2><p>{error}</p><button className="button primary" onClick={()=>navigate('/')}>Back home</button></div>
  if (!join) return <div className="center-state"><div className="pulse-ring"/><h2>Finding your room…</h2></div>
  if (join.status === 'waiting') return <div className="waiting-page">
    <div className="waiting-visual"><div className="avatar-large">{user.avatar ? <img src={user.avatar} alt=""/> : user.displayName[0]}</div><span className="orbit one"/><span className="orbit two"/></div>
    <h1>You’re ready to join</h1><p>The host knows you’re here. We’ll bring you in as soon as they approve your request.</p>
    <div className="waiting-code"><span>{id}</span><button onClick={copy}>{copied?<Check/>:<Copy/>}</button></div>
    <button className="text-button" onClick={leave}>Cancel request</button>
  </div>

  return <LiveKitRoom token={join.livekitToken} serverUrl={join.livekitUrl} connect audio video data-lk-theme="default" onDisconnected={()=>navigate('/')}>
    <div className="meeting-shell">
      <header className="meeting-header"><a className="brand compact" href="/"><span className="brand-mark">I</span><span>InstantMeet</span></a><div className="meeting-title"><span className="live-dot"/> Live <span>·</span> {id}</div><button className="icon-button" onClick={copy} title="Copy meeting link">{copied?<Check/>:<Copy/>}</button></header>
      <VideoGrid/>
      <MeetingControls meeting={join.meeting} id={id!} panel={panel} setPanel={setPanel} leave={leave} end={end}/>
      {panel && <aside className="side-panel">
        <div className="panel-head"><h2>{panel==='people' ? `People (${people.length})` : 'In-call messages'}</h2><button onClick={()=>setPanel(null)}><X/></button></div>
        {panel==='people' ? <div className="people-list">
          {join.meeting.isHost && Object.values(join.meeting.waitingRoom).length>0 && <section><h3>Waiting to join</h3>{Object.values(join.meeting.waitingRoom).map(p=><div className="person" key={p.userId}><Avatar name={p.displayName} src={p.avatar}/><span>{p.displayName}</span><button className="accept" onClick={()=>api.action(id!,'admit',p.userId)}><Check/></button><button className="reject" onClick={()=>api.action(id!,'reject',p.userId)}><X/></button></div>)}</section>}
          <section><h3>In this meeting</h3>{people.map(p=><div className="person" key={p.userId}><Avatar name={p.displayName} src={p.avatar}/><span>{p.displayName}{p.userId===user.id && ' (you)'}<small>{p.isHost && <><Shield/> Host</>}</small></span>{!p.micEnabled && <MicOff className="muted-icon"/>}{join.meeting.isHost && !p.isHost && <div className="person-actions"><button onClick={()=>api.action(id!,'mute',p.userId)} title="Mute"><MicOff/></button><button onClick={()=>api.action(id!,'remove',p.userId)} title="Remove"><LogOut/></button></div>}</div>)}</section>
        </div> : <><div className="messages">{join.meeting.chat.length===0 ? <div className="empty-chat"><MessageSquare/><p>Messages are visible to everyone here and disappear when the meeting ends.</p></div> : join.meeting.chat.map(m=><div className="message" key={m.id}><strong>{m.displayName}<time>{new Date(m.sentAt).toLocaleTimeString([],{hour:'2-digit',minute:'2-digit'})}</time></strong><p>{m.text}</p></div>)}</div><form className="chat-form" onSubmit={send}><input maxLength={1000} placeholder="Send a message" value={message} onChange={e=>setMessage(e.target.value)}/><button disabled={!message.trim()}><Send/></button></form></>}
      </aside>}
      <RoomAudioRenderer/><StartAudio label="Enable meeting audio"/>
    </div>
  </LiveKitRoom>
}

function MeetingControls({meeting,id,panel,setPanel,leave,end}:{meeting:Meeting;id:string;panel:string|null;setPanel:(p:'people'|'chat'|null)=>void;leave:()=>void;end:()=>void}) {
  const [mic,setMic]=useState(true), [camera,setCamera]=useState(true), [screen,setScreen]=useState(false)
  const { localParticipant } = useLocalParticipant()
  useEffect(() => {
    const mute = () => { void localParticipant.setMicrophoneEnabled(false); setMic(false) }
    window.addEventListener('instantmeet:mute', mute)
    return () => window.removeEventListener('instantmeet:mute', mute)
  }, [localParticipant])
  const toggle = async (kind:'mic'|'camera'|'screen') => {
    if(kind==='mic'){const next=!mic; await localParticipant.setMicrophoneEnabled(next); setMic(next); await api.media(id,{mic:next})}
    if(kind==='camera'){const next=!camera; await localParticipant.setCameraEnabled(next); setCamera(next); await api.media(id,{camera:next})}
    if(kind==='screen'){const next=!screen; await localParticipant.setScreenShareEnabled(next); setScreen(next); await api.media(id,{screen:next})}
  }
  return <div className="controls">
    <div className="meeting-clock">{new Date().toLocaleTimeString([],{hour:'2-digit',minute:'2-digit'})}</div>
    <div className="control-center">
      <button className={!mic?'off':''} onClick={()=>toggle('mic')} title={mic?'Mute':'Unmute'}>{mic?<Mic/>:<MicOff/>}</button>
      <button className={!camera?'off':''} onClick={()=>toggle('camera')} title={camera?'Turn camera off':'Turn camera on'}>{camera?<Video/>:<VideoOff/>}</button>
      <button className={screen?'active':''} onClick={()=>toggle('screen')} title="Share screen"><MonitorUp/></button>
      <button className="end-call" onClick={meeting.isHost?end:leave} title={meeting.isHost?'End for everyone':'Leave'}><PhoneOff/></button>
    </div>
    <div className="control-right">
      <button onClick={()=>setPanel(panel==='people'?null:'people')} className={panel==='people'?'active':''}><Users/><span>{Object.keys(meeting.participants).length}</span>{Object.keys(meeting.waitingRoom).length>0&&<b>{Object.keys(meeting.waitingRoom).length}</b>}</button>
      <button onClick={()=>setPanel(panel==='chat'?null:'chat')} className={panel==='chat'?'active':''}><MessageSquare/></button>
      <button><Settings/></button>
    </div>
  </div>
}
function Avatar({name,src}:{name:string;src:string}) { return <span className="avatar">{src?<img src={src} alt=""/>:name[0]}</span> }
