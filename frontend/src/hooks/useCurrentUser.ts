import { useEffect, useState } from 'react'
import { API_BASE_URL } from '@/lib/api'
import { fetchWithDemoRecovery } from '@/lib/fetchWithDemoRecovery'
import type { User } from '@/opengym/client'

export function useCurrentUser() {
  const [user, setUser] = useState<User | null>(null)
  const [isLoading, setIsLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    const fetchUser = async () => {
      try {
        const response = await fetchWithDemoRecovery(`${API_BASE_URL}/api/auth/me`, {
          credentials: 'include',
        })

        if (response.ok) {
          const userData = await response.json()
          setUser(userData)
        } else {
          setError('Not authenticated')
        }
      } catch (err) {
        setError('Failed to fetch user')
        console.error('Error fetching current user:', err)
      } finally {
        setIsLoading(false)
      }
    }

    fetchUser()
  }, [])

  return { user, isLoading, error, setUser }
}
