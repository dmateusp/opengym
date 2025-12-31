import { useEffect, useState } from 'react'
import { Button } from '@/components/ui/button'
import CreateGameModal from '@/components/games/CreateGameModal'
import GamesList from '@/components/games/GamesList'
import UserProfileMenu from '@/components/auth/UserProfileMenu'
import { useCurrentUser } from '@/hooks/useCurrentUser'
import { API_BASE_URL } from '@/lib/api'
import { fetchWithDemoRecovery } from '@/lib/fetchWithDemoRecovery'

export default function HomePage() {
  const { user, setUser } = useCurrentUser()
  const [isModalOpen, setIsModalOpen] = useState(false)
  const [hasAnyGames, setHasAnyGames] = useState<boolean>(false)

  const handleUserChange = (newUser: any) => {
    setUser(newUser)
    // Refetch games when user changes
    checkGames()
  }

  const checkGames = async () => {
    try {
      const params = new URLSearchParams()
      params.set('page', '1')
      params.set('pageSize', '1')
      const resp = await fetchWithDemoRecovery(
        `${API_BASE_URL}/api/games?${params.toString()}`,
        {
          credentials: 'include',
        }
      )
      if (!resp.ok) return
      const data = await resp.json()
      setHasAnyGames((data?.items?.length || 0) > 0)
    } catch {
      // ignore
    }
  }

  useEffect(() => {
    // Quick check to decide whether to show empty state or list
    checkGames()
  }, [])

  return (
    <div className="min-h-screen bg-gradient-to-br from-blue-50 to-indigo-100">
      <div className="container mx-auto px-4 py-12 max-w-5xl">
        <div className="flex items-center justify-between mb-6">
          <h1 className="text-3xl font-bold text-gray-900">Games</h1>
          <div className="flex items-center gap-4">
            <Button
              size="sm"
              onClick={() => setIsModalOpen(true)}
              className="bg-indigo-600 hover:bg-indigo-700"
            >
              Organize a Game
            </Button>
            <UserProfileMenu user={user} onUserChange={handleUserChange} />
          </div>
        </div>

        {hasAnyGames ? (
          <GamesList />
        ) : (
          <div className="flex flex-col items-center justify-center py-20">
            <div className="text-center">
              <h2 className="text-2xl font-semibold text-gray-900 mb-3">No games yet 🏐</h2>
              <p className="text-gray-600 mb-6">
                You haven't organized any games yet. Ready to get started?
              </p>
              <Button
                size="lg"
                onClick={() => setIsModalOpen(true)}
                className="bg-indigo-600 hover:bg-indigo-700"
              >
                Organize a Game
              </Button>
            </div>
          </div>
        )}
      </div>

      <CreateGameModal
        isOpen={isModalOpen}
        onClose={() => setIsModalOpen(false)}
      />
    </div>
  )
}
