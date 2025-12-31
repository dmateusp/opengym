import { useEffect, useMemo, useRef, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { API_BASE_URL, redirectToLogin } from '@/lib/api'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Loader2, CheckCircle2, CircleDashed, Crown, Clock } from 'lucide-react'
import { Popover, PopoverAnchor, PopoverContent } from '@/components/ui/popover'
import { formatDistanceToNow, format } from 'date-fns'
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
    <Card className="bg-white shadow-lg">
      <CardHeader>
        <CardTitle>Your Games</CardTitle>
      </CardHeader>
      <CardContent>
        {/* Table Header */}
        <div className="grid grid-cols-12 gap-4 px-3 py-2 text-xs font-semibold text-gray-500 border-b">
          <div className="col-span-5">Name</div>
          <div className="col-span-2">Location</div>
          <div className="col-span-2">When</div>
          <div className="col-span-2">Status</div>
          <div className="col-span-1 text-right">Updated</div>
        </div>

        {/* Rows */}
        <div>
          {items.map((it) => {
            const status = getPublicationState(it)
            return (
              <button
                key={it.id}
                className="w-full grid grid-cols-12 gap-4 px-3 py-4 border-b hover:bg-gray-50 transition text-left"
                onClick={() => navigate(`/games/${it.id}`)}
              >
                <div className="col-span-5 flex items-center gap-2">
                  {it.isOrganizer && (
                    <div
                      onMouseEnter={() => setHoveredCrowId(it.id)}
                      onMouseLeave={() => setHoveredCrowId(null)}
                    >
                      <Popover open={hoveredCrowId === it.id}>
                        <PopoverAnchor asChild>
                          <span className="inline-flex items-center" aria-label="Organizer">
                            <Crown className="h-4 w-4 text-amber-500" aria-hidden="true" />
                          </span>
                        </PopoverAnchor>
                        <PopoverContent side="bottom" className="text-gray-800">
                          You are the organizer of this game.
                        </PopoverContent>
                      </Popover>
                    </div>
                  )}
                  <span className="text-gray-900 font-medium">{it.name}</span>
                </div>
                <div className="col-span-2 text-gray-600 truncate">
                  {it.location || '—'}
                </div>
                <div className="col-span-2 text-gray-600 text-xs">
                  {it.startsAt ? (
                    <TimeDisplay 
                      timestamp={it.startsAt} 
                      displayFormat="friendly"
                      className="text-gray-600 decoration-gray-400"
                    />
                  ) : (
                    '—'
                  )}
                </div>
                <div className="col-span-2">
                  {status.state === 'published' ? (
                    <span className="inline-flex items-center gap-1 text-xs px-2 py-0.5 rounded-full bg-green-100 text-green-700 border border-green-200">
                      <CheckCircle2 className="h-3 w-3" />
                      Published
                    </span>
                  ) : status.state === 'scheduled' ? (
                    <div className="space-y-1">
                      <span className="inline-flex items-center gap-1 text-xs px-2 py-0.5 rounded-full bg-blue-100 text-blue-700 border border-blue-200">
                        <Clock className="h-3 w-3" />
                        Scheduled
                      </span>
                      {status.timestamp && (
                        <div className="text-[10px] text-blue-700">
                          <TimeDisplay 
                            timestamp={status.timestamp}
                            displayFormat="relative"
                            prefix="Publishes"
                            className="text-blue-700 decoration-blue-400"
                          />
                        </div>
                      )}
                    </div>
                  ) : (
                    <span className="inline-flex items-center gap-1 text-xs px-2 py-0.5 rounded-full bg-gray-100 text-gray-700 border border-gray-200">
                      <CircleDashed className="h-3 w-3" />
                      Draft
                    </span>
                  )}
                </div>
                <div className="col-span-1 text-right text-gray-500 text-xs">
                  <TimeDisplay 
                    timestamp={it.updatedAt} 
                    displayFormat="relative"
                    className="text-gray-500 decoration-gray-400"
                  />
                </div>
              </button>
            )
          })}
        </div>

        {/* Loading More */}
        <div ref={sentinelRef} className="flex justify-center py-4">
          {isLoading && (
            <Loader2 className="h-5 w-5 animate-spin text-indigo-600" />
          )}
        </div>
      </CardContent>
    </Card>
  )
}
