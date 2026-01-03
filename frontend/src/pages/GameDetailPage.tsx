import { useParams, useNavigate } from 'react-router-dom'
import { useEffect, useMemo, useRef, useState } from 'react'
import { Button } from '@/components/ui/button'
import { Card } from '@/components/ui/card'
import { API_BASE_URL, redirectToLogin } from '@/lib/api'
import { fetchWithDemoRecovery } from '@/lib/fetchWithDemoRecovery'
import { ArrowLeft, Loader2, CheckCircle2, Clock, Users, Crown, XCircle, Rocket, Circle as CircleDashed } from 'lucide-react'
import { Input } from '@/components/ui/input'
import { MarkdownRenderer } from '@/components/ui/MarkdownRenderer'
import { PriceDisplay } from '@/components/games/PriceDisplay'
import { Popover, PopoverContent, PopoverAnchor } from '@/components/ui/popover'
import { TimeDisplay } from '@/components/ui/TimeDisplay'
import UserProfileMenu from '@/components/auth/UserProfileMenu'

interface Game {
  id: string
  name: string
  organizerId: number
  description?: string
  location?: string
  startsAt?: string
  durationMinutes?: number
  maxPlayers?: number
  totalPriceCents?: number
  createdAt: string
  updatedAt: string
  publishedAt?: string | null
}

interface AuthUser {
  id: string
  email: string
  name?: string
  picture?: string
  isDemo: boolean
}

interface Participant {
  status: 'going' | 'not_going' | 'waitlisted'
  user: AuthUser
  createdAt: string
  updatedAt: string
}

