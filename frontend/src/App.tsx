import { lazy, Suspense } from 'react'
import { Navigate, Route, Routes } from 'react-router-dom'
import { useAuth } from './hooks/useAuth'
import { HomePage } from './pages/HomePage'

const MeetingPage = lazy(() => import('./pages/MeetingPage'))

function Loader() {
  return <div className="app-loader"><span className="brand-mark">I</span></div>
}

export default function App() {
  const auth = useAuth()
  if (auth.loading) return <Loader />
  return (
    <Suspense fallback={<Loader />}>
      <Routes>
        <Route path="/" element={<HomePage user={auth.user} onLogout={auth.logout} />} />
        <Route path="/meet/:id" element={auth.user ? <MeetingPage user={auth.user} /> : <Navigate to="/" replace />} />
        <Route path="*" element={<Navigate to="/" replace />} />
      </Routes>
    </Suspense>
  )
}
