interface PriceDisplayProps {
  totalPriceCents?: number
  maxPlayers?: number
  className?: string
}

function formatNumber(n?: number) {
  if (typeof n !== 'number' || Number.isNaN(n)) return '—'
  return String(n)
}

export function PriceDisplay({ totalPriceCents, maxPlayers, className }: PriceDisplayProps) {
  const total = typeof totalPriceCents === 'number' ? totalPriceCents : undefined
  const players = typeof maxPlayers === 'number' ? maxPlayers : undefined

  let perPlayer: number | undefined
  if (typeof total === 'number' && typeof players === 'number' && players > 0) {
    perPlayer = Math.round(total / players)
  }

  return (
    <div className={className}>
      <span className="font-medium text-gray-900">
        {perPlayer !== undefined ? formatNumber(perPlayer) : 'n/a'}
      </span>
      <span className="text-gray-500">{' '}({formatNumber(total)} total)</span>
    </div>
  )
}
