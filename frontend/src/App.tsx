import { Navigate, Route, Routes } from 'react-router-dom'
import { useAuth } from './hooks/useAuth'
import { HomePage } from './pages/HomePage'
import { MeetingPage } from './pages/MeetingPage'

export default function App() {
  const auth = useAuth()
  if (auth.loading) return <div className="app-loader"><span className="brand-mark">I</span></div>
  return (
    <Routes>
      <Route path="/" element={<HomePage user={auth.user} onLogout={auth.logout} />} />
      <Route path="/meet/:id" element={auth.user ? <MeetingPage user={auth.user} /> : <Navigate to="/" replace />} />
      <Route path="*" element={<Navigate to="/" replace />} />
    </Routes>
  )
}