export default function GameDetailPage() {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()
  
  // Helper function to get initials from name
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

  const [game, setGame] = useState<Game | null>(null)
  const [organizer, setOrganizer] = useState<AuthUser | null>(null)
  const [isLoading, setIsLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [user, setUser] = useState<AuthUser | null>(null)
  const [organizerHintOpen, setOrganizerHintOpen] = useState(false)
  const [isPublishing, setIsPublishing] = useState(false)
  const [publishError, setPublishError] = useState<string | null>(null)
  const [publishAtInput, setPublishAtInput] = useState('')
  const [nowTs, setNowTs] = useState(() => Date.now())
  const [isEditingSchedule, setIsEditingSchedule] = useState(false)

  // Participants state
  const [participants, setParticipants] = useState<Participant[]>([])
  const [isLoadingParticipants, setIsLoadingParticipants] = useState(false)
  const [participantsError, setParticipantsError] = useState<string | null>(null)
  const [isUpdatingParticipation, setIsUpdatingParticipation] = useState(false)

  // Editing state
  const [editingField, setEditingField] = useState<string | null>(null)
  const [editValue, setEditValue] = useState<string>('')

  // Autosave status per field
  const saveTimersRef = useRef<Record<string, number | undefined>>({})
  const inputRef = useRef<HTMLInputElement | HTMLTextAreaElement>(null)

  useEffect(() => {
    const fetchAll = async () => {
      try {
        setIsLoading(true)
        setError(null)

        const response = await fetchWithDemoRecovery(`${API_BASE_URL}/api/games/${id}`, {
          credentials: 'include',
        })

        if (!response.ok) {
          if (response.status === 401) {
            redirectToLogin()
            return
          }
          if (response.status === 404) {
            throw new Error('Game not found')
          }
          throw new Error('Failed to load game')
        }

        const gameData = await response.json()
        setGame(gameData.game)
        setOrganizer(gameData.organizer)

        // Fetch authenticated user (optional if unauthenticated)
        try {
          const meResp = await fetchWithDemoRecovery(`${API_BASE_URL}/api/auth/me`, {
            credentials: 'include',
          })
          if (meResp.ok) {
            const me = await meResp.json()
            setUser(me)
          }
        } catch {
          // ignore user fetch errors, treat as not logged in
        }
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Something went wrong')
      } finally {
        setIsLoading(false)
      }
    }

    if (id) {
      fetchAll()
    }
  }, [id])

  useEffect(() => {
    const timer = window.setInterval(() => setNowTs(Date.now()), 30000)
    return () => window.clearInterval(timer)
  }, [])

  const refreshGame = async () => {
    if (!id) return
    try {
      const response = await fetchWithDemoRecovery(`${API_BASE_URL}/api/games/${id}`, {
        credentials: 'include',
      })
      if (response.ok) {
        const gameData = await response.json()
        setGame(gameData.game)
        setOrganizer(gameData.organizer)
      }
    } catch (err) {
      console.error('Error refreshing game:', err)
    }
  }

  const fetchParticipants = async () => {
    if (!id || !game?.publishedAt) return
    try {
      setIsLoadingParticipants(true)
      setParticipantsError(null)
      const response = await fetchWithDemoRecovery(`${API_BASE_URL}/api/games/${id}/participants`, {
        credentials: 'include',
      })
      if (response.ok) {
        const data = await response.json()
        setParticipants(data)
      } else if (response.status === 401) {
        redirectToLogin()
      } else {
        throw new Error('Failed to load participants')
      }
    } catch (err) {
      setParticipantsError(err instanceof Error ? err.message : 'Failed to load participants')
    } finally {
      setIsLoadingParticipants(false)
    }
  }

  useEffect(() => {
    if (game?.publishedAt) {
      fetchParticipants()
    }
  }, [game?.publishedAt, id])

  const handleUserChange = (newUser: any) => {
    setUser(newUser)
    // Refetch game when user changes
    refreshGame()
    if (game?.publishedAt) {
      fetchParticipants()
    }
  }

  const updateParticipation = async (status: 'going' | 'not_going') => {
    if (!id || !user) return
    try {
      setIsUpdatingParticipation(true)
      const response = await fetchWithDemoRecovery(`${API_BASE_URL}/api/games/${id}/participants`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        credentials: 'include',
        body: JSON.stringify({ status }),
      })
      if (response.ok) {
        await fetchParticipants()
      } else if (response.status === 401) {
        redirectToLogin()
      } else {
        const txt = await response.text()
        throw new Error(txt || 'Failed to update participation')
      }
    } catch (err) {
      setParticipantsError(err instanceof Error ? err.message : 'Failed to update participation')
    } finally {
      setIsUpdatingParticipation(false)
    }
  }

  const isOrganizer = useMemo(() => {
    if (!game || !user) return false
    const userIdNum = Number.parseInt(user.id, 10)
    if (Number.isNaN(userIdNum)) return false
    return userIdNum === game.organizerId
  }, [game, user])

  const currentUserParticipation = useMemo(() => {
    if (!user) return null
    return participants.find(p => p.user.id === user.id)
  }, [participants, user])

  const participantCounts = useMemo(() => {
    const going = participants.filter(p => p.status === 'going').length
    const waitlisted = participants.filter(p => p.status === 'waitlisted').length
    const notGoing = participants.filter(p => p.status === 'not_going').length
    return { going, waitlisted, notGoing }
  }, [participants])

  // Check if all publish requirements are met
  const publishRequirements = useMemo(() => {
    if (!game) return []
    return [
      {
        label: 'Location set',
        met: !!game.location,
        field: 'location'
      },
      {
        label: 'Start time set',
        met: !!game.startsAt,
        field: 'startsAt'
      },
      {
        label: 'Duration set',
        met: typeof game.durationMinutes === 'number' && game.durationMinutes > 0,
        field: 'durationMinutes'
      },
      {
        label: 'Max players set',
        met: typeof game.maxPlayers === 'number' && game.maxPlayers > 0,
        field: 'maxPlayers'
      },
      {
        label: 'Pricing set',
        met: typeof game.totalPriceCents === 'number' && game.totalPriceCents >= 0,
        field: 'totalPriceCents'
      }
    ]
  }, [game])

  const canPublish = useMemo(() => {
    return publishRequirements.every(req => req.met)
  }, [publishRequirements])

  const publishedAtDate = useMemo(() => {
    if (!game?.publishedAt) return null
    const d = new Date(game.publishedAt)
    if (Number.isNaN(d.getTime())) return null
    return d
  }, [game?.publishedAt])

  const isScheduled = useMemo(() => {
    if (!publishedAtDate) return false
    return publishedAtDate.getTime() > nowTs
  }, [publishedAtDate, nowTs])

  const isPublished = useMemo(() => {
    if (!publishedAtDate) return false
    return publishedAtDate.getTime() <= nowTs
  }, [publishedAtDate, nowTs])

  useEffect(() => {
    if (game?.publishedAt) {
      setPublishAtInput(toLocalInputValue(game.publishedAt))
      setIsEditingSchedule(false)
    } else {
      setPublishAtInput('')
    }
  }, [game?.publishedAt])

  // Focus input when editing starts
  useEffect(() => {
    if (editingField && inputRef.current) {
      inputRef.current.focus()
    }
  }, [editingField])

  function startEditing(field: string, currentValue: unknown) {
    if (!isOrganizer) return
    setEditingField(field)
    if (field === 'totalPriceCents' && typeof currentValue === 'number') {
      setEditValue(formatCentsAsDollars(currentValue))
    } else if (typeof currentValue === 'number') {
      setEditValue(String(currentValue))
    } else if (typeof currentValue === 'string') {
      setEditValue(currentValue)
    } else {
      setEditValue('')
    }
  }

  function cancelEditing() {
    setEditingField(null)
    setEditValue('')
    cancelAllDebounces()
  }

  function cancelAllDebounces() {
    Object.keys(saveTimersRef.current).forEach(field => {
      const timerId = saveTimersRef.current[field]
      if (timerId !== undefined) {
        clearTimeout(timerId)
        saveTimersRef.current[field] = undefined
      }
    })
  }

  async function updatePublishTime(publishedAtValue: string | null, requireReady = true) {
    if (!isOrganizer || !id) return false
    if (requireReady && !canPublish) {
      setPublishError('Complete all required fields before publishing.')
      return false
    }

    setIsPublishing(true)
    setPublishError(null)

    try {
      const resp = await fetch(`${API_BASE_URL}/api/games/${id}`, {
        method: 'PATCH',
        headers: { 'Content-Type': 'application/json' },
        credentials: 'include',
        body: JSON.stringify({ publishedAt: publishedAtValue }),
      })

      if (!resp.ok) {
        if (resp.status === 401) {
          redirectToLogin()
          return false
        }
        const txt = await resp.text()
        throw new Error(txt || 'Failed to update publish time')
      }

      const updated = await resp.json()
      setGame(updated.game)
      setOrganizer(updated.organizer)
      return true
    } catch (e) {
      setPublishError(e instanceof Error ? e.message : 'Failed to update publish time')
      return false
    } finally {
      setIsPublishing(false)
    }
  }

  async function handlePublishNow() {
    const ok = await updatePublishTime(new Date().toISOString())
    if (ok) setIsEditingSchedule(false)
  }

  async function handleSchedulePublish() {
    if (!publishAtInput) {
      setPublishError('Select a date and time to schedule publishing.')
      return
    }
    const iso = fromLocalInputValue(publishAtInput)
    if (!iso) {
      setPublishError('Invalid date and time. Please pick a valid value.')
      return
    }
    const ok = await updatePublishTime(iso)
    if (ok) setIsEditingSchedule(false)
  }

  async function saveField(field: string, value: unknown) {
    if (!isOrganizer || !id) return
    // Skip sending empty strings to avoid accidental clearing until supported
    if (typeof value === 'string' && value.trim() === '') return
    if (value === undefined) return

    try {
      const resp = await fetch(`${API_BASE_URL}/api/games/${id}`, {
        method: 'PATCH',
        headers: { 'Content-Type': 'application/json' },
        credentials: 'include',
        body: JSON.stringify({ [field]: value }),
      })
      if (!resp.ok) {
        if (resp.status === 401) {
          redirectToLogin()
          return
        }
        const txt = await resp.text()
        throw new Error(txt || 'Failed to save')
      }
      const updated = await resp.json()
      setGame(updated)
      setEditingField(null)
    } catch (e) {
      console.error('Failed to save field:', field, e)
    }
  }

  function handleBlur(field: string) {
    if (editingField !== field) return
    cancelDebouncedSave(field)
    
    let valueToSave: unknown = editValue
    if (field === 'totalPriceCents') {
      // Convert dollars.cents format to cents
      const dollars = parseFloat(editValue)
      if (isNaN(dollars)) {
        valueToSave = undefined
      } else {
        // Round to ensure we have whole cents
        valueToSave = Math.round(dollars * 100)
      }
    } else if (field === 'maxPlayers' || field === 'durationMinutes') {
      const num = Number(editValue)
      valueToSave = isNaN(num) ? undefined : num
    } else if (field === 'startsAt') {
      valueToSave = fromLocalInputValue(editValue)
    }
    
    if (valueToSave !== undefined && valueToSave !== '') {
      saveField(field, valueToSave)
    } else {
      cancelEditing()
    }
  }

  function cancelDebouncedSave(field: string) {
    const timers = saveTimersRef.current
    const existing = timers[field]
    if (existing !== undefined) {
      clearTimeout(existing)
      timers[field] = undefined
    }
  }

  if (isLoading) {
    return (
      <div className="min-h-screen bg-gradient-to-br from-yellow-50 via-orange-50 to-blue-50 flex items-center justify-center">
        <Loader2 className="h-8 w-8 animate-spin text-primary" />
      </div>
    )
  }

  if (error) {
    return (
      <div className="min-h-screen bg-gradient-to-br from-yellow-50 via-orange-50 to-blue-50">
        <div className="container mx-auto px-4 py-8">
          <Button
            variant="ghost"
            onClick={() => navigate('/')}
            className="mb-8"
          >
            <ArrowLeft className="mr-2 h-4 w-4" />
            Back to Games
          </Button>

          <div className="bg-red-50 border border-red-200 rounded-xl p-8 text-center">
            <h1 className="text-2xl font-bold text-red-900 mb-2">Error</h1>
            <p className="text-red-700">{error}</p>
          </div>
        </div>
      </div>
    )
  }

  if (!game) {
    return (
      <div className="min-h-screen bg-gradient-to-br from-yellow-50 via-orange-50 to-blue-50">
        <div className="container mx-auto px-4 py-8">
          <Button
            variant="ghost"
            onClick={() => navigate('/')}
            className="mb-8"
          >
            <ArrowLeft className="mr-2 h-4 w-4" />
            Back to Games
          </Button>

          <div className="text-center py-20">
            <p className="text-gray-600">Game not found</p>
          </div>
        </div>
      </div>
    )
  }

  return (
    <div className="min-h-screen bg-gradient-to-br from-yellow-50 via-orange-50 to-blue-50">
      <div className="container mx-auto px-4 py-8 max-w-5xl">
        {/* Header with Back Button */}
        <div className="flex items-center justify-between mb-8">
          <Button
            variant="ghost"
            onClick={() => navigate('/')}
            className="text-gray-600 hover:text-gray-900"
          >
            <ArrowLeft className="mr-2 h-5 w-5" />
            Back
          </Button>
          <UserProfileMenu user={user} onUserChange={handleUserChange} />
        </div>

        {/* Main Game Card */}
        <Card className="overflow-hidden border-l-8 border-l-primary mb-8">
          {/* Hero Section with Status Badge */}
          <div className="relative p-8 pb-6">
            <div className="flex justify-between items-start gap-4 mb-4">
              <div className="flex-1">
                <div className="flex items-center gap-3 mb-3">
                  {isOrganizer && (
                    <div
                      onMouseEnter={() => setOrganizerHintOpen(true)}
                      onMouseLeave={() => setOrganizerHintOpen(false)}
                    >
                      <Popover open={organizerHintOpen}>
                        <PopoverAnchor asChild>
                          <span className="inline-flex items-center bg-secondary/20 rounded-full p-1.5 cursor-help">
                            <Crown className="h-5 w-5 text-primary" />
                          </span>
                        </PopoverAnchor>
                        <PopoverContent side="right" className="text-sm rounded-xl">
                          <p className="font-semibold mb-1">You&apos;re organizing</p>
                          <p className="text-gray-600">You can edit everything on this page</p>
                        </PopoverContent>
                      </Popover>
                    </div>
                  )}
                  {editingField === 'name' && isOrganizer ? (
                    <Input
                      value={editValue}
                      onChange={(e) => setEditValue(e.target.value)}
                      onBlur={() => handleBlur('name')}
                      onKeyDown={(e) => {
                        if (e.key === 'Enter') handleBlur('name')
                        if (e.key === 'Escape') setEditingField(null)
                      }}
                      className="text-3xl font-bold"
                    />
                  ) : (
                    <h1 
                      className="text-4xl font-bold text-gray-900 cursor-text transition"
                      onClick={() => startEditing('name', game?.name || '')}
                    >
                      {game?.name || 'Game'}
                    </h1>
                  )}
                </div>

                {/* Status Badge */}
                <div className="flex flex-wrap gap-2 items-center">
                  {isPublished && (
                    <span className="inline-flex items-center gap-1.5 px-4 py-2 bg-success text-white text-sm font-semibold rounded-full shadow-md">
                      <CheckCircle2 className="h-4 w-4" />
                      Game locked
                    </span>
                  )}
                  {isScheduled && (
                    <span className="inline-flex items-center gap-1.5 px-4 py-2 bg-secondary text-secondary-foreground text-sm font-semibold rounded-full shadow-md">
                      <Clock className="h-4 w-4" />
                      Publishing soon
                    </span>
                  )}
                  {!isPublished && !isScheduled && (
                    <span className="inline-flex items-center gap-1.5 px-4 py-2 bg-gray-300 text-gray-700 text-sm font-semibold rounded-full shadow-md">
                      <CircleDashed className="h-4 w-4" />
                      Draft
                    </span>
                  )}
                </div>

                {/* Organizer Info */}
                {organizer && (
                  <div className="flex items-center gap-3 mt-4 pt-4 border-t border-gray-200">
                    <div className="w-8 h-8 rounded-full overflow-hidden flex-shrink-0">
                      {organizer.picture ? (
                        <img 
                          src={organizer.picture} 
                          alt={organizer.name || organizer.email}
                          className="w-full h-full object-cover"
                        />
                      ) : (
                        <div className="w-full h-full bg-primary text-white text-xs flex items-center justify-center font-bold">
                          {getInitials(organizer.name, organizer.email)}
                        </div>
                      )}
                    </div>
                    <div>
                      <p className="text-xs text-gray-500 font-medium">Organizer</p>
                      <p className="text-sm font-semibold text-gray-900">{organizer.name || organizer.email}</p>
                    </div>
                  </div>
                )}
              </div>
            </div>

            {/* Quick Stats */}
            {isPublished && (
              <div className="grid grid-cols-3 gap-4 mt-6 pt-6 border-t border-gray-200">
                <div className="text-center">
                  <div className="text-2xl font-bold text-primary mb-1">{participantCounts.going}/{game?.maxPlayers || '?'}</div>
                  <div className="text-xs text-gray-600 font-medium">Going</div>
                </div>
                <div className="text-center">
                  <div className="text-2xl font-bold text-accent mb-1">{participantCounts.waitlisted}</div>
                  <div className="text-xs text-gray-600 font-medium">Waitlist</div>
                </div>
                <div className="text-center">
                  <div className="text-2xl font-bold text-gray-400 mb-1">{participantCounts.notGoing}</div>
                  <div className="text-xs text-gray-600 font-medium">Not Going</div>
                </div>
              </div>
            )}
          </div>

          {/* Game Details */}
          <div className="grid md:grid-cols-2 gap-6 p-8 border-t border-gray-100">
            {/* Left Column */}
            <div className="space-y-6">
              {/* Location */}
              <div>
                <label className="text-xs font-semibold text-gray-500 uppercase tracking-wide mb-2 block">Location</label>
                {editingField === 'location' && isOrganizer ? (
                  <Input
                    type="text"
                    value={editValue}
                    onChange={(e) => setEditValue(e.target.value)}
                    onBlur={() => handleBlur('location')}
                    onKeyDown={(e) => {
                      if (e.key === 'Enter') handleBlur('location')
                      if (e.key === 'Escape') setEditingField(null)
                    }}
                    placeholder="Where are you playing?"
                  />
                ) : (
                  <div
                    onClick={() => startEditing('location', game?.location || '')}
                    className={`text-lg font-semibold cursor-text transition ${game?.location ? 'text-gray-900' : 'text-gray-400'}`}
                  >
                    {game?.location || (isOrganizer ? 'Click to add location' : '—')}
                  </div>
                )}
              </div>

              {/* Date & Time */}
              <div>
                <label className="text-xs font-semibold text-gray-500 uppercase tracking-wide mb-2 block">When</label>
                {editingField === 'startsAt' && isOrganizer ? (
                  <Input
                    type="datetime-local"
                    value={editValue}
                    onChange={(e) => setEditValue(e.target.value)}
                    onBlur={() => handleBlur('startsAt')}
                    onKeyDown={(e) => {
                      if (e.key === 'Enter') handleBlur('startsAt')
                      if (e.key === 'Escape') setEditingField(null)
                    }}
                  />
                ) : (
                  <div
                    onClick={() => startEditing('startsAt', game?.startsAt || '')}
                    className={`text-lg font-semibold cursor-text transition ${game?.startsAt ? 'text-gray-900' : 'text-gray-400'}`}
                  >
                    {game?.startsAt ? (
                      <TimeDisplay 
                        timestamp={game.startsAt} 
                        displayFormat="friendly"
                        className="text-gray-900"
                      />
                    ) : (
                      (isOrganizer ? 'Click to set time' : '—')
                    )}
                  </div>
                )}
              </div>

              {/* Duration */}
              <div>
                <label className="text-xs font-semibold text-gray-500 uppercase tracking-wide mb-2 block">Duration</label>
                {editingField === 'durationMinutes' && isOrganizer ? (
                  <Input
                    type="number"
                    value={editValue}
                    onChange={(e) => setEditValue(e.target.value)}
                    onBlur={() => handleBlur('durationMinutes')}
                    onKeyDown={(e) => {
                      if (e.key === 'Enter') handleBlur('durationMinutes')
                      if (e.key === 'Escape') setEditingField(null)
                    }}
                    placeholder="Minutes"
                  />
                ) : (
                  <div
                    onClick={() => startEditing('durationMinutes', game?.durationMinutes || '')}
                    className={`text-lg font-semibold cursor-text transition ${game?.durationMinutes ? 'text-gray-900' : 'text-gray-400'}`}
                  >
                    {game?.durationMinutes ? `${game.durationMinutes} min` : (isOrganizer ? 'Click to set' : '—')}
                  </div>
                )}
              </div>
            </div>

            {/* Right Column */}
            <div className="space-y-6">
              {/* Max Players */}
              <div>
                <label className="text-xs font-semibold text-gray-500 uppercase tracking-wide mb-2 block">Players</label>
                {editingField === 'maxPlayers' && isOrganizer ? (
                  <Input
                    type="number"
                    value={editValue}
                    onChange={(e) => setEditValue(e.target.value)}
                    onBlur={() => handleBlur('maxPlayers')}
                    onKeyDown={(e) => {
                      if (e.key === 'Enter') handleBlur('maxPlayers')
                      if (e.key === 'Escape') setEditingField(null)
                    }}
                    placeholder="How many players?"
                  />
                ) : (
                  <div
                    onClick={() => startEditing('maxPlayers', game?.maxPlayers || '')}
                    className={`text-lg font-semibold cursor-text transition ${game?.maxPlayers ? 'text-gray-900' : 'text-gray-400'}`}
                  >
                    {game?.maxPlayers ? `Up to ${game.maxPlayers}` : (isOrganizer ? 'Click to set' : '—')}
                  </div>
                )}
              </div>

              {/* Price */}
              <div>
                <label className="text-xs font-semibold text-gray-500 uppercase tracking-wide mb-2 block">Price</label>
                {editingField === 'totalPriceCents' && isOrganizer ? (
                  <Input
                    type="number"
                    step="0.01"
                    value={editValue}
                    onChange={(e) => setEditValue(e.target.value)}
                    onBlur={() => handleBlur('totalPriceCents')}
                    onKeyDown={(e) => {
                      if (e.key === 'Enter') handleBlur('totalPriceCents')
                      if (e.key === 'Escape') setEditingField(null)
                    }}
                    placeholder="e.g., 15.50"
                  />
                ) : (
                  <div
                    onClick={() => startEditing('totalPriceCents', game?.totalPriceCents || '')}
                    className={`text-lg font-semibold cursor-text transition ${game?.totalPriceCents !== undefined && game.totalPriceCents >= 0 ? 'text-gray-900' : 'text-gray-400'}`}
                  >
                    {game?.totalPriceCents !== undefined && game.totalPriceCents >= 0 ? (
                      <PriceDisplay
                        totalPriceCents={game.totalPriceCents}
                        maxPlayers={game.maxPlayers}
                      />
                    ) : (
                      (isOrganizer ? 'Click to set price' : '—')
                    )}
                  </div>
                )}
              </div>
            </div>
          </div>

          {/* Description */}
          {(game?.description || isOrganizer) && (
            <div className="px-8 py-6 border-t border-gray-100">
              <div className="flex items-center justify-between mb-3">
                <label className="text-xs font-semibold text-gray-500 uppercase tracking-wide block">About this game</label>
                <span className="text-xs text-gray-400">Markdown supported</span>
              </div>
              {editingField === 'description' && isOrganizer ? (
                <textarea
                  ref={inputRef as unknown as React.RefObject<HTMLTextAreaElement>}
                  value={editValue}
                  onChange={(e) => setEditValue(e.target.value)}
                  onBlur={() => handleBlur('description')}
                  className="w-full p-3 border-2 border-primary rounded-xl resize-none focus:outline-none"
                  rows={3}
                  placeholder="Tell people about your game..."
                />
              ) : (
                <div
                  onClick={() => startEditing('description', game?.description || '')}
                  className={`cursor-text transition text-sm leading-relaxed ${game?.description ? 'text-gray-700' : 'text-gray-400'}`}
                >
                  {game?.description ? (
                    <MarkdownRenderer value={game.description} />
                  ) : (
                    (isOrganizer ? 'Click to add description' : 'No description')
                  )}
                </div>
              )}
            </div>
          )}

          {/* Participants Section */}
          {isPublished && (
            <div className="px-8 py-6 border-t border-gray-100">
              <h2 className="text-lg font-bold text-gray-900 mb-4 flex items-center gap-2">
                <Users className="h-5 w-5 text-primary" />
                Who&apos;s coming ({participantCounts.going}/{game?.maxPlayers || '?'})
              </h2>

              {isLoadingParticipants ? (
                <div className="flex justify-center py-8">
                  <Loader2 className="h-6 w-6 animate-spin text-primary" />
                </div>
              ) : participantsError ? (
                <div className="bg-red-50 border border-red-200 rounded-xl p-4 text-red-700 text-sm">
                  {participantsError}
                </div>
              ) : participants.length > 0 ? (
                <div>
                  {/* People Grid */}
                  <div className="flex flex-wrap gap-4 mb-6">
                    {participants.filter(p => p.status === 'going').map((p) => {
                      const isOrganizerParticipating = organizer && p.user.id === organizer.id
                      return (
                        <div key={p.user.id} className="flex flex-col items-center gap-2">
                          <div className="relative w-16 h-16">
                            <div className="w-16 h-16 rounded-full overflow-hidden ring-4 ring-success/30 shadow-md">
                              {p.user.picture ? (
                                <img 
                                  src={p.user.picture} 
                                  alt={p.user.name || p.user.email}
                                  className="w-full h-full object-cover"
                                />
                              ) : (
                                <div className="w-full h-full bg-primary text-white flex items-center justify-center font-bold">
                                  {getInitials(p.user.name, p.user.email)}
                                </div>
                              )}
                            </div>
                            {isOrganizerParticipating && (
                              <div className="absolute -bottom-1 -right-1 bg-primary rounded-full p-1.5 shadow-lg border-2 border-white">
                                <Crown className="h-3 w-3 text-white" />
                              </div>
                            )}
                          </div>
                          <span className="text-xs text-center text-gray-700 font-medium max-w-16 truncate">
                            {p.user.name || p.user.email}
                          </span>
                        </div>
                      )
                    })}
                    {/* Empty slots */}
                    {game?.maxPlayers && Array.from({ length: Math.max(0, game.maxPlayers - participantCounts.going) }).map((_, i) => (
                      <div key={`empty-${i}`} className="flex flex-col items-center gap-2">
                        <div className="w-16 h-16 rounded-full border-4 border-dashed border-gray-300 bg-gray-50 flex items-center justify-center">
                          <span className="text-gray-400 text-sm">?</span>
                        </div>
                        <span className="text-xs text-gray-400 font-medium max-w-16 truncate">Open</span>
                      </div>
                    ))}
                  </div>

                  {/* Waitlist */}
                  {participantCounts.waitlisted > 0 && (
                    <div className="mt-6 pt-6 border-t border-gray-200">
                      <p className="text-sm font-semibold text-gray-700 mb-3">Waitlist ({participantCounts.waitlisted})</p>
                      <div className="space-y-2">
                        {participants.filter(p => p.status === 'waitlisted').map((p) => (
                          <div key={p.user.id} className="flex items-center gap-3 p-2 rounded-lg hover:bg-gray-50">
                            <div className="w-8 h-8 rounded-full overflow-hidden">
                              {p.user.picture ? (
                                <img 
                                  src={p.user.picture} 
                                  alt={p.user.name || p.user.email}
                                  className="w-full h-full object-cover"
                                />
                              ) : (
                                <div className="w-full h-full bg-primary/50 text-white text-xs flex items-center justify-center font-bold">
                                  {getInitials(p.user.name, p.user.email)}
                                </div>
                              )}
                            </div>
                            <span className="text-sm text-gray-700">{p.user.name || p.user.email}</span>
                          </div>
                        ))}
                      </div>
                    </div>
                  )}
                </div>
              ) : (
                <div className="bg-yellow-50 border-2 border-dashed border-secondary rounded-xl p-8 text-center">
                  <Users className="h-12 w-12 mx-auto mb-3 text-secondary/40" />
                  <p className="text-gray-600 font-medium mb-2">Almost there</p>
                  <p className="text-sm text-gray-500">Be the first to sign up and get people excited!</p>
                </div>
              )}
            </div>
          )}

          {/* Action Buttons */}
          <div className="px-8 py-6 border-t border-gray-100 bg-gray-50/50">
            {isOrganizer && !isPublished ? (
              <div className="space-y-4">
                {/* Requirements Checklist */}
                <div className="bg-white p-4 rounded-xl border-2 border-primary/20">
                  <h3 className="font-semibold text-gray-900 mb-3 flex items-center gap-2">
                    <Rocket className="h-5 w-5 text-primary" />
                    Ready to share this game?
                  </h3>
                  <div className="space-y-2 mb-4">
                    {publishRequirements.map((req) => (
                      <div key={req.field} className="flex items-center gap-2 text-sm">
                        {req.met ? (
                          <CheckCircle2 className="h-4 w-4 text-success flex-shrink-0" />
                        ) : (
                          <XCircle className="h-4 w-4 text-gray-300 flex-shrink-0" />
                        )}
                        <span className={req.met ? 'text-gray-700' : 'text-gray-400'}>
                          {req.label}
                        </span>
                      </div>
                    ))}
                  </div>

                  <div className="flex gap-2">
                    <Button
                      onClick={handlePublishNow}
                      disabled={isPublishing || !canPublish}
                      className={canPublish ? 'bg-success' : 'bg-gray-400'}
                    >
                      {isPublishing ? (
                        <>
                          <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                          Publishing...
                        </>
                      ) : (
                        <>
                          <Rocket className="mr-2 h-4 w-4" />
                          Publish
                        </>
                      )}
                    </Button>
                    {canPublish && (
                      <Button
                        variant="outline"
                        onClick={() => setIsEditingSchedule(!isEditingSchedule)}
                      >
                        Schedule
                      </Button>
                    )}
                  </div>

                  {isEditingSchedule && canPublish && (
                    <div className="mt-4 pt-4 border-t border-gray-200 space-y-2">
                      <Input
                        type="datetime-local"
                        value={publishAtInput}
                        onChange={(e) => setPublishAtInput(e.target.value)}
                        placeholder="Pick a time to publish"
                      />
                      <Button
                        onClick={handleSchedulePublish}
                        disabled={isPublishing || !publishAtInput}
                        size="sm"
                        className="w-full"
                      >
                        {isScheduled ? 'Update schedule' : 'Schedule publish'}
                      </Button>
                    </div>
                  )}

                  {publishError && (
                    <div className="mt-3 text-red-600 text-sm">{publishError}</div>
                  )}
                </div>
              </div>
            ) : user && !isOrganizer && isPublished ? (
              <div>
                {currentUserParticipation?.status === 'going' ? (
                  <Button
                    variant="outline"
                    onClick={() => updateParticipation('not_going')}
                    disabled={isUpdatingParticipation}
                    className="w-full bg-accent/10 border-accent text-accent hover:bg-accent/20"
                  >
                    {isUpdatingParticipation ? 'Updating...' : '✓ You&apos;re going'}
                  </Button>
                ) : (
                  <Button
                    onClick={() => updateParticipation('going')}
                    disabled={isUpdatingParticipation || participantCounts.going >= (game?.maxPlayers || Infinity)}
                    className="w-full bg-accent"
                  >
                    {isUpdatingParticipation ? 'Signing up...' : 'Count me in!'}
                  </Button>
                )}
              </div>
            ) : null}
          </div>
        </Card>
      </div>
    </div>
  )
}

function toLocalInputValue(iso: string) {
  try {
    const d = new Date(iso)
    const pad = (n: number) => String(n).padStart(2, '0')
    const yyyy = d.getFullYear()
    const mm = pad(d.getMonth() + 1)
    const dd = pad(d.getDate())
    const hh = pad(d.getHours())
    const mi = pad(d.getMinutes())
    return `${yyyy}-${mm}-${dd}T${hh}:${mi}`
  } catch {
    return ''
  }
}

function fromLocalInputValue(val: string) {
  // Convert local datetime input back to ISO string
  try {
    const d = new Date(val)
    return d.toISOString()
  } catch {
    return ''
  }
}

function formatCentsAsDollars(cents?: number) {
  if (typeof cents !== 'number' || Number.isNaN(cents)) return ''
  if (cents === 0) return '0'
  const dollars = cents / 100
  return dollars.toFixed(2)
}
