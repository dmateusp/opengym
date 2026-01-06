import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Button } from '@/components/ui/button'
import CreateGameModal from '@/components/games/CreateGameModal'
import GamesList from '@/components/games/GamesList'
import UserProfileMenu from '@/components/auth/UserProfileMenu'
import { LanguageSwitcher } from '@/components/LanguageSwitcher'
import { useCurrentUser } from '@/hooks/useCurrentUser'
import { API_BASE_URL } from '@/lib/api'
import { fetchWithDemoRecovery } from '@/lib/fetchWithDemoRecovery'

export default function HomePage() {
  const { t } = useTranslation()
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
    <div className="min-h-screen bg-gradient-to-br from-orange-50 via-yellow-50 to-blue-50">
      <div className="container mx-auto px-4 py-8 max-w-5xl">
        <div className="flex items-center justify-between mb-12">
          <div>
            <Button
              size="lg"
              onClick={() => setIsModalOpen(true)}
              className="bg-accent hover:bg-accent/90"
            >
              {t('home.organizeGame')}
            </Button>
          </div>
          <div className="flex items-center gap-4">
            <LanguageSwitcher />
            <UserProfileMenu user={user} onUserChange={handleUserChange} />
          </div>
        </div>

        {hasAnyGames ? (
          <GamesList />
        ) : (
          <div className="flex flex-col items-center justify-center py-24">
            <div className="text-center max-w-md">
              <div className="mb-6 text-6xl">🏐</div>
              <h2 className="text-3xl font-bold text-gray-900 mb-3">{t('home.noGamesYet')}</h2>
              <p className="text-gray-600 mb-8">
                {t('home.noGamesDescription')}
              </p>
              <Button
                size="lg"
                onClick={() => setIsModalOpen(true)}
                className="bg-accent hover:bg-accent/90 w-full"
              >
                {t('home.organizeGame')}
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
