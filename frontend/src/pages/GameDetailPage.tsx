import { useParams, useNavigate } from 'react-router-dom'
import { useEffect, useMemo, useRef, useState } from 'react'
import { Button } from '@/components/ui/button'
import { API_BASE_URL, redirectToLogin } from '@/lib/api'
import { fetchWithDemoRecovery } from '@/lib/fetchWithDemoRecovery'
import { ArrowLeft, Loader2, CheckCircle2, Edit2, Calendar, Clock, MapPin, Users, DollarSign, Crown, AlertCircle, XCircle, Rocket } from 'lucide-react'
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
}

export default function GameDetailPage() {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const [game, setGame] = useState<Game | null>(null)
  const [isLoading, setIsLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [user, setUser] = useState<AuthUser | null>(null)
  const [organizerHintOpen, setOrganizerHintOpen] = useState(false)
  const [draftHintOpen, setDraftHintOpen] = useState(false)
  const [isPublishing, setIsPublishing] = useState(false)
  const [publishError, setPublishError] = useState<string | null>(null)
  const [publishAtInput, setPublishAtInput] = useState('')
  const [nowTs, setNowTs] = useState(() => Date.now())
  const [isEditingSchedule, setIsEditingSchedule] = useState(false)

  // Editing state
  const [editingField, setEditingField] = useState<string | null>(null)
  const [editValue, setEditValue] = useState<string>('')

  // Autosave status per field
  const [saveErrors, setSaveErrors] = useState<Record<string, string | null>>({})
  const saveTimersRef = useRef<Record<string, number | undefined>>({})
  const [saved, setSaved] = useState<Record<string, boolean>>({})
  const savedTimersRef = useRef<Record<string, number | undefined>>({})
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
        setGame(gameData)

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
        setGame(gameData)
      }
    } catch (err) {
      console.error('Error refreshing game:', err)
    }
  }

  const handleUserChange = (newUser: any) => {
    setUser(newUser)
    // Refetch game when user changes
    refreshGame()
  }

  const isOrganizer = useMemo(() => {
    if (!game || !user) return false
    const userIdNum = Number.parseInt(user.id, 10)
    if (Number.isNaN(userIdNum)) return false
    return userIdNum === game.organizerId
  }, [game, user])

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
    if (typeof currentValue === 'number') {
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
      setGame(updated)
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

  async function handleClearSchedule() {
    const ok = await updatePublishTime(null, false)
    if (ok) setIsEditingSchedule(false)
  }

  async function saveField(field: string, value: unknown) {
    if (!isOrganizer || !id) return
    // Skip sending empty strings to avoid accidental clearing until supported
    if (typeof value === 'string' && value.trim() === '') return
    if (value === undefined) return

    setSaveErrors((prev) => ({ ...prev, [field]: null }))
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
      // Show 'Saved' briefly
      setSaved((prev) => ({ ...prev, [field]: true }))
      const existing = savedTimersRef.current[field]
      if (existing !== undefined) {
        clearTimeout(existing)
      }
      const t = window.setTimeout(() => {
        setSaved((prev) => ({ ...prev, [field]: false }))
        const current = savedTimersRef.current[field]
        if (current !== undefined) {
          clearTimeout(current)
          savedTimersRef.current[field] = undefined
        }
      }, 1500)
      savedTimersRef.current[field] = t
    } catch (e) {
      setSaveErrors((prev) => ({ ...prev, [field]: e instanceof Error ? e.message : 'Failed to save' }))
    } finally {
    }
  }

  function handleBlur(field: string) {
    if (editingField !== field) return
    cancelDebouncedSave(field)
    
    let valueToSave: unknown = editValue
    if (field === 'totalPriceCents' || field === 'maxPlayers' || field === 'durationMinutes') {
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

  function handleKeyDown(e: React.KeyboardEvent, field: string) {
    if (e.key === 'Escape') {
      cancelEditing()
    } else if (e.key === 'Enter' && field !== 'description') {
      handleBlur(field)
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
      <div className="min-h-screen bg-gradient-to-br from-blue-50 to-indigo-100 flex items-center justify-center">
        <Loader2 className="h-8 w-8 animate-spin text-indigo-600" />
      </div>
    )
  }

  if (error) {
    return (
      <div className="min-h-screen bg-gradient-to-br from-blue-50 to-indigo-100">
        <div className="container mx-auto px-4 py-8">
          <Button
            variant="ghost"
            onClick={() => navigate('/')}
            className="mb-8"
          >
            <ArrowLeft className="mr-2 h-4 w-4" />
            Back to Games
          </Button>

          <div className="bg-red-50 border border-red-200 rounded-lg p-8 text-center">
            <h1 className="text-2xl font-bold text-red-900 mb-2">Error</h1>
            <p className="text-red-700">{error}</p>
          </div>
        </div>
      </div>
    )
  }

  if (!game) {
    return (
      <div className="min-h-screen bg-gradient-to-br from-blue-50 to-indigo-100">
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
    <div className="min-h-screen bg-gradient-to-br from-blue-50 to-indigo-100">
      <div className="container mx-auto px-4 py-8 max-w-4xl">
        <div className="flex items-center justify-between mb-8">
          <Button
            variant="ghost"
            onClick={() => navigate('/')}
          >
            <ArrowLeft className="mr-2 h-4 w-4" />
            Back to Games
          </Button>
          <UserProfileMenu user={user} onUserChange={handleUserChange} />
        </div>

        <div className="bg-white rounded-lg shadow-lg overflow-hidden">
          {/* Hero Header */}
          <div className="bg-gradient-to-r from-indigo-600 to-purple-600 px-8 py-12 text-white relative">
            <div className="absolute top-4 right-4 flex items-center gap-2">
              {!isPublished && (
                <Popover open={draftHintOpen}>
                  <PopoverAnchor asChild>
                    <div
                      className={`inline-flex items-center px-3 py-1 rounded-full backdrop-blur-sm border transition-colors cursor-help ${
                        isScheduled
                          ? 'bg-blue-500/15 border-blue-300/50 hover:bg-blue-500/25'
                          : 'bg-amber-400/20 border-amber-300/50 hover:bg-amber-400/30'
                      }`}
                      onMouseEnter={() => setDraftHintOpen(true)}
                      onMouseLeave={() => setDraftHintOpen(false)}
                      aria-label="Publish status"
                    >
                      <AlertCircle className={`h-4 w-4 mr-1.5 ${isScheduled ? 'text-blue-200' : 'text-amber-200'}`} aria-hidden="true" />
                      <span className={`text-xs font-semibold ${isScheduled ? 'text-blue-100' : 'text-amber-100'}`}>
                        {isScheduled ? 'Scheduled' : 'Draft'}
                      </span>
                    </div>
                  </PopoverAnchor>
                  <PopoverContent side="left" sideOffset={12} className="text-gray-800 w-64 space-y-1">
                    {isScheduled ? (
                      <>
                        <p className="font-semibold">Publishing scheduled</p>
                        <p className="text-sm text-gray-700">Other users will see the game once it publishes.</p>
                        {publishedAtDate && (
                          <TimeDisplay
                            timestamp={publishedAtDate.toISOString()}
                            displayFormat="friendly"
                            prefix="Publishes"
                            className="text-gray-700 decoration-gray-400"
                          />
                        )}
                      </>
                    ) : (
                      <>
                        <p className="font-semibold mb-1">Game is a draft</p>
                        <p className="text-sm">Other users won't be able to view or join this game until it's published.</p>
                      </>
                    )}
                  </PopoverContent>
                </Popover>
              )}
              {isOrganizer && (
                <Popover open={organizerHintOpen}>
                  <PopoverAnchor asChild>
                    <div
                      className="inline-flex items-center px-2 py-1 rounded-full bg-white/20 backdrop-blur-sm border border-white/30 cursor-help"
                      onMouseEnter={() => setOrganizerHintOpen(true)}
                      onMouseLeave={() => setOrganizerHintOpen(false)}
                      aria-label="Organizer"
                    >
                      <Crown className="h-4 w-4 text-amber-300" aria-hidden="true" />
                    </div>
                  </PopoverAnchor>
                  <PopoverContent side="left" sideOffset={12} className="text-gray-800">
                    <p className="font-semibold mb-1\">You are the organizer</p>
                    <p className="text-sm">You can edit the fields on this page.</p>
                  </PopoverContent>
                </Popover>
              )}
            </div>
            
            {/* Title - Inline Editable */}
            {editingField === 'name' && isOrganizer ? (
              <div className="relative">
                <Input
                  ref={inputRef as React.RefObject<HTMLInputElement>}
                  value={editValue}
                  onChange={(e) => setEditValue(e.target.value)}
                  onBlur={() => handleBlur('name')}
                  onKeyDown={(e) => handleKeyDown(e, 'name')}
                  className="text-4xl font-bold bg-white/10 border-white/30 text-white placeholder:text-white/50"
                />
                {saveErrors['name'] && (
                  <div className="text-xs text-red-200 mt-2">{saveErrors['name']}</div>
                )}
              </div>
            ) : (
              <h1 
                className={`text-4xl font-bold mb-4 ${isOrganizer ? 'cursor-pointer hover:text-white/90 transition-colors' : ''}`}
                onClick={() => isOrganizer && startEditing('name', game?.name)}
              >
                {game?.name || 'Untitled Game'}
              </h1>
            )}
            
            <div className="flex items-center gap-2 text-sm text-white/80">
              <Calendar className="h-4 w-4" />
              <TimeDisplay 
                timestamp={game?.createdAt || ''} 
                displayFormat="relative" 
                prefix="Created"
                className="text-white/80 decoration-white/40"
              />
            </div>
          </div>

          {/* Main Content */}
          <div className="p-8 space-y-8">
            {/* Description Section */}
            <section>
              <div className="flex items-center gap-2 mb-4">
                <h2 className="text-xl font-semibold text-gray-900">About</h2>
                {isOrganizer && editingField !== 'description' && (
                  <Button
                    variant="ghost"
                    size="sm"
                    onClick={() => startEditing('description', game?.description || '')}
                    className="h-7 px-2 text-gray-500 hover:text-gray-900"
                  >
                    <Edit2 className="h-3 w-3" />
                  </Button>
                )}
                <Popover open={!!saved['description']}>
                  <PopoverContent side="right" className="flex items-center gap-2 text-green-700 w-auto">
                    <CheckCircle2 className="h-4 w-4" />
                    <span>Saved</span>
                  </PopoverContent>
                </Popover>
              </div>
              
              {editingField === 'description' && isOrganizer ? (
                <div className="space-y-3">
                  <textarea
                    ref={inputRef as React.RefObject<HTMLTextAreaElement>}
                    className="w-full min-h-[200px] rounded-lg border border-gray-300 bg-white px-4 py-3 text-base text-gray-900 placeholder:text-gray-400 focus:border-indigo-500 focus:outline-none focus:ring-2 focus:ring-indigo-500/20"
                    value={editValue}
                    onChange={(e) => setEditValue(e.target.value)}
                    onBlur={() => handleBlur('description')}
                    placeholder="Describe your game (Markdown supported)..."
                  />
                  <div className="flex gap-2">
                    <Button size="sm" onClick={() => handleBlur('description')}>
                      Save
                    </Button>
                    <Button size="sm" variant="outline" onClick={cancelEditing}>
                      Cancel
                    </Button>
                  </div>
                  {saveErrors['description'] && (
                    <div className="text-xs text-red-600">{saveErrors['description']}</div>
                  )}
                </div>
              ) : (
                <div className={`prose prose-slate max-w-none ${isOrganizer && !game?.description ? 'cursor-pointer' : ''}`}
                     onClick={() => isOrganizer && !game?.description && startEditing('description', '')}>
                  {game?.description ? (
                    <MarkdownRenderer value={game.description} />
                  ) : (
                    <div className="text-gray-400 italic py-8 text-center border-2 border-dashed border-gray-200 rounded-lg">
                      {isOrganizer ? 'Click to add a description...' : 'No description yet.'}
                    </div>
                  )}
                </div>
              )}
            </section>

            {/* Details Grid */}
            <section className="grid grid-cols-1 md:grid-cols-2 gap-6">
              {/* Location */}
              <div className="bg-gray-50 rounded-lg p-6">
                <div className="flex items-center gap-3 mb-3">
                  <div className="bg-indigo-100 rounded-full p-2">
                    <MapPin className="h-5 w-5 text-indigo-600" />
                  </div>
                  <h3 className="font-semibold text-gray-900">Location</h3>
                  <Popover open={!!saved['location']}>
                    <PopoverContent side="right" className="flex items-center gap-2 text-green-700 w-auto">
                      <CheckCircle2 className="h-4 w-4" />
                      <span>Saved</span>
                    </PopoverContent>
                  </Popover>
                </div>
                
                {editingField === 'location' && isOrganizer ? (
                  <div>
                    <Input
                      ref={inputRef as React.RefObject<HTMLInputElement>}
                      value={editValue}
                      onChange={(e) => setEditValue(e.target.value)}
                      onBlur={() => handleBlur('location')}
                      onKeyDown={(e) => handleKeyDown(e, 'location')}
                      placeholder="e.g., 123 Main St, Downtown"
                      className="mb-2"
                    />
                    {saveErrors['location'] && (
                      <div className="text-xs text-red-600">{saveErrors['location']}</div>
                    )}
                  </div>
                ) : (
                  <p 
                    className={`text-gray-700 ${isOrganizer ? 'cursor-pointer hover:text-gray-900' : ''}`}
                    onClick={() => isOrganizer && startEditing('location', game?.location || '')}
                  >
                    {game?.location || (isOrganizer ? 'Click to add location...' : '—')}
                  </p>
                )}
              </div>

              {/* Start Time */}
              <div className="bg-gray-50 rounded-lg p-6">
                <div className="flex items-center gap-3 mb-3">
                  <div className="bg-green-100 rounded-full p-2">
                    <Clock className="h-5 w-5 text-green-600" />
                  </div>
                  <h3 className="font-semibold text-gray-900">Start Time</h3>
                  <Popover open={!!saved['startsAt']}>
                    <PopoverContent side="right" className="flex items-center gap-2 text-green-700 w-auto">
                      <CheckCircle2 className="h-4 w-4" />
                      <span>Saved</span>
                    </PopoverContent>
                  </Popover>
                </div>
                
                {editingField === 'startsAt' && isOrganizer ? (
                  <div>
                    <Input
                      ref={inputRef as React.RefObject<HTMLInputElement>}
                      type="datetime-local"
                      value={editValue}
                      onChange={(e) => setEditValue(e.target.value)}
                      onBlur={() => handleBlur('startsAt')}
                      onKeyDown={(e) => handleKeyDown(e, 'startsAt')}
                      className="mb-2"
                    />
                    {saveErrors['startsAt'] && (
                      <div className="text-xs text-red-600">{saveErrors['startsAt']}</div>
                    )}
                  </div>
                ) : (
                  <div
                    className={isOrganizer ? 'cursor-pointer hover:text-gray-900' : ''}
                    onClick={() => isOrganizer && startEditing('startsAt', game?.startsAt ? toLocalInputValue(game.startsAt) : '')}
                  >
                    {game?.startsAt ? (
                      <TimeDisplay 
                        timestamp={game.startsAt} 
                        displayFormat="friendly"
                        className="text-gray-700 decoration-gray-400"
                      />
                    ) : (
                      <p className="text-gray-700">{isOrganizer ? 'Click to set time...' : '—'}</p>
                    )}
                  </div>
                )}
              </div>

              {/* Duration */}
              <div className="bg-gray-50 rounded-lg p-6">
                <div className="flex items-center gap-3 mb-3">
                  <div className="bg-amber-100 rounded-full p-2">
                    <Clock className="h-5 w-5 text-amber-600" />
                  </div>
                  <h3 className="font-semibold text-gray-900">Duration</h3>
                  <Popover open={!!saved['durationMinutes']}>
                    <PopoverContent side="right" className="flex items-center gap-2 text-green-700 w-auto">
                      <CheckCircle2 className="h-4 w-4" />
                      <span>Saved</span>
                    </PopoverContent>
                  </Popover>
                </div>
                
                {editingField === 'durationMinutes' && isOrganizer ? (
                  <div>
                    <Input
                      ref={inputRef as React.RefObject<HTMLInputElement>}
                      type="number"
                      value={editValue}
                      onChange={(e) => setEditValue(e.target.value)}
                      onBlur={() => handleBlur('durationMinutes')}
                      onKeyDown={(e) => handleKeyDown(e, 'durationMinutes')}
                      placeholder="Minutes"
                      className="mb-2"
                    />
                    {saveErrors['durationMinutes'] && (
                      <div className="text-xs text-red-600">{saveErrors['durationMinutes']}</div>
                    )}
                  </div>
                ) : (
                  <p 
                    className={`text-gray-700 ${isOrganizer ? 'cursor-pointer hover:text-gray-900' : ''}`}
                    onClick={() => isOrganizer && startEditing('durationMinutes', game?.durationMinutes)}
                  >
                    {typeof game?.durationMinutes === 'number' ? `${game.durationMinutes} minutes` : (isOrganizer ? 'Click to set duration...' : '—')}
                  </p>
                )}
              </div>

              {/* Max Players */}
              <div className="bg-gray-50 rounded-lg p-6">
                <div className="flex items-center gap-3 mb-3">
                  <div className="bg-blue-100 rounded-full p-2">
                    <Users className="h-5 w-5 text-blue-600" />
                  </div>
                  <h3 className="font-semibold text-gray-900">Max Players</h3>
                  <Popover open={!!saved['maxPlayers']}>
                    <PopoverContent side="right" className="flex items-center gap-2 text-green-700 w-auto">
                      <CheckCircle2 className="h-4 w-4" />
                      <span>Saved</span>
                    </PopoverContent>
                  </Popover>
                </div>
                
                {editingField === 'maxPlayers' && isOrganizer ? (
                  <div>
                    <Input
                      ref={inputRef as React.RefObject<HTMLInputElement>}
                      type="number"
                      value={editValue}
                      onChange={(e) => setEditValue(e.target.value)}
                      onBlur={() => handleBlur('maxPlayers')}
                      onKeyDown={(e) => handleKeyDown(e, 'maxPlayers')}
                      placeholder="Number of players"
                      className="mb-2"
                    />
                    {saveErrors['maxPlayers'] && (
                      <div className="text-xs text-red-600">{saveErrors['maxPlayers']}</div>
                    )}
                  </div>
                ) : (
                  <p 
                    className={`text-gray-700 ${isOrganizer ? 'cursor-pointer hover:text-gray-900' : ''}`}
                    onClick={() => isOrganizer && startEditing('maxPlayers', game?.maxPlayers)}
                  >
                    {typeof game?.maxPlayers === 'number' ? `${game.maxPlayers} players` : (isOrganizer ? 'Click to set max...' : '—')}
                  </p>
                )}
              </div>
            </section>

            {/* Pricing Section */}
            <section className="bg-gradient-to-br from-purple-50 to-indigo-50 rounded-lg p-6 border border-purple-100">
              <div className="flex items-center gap-3 mb-4">
                <div className="bg-purple-100 rounded-full p-2">
                  <DollarSign className="h-5 w-5 text-purple-600" />
                </div>
                <h3 className="font-semibold text-gray-900">Pricing</h3>
                <Popover open={!!saved['totalPriceCents']}>
                  <PopoverContent side="right" className="flex items-center gap-2 text-green-700 w-auto">
                    <CheckCircle2 className="h-4 w-4" />
                    <span>Saved</span>
                  </PopoverContent>
                </Popover>
              </div>
              
              {editingField === 'totalPriceCents' && isOrganizer ? (
                <div>
                  <div className="flex gap-2 items-center mb-2">
                    <Input
                      ref={inputRef as React.RefObject<HTMLInputElement>}
                      type="number"
                      value={editValue}
                      onChange={(e) => setEditValue(e.target.value)}
                      onBlur={() => handleBlur('totalPriceCents')}
                      onKeyDown={(e) => handleKeyDown(e, 'totalPriceCents')}
                      placeholder="Price in cents"
                      className="max-w-xs"
                    />
                    <span className="text-sm text-gray-500">cents</span>
                  </div>
                  {saveErrors['totalPriceCents'] && (
                    <div className="text-xs text-red-600">{saveErrors['totalPriceCents']}</div>
                  )}
                </div>
              ) : (
                <div 
                  className={isOrganizer ? 'cursor-pointer' : ''}
                  onClick={() => isOrganizer && startEditing('totalPriceCents', game?.totalPriceCents)}
                >
                  <PriceDisplay totalPriceCents={game?.totalPriceCents} maxPlayers={game?.maxPlayers} />
                </div>
              )}
            </section>

            {/* Action Area */}
            <section className="border-t pt-6">
              {isOrganizer ? (
                !isPublished ? (
                  <div className="space-y-6">
                    {/* Publish Requirements Checklist */}
                    <div className="bg-gradient-to-br from-blue-50 to-indigo-50 rounded-lg p-6 border border-blue-100">
                      <h3 className="font-semibold text-gray-900 mb-4 flex items-center gap-2">
                        <Rocket className="h-5 w-5 text-indigo-600" />
                        Requirements to publish the game
                      </h3>
                      <div className="space-y-3 mb-6">
                        {publishRequirements.map((req) => (
                          <div key={req.field} className="flex items-center gap-3">
                            {req.met ? (
                              <CheckCircle2 className="h-5 w-5 text-green-600 flex-shrink-0" />
                            ) : (
                              <XCircle className="h-5 w-5 text-gray-400 flex-shrink-0" />
                            )}
                            <span className={`text-sm ${req.met ? 'text-gray-700' : 'text-gray-500'}`}>
                              {req.label}
                            </span>
                          </div>
                        ))}
                      </div>
                      
                      {canPublish || isScheduled ? (
                        <div className="space-y-4">
                          <div className="bg-white rounded-lg p-4 border border-green-200 space-y-3">
                            <div className="flex items-start gap-3">
                              {isScheduled ? (
                                <Clock className="h-5 w-5 text-blue-600 mt-0.5 flex-shrink-0" />
                              ) : (
                                <CheckCircle2 className="h-5 w-5 text-green-600 mt-0.5 flex-shrink-0" />
                              )}
                              <div className="space-y-1">
                                <p className="font-medium text-gray-900">
                                  {isScheduled ? 'Publishing scheduled' : 'Ready to publish'}
                                </p>
                                <p className="text-sm text-gray-600">
                                  {isScheduled
                                    ? (canPublish
                                      ? 'Change the scheduled time, publish now, or cancel scheduling.'
                                      : 'Publishing is scheduled. Complete the requirements above before changing the publish time or publishing now.')
                                    : 'Your game meets all requirements and can be published. Once published, other users will be able to view and join your game.'}
                                </p>
                                {isScheduled && publishedAtDate && (
                                  <div className="text-sm text-gray-700">
                                    <TimeDisplay 
                                      timestamp={publishedAtDate.toISOString()}
                                      displayFormat="friendly"
                                      prefix="Publishes"
                                      className="text-gray-700 decoration-gray-400"
                                    />
                                  </div>
                                )}
                                {!isScheduled && !isPublished && (
                                  <p className="text-xs text-gray-500">You can publish now or schedule it for later.</p>
                                )}
                              </div>
                            </div>

                            <div className="flex flex-wrap gap-2">
                              <Button
                                onClick={handlePublishNow}
                                disabled={isPublishing || !canPublish}
                                className="bg-gradient-to-r from-indigo-600 to-purple-600 hover:from-indigo-700 hover:to-purple-700"
                              >
                                {isPublishing ? (
                                  <>
                                    <Loader2 className="mr-2 h-5 w-5 animate-spin" />
                                    Publishing...
                                  </>
                                ) : (
                                  <>
                                    <Rocket className="mr-2 h-5 w-5" />
                                    Publish now
                                  </>
                                )}
                              </Button>
                              <Button
                                variant="outline"
                                onClick={() => setIsEditingSchedule(true)}
                                disabled={isPublishing || !canPublish}
                              >
                                {isScheduled ? 'Reschedule' : 'Schedule publish'}
                              </Button>
                              {isScheduled && (
                                <Button
                                  variant="ghost"
                                  onClick={handleClearSchedule}
                                  disabled={isPublishing}
                                >
                                  Clear schedule
                                </Button>
                              )}
                            </div>

                            {isEditingSchedule && (
                              <div className="space-y-2 border-t border-gray-200 pt-3">
                                <label className="text-sm font-semibold text-gray-900">Publish time</label>
                                <div className="flex flex-col gap-3 md:flex-row md:items-center">
                                  <Input
                                    type="datetime-local"
                                    value={publishAtInput}
                                    onChange={(e) => setPublishAtInput(e.target.value)}
                                    className="md:w-64"
                                  />
                                  <div className="flex flex-wrap gap-2">
                                    <Button
                                      variant="outline"
                                      onClick={handleSchedulePublish}
                                      disabled={isPublishing || !publishAtInput || !canPublish}
                                    >
                                      {isPublishing ? 'Saving...' : (isScheduled ? 'Update schedule' : 'Save schedule')}
                                    </Button>
                                    <Button
                                      variant="ghost"
                                      onClick={() => setIsEditingSchedule(false)}
                                      disabled={isPublishing}
                                    >
                                      Cancel
                                    </Button>
                                  </div>
                                </div>
                                <p className="text-xs text-gray-500">
                                  Pick a future time to schedule (uses your local timezone). Leave empty and use Publish now for immediate release.
                                </p>
                              </div>
                            )}
                          </div>
                        </div>
                      ) : (
                        <div className="bg-blue-50 rounded-lg p-4 border border-blue-200">
                          <div className="flex items-start gap-3">
                            <AlertCircle className="h-5 w-5 text-blue-600 mt-0.5 flex-shrink-0" />
                            <div>
                              <p className="text-sm text-gray-600">
                                After publishing, you'll still be able to change game details. However, changes to 
                                important details like Location, Time, and Price will require participants to re-cast their vote.
                              </p>
                            </div>
                          </div>
                        </div>
                      )}
                      
                      {publishError && (
                        <div className="mt-3 bg-red-50 border border-red-200 rounded-lg p-4">
                          <div className="flex items-center gap-2 text-red-800">
                            <AlertCircle className="h-4 w-4" />
                            <span className="text-sm font-medium">Error: {publishError}</span>
                          </div>
                        </div>
                      )}
                    </div>
                    
                    <div className="text-center text-sm text-gray-500">
                      Click any field above to edit. Changes save automatically.
                    </div>
                  </div>
                ) : (
                  <div className="space-y-4">
                    <div className="bg-green-50 rounded-lg p-6 border border-green-200">
                      <div className="flex items-center gap-3 mb-2">
                        <CheckCircle2 className="h-6 w-6 text-green-600" />
                        <h3 className="font-semibold text-gray-900">Game Published</h3>
                      </div>
                      <p className="text-sm text-gray-600">
                        Your game is now live! Other users can view and join your game.
                      </p>
                      {game?.publishedAt && (
                        <div className="mt-3 text-sm text-gray-500">
                          <TimeDisplay 
                            timestamp={game.publishedAt} 
                            displayFormat="relative" 
                            prefix="Published"
                            className="text-gray-500 decoration-gray-400"
                          />
                        </div>
                      )}
                    </div>
                    <div className="text-center text-sm text-gray-500">
                      You can still edit game details. Changes will be visible immediately.
                    </div>
                  </div>
                )
              ) : (
                <div className="text-center py-4 text-gray-500">
                  <p className="text-sm">Player registration coming soon!</p>
                </div>
              )}
            </section>
          </div>
        </div>
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
