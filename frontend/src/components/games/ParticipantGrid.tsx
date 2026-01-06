import { Crown } from 'lucide-react'
import type { LucideIcon } from 'lucide-react'

interface User {
  id: string
  email: string
  name?: string
  picture?: string
}

interface ParticipantGridProps {
  participants: User[]
  organizerId?: number
  maxCount?: number
  icon?: LucideIcon
  size?: 'sm' | 'md' | 'lg'
  opacity?: number
  emptySlotLabel?: string
}

const sizeConfig = {
  sm: {
    avatar: 'w-12 h-12',
    crown: 'h-2.5 w-2.5',
    crownPadding: 'p-1',
    text: 'text-[10px] max-w-12',
    emptyIcon: 'text-xs',
  },
  md: {
    avatar: 'w-16 h-16',
    crown: 'h-3 w-3',
    crownPadding: 'p-1.5',
    text: 'text-xs max-w-16',
    emptyIcon: 'text-sm',
  },
  lg: {
    avatar: 'w-20 h-20',
    crown: 'h-4 w-4',
    crownPadding: 'p-2',
    text: 'text-sm max-w-20',
    emptyIcon: 'text-base',
  },
}

export function ParticipantGrid({
  participants,
  organizerId,
  maxCount,
  icon: Icon,
  size = 'md',
  opacity = 1,
  emptySlotLabel = 'Open',
}: ParticipantGridProps) {
  const getInitials = (name?: string, email?: string) => {
    if (name) {
      const parts = name.split(' ').filter((p) => p.length > 0)
      if (parts.length > 0) {
        return parts
          .slice(0, 3)
          .map((p) => p[0].toUpperCase())
          .join('')
      }
      return name.slice(0, 2).toUpperCase()
    }
    if (email) {
      return email.slice(0, 2).toUpperCase()
    }
    return '??'
  }

  const config = sizeConfig[size]
  const containerStyle = opacity < 1 ? { opacity } : undefined

  return (
    <div className="flex flex-wrap gap-4" style={containerStyle}>
      {participants.map((participant) => {
        const isOrganizerParticipating =
          organizerId && participant.id === String(organizerId)
        return (
          <div
            key={participant.id}
            className="flex flex-col items-center gap-2"
          >
            <div className={`relative ${config.avatar}`}>
              <div
                className={`${config.avatar} rounded-full overflow-hidden ring-4 ring-success/30 shadow-md`}
              >
                {participant.picture ? (
                  <img
                    src={participant.picture}
                    alt={participant.name || participant.email}
                    className="w-full h-full object-cover"
                  />
                ) : (
                  <div className="w-full h-full bg-primary text-white flex items-center justify-center font-bold">
                    {getInitials(participant.name, participant.email)}
                  </div>
                )}
              </div>
              {isOrganizerParticipating && Icon && (
                <div
                  className={`absolute -bottom-1 -right-1 bg-primary rounded-full ${config.crownPadding} shadow-lg border-2 border-white`}
                >
                  <Icon className={`${config.crown} text-white`} />
                </div>
              )}
              {isOrganizerParticipating && !Icon && (
                <div
                  className={`absolute -bottom-1 -right-1 bg-primary rounded-full ${config.crownPadding} shadow-lg border-2 border-white`}
                >
                  <Crown className={`${config.crown} text-white`} />
                </div>
              )}
            </div>
            <span
              className={`${config.text} text-center text-gray-700 font-medium truncate`}
            >
              {participant.name || participant.email}
            </span>
          </div>
        )
      })}
      {/* Empty slots */}
      {maxCount &&
        maxCount > 0 &&
        Array.from({
          length: Math.max(0, maxCount - participants.length),
        }).map((_, i) => (
          <div
            key={`empty-${i}`}
            className="flex flex-col items-center gap-2"
          >
            <div
              className={`${config.avatar} rounded-full border-4 border-dashed border-gray-300 bg-gray-50 flex items-center justify-center`}
            >
              <span className={`text-gray-400 ${config.emptyIcon}`}>?</span>
            </div>
            <span
              className={`${config.text} text-center text-gray-400 font-medium truncate`}
            >
              {emptySlotLabel}
            </span>
          </div>
        ))}
    </div>
  )
}
