import { useParams, useNavigate } from 'react-router-dom'
import { useEffect, useMemo, useRef, useState } from 'react'
import { Button } from '@/components/ui/button'
import { API_BASE_URL, redirectToLogin } from '@/lib/api'
import { ArrowLeft, Loader2, CheckCircle2, Edit2, Calendar, Clock, MapPin, Users, DollarSign, Crown } from 'lucide-react'
import { Input } from '@/components/ui/input'
import { MarkdownRenderer } from '@/components/ui/MarkdownRenderer'
import { PriceDisplay } from '@/components/games/PriceDisplay'
import { Popover, PopoverContent, PopoverAnchor } from '@/components/ui/popover'

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
  publishedAt?: string
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

        const response = await fetch(`${API_BASE_URL}/api/games/${id}`, {
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
          const meResp = await fetch(`${API_BASE_URL}/api/auth/me`, {
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

  const isOrganizer = useMemo(() => {
    if (!game || !user) return false
    const userIdNum = Number.parseInt(user.id, 10)
    if (Number.isNaN(userIdNum)) return false
    return userIdNum === game.organizerId
  }, [game, user])

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
        <Button
          variant="ghost"
          onClick={() => navigate('/')}
          className="mb-8"
        >
          <ArrowLeft className="mr-2 h-4 w-4" />
          Back to Games
        </Button>

        <div className="bg-white rounded-lg shadow-lg overflow-hidden">
          {/* Hero Header */}
          <div className="bg-gradient-to-r from-indigo-600 to-purple-600 px-8 py-12 text-white relative">
            {isOrganizer && (
              <div
                className="absolute top-4 right-4"
                onMouseEnter={() => setOrganizerHintOpen(true)}
                onMouseLeave={() => setOrganizerHintOpen(false)}
              >
                <Popover open={organizerHintOpen}>
                  <PopoverAnchor asChild>
                    <span className="inline-flex items-center px-2 py-1 rounded-full bg-white/20 backdrop-blur-sm border border-white/30" aria-label="Organizer">
                      <Crown className="h-4 w-4 text-amber-300" aria-hidden="true" />
                    </span>
                  </PopoverAnchor>
                  <PopoverContent side="bottom" className="text-gray-800">
                    You are the organizer of this game.
                    <br />
                    You can edit the fields on this page.
                  </PopoverContent>
                </Popover>
              </div>
            )}
            
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
              <span>Created {new Date(game?.createdAt || '').toLocaleDateString()}</span>
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
                  <p 
                    className={`text-gray-700 ${isOrganizer ? 'cursor-pointer hover:text-gray-900' : ''}`}
                    onClick={() => isOrganizer && startEditing('startsAt', game?.startsAt ? toLocalInputValue(game.startsAt) : '')}
                  >
                    {game?.startsAt ? new Date(game.startsAt).toLocaleString() : (isOrganizer ? 'Click to set time...' : '—')}
                  </p>
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
              <div className="text-center py-4 text-gray-500">
                {isOrganizer ? (
                  <p className="text-sm">Click any field above to edit. Changes save automatically.</p>
                ) : (
                  <p className="text-sm">Player registration coming soon!</p>
                )}
              </div>
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
