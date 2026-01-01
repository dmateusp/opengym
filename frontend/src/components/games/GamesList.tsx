import { useEffect, useMemo, useRef, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { API_BASE_URL, redirectToLogin } from '@/lib/api'
import { Card } from '@/components/ui/card'
import { Loader2, CheckCircle2, CircleDashed, Crown, Clock, MapPin } from 'lucide-react'
import { Popover, PopoverAnchor, PopoverContent } from '@/components/ui/popover'
import { TimeDisplay } from '@/components/ui/TimeDisplay'

interface GameListItem {
  id: string
  name: string
  isOrganizer: boolean
  location?: string
  startsAt?: string | null
  publishedAt?: string | null
  updatedAt: string
}

interface GameListResponse {
  total: number
  page: number
  pageSize: number
  items: GameListItem[]
}

export default function GamesList() {
  const navigate = useNavigate()
  const [items, setItems] = useState<GameListItem[]>([])
  const [total, setTotal] = useState<number>(0)
  const [page, setPage] = useState<number>(1)
  const [pageSize] = useState<number>(10)
  const [isLoading, setIsLoading] = useState<boolean>(false)
  const [error, setError] = useState<string | null>(null)
  const sentinelRef = useRef<HTMLDivElement | null>(null)
  const hasMore = useMemo(() => items.length < total, [items.length, total])
  const [hoveredCrowId, setHoveredCrowId] = useState<string | null>(null)
  const [nowTs, setNowTs] = useState<number>(() => Date.now())

  async function fetchPage(p: number) {
    try {
      setIsLoading(true)
      setError(null)
      const params = new URLSearchParams()
      params.set('page', String(p))
      params.set('pageSize', String(pageSize))
      const resp = await fetch(`${API_BASE_URL}/api/games?${params.toString()}`, {
        credentials: 'include',
      })
      if (!resp.ok) {
        if (resp.status === 401) {
          redirectToLogin()
          return
        }
        const txt = await resp.text()
        throw new Error(txt || 'Failed to load games')
      }
      const data: GameListResponse = await resp.json()
      setTotal(data.total)
      setPage(data.page)
      setItems(prev => (p === 1 ? data.items : [...prev, ...data.items]))
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Failed to load games')
    } finally {
      setIsLoading(false)
    }
  }

  useEffect(() => {
    fetchPage(1)
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [])

  useEffect(() => {
    const timer = window.setInterval(() => setNowTs(Date.now()), 30000)
    return () => window.clearInterval(timer)
  }, [])

  useEffect(() => {
    if (!sentinelRef.current) return
    const el = sentinelRef.current
    const observer = new IntersectionObserver((entries) => {
      for (const entry of entries) {
        if (entry.isIntersecting && !isLoading && hasMore) {
          const next = page + 1
          fetchPage(next)
        }
      }
    }, { rootMargin: '300px' })
    observer.observe(el)
    return () => observer.disconnect()
  }, [hasMore, isLoading, page])

  if (error) {
    return (
      <div className="bg-red-50 border border-red-200 rounded-lg p-6 text-center">
        <p className="text-red-700">{error}</p>
      </div>
    )
  }

  if (!isLoading && items.length === 0) {
    // Let the parent render its empty state; return null to avoid layout noise
    return null
  }

  function getPublicationState(item: GameListItem) {
    if (!item.publishedAt) return { state: 'draft' as const }
    const date = new Date(item.publishedAt)
    if (Number.isNaN(date.getTime())) return { state: 'draft' as const }
    if (date.getTime() > nowTs) {
      return { state: 'scheduled' as const, timestamp: item.publishedAt }
    }
    return { state: 'published' as const, timestamp: item.publishedAt }
  }

  return (
    <div className="space-y-4">
      {items.map((it) => {
        const status = getPublicationState(it)
        
        return (
          <button
            key={it.id}
            onClick={() => navigate(`/games/${it.id}`)}
            className="block w-full text-left transform transition-all hover:scale-102 active:scale-98"
          >
            <Card className="p-6 hover:shadow-2xl cursor-pointer border-l-4 border-l-primary">
              <div className="flex items-start justify-between gap-4">
                <div className="flex-1">
                  <div className="flex items-center gap-3 mb-2">
                    {it.isOrganizer && (
                      <div
                        onMouseEnter={() => setHoveredCrowId(it.id)}
                        onMouseLeave={() => setHoveredCrowId(null)}
                      >
                        <Popover open={hoveredCrowId === it.id}>
                          <PopoverAnchor asChild>
                            <span className="inline-flex items-center bg-secondary/20 rounded-full p-1" aria-label="Organizer">
                              <Crown className="h-4 w-4 text-primary" aria-hidden="true" />
                            </span>
                          </PopoverAnchor>
                          <PopoverContent side="bottom" className="text-gray-800 text-sm rounded-xl">
                            You are organizing this game
                          </PopoverContent>
                        </Popover>
                      </div>
                    )}
                    <h3 className="text-xl font-bold text-gray-900">{it.name}</h3>
                  </div>
                  
                  <div className="grid grid-cols-2 gap-4 text-sm text-gray-600 mt-3">
                    {it.location && (
                      <div className="flex items-center gap-2">
                        <MapPin className="h-4 w-4 text-gray-400" />
                        <span>{it.location}</span>
                      </div>
                    )}
                    {it.startsAt && (
                      <div className="flex items-center gap-2">
                        <Clock className="h-4 w-4 text-gray-400" />
                        <TimeDisplay 
                          timestamp={it.startsAt} 
                          displayFormat="friendly"
                          className="text-gray-600"
                        />
                      </div>
                    )}
                  </div>
                </div>

                <div className="flex flex-col items-end gap-3">
                  {status.state === 'published' ? (
                    <span className="inline-flex items-center gap-2 text-sm font-semibold px-4 py-2 rounded-full bg-success text-white shadow-md">
                      <CheckCircle2 className="h-4 w-4" />
                      Published
                    </span>
                  ) : status.state === 'scheduled' ? (
                    <div className="space-y-2 text-right">
                      <span className="inline-flex items-center gap-2 text-sm font-semibold px-4 py-2 rounded-full bg-secondary text-secondary-foreground shadow-md">
                        <Clock className="h-4 w-4" />
                        Scheduled
                      </span>
                      {status.timestamp && (
                        <div className="text-xs text-gray-500">
                          <TimeDisplay 
                            timestamp={status.timestamp}
                            displayFormat="relative"
                            prefix="In"
                            className="text-gray-500"
                          />
                        </div>
                      )}
                    </div>
                  ) : (
                    <span className="inline-flex items-center gap-2 text-sm font-semibold px-4 py-2 rounded-full bg-gray-200 text-gray-700 shadow-md">
                      <CircleDashed className="h-4 w-4" />
                      Draft
                    </span>
                  )}
                  
                  <div className="text-xs text-gray-400">
                    Updated <TimeDisplay 
                      timestamp={it.updatedAt} 
                      displayFormat="relative"
                      className="text-gray-400"
                    />
                  </div>
                </div>
              </div>
            </Card>
          </button>
        )
      })}

      {/* Loading More */}
      <div ref={sentinelRef} className="flex justify-center py-4">
        {isLoading && (
          <Loader2 className="h-5 w-5 animate-spin text-primary" />
        )}
      </div>
    </div>
  )
}
