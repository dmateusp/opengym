import { lazy, Suspense } from 'react'
import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom'

// Lazy load pages for code splitting
const HomePage = lazy(() => import('@/pages/HomePage'))
const GameDetailPage = lazy(() => import('@/pages/GameDetailPage'))
const LoginPage = lazy(() => import('@/pages/LoginPage'))

function App() {
  return (
    <BrowserRouter>
      <Suspense fallback={<div className="flex items-center justify-center min-h-screen">Loading...</div>}>
        <Routes>
          <Route path="/" element={<HomePage />} />
          <Route path="/auth/login" element={<LoginPage />} />
          <Route path="/games/:id" element={<GameDetailPage />} />
          <Route path="*" element={<Navigate to="/" replace />} />
        </Routes>
      </Suspense>
    </BrowserRouter>
  )
}

export default App
