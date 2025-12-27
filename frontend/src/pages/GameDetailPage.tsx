import { useParams, useNavigate } from 'react-router-dom'
import { useEffect, useMemo, useRef, useState } from 'react'
import { Button } from '@/components/ui/button'
import { API_BASE_URL, redirectToLogin } from '@/lib/api'
import { ArrowLeft, Loader2, CheckCircle2 } from 'lucide-react'
import { Input } from '@/components/ui/input'
import { MarkdownRenderer } from '@/components/ui/MarkdownRenderer'
import { PriceDisplay } from '@/components/games/PriceDisplay'
import { Popover, PopoverTrigger, PopoverContent, PopoverAnchor } from '@/components/ui/popover'

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

  // Local editable state (no persistence yet)
  const [name, setName] = useState('')
  const [description, setDescription] = useState('')
  const [location, setLocation] = useState('')
  const [totalPriceCents, setTotalPriceCents] = useState<number | undefined>(undefined)
  const [maxPlayers, setMaxPlayers] = useState<number | undefined>(undefined)
  const [startsAt, setStartsAt] = useState('')
  const [durationMinutes, setDurationMinutes] = useState<number | undefined>(undefined)

  // Autosave status per field
  const [saving, setSaving] = useState<Record<string, boolean>>({})
  const [saveErrors, setSaveErrors] = useState<Record<string, string | null>>({})
  const saveTimersRef = useRef<Record<string, number | undefined>>({})
  const [saved, setSaved] = useState<Record<string, boolean>>({})
  const savedTimersRef = useRef<Record<string, number | undefined>>({})

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

        // Initialize local editable state
        setName(gameData.name || '')
        setDescription(gameData.description || '')
        setLocation(gameData.location || '')
        setTotalPriceCents(
          typeof gameData.totalPriceCents === 'number' ? gameData.totalPriceCents : undefined
        )
        setMaxPlayers(
          typeof gameData.maxPlayers === 'number' ? gameData.maxPlayers : undefined
        )
        setStartsAt(gameData.startsAt || '')
        setDurationMinutes(
          typeof gameData.durationMinutes === 'number' ? gameData.durationMinutes : undefined
        )
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

  async function saveField(field: string, value: unknown) {
    if (!isOrganizer || !id) return
    // Skip sending empty strings to avoid accidental clearing until supported
    if (typeof value === 'string' && value.trim() === '') return
    if (value === undefined) return

    setSaving((prev) => ({ ...prev, [field]: true }))
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
      setSaving((prev) => ({ ...prev, [field]: false }))
    }
  }

  function scheduleDebouncedSave(field: string, value: unknown, delay = 800) {
    const timers = saveTimersRef.current
    const existing = timers[field]
    if (existing !== undefined) {
      clearTimeout(existing)
    }
    // Hide any previous 'Saved' when new edits begin
    setSaved((prev) => ({ ...prev, [field]: false }))
    const timerId = window.setTimeout(() => {
      saveField(field, value)
      // Clear the timer reference after running
      const t = saveTimersRef.current[field]
      if (t !== undefined) {
        clearTimeout(t)
        saveTimersRef.current[field] = undefined
      }
    }, delay)
    saveTimersRef.current[field] = timerId
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
      <div className="container mx-auto px-4 py-8">
        <Button
          variant="ghost"
          onClick={() => navigate('/')}
          className="mb-8"
        >
          <ArrowLeft className="mr-2 h-4 w-4" />
          Back to Games
        </Button>

        <div className="bg-white rounded-lg shadow-lg p-8 space-y-8">
          {/* Header: Name */}
          <div>
            <div className="flex items-center justify-between">
              <h1 className="text-4xl font-bold text-gray-900 mb-2">Game</h1>
              {isOrganizer && (
                <span className="text-xs px-2 py-1 rounded bg-indigo-50 text-indigo-700 border border-indigo-200">Organizer view</span>
              )}
            </div>
            <div className="text-sm text-gray-500 mb-4">
              Created on {new Date(game.createdAt).toLocaleDateString()}
            </div>

            {isOrganizer ? (
              <div className="space-y-2">
                <label className="text-sm font-medium text-gray-700 flex items-center gap-2">
                  <Popover open={!!saved['name'] && !saving['name'] && !saveErrors['name']}>
                    <PopoverTrigger asChild>
                      <span>Name</span>
                    </PopoverTrigger>
                    <PopoverContent side="right" className="flex items-center gap-2 text-green-700">
                      <CheckCircle2 className="h-4 w-4" />
                      <span>Saved</span>
                    </PopoverContent>
                  </Popover>
                  {/* Removed inline saving indicator; popover handles saved feedback */}
                </label>
                <Input
                  value={name}
                  onChange={(e) => {
                    const v = e.target.value
                    setName(v)
                    scheduleDebouncedSave('name', v)
                  }}
                  onBlur={() => {
                    cancelDebouncedSave('name')
                    saveField('name', name)
                  }}
                />
                {saveErrors['name'] && (
                  <div className="text-xs text-red-600">{saveErrors['name']}</div>
                )}
              </div>
            ) : (
              <h2 className="text-3xl font-semibold text-gray-900">{game.name}</h2>
            )}
          </div>

          {/* Description */}
          <div>
            <h3 className="text-lg font-semibold text-gray-800 mb-2">Description</h3>
            {isOrganizer ? (
              <div className="space-y-3">
                <Popover open={!!saved['description'] && !saving['description'] && !saveErrors['description']}>
                  <PopoverAnchor asChild>
                    <textarea
                      className="w-full min-h-[140px] rounded-md border border-gray-300 bg-white px-3 py-2 text-base text-gray-900 placeholder:text-gray-400 focus:border-indigo-500 focus:outline-none focus:ring-1 focus:ring-indigo-500"
                      value={description}
                      onChange={(e) => {
                        const v = e.target.value
                        setDescription(v)
                        scheduleDebouncedSave('description', v)
                      }}
                      placeholder="Markdown supported"
                      onBlur={() => {
                        cancelDebouncedSave('description')
                        saveField('description', description)
                      }}
                    />
                  </PopoverAnchor>
                  <PopoverContent side="right" className="flex items-center gap-2 text-green-700">
                    <CheckCircle2 className="h-4 w-4" />
                    <span>Saved</span>
                  </PopoverContent>
                </Popover>
                {saveErrors['description'] && (
                  <div className="text-xs text-red-600">{saveErrors['description']}</div>
                )}
                <div className="border-t pt-3">
                  <div className="text-sm text-gray-500 mb-2">Preview</div>
                  <MarkdownRenderer value={description || ''} />
                </div>
              </div>
            ) : (
              game.description ? (
                <MarkdownRenderer value={game.description} />
              ) : (
                <div className="bg-gray-50 rounded-lg p-6 text-center text-gray-500">
                  <p>No description yet.</p>
                </div>
              )
            )}
          </div>

          {/* Location */}
          <div>
            <h3 className="text-lg font-semibold text-gray-800 mb-2">Location</h3>
            {isOrganizer ? (
              <div className="space-y-2">
                <Popover open={!!saved['location'] && !saving['location'] && !saveErrors['location']}>
                  <PopoverAnchor asChild>
                    <Input
                      value={location}
                      onChange={(e) => {
                        const v = e.target.value
                        setLocation(v)
                        scheduleDebouncedSave('location', v)
                      }}
                      onBlur={() => {
                        cancelDebouncedSave('location')
                        saveField('location', location)
                      }}
                      placeholder="e.g., 123 Main St, Downtown"
                    />
                  </PopoverAnchor>
                  <PopoverContent side="right" className="flex items-center gap-2 text-green-700">
                    <CheckCircle2 className="h-4 w-4" />
                    <span>Saved</span>
                  </PopoverContent>
                </Popover>
                {/* Removed inline saving indicator; popover handles saved feedback */}
                {saveErrors['location'] && (
                  <div className="text-xs text-red-600">{saveErrors['location']}</div>
                )}
              </div>
            ) : (
              <div className="text-gray-800">{game.location || '—'}</div>
            )}
          </div>

          {/* Pricing */}
          <div>
            <h3 className="text-lg font-semibold text-gray-800 mb-2">Price</h3>
            {isOrganizer ? (
              <div className="grid grid-cols-1 sm:grid-cols-3 gap-4 items-end">
                <div className="space-y-2">
                  <label className="text-sm font-medium text-gray-700">Total Price (cents)</label>
                  <Popover open={!!saved['totalPriceCents'] && !saving['totalPriceCents'] && !saveErrors['totalPriceCents']}>
                    <PopoverAnchor asChild>
                      <Input
                        type="number"
                        value={typeof totalPriceCents === 'number' ? totalPriceCents : ''}
                        onChange={(e) => {
                          const v = e.target.value === '' ? undefined : Number(e.target.value)
                          setTotalPriceCents(v)
                          if (v !== undefined) scheduleDebouncedSave('totalPriceCents', v)
                        }}
                        onBlur={() => {
                          cancelDebouncedSave('totalPriceCents')
                          if (totalPriceCents !== undefined) saveField('totalPriceCents', totalPriceCents)
                        }}
                      />
                    </PopoverAnchor>
                    <PopoverContent side="right" className="flex items-center gap-2 text-green-700">
                      <CheckCircle2 className="h-4 w-4" />
                      <span>Saved</span>
                    </PopoverContent>
                  </Popover>
                  {/* Removed inline saving indicator; popover handles saved feedback */}
                  {saveErrors['totalPriceCents'] && (
                    <div className="text-xs text-red-600">{saveErrors['totalPriceCents']}</div>
                  )}
                </div>
                <div className="space-y-2">
                  <label className="text-sm font-medium text-gray-700">Max Players</label>
                  <Popover open={!!saved['maxPlayers'] && !saving['maxPlayers'] && !saveErrors['maxPlayers']}>
                    <PopoverAnchor asChild>
                      <Input
                        type="number"
                        value={typeof maxPlayers === 'number' ? maxPlayers : ''}
                        onChange={(e) => {
                          const v = e.target.value === '' ? undefined : Number(e.target.value)
                          setMaxPlayers(v)
                          if (v !== undefined) scheduleDebouncedSave('maxPlayers', v)
                        }}
                        onBlur={() => {
                          cancelDebouncedSave('maxPlayers')
                          if (maxPlayers !== undefined) saveField('maxPlayers', maxPlayers)
                        }}
                      />
                    </PopoverAnchor>
                    <PopoverContent side="right" className="flex items-center gap-2 text-green-700">
                      <CheckCircle2 className="h-4 w-4" />
                      <span>Saved</span>
                    </PopoverContent>
                  </Popover>
                  {/* Removed inline saving indicator; popover handles saved feedback */}
                  {saveErrors['maxPlayers'] && (
                    <div className="text-xs text-red-600">{saveErrors['maxPlayers']}</div>
                  )}
                </div>
                <div className="text-sm text-gray-700">
                  <PriceDisplay totalPriceCents={totalPriceCents} maxPlayers={maxPlayers} />
                </div>
              </div>
            ) : (
              <PriceDisplay totalPriceCents={game.totalPriceCents} maxPlayers={game.maxPlayers} />
            )}
          </div>

          {/* Timing */}
          <div>
            <h3 className="text-lg font-semibold text-gray-800 mb-2">Schedule</h3>
            {isOrganizer ? (
              <div className="grid grid-cols-1 sm:grid-cols-2 gap-4 items-end">
                <div className="space-y-2">
                  <label className="text-sm font-medium text-gray-700">Starts At</label>
                  <Popover open={!!saved['startsAt'] && !saving['startsAt'] && !saveErrors['startsAt']}>
                    <PopoverAnchor asChild>
                      <Input
                        type="datetime-local"
                        value={startsAt ? toLocalInputValue(startsAt) : ''}
                        onChange={(e) => {
                          const v = fromLocalInputValue(e.target.value)
                          setStartsAt(v)
                          if (v) scheduleDebouncedSave('startsAt', v)
                        }}
                        onBlur={() => {
                          cancelDebouncedSave('startsAt')
                          if (startsAt) saveField('startsAt', startsAt)
                        }}
                      />
                    </PopoverAnchor>
                    <PopoverContent side="right" className="flex items-center gap-2 text-green-700">
                      <CheckCircle2 className="h-4 w-4" />
                      <span>Saved</span>
                    </PopoverContent>
                  </Popover>
                  {/* Removed inline saving indicator; popover handles saved feedback */}
                  {saveErrors['startsAt'] && (
                    <div className="text-xs text-red-600">{saveErrors['startsAt']}</div>
                  )}
                </div>
                <div className="space-y-2">
                  <label className="text-sm font-medium text-gray-700">Duration (minutes)</label>
                  <Popover open={!!saved['durationMinutes'] && !saving['durationMinutes'] && !saveErrors['durationMinutes']}>
                    <PopoverAnchor asChild>
                      <Input
                        type="number"
                        value={typeof durationMinutes === 'number' ? durationMinutes : ''}
                        onChange={(e) => {
                          const v = e.target.value === '' ? undefined : Number(e.target.value)
                          setDurationMinutes(v)
                          if (v !== undefined) scheduleDebouncedSave('durationMinutes', v)
                        }}
                        onBlur={() => {
                          cancelDebouncedSave('durationMinutes')
                          if (durationMinutes !== undefined) saveField('durationMinutes', durationMinutes)
                        }}
                      />
                    </PopoverAnchor>
                    <PopoverContent side="right" className="flex items-center gap-2 text-green-700">
                      <CheckCircle2 className="h-4 w-4" />
                      <span>Saved</span>
                    </PopoverContent>
                  </Popover>
                  {/* Removed inline saving indicator; popover handles saved feedback */}
                  {saveErrors['durationMinutes'] && (
                    <div className="text-xs text-red-600">{saveErrors['durationMinutes']}</div>
                  )}
                </div>
              </div>
            ) : (
              <div className="text-gray-800">
                <span className="font-medium">Start:</span>{' '}
                {game.startsAt ? new Date(game.startsAt).toLocaleString() : '—'}
                <span className="ml-4 font-medium">Duration:</span>{' '}
                {typeof game.durationMinutes === 'number' ? `${game.durationMinutes} min` : '—'}
              </div>
            )}
          </div>

          {/* Placeholder for future organizer controls */}
          <div className="border-t pt-6 text-sm text-gray-500">
            {isOrganizer ? 'Organizer actions will appear here.' : 'Join and participation controls will appear here.'}
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
