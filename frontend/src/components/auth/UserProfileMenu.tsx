import { useEffect, useState } from 'react'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import { Avatar, AvatarFallback, AvatarImage } from '@/components/ui/avatar'
import { API_BASE_URL } from '@/lib/api'
import type { User } from '@/opengym/client'

interface UserProfileMenuProps {
  user: User | null
  onUserChange?: (user: User) => void
}

export default function UserProfileMenu({ user, onUserChange }: UserProfileMenuProps) {
  const [isDemoMode, setIsDemoMode] = useState(false)
  const [demoUsers, setDemoUsers] = useState<User[]>([])
  const [isLoadingDemo, setIsLoadingDemo] = useState(true)

  useEffect(() => {
    // Check if demo mode is available
    const checkDemoMode = async () => {
      try {
        const response = await fetch(`${API_BASE_URL}/api/demo/users`, {
          credentials: 'include',
        })
        
        if (response.ok) {
          const users = await response.json()
          setDemoUsers(users)
          setIsDemoMode(true)
          
          // Auto-impersonate first user if not already logged in and haven't already tried
          if (!user && users.length > 0 && !sessionStorage.getItem('demo_auto_impersonate_attempted')) {
            sessionStorage.setItem('demo_auto_impersonate_attempted', 'true')
            await impersonateUser(users[0].id)
          }
        }
      } catch (error) {
        // Demo mode not available or error occurred
        console.debug('Demo mode not available:', error)
      } finally {
        setIsLoadingDemo(false)
      }
    }

    checkDemoMode()
  }, [])

  const impersonateUser = async (userId: string) => {
    try {
      const response = await fetch(
        `${API_BASE_URL}/api/demo/users/${userId}/impersonate`,
        {
          method: 'POST',
          credentials: 'include',
        }
      )

      if (response.ok) {
        const impersonatedUser = await response.json()
        onUserChange?.(impersonatedUser)
        // Don't reload - just update the user state and let React handle the re-render
      }
    } catch (error) {
      console.error('Failed to impersonate user:', error)
    }
  }

  const getInitials = (name?: string, email?: string) => {
    if (name) {
      const parts = name.split(' ').filter(p => p.length > 0)
      if (parts.length > 0) {
        return parts.slice(0, 3).map(p => p[0].toUpperCase()).join('')
      }
      return name.slice(0, 2).toUpperCase()
    }
    if (email) {
      return email.slice(0, 2).toUpperCase()
    }
    return '??'
  }

  const handleLogout = async () => {
    try {
      const response = await fetch(`${API_BASE_URL}/api/auth/logout`, {
        method: 'POST',
        credentials: 'include',
      })

      if (response.ok) {
        // Redirect to home page - the backend has invalidated the session cookie
        window.location.href = '/'
      } else {
        console.error('Logout failed:', response.statusText)
      }
    } catch (error) {
      console.error('Logout error:', error)
    }
  }

  if (isLoadingDemo || !user) {
    return null
  }

  return (
    <DropdownMenu>
      <DropdownMenuTrigger className="focus:outline-none">
        <div className="flex items-center gap-2 cursor-pointer hover:opacity-80 transition-opacity">
          <Avatar className="h-9 w-9">
            <AvatarImage src={user.picture} alt={user.name || user.email} />
            <AvatarFallback className="bg-primary text-white text-sm font-semibold">
              {getInitials(user.name, user.email)}
            </AvatarFallback>
          </Avatar>
          {user.name && (
            <span className="text-sm font-semibold text-gray-700 hidden sm:inline">
              {user.name}
            </span>
          )}
        </div>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end" className="w-56 rounded-xl">
        <DropdownMenuLabel>
          <div className="flex flex-col space-y-1">
            <p className="text-sm font-semibold">{user.name || 'User'}</p>
            <p className="text-xs text-gray-500">{user.email}</p>
            {user.isDemo && (
              <p className="text-xs text-accent font-semibold">Demo User</p>
            )}
          </div>
        </DropdownMenuLabel>
        
        {isDemoMode && demoUsers.length > 0 && (
          <>
            <DropdownMenuSeparator />
            <DropdownMenuLabel className="text-xs text-gray-500">
              Switch Demo User
            </DropdownMenuLabel>
            {demoUsers.map((demoUser) => (
              <DropdownMenuItem
                key={demoUser.id}
                onClick={() => impersonateUser(demoUser.id)}
                disabled={demoUser.id === user.id}
                className="cursor-pointer"
              >
                <div className="flex items-center gap-2 w-full">
                  <Avatar className="h-6 w-6">
                    <AvatarImage src={demoUser.picture} alt={demoUser.name || demoUser.email} />
                    <AvatarFallback className="bg-primary text-white text-xs">
                      {getInitials(demoUser.name, demoUser.email)}
                    </AvatarFallback>
                  </Avatar>
                  <div className="flex flex-col flex-1 min-w-0">
                    <span className="text-sm truncate">
                      {demoUser.name || demoUser.email}
                    </span>
                    {demoUser.id === user.id && (
                      <span className="text-xs text-primary font-semibold">Current</span>
                    )}
                  </div>
                </div>
              </DropdownMenuItem>
            ))}
          </>
        )}

        {!isDemoMode && (
          <>
            <DropdownMenuSeparator />
            <DropdownMenuItem
              onClick={handleLogout}
              className="cursor-pointer text-red-600"
            >
              Logout
            </DropdownMenuItem>
          </>
        )}
      </DropdownMenuContent>
    </DropdownMenu>
  )
}
